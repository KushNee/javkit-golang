package javkit

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/imroc/req"
	"github.com/pelletier/go-toml"
	"github.com/thoas/go-funk"
	"gopkg.in/ini.v1"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

var yellow = color.New(color.FgYellow)

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
func CreateNfo(path string, javinfo JavInfo, config IniConfig) {
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

	var buffer bytes.Buffer
	buffer.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?>\n")
	buffer.WriteString("<movie>\n")
	buffer.WriteString("  <title>" + customTitle + "</title>\n")
	buffer.WriteString("  <director>" + javinfo.Director + "</director>\n")
	if len(javinfo.AllActress) > 0 {
		for _, actress := range javinfo.AllActress {
			buffer.WriteString("  <actor>\n    <name>" + actress + "</name>\n    <type>Actor</type>\n  </actor>\n")
		}
	}
	buffer.WriteString("  <year>" + javinfo.Release.Year + "</year>\n")
	buffer.WriteString("  <mpaa>NC-17</mpaa>\n")
	buffer.WriteString("  <customrating>NC-17</customrating>\n")
	buffer.WriteString("  <countrycode>JP</countrycode>\n")
	buffer.WriteString("  <premiered>" + javinfo.Release.FullDate + "</premiered>\n")
	buffer.WriteString("  <releasedate>" + javinfo.Release.FullDate + "</releasedate>\n")
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
	buffer.WriteString("</movie>\n")

	buffer.WriteTo(nfoFile)

}

// RenameAndMoveVideo 重命名并移动影片到新的文件夹
func RenameAndMoveVideo(file JavFile, info JavInfo, config IniConfig, path string, log func(messages ...string)) (string, error) {
	prefix := filepath.Ext(file.Path)
	newName := renameVideo(info, config) + prefix
	newPath := filepath.Join(path, newName)
	err := os.Rename(file.Path, newPath)
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "cross-device link") {
			yellow.Printf("%s -> %s 是一个跨卷移动操作，请耐心等待\n", file.Path, newPath)
			err = MoveFile(file.Path, newPath)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return newPath, nil
}

// CreateNewFolder	对每个 Jav 创建单独的文件夹
func CreateNewFolder(file JavFile, info JavInfo, config IniConfig) string {
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

	newBasePath := filepath.Dir(newFolderPath)
	if !Exists(newBasePath) {
		os.Mkdir(newBasePath, 0776)
	}

	os.Mkdir(newFolderPath, 0776)

	return newFolderPath
}

// renameVideo	根据配置对影片重命名
func renameVideo(info JavInfo, config IniConfig) string {
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

// TODO: 使用 arzon 获取所有信息
//func GetJavInfoByArzon(url string, config IniConfig, arzonRequest *req.Req) (JavInfo, error) {
//	javInfo := CreateDefaultJavInfo()
//
//	searchHtml, err := GetArzonHtml(url, arzonRequest, &config)
//	if err != nil {
//		arzonRequest, err = GetArzonCookie(&config)
//		if err != nil {
//			return javInfo, err
//		}
//		searchHtml, err = GetArzonHtml(url, arzonRequest, &config)
//	}
//	if searchHtml != "" && err == nil {
//		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(searchHtml))
//
//		detail,_:=doc.Find("div#item div.pictlist dl.hentry dd.entry-title h2 a").First().Attr("href")
//		detailUrl:=arzonBaseUrl+detail
//
//
//	}
//	return javInfo, err
//}

func getInfoFromArzon(url string, arzonRequest *req.Req, config IniConfig, info *JavInfo) error {
	detailHtml, err := GetArzonHtml(url, arzonRequest, &config)
	if err != nil {
		arzonRequest, err = GetArzonCookie(&config)
		if err != nil {
			return err
		}
		detailHtml, err = GetArzonHtml(url, arzonRequest, &config)
	}
	if detailHtml != "" && err == nil { // 获取相关信息
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(detailHtml))

		title := []rune(doc.Find("title").Text())
		prefix := []rune("Arzon： ")
		title = title[len(prefix):]
		changedTitle := string(title)
		info.Title = changedTitle

	}
	return err
}

