package javkit

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/imroc/req"
	"github.com/thoas/go-funk"
	"gopkg.in/ini.v1"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	arzonBaseUrl       = "https://www.arzon.jp"
	arzonUrl           = "https://www.arzon.jp/index.php"
	arzonSearchBaseUrl = "https://www.arzon.jp/itemlist.html?t=&m=all&s=&q="
	cutPercent         = 0.52625
)

func CreateSymlink(path, newFolderPath string, info JavInfo) {
	for _, actress := range info.AllActress {
		folderName := filepath.Base(newFolderPath)
		actressPath := filepath.Join(path, actress)
		if !Exists(actressPath) {
			os.Mkdir(actressPath, 0776)
		}
		symlinkPath := filepath.Join(actressPath, folderName)
		os.Symlink(newFolderPath, symlinkPath)
	}
}

// CreateNfo	创建相应影片的 nfo 信息
func CreateNfo(path string, javinfo JavInfo, config Config) {
	newName := renameVideo(javinfo, config)
	nfoFile, _ := os.Create(filepath.Join(path, newName) + ".nfo")
	defer nfoFile.Close()

	var customTitle string
	titleRules := strings.Split(config.CustomTitle, "+")
	for _, rule := range titleRules {
		if rule == "车牌" {
			customTitle += javinfo.License
		} else if rule == "空格" {
			customTitle += " "
		} else if rule == "标题" {
			customTitle += javinfo.Title
		} else if rule == "完整标题" {
			customTitle += javinfo.FullTitle
		}
	}

	scoreNum, _ := strconv.Atoi(javinfo.Score)
	criticrating := string(scoreNum * 10)

	var buffer bytes.Buffer
	buffer.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?>\n")
	buffer.WriteString("<movie>\n")
	buffer.WriteString("  <plot>" + javinfo.Introduction + javinfo.Review + "</plot>\n")
	buffer.WriteString("  <title>" + customTitle + "</title>\n")
	buffer.WriteString("  <director>" + javinfo.Director + "</director>\n")
	buffer.WriteString("  <rating>" + javinfo.Score + "</rating>\n")
	buffer.WriteString("  <criticrating>" + criticrating + "</criticrating>\n")
	buffer.WriteString("  <year>" + javinfo.Release.Year + "</year>\n")
	buffer.WriteString("  <mpaa>NC-17</mpaa>\n")
	buffer.WriteString("  <customrating>NC-17</customrating>\n")
	buffer.WriteString("  <countrycode>JP</countrycode>\n")
	buffer.WriteString("  <premiered>" + javinfo.Release.FullDate + "</premiered>\n")
	buffer.WriteString("  <release>" + javinfo.Release.FullDate + "</release>\n")
	buffer.WriteString("  <runtime>" + strconv.Itoa(javinfo.Length) + "</runtime>\n")
	buffer.WriteString("  <country>日本</country>\n")
	buffer.WriteString("  <studio>" + javinfo.Studio + "</studio>\n")
	buffer.WriteString("  <id>" + javinfo.License + "</id>\n")
	buffer.WriteString("  <num>" + javinfo.License + "</num>\n")
	for _, genre := range javinfo.Genres {
		buffer.WriteString("  <genre>" + genre + "</genre>\n")
	}
	for _, genre := range javinfo.Genres {
		buffer.WriteString("  <tag>" + genre + "</tag>\n")
	}
	for _, actress := range javinfo.AllActress {
		buffer.WriteString("  <actor>\n    <name>" + actress + "</name>\n    <type>Actor</type>\n  </actor>\n")
	}
	buffer.WriteString("</movie>\n")

	buffer.WriteTo(nfoFile)

}

// RenameAndMoveVideo 重命名并移动影片到新的文件夹
func RenameAndMoveVideo(file JavFile, info JavInfo, config Config, path string) (string, error) {
	prefix := filepath.Ext(file.Path)
	newName := renameVideo(info, config) + prefix
	newPath := filepath.Join(path, newName)
	err := os.Rename(file.Path, newPath)
	if err != nil {
		return "", err
	}
	return newPath, nil
}

