package javkit

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

// GetVideoTitle	正则匹配找出番号
func GetVideoTitle(name string) (string,error) {
	titleRegexp := regexp.MustCompile(`([a-zA-Z]{2,6})-? ?(\d{2,5})`)
	titles := titleRegexp.FindStringSubmatch(name)
	if len(titles) == 0{
		return "",errors.New(" 不是影片")
	}
	licensePrefix := strings.ToUpper(titles[1])
	license := licensePrefix + "-" + titles[2]
	return license,nil
}

// 创建带默认值的 javinfo
func CreateDefaultJavInfo() JavInfo {
	javInfo := JavInfo{
		License:       "ABC-123",
		LicensePrefix: "ABC",
		Title:         "未知标题",
		FullTitle:     "完整标题",
		Director:      "未知导演",
		Release: JavReleaseDate{
			Year:     "1970",
			Month:    "01",
			Day:      "01",
			FullDate: "1970-01-01",
		},
		Studio:       "未知片商",
		Score:        "0",
		FirstActress: "未知演员",
		AllActress:   []string{"未知演员"},
		Length:       0,
		ChineseSub:   false,
		VideoName:    "ABC-123",
		CoverUrl:     "",
		Review:       "",
		Introduction: "",
		Genres:       []string{},
	}

	return javInfo
}

// Exists 判断路径是否存在
func Exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// JavLibraryCatchError	判断 javlibrary 页面是否抓取成功。目前使用 python 脚本完成 cloudflare challenge，以后可能去除
func JavLibraryCatchError(title string) bool {
	if strings.Contains(title, "404") || strings.Contains(title, "502") || strings.Contains(title, "503") {
		return true
	}else {
		return false
	}
}