// GetJavInfo	获取影片信息
func GetJavInfo(url string, config IniConfig, log func(messages ...string)) (JavInfo, error) {

	javInfo := CreateDefaultJavInfo()

	libraryError := new(error)

	getJavBusInfo(url, config, &javInfo, libraryError)

	if reflect.ValueOf(libraryError).String() != "" {
		return javInfo, *libraryError
	}

	return javInfo, nil
}

func getJavBusInfo(url string, config IniConfig, javInfo *JavInfo, busError *error) {

	request := makeRequest(&config)
	javBusHtml, err := request.Get(url)
	if err != nil {
		busError = &err
		return
	}
	javBusSearch, err := javBusHtml.ToString()
	if err != nil {
		busError = &err
		return
	}
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(javBusSearch))

	var javBus string
	singleUrl, exist := doc.Find("div#waterfall div#waterfall div.item a.movie-box").First().Attr("href")
	if exist {
		url = singleUrl
		javBusHtml, err := request.Get(url)
		if err != nil {
			busError = &err
			return
		}
		javBus, err = javBusHtml.ToString()
		if err != nil {
			busError = &err
			return
		}
		doc, _ = goquery.NewDocumentFromReader(strings.NewReader(javBus))
	} else {
		notFoundError := errors.New("此影片无法在 JavBus 中找到")
		busError = &notFoundError
		return
	}

	title := doc.Find("title").Text()
	getTitleAndLicense(title, javInfo, config)

	infoPList := doc.Find("div.col-md-3.info p")

	if studio := infoPList.Eq(4).Find("a").Text(); studio != "" {
		javInfo.Studio = studio
	}

	if releaseDate := infoPList.Eq(1).Last().Text(); releaseDate != "" {
		timeCompiler := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
		release := timeCompiler.FindString(releaseDate)
		releaseDate = strings.TrimSpace(release)
		dateSlice := strings.Split(releaseDate, "-")
		year := dateSlice[0]
		month := dateSlice[1]
		day := dateSlice[2]
		javInfo.Release.Year = year
		javInfo.Release.Month = month
		javInfo.Release.Day = day
		javInfo.Release.FullDate = releaseDate
	}

	if videoLength := infoPList.Eq(2).Last().Text(); videoLength != "" {
		regRule := regexp.MustCompile(`\d{2,3}`)
		number := regRule.FindString(videoLength)
		length, err := strconv.Atoi(number)
		if err == nil {
			javInfo.Length = length
		}
	}

	if director := infoPList.Eq(3).Find("a").Text(); director != "" {
		javInfo.Director = director
	}

	var actresses []string
	doc.Find("div#avatar-waterfall a.avatar-box").Each(func(i int, selection *goquery.Selection) {
		name := selection.Find("span").Text()
		actresses = append(actresses, name)
	})

	// 无演员时不添加
	if len(actresses) > 0 {
		javInfo.FirstActress = actresses[0]
		javInfo.AllActress = actresses
	}

	var genres []string
	infoPList.NextFiltered("p.header").Next().Find("span.genre").Each(func(i int, selection *goquery.Selection) {
		genres = append(genres, selection.Find("a").First().Text())
	})
	javInfo.Genres = genres

	if coverUrl, exist := doc.Find("div.col-md-9.screencap a").Attr("href"); exist {
		javInfo.CoverUrl = coverUrl
	}

}