// CreateNewFolder	对每个 Jav 创建单独的文件夹
func CreateNewFolder(file JavFile, info JavInfo, config Config) string {
	var newFolder string
	renameRules := strings.Split(config.RenameFolder, "+")
	for _, rule := range renameRules {
		switch rule {
		case "车牌":
			newFolder += info.License
		case "[", "]":
			newFolder += rule
		case "全部女优":
			newFolder += strings.Join(info.AllActress, " ")
		}
	}

	var basePath string
	if config.ClassifyRoot != "" {
		basePath = config.ClassifyRoot
	} else {
		basePath = filepath.Dir(file.Path)
	}

	newFolderPath := filepath.Join(basePath, info.LicensePrefix, newFolder)

	os.Mkdir(newFolderPath, 0776)

	return newFolderPath
}

// renameVideo	根据配置对影片重命名
func renameVideo(info JavInfo, config Config) string {
	var name string

	renameRules := strings.Split(config.RenameMP4, "+")
	for _, rule := range renameRules {
		switch rule {
		case "车牌":
			name += info.License + " "
		case "女优":
			name += info.FirstActress + " "
		case "全部女优":
			name += strings.Join(info.AllActress, " ") + " "
		}
	}
	name = strings.Trim(name, " ")

	return name
}

// GetJavInfo	获取影片信息
func GetJavInfo(url string, config Config, r *req.Req) (JavInfo, error) {
	javInfo := CreateDefaultJavInfo()
	javlibraryhtml, err := getJavLibraryHtml(url, config)
	if err != nil {
		return javInfo, err
	}
	javLibrary := string(javlibraryhtml)
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(javLibrary))

	// TODO: 有多个页面待选择的处理
	title := doc.Find("title").Text()

	if JavLibraryCatchError(title) {
		log.Println(url, " 查询 JavLibrary 失败，等待 5 秒后继续")
		time.Sleep(time.Second * 5)
		javlibraryhtml, err = getJavLibraryHtml(url, config)
		if err != nil {
			return javInfo, err
		}
		javLibrary = string(javlibraryhtml)
		doc, _ = goquery.NewDocumentFromReader(strings.NewReader(javLibrary))

		title = doc.Find("title").Text()
	}

	if JavLibraryCatchError(title) {
		return javInfo, errors.New(url + " 获取失败，请稍后手动重试")
	}

	getTitleAndLicense(title, &javInfo, config)

	if sutdio := doc.Find("div#video_maker table tr td.text a").Text(); sutdio != "" {
		javInfo.Studio = sutdio
	}

	if releaseDate := doc.Find("div#video_date table tr td.text").Text(); releaseDate != "" {
		dateSlice := strings.Split(releaseDate, "-")
		year := dateSlice[0]
		month := dateSlice[1]
		day := dateSlice[2]
		javInfo.Release.Year = year
		javInfo.Release.Month = month
		javInfo.Release.Day = day
		javInfo.Release.FullDate = releaseDate
	}

	if videoLength := doc.Find("div#video_length table tr td.text").Text(); videoLength != "" {
		length, err := strconv.Atoi(videoLength)
		if err == nil {
			javInfo.Length = length
		}
	}

	if director := doc.Find("div#video_director table tr td.text").Text(); director != "" && director[0] != '-' {
		javInfo.Director = director
	}

	actresses := []string{}
	doc.Find("div#video_cast table tr td.text span.star a").Each(func(i int, selection *goquery.Selection) {
		actresses = append(actresses, selection.Text())
	})
	javInfo.FirstActress = actresses[0]
	javInfo.AllActress = actresses

	genres := []string{}
	doc.Find("div#video_genres table tr td.text span.genre a").Each(func(i int, selection *goquery.Selection) {
		genres = append(genres, selection.Text())
	})
	javInfo.Genres = genres
	// TODO: 中文字幕检测

	if coverUrl, exist := doc.Find("div#video_jacket img").Attr("src"); exist {
		javInfo.CoverUrl = "https:" + coverUrl
	}

	if score := doc.Find("div#video_review table tr td.text span.score").Text(); score != "" {
		score = score[1 : len(score)-1]
		javInfo.Score = score
	}

	reviews := []string{}
	doc.Find("div#video_reviews table.review td.t textarea.hidden").Each(func(i int, selection *goquery.Selection) {
		reviews = append(reviews, selection.Text())
	})
	plotReview := "\n[精彩影评]：" + strings.Join(reviews, "\n")
	javInfo.Review = plotReview

	var introduction string
	if config.IfNfo == "是" && config.IfPlot == "是" {
		arzonSearchUrl := arzonSearchBaseUrl + javInfo.License
		introduction, err = getArzonIntroduction(arzonSearchUrl, r, config, &javInfo)
		if err == nil {
			javInfo.Introduction = introduction
		}
	}

	return javInfo, nil
}

