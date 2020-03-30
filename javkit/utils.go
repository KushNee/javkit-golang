package javkit

import (
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// GetVideoTitle	正则匹配找出番号
func GetVideoTitle(name string) (string, error) {
	titleRegexp := regexp.MustCompile(`([a-zA-Z]{2,6})-? ?(\d{2,5})`)
	titles := titleRegexp.FindStringSubmatch(name)
	if len(titles) == 0 {
		return "", errors.New(" 不是影片")
	}
	licensePrefix := strings.ToUpper(titles[1])
	license := licensePrefix + "-" + titles[2]
	return license, nil
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
	_, err := os.Stat(path) //os.Stat获取文件信息
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
	} else {
		return false
	}
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("无法打开源文件: %s", err)
	}
	defer inputFile.Close()
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("无法创建目标文件: %s", err)
	}
	defer outputFile.Close()

	fileInfo, _ := os.Stat(sourcePath)
	processBar := pb.New(int(fileInfo.Size()))
	processBar.Set(pb.SIBytesPrefix, true)
	proxyReader := processBar.NewProxyReader(inputFile)
	_, err = io.Copy(outputFile, proxyReader)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("写入目标文件失败: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("删除源文件失败: %s", err)
	}
	return nil
}

func PrintWithTime(message ...string) {
	now := time.Now().Format("2006/01/02 15:04:05")
	fmt.Println(now, " ", message)
}