func getJavLibraryInfo(url string, config IniConfig, log func(messages ...string), javInfo *JavInfo, libraryError *error) {
	javlibraryhtml, err := getJavLibraryHtml(url, config, log)
	if err != nil {
		libraryError = &err
		return
	}
	javLibrary := string(javlibraryhtml)
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(javLibrary))

	title := doc.Find("title").Text()

	if strings.Contains(title, "识别码搜寻结果") {
		log("未进入详情页，尝试寻找第一个结果继续查找")
		singleUrl, exist := doc.Find("div#rightcolumn div.videothumblist div.videos div.video").First().Find("a").Attr("href")
		if exist {
			url = config.LibraryUrl + "cn/" + singleUrl[2:]
			javlibraryhtml, err = getJavLibraryHtml(url, config, log)
			if err != nil {
				libraryError = &err
				return
			}
			javLibrary = string(javlibraryhtml)
			doc, _ = goquery.NewDocumentFromReader(strings.NewReader(javLibrary))
			title = doc.Find("title").Text()
		} else {
			notFoundError := errors.New("此影片无法在 JavLibrary 中找到")
			libraryError = &notFoundError
			return

		}
	}

	//if JavLibraryCatchError(title) {
	//	log(url, " 查询 JavLibrary 失败，等待 5 秒后继续")
	//	time.Sleep(time.Second * 5)
	//	javlibraryhtml, err = getJavLibraryHtml(url, config, log)
	//	if err != nil {
	//		libraryError = &err
	//		return
	//	}
	//	javLibrary = string(javlibraryhtml)
	//	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(javLibrary))
	//
	//	title = doc.Find("title").Text()
	//}

	//if JavLibraryCatchError(title) {
	//	searchError := errors.New(url + " 获取失败，请稍后手动重试")
	//	libraryError = &searchError
	//	return
	//}

	getTitleAndLicense(title, javInfo, config)

	if sutdio := doc.Find("div#video_maker table tbody tr td.text a").Text(); sutdio != "" {
		javInfo.Studio = sutdio
	}

	if releaseDate := doc.Find("div#video_date table tbody tr td.text").Text(); releaseDate != "" {
		dateSlice := strings.Split(releaseDate, "-")
		year := dateSlice[0]
		month := dateSlice[1]
		day := dateSlice[2]
		javInfo.Release.Year = year
		javInfo.Release.Month = month
		javInfo.Release.Day = day
		javInfo.Release.FullDate = releaseDate
	}

	if videoLength := doc.Find("div#video_length table tbody tr td span.text").Text(); videoLength != "" {
		length, err := strconv.Atoi(videoLength)
		if err == nil {
			javInfo.Length = length
		}
	}

	if director := doc.Find("div#video_director table tbody tr td.text").Text(); director != "" && director[0] != '-' {
		javInfo.Director = director
	}

	actresses := []string{}
	doc.Find("div#video_cast table tbody tr td.text span.cast span.star a").Each(func(i int, selection *goquery.Selection) {
		actresses = append(actresses, selection.Text())
	})
	// 无演员时不添加
	if len(actresses) > 0 {
		javInfo.FirstActress = actresses[0]
		javInfo.AllActress = actresses
	}

	genres := []string{}
	doc.Find("div#video_genres table tbody tr td.text span.genre a").Each(func(i int, selection *goquery.Selection) {
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
}

func getArzonInfo(arzonSearchUrl string, arzonRequest *req.Req, config IniConfig, javInfo *JavInfo, done func()) {
	defer done()
	introduction, err := getArzonIntroduction(arzonSearchUrl, arzonRequest, config, javInfo)
	if err == nil {
		javInfo.Introduction = introduction
	}
}

// getArzonIntroduction	获取 arzon 的作品介绍
func getArzonIntroduction(url string, request *req.Req, config IniConfig, info *JavInfo) (string, error) {
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
func cycleSearchIntroduction(url string, request *req.Req, config IniConfig) (string, error) {
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
		introduction := detail.Find("div#detail_new table tbody tr td table.item_detail tbody tr:eq(1) td.text div.item_text").Text()

		return introduction, nil

	}
	return "", err
}

// getTitleAndLicense	对标题和车牌进行获取和清理
func getTitleAndLicense(originalTitle string, info *JavInfo, config IniConfig) {
	originalTitle = strings.ReplaceAll(originalTitle, " - JavBus", "")
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
func getJavLibraryHtml(url string, config IniConfig, log func(messages ...string)) ([]byte, error) {
	var interpreter string
	args := []string{config.Script, "--url=" + url}
	if "" == config.Interpreter {
		interpreter = "python"
	} else {
		interpreter = config.Interpreter
	}
	out, err := exec.Command(interpreter, args...).Output()
	return out, err
}

// makeRequest 生成一个 10 秒超时、装配代理的空 request
func makeRequest(config *IniConfig) *req.Req {
	request := req.New()
	request.SetTimeout(time.Second * 10)
	if config.IfProxy == "是" && config.Proxy != "" {
		_ = request.SetProxyUrl(config.Proxy)
	}
	return request
}

// GetConfig	根据配置后缀名使用不同工具加载并统一， **目前不可用**
func GetConfig(configType string, path string) (interface{}, error) {
	suffix := filepath.Ext(path)
	var config interface{}
	if suffix == ".ini" {
		config, _ = GetIniConfig(configType, path)
	} else if suffix == ".toml" {
		config, _ = getTomlConfig(configType, path)
	}

	switch realConfig := config.(type) {
	case IniConfig:
		fmt.Println(realConfig)
	case TomlConfig:
		fmt.Println(realConfig)
	}

	return config, nil
}

// getTomlConfig	获取 toml 配置并根据类型转换为不同的 config， **目前不可用**
func getTomlConfig(configType, path string) (TomlConfig, error) {
	var config TomlConfig

	if configType == "javlibrary" {
		file, err := os.Open(path)
		if err != nil {
			return config, err
		}
		defer file.Close()

		data, err := ioutil.ReadAll(file)
		if err != nil {
			return config, err
		}

		err = toml.Unmarshal(data, &config)
		if err != nil {
			return config, err
		}

		return config, nil
	} else {
		err := errors.New(configType + " 类型的配置暂时未支持")
		return config, err
	}

}

// GetIniConfig 获取 ini 配置并根据类型转换为不同的 config
func GetIniConfig(configType string, path string) (IniConfig, error) {
	var config IniConfig

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
		config.Interpreter = otherConfig.Key("python解释器位置").String()

	}

	return config, nil
}

// GetArzonCookie	通过 arzon 的成人认证，并返回相应的 request
func GetArzonCookie(config *IniConfig) (*req.Req, error) {
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
func GetArzonHtml(url string, arzonRequest *req.Req, config *IniConfig) (string, error) {

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

// SavePic	下载封面
func SavePic(picPath string, picData []byte) error {

	if len(picData) == 0 {
		return errors.New("未正确下载图片数据")
	}

	file, err := os.Create(picPath)
	if err != nil {
		return err
	}
	defer file.Close()

	picBuffer := bytes.NewBuffer(picData)
	_, err = picBuffer.WriteTo(file)
	if err != nil {
		return err
	}

	return nil
}

func DownloadPicAsync(errorTimes int, picUrl string, config *IniConfig, pPicData *[]byte, pDownloadErr *error, done func()) {
	defer done()

	request := makeRequest(config)
	var response *req.Resp
	var downloadErr error
	tryTimes := 0
	for ; tryTimes < errorTimes; tryTimes++ {
		resp, err := request.Get(picUrl)
		downloadErr = err
		if err == nil {
			response = resp
			break
		}
		time.Sleep(time.Second * 1)
		continue
	}
	if downloadErr != nil {
		*pDownloadErr = downloadErr
		pPicData = nil
	}
	data, err := response.ToBytes()
	if err != nil {
		*pDownloadErr = err
		*pPicData = nil
	}
	*pDownloadErr = nil
	*pPicData = data
}

func DownloadPic(errorTimes int, picUrl string, config *IniConfig) ([]byte, error) {
	request := makeRequest(config)
	var response *req.Resp
	var downloadErr error
	tryTimes := 0
	for ; tryTimes < errorTimes; tryTimes++ {
		resp, err := request.Get(picUrl)
		downloadErr = err
		if err == nil {
			response = resp
			break
		}
		time.Sleep(time.Second * 1)
		continue
	}
	if downloadErr != nil {
		return nil, downloadErr
	}
	data, err := response.ToBytes()
	if err != nil {
		return nil, err
	}
	return data, nil
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
func GetJavFromFolder(path string, config IniConfig) []JavFile {
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
						PrintWithTime(filename, err.Error(), " 跳过")
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