// getArzonIntroduction	获取 arzon 的作品介绍
func getArzonIntroduction(url string, request *req.Req, config Config, info *JavInfo) (string, error) {
	searchHtml, err := GetArzonHtml(url, request, &config)
	if err != nil {
		request, err = GetArzonCookie(&config)
		if err != nil {
			return "", err
		}
		searchHtml, err = GetArzonHtml(url, request, &config)
	}
	if searchHtml != "" && err == nil {
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(searchHtml))

		var introduction string
		doc.Find("div#item div.pictlist dl.hentry dd.entry-title h2 a").Each(func(i int, selection *goquery.Selection) {
			urlSuffix, _ := selection.Attr("href")
			detailUrl := arzonBaseUrl + urlSuffix
			introduction, err = cycleSearchIntroduction(detailUrl, request, config)
		})

		return introduction, err

	}
	return "", err
}

// cycleSearchIntroduction	循环查询每个详情页的介绍
func cycleSearchIntroduction(url string, request *req.Req, config Config) (string, error) {
	detailHtml, err := GetArzonHtml(url, request, &config)
	if err != nil {
		request, err = GetArzonCookie(&config)
		if err != nil {
			return "", err
		}
		detailHtml, err = GetArzonHtml(url, request, &config)
	}
	if detailHtml != "" && err == nil {

		detail, _ := goquery.NewDocumentFromReader(strings.NewReader(detailHtml))
		introduction := detail.Find("div#detail_new table tr td tr td.text div.item_text").Text()

		return introduction, nil

	}
	return "", err
}

// getTitleAndLicense	对标题和车牌进行获取和清理
func getTitleAndLicense(originalTitle string, info *JavInfo, config Config) {
	originalTitle = strings.ReplaceAll(originalTitle, " - JAVLibrary", "")
	originalTitle = strings.ReplaceAll(originalTitle, "\n", "")
	originalTitle = strings.ReplaceAll(originalTitle, "&", "和")
	originalTitle = strings.ReplaceAll(originalTitle, "\\", "#")
	originalTitle = strings.ReplaceAll(originalTitle, "/", "#")
	originalTitle = strings.ReplaceAll(originalTitle, ":", "：")
	originalTitle = strings.ReplaceAll(originalTitle, "*", "#")
	originalTitle = strings.ReplaceAll(originalTitle, "?", "？")
	originalTitle = strings.ReplaceAll(originalTitle, "\"", "#")
	originalTitle = strings.ReplaceAll(originalTitle, "<", "[")
	originalTitle = strings.ReplaceAll(originalTitle, ">", "]")
	originalTitle = strings.ReplaceAll(originalTitle, "|", "#")
	originalTitle = strings.ReplaceAll(originalTitle, "＜", "[")
	originalTitle = strings.ReplaceAll(originalTitle, "＞", "]")
	originalTitle = strings.ReplaceAll(originalTitle, "〈", "[")
	originalTitle = strings.ReplaceAll(originalTitle, "〉", "]")
	originalTitle = strings.ReplaceAll(originalTitle, ".", "。")
	title := strings.ReplaceAll(originalTitle, "＆", "和")

	var subTitle string
	changeTitle := []rune(title)
	if len(changeTitle) > config.TitleLen {
		subTitle = string(changeTitle[:config.TitleLen])
	} else {
		subTitle = title
	}
	info.FullTitle = subTitle

	titleSplit := regexp.MustCompile(`(.+?) (.+)`)
	titleList := titleSplit.FindStringSubmatch(title)
	info.Title = titleList[2]

	license := titleList[1]
	info.License = license
	prefix := strings.Split(license, "-")[0]
	info.LicensePrefix = prefix

}

