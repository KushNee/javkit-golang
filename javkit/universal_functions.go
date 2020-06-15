package javkit

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/imroc/req"
	"github.com/pelletier/go-toml"
	"github.com/thoas/go-funk"
	"gopkg.in/ini.v1"
)

const (
	cutPercent = 0.52625
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
func CreateNfo(path string, javInfo JavInfo, config IniConfig) {
	newName := renameVideo(javInfo, config)
	nfoFile, _ := os.Create(filepath.Join(path, newName) + ".nfo")
	defer nfoFile.Close()

	var customTitle string
	titleRules := strings.Split(config.CustomTitle, "+")
	for _, rule := range titleRules {
		if rule == "车牌" {
			customTitle += javInfo.License
		} else if rule == "空格" {
			customTitle += " "
		} else if rule == "标题" {
			customTitle += javInfo.Title
		} else if rule == "完整标题" {
			customTitle += javInfo.FullTitle
		}
	}

	var buffer bytes.Buffer
	buffer.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?>\n")
	buffer.WriteString("<movie>\n")
	buffer.WriteString("  <title>" + customTitle + "</title>\n")
	buffer.WriteString("  <director>" + javInfo.Director + "</director>\n")
	if len(javInfo.AllActress) > 0 {
		for _, actress := range javInfo.AllActress {
			buffer.WriteString("  <actor>\n    <name>" + actress + "</name>\n    <type>Actor</type>\n  </actor>\n")
		}
	}
	buffer.WriteString("  <year>" + javInfo.Release.Year + "</year>\n")
	buffer.WriteString("  <mpaa>NC-17</mpaa>\n")
	buffer.WriteString("  <customrating>NC-17</customrating>\n")
	buffer.WriteString("  <countrycode>JP</countrycode>\n")
	buffer.WriteString("  <premiered>" + javInfo.Release.FullDate + "</premiered>\n")
	buffer.WriteString("  <releasedate>" + javInfo.Release.FullDate + "</releasedate>\n")
	buffer.WriteString("  <runtime>" + strconv.Itoa(javInfo.Length) + "</runtime>\n")
	buffer.WriteString("  <country>日本</country>\n")
	buffer.WriteString("  <studio>" + javInfo.Studio + "</studio>\n")
	buffer.WriteString("  <id>" + javInfo.License + "</id>\n")
	buffer.WriteString("  <num>" + javInfo.License + "</num>\n")
	for _, genre := range javInfo.Genres {
		buffer.WriteString("  <genre>" + genre + "</genre>\n")
	}
	for _, genre := range javInfo.Genres {
		buffer.WriteString("  <tag>" + genre + "</tag>\n")
	}
	buffer.WriteString("</movie>\n")

	buffer.WriteTo(nfoFile)

}

// RenameAndMoveVideo 重命名并移动影片到新的文件夹
func RenameAndMoveVideo(videoPath string, info JavInfo, config IniConfig, path string) (string, error) {
	prefix := filepath.Ext(videoPath)
	newName := renameVideo(info, config) + prefix
	newPath := filepath.Join(path, newName)
	err := os.Rename(videoPath, newPath)
	if err != nil {
		errString := err.Error()
		if strings.Contains(errString, "cross-device link") {
			yellow.Printf("%s -> %s 是一个跨卷移动操作，请耐心等待\n", videoPath, newPath)
			err = MoveFile(videoPath, newPath)
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
func CreateNewFolder(path string, info JavInfo, config IniConfig) string {
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
		basePath = filepath.Dir(path)
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

// GetJavInfo	获取影片信息
func GetJavInfo(url string, config IniConfig) (JavInfo, error) {

	javInfo := CreateDefaultJavInfo()

	err := getJavBusInfo(url, config, &javInfo)

	return javInfo, err
}

func getJavBusInfo(url string, config IniConfig, javInfo *JavInfo) error {

	request := makeRequest(&config)
	javBusHtml, err := request.Get(url)
	if err != nil {
		return err
	}
	javBusSearch, err := javBusHtml.ToString()
	if err != nil {
		return err
	}
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(javBusSearch))

	var javBus string
	singleUrl, exist := doc.Find("div#waterfall div#waterfall div.item a.movie-box").First().Attr("href")
	if exist {
		url = singleUrl
		javBusHtml, err := request.Get(url)
		if err != nil {
			return err
		}
		javBus, err = javBusHtml.ToString()
		if err != nil {
			return err
		}
		doc, _ = goquery.NewDocumentFromReader(strings.NewReader(javBus))
	} else {
		notFoundError := errors.New("此影片无法在 JavBus 中找到")
		return notFoundError
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

	return nil
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
		config.Proxy = proxyConfig.Key("代理ip及端口").String()

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
		*pPicData = nil
	}
	data, err := response.ToBytes()
	if err != nil {
		*pDownloadErr = err
		*pPicData = nil
	}
	*pDownloadErr = nil
	*pPicData = data
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

	return poster, err

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
					if strings.Contains(license, "-") {
						license = strings.Replace(license, "-", "", -1)
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