// TODO: cloudflare 的防护突破
// getJavLibraryHtml	获取 javlibrary 页面信息
func getJavLibraryHtml(url string, config Config) ([]byte, error) {

	args := []string{config.Script, "--url=" + url}
	out, err := exec.Command("/Users/kushnee/.virtualenvs/default/bin/python", args...).Output()
	return out, err
}

// makeRequest 生成一个 10 秒超时、装配代理的空 request
func makeRequest(config *Config) *req.Req {
	request := req.New()
	request.SetTimeout(time.Second * 10)
	if config.IfProxy == "是" && config.Proxy != "" {
		request.SetProxyUrl(config.Proxy)
	}
	return request
}

// GetConfig 获取 ini 配置并根据类型转换为不同的 config
func GetConfig(configType string, path string) (Config, error) {
	var config Config

	configSource, err := ini.Load(path)
	if err != nil {
		return config, err
	}

	if configType == "javlibrary" {
		nfoConfig := configSource.Section("收集nfo")
		config.IfNfo = nfoConfig.Key("是否收集nfo？").String()
		config.IfExnfo = nfoConfig.Key("是否跳过已存在nfo的文件夹？").String()
		config.IfReview = nfoConfig.Key("是否收集javlibrary上的影评？").String()
		config.CustomTitle = nfoConfig.Key("nfo中title的格式").String()
		config.CustomSubtitle = nfoConfig.Key("是否中字的表现形式").String()

		renameConfig := configSource.Section("重命名影片")
		config.IfMP4 = renameConfig.Key("是否重命名影片？").String()
		config.RenameMP4 = renameConfig.Key("重命名影片的格式").String()

		folderConfig := configSource.Section("修改文件夹")
		config.IfFolder = folderConfig.Key("是否重命名或创建独立文件夹？").String()
		config.RenameFolder = folderConfig.Key("新文件夹的格式").String()

		archiveConfig := configSource.Section("归类影片")
		config.IfClassify = archiveConfig.Key("是否归类影片？").String()
		config.IgnoreParentErr = archiveConfig.Key("忽略上层路径报错").String()
		config.ClassifyRoot = archiveConfig.Key("归类的根目录").String()
		config.ClassifyBasis = archiveConfig.Key("归类的标准").String()
		config.CreateSymboliclink = archiveConfig.Key("创建软链接").String()
		config.SymboliclinkDirectory = archiveConfig.Key("软链接目录").String()

		posterConfig := configSource.Section("下载封面")
		config.IfJpg = posterConfig.Key("是否下载封面海报？").String()
		config.CustomFanart = posterConfig.Key("dvd封面的格式").String()
		config.CustomPoster = posterConfig.Key("海报的格式").String()

		kodiConfig := configSource.Section("kodi专用")
		config.IfSculpture = kodiConfig.Key("是否收集女优头像").String()

		proxyConfig := configSource.Section("代理")
		config.IfProxy = proxyConfig.Key("是否使用代理？").String()
		config.Proxy = proxyConfig.Key("代理IP及端口").String()

		transConfig := configSource.Section("百度翻译API")
		config.IfPlot = transConfig.Key("是否需要日语简介？").String()
		config.IfTran = transConfig.Key("是否翻译为中文？").String()
		config.TransId = transConfig.Key("APP ID").String()
		config.TransSk = transConfig.Key("密钥").String()

		otherConfig := configSource.Section("其他设置")
		config.SimpTrad = otherConfig.Key("简繁中文？").String()
		config.LibraryUrl = otherConfig.Key("javlibrary网址").String()
		config.BusUrl = otherConfig.Key("javbus网址").String()
		config.SurenPref = otherConfig.Key("素人车牌(若有新车牌请自行添加)").String()
		config.FileType = otherConfig.Key("扫描文件类型").String()
		config.TitleLen = otherConfig.Key("重命名中的标题长度（50~150）").MustInt()
		config.Script = otherConfig.Key("python脚本位置").String()

	}

	return config, nil
}

// GetArzonCookie	通过 arzon 的成人认证，并返回相应的 request
func GetArzonCookie(config *Config) (*req.Req, error) {
	request := req.New()
	request.SetTimeout(time.Second * 10)
	if config.IfProxy == "是" && config.Proxy != "" {
		err := request.SetProxyUrl(config.Proxy)
		if err != nil {
			return nil, err
		}
	}

	params := req.Param{
		"action":   "adult_customer_agecheck",
		"agecheck": 1,
		"redirect": "https://www.arzon.jp",
	}
	_, err := request.Post(arzonUrl, params)
	if err != nil {
		return nil, err
	}

	return request, nil

}

// GetArzonHtml	获取 arzon 页面信息
func GetArzonHtml(url string, arzonRequest *req.Req, config *Config) (string, error) {

	response, err := arzonRequest.Get(url)
	if err != nil {
		return "", err
	}

	result, err := response.ToString()
	if err != nil {
		return "", err
	}

	return result, nil

}

// DownloadPic	下载封面
func DownloadPic(errorTimes int, picUrl string, picPath string, config *Config) error {
	request := makeRequest(config)
	var finalError error
	for tryTimes := 0; tryTimes < errorTimes; tryTimes++ {
		response, err := request.Get(picUrl)
		if err != nil {
			finalError = err
			continue
		}

		err = response.ToFile(picPath)
		if err != nil {
			finalError = err
			continue
		}

		return nil
	}
	return finalError

}

// MakePoster	用来生成 poster，并且完成封面的重命名
func MakePoster(fanartPath string) error {

	poster, err := cutPoster(fanartPath)
	if err != nil {
		return err
	}

	posterPath := strings.Replace(fanartPath, "fanart", "poster", -1)
	posterFile, err := os.Create(posterPath)
	if err != nil {
		return err
	}
	defer posterFile.Close()

	suffix := filepath.Ext(posterPath)

	switch suffix {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(posterFile, poster, &jpeg.Options{Quality: 95})
	case ".png":
		err = png.Encode(posterFile, poster)
	}
	return err

}

// cutPoster	裁剪只有右边的 poster
func cutPoster(fanartPath string) (image.Image, error) {
	fanart, err := os.Open(fanartPath)
	if err != nil {
		return nil, err
	}
	defer fanart.Close()

	img, _, err := image.Decode(fanart)
	if err != nil {
		return nil, err
	}

	poster, err := cutPic(img)
	if err != nil {
		return nil, err
	}

	return poster, nil

}

// cutPic 实际的裁剪操作
func cutPic(src image.Image) (image.Image, error) {
	var smallPic image.Image
	leftBottom := src.Bounds().Min
	rightTop := src.Bounds().Max

	width := float64(src.Bounds().Size().X)
	newWidth := int(width * cutPercent)
	newLeftBotton := image.Point{
		X: newWidth,
		Y: leftBottom.Y,
	}

	switch t := src.(type) {
	case *image.YCbCr:
		smallPic = t.SubImage(image.Rect(newLeftBotton.X, newLeftBotton.Y, rightTop.X, rightTop.Y)).(*image.YCbCr)
	case *image.RGBA:
		smallPic = t.SubImage(image.Rect(newLeftBotton.X, newLeftBotton.Y, rightTop.X, rightTop.Y)).(*image.RGBA)
	case *image.NRGBA:
		smallPic = t.SubImage(image.Rect(newLeftBotton.X, newLeftBotton.Y, rightTop.X, rightTop.Y)).(*image.NRGBA)
	default:
		return nil, errors.New("图片解码失败")
	}

	return smallPic, nil
}

// GetJavFromFolder	从给定目录中查找影片，创建 Jav 对象
func GetJavFromFolder(path string, config Config) []JavFile {
	var javList []JavFile
	javDic := map[string]int{}
	typeList := strings.Split(config.FileType, "、")
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		fileInfo, _ := os.Stat(path)
		if err != nil {
			return err
		}
		isDir := fileInfo.IsDir()
		if !isDir {
			filename := filepath.Base(path)
			ext := filepath.Ext(filename)

			for _, videoType := range typeList {
				if ext == "."+videoType && filename[0] != '.' {
					license, err := GetVideoTitle(filepath.Base(filename))
					if err != nil {
						log.Println(filename, err, " 跳过")
						continue
					}
					if !funk.Contains(javDic, license) {
						javDic[license] = 1
					} else {
						javDic[license] += 1
					}
					javFile := JavFile{path, license, javDic[license]}
					javList = append(javList, javFile)
				}
			}

		}
		return nil
	})
	return javList
}
