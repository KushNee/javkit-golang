package main

import (
	"bufio"
	"fmt"
	"github.com/KushNee/javkit-golang/javkit"
	"github.com/imroc/req"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {

	var path string

	config, err := javkit.GetConfig("javlibrary")
	if err != nil {
		log.Fatalln("无法获取配置，原因：", err)
	}

	inputReader := bufio.NewReader(os.Stdin)
	fmt.Println("请输入路径：")
	// TODO: 选择路径
	path, err = inputReader.ReadString('\n')
	if err != nil {
		log.Fatalln("获取路径失败，原因：", err)
	}
	path = strings.Trim(path, "\n")

	javList := javkit.GetJavFromFolder(path, config)

	searchBaseUrl := config.LibraryUrl + "cn/" + "vl_searchbyid.php?keyword="
	arzonRequest, _ := javkit.GetArzonCookie(&config)
	var wg sync.WaitGroup
	wg.Add(len(javList))
	for _, jav := range javList {
		go processJav(config, jav, searchBaseUrl, arzonRequest,wg.Done)
	}
	wg.Wait()

}

func processJav(config javkit.Config, jav javkit.JavFile, searchBaseUrl string, arzonRequest *req.Req, done func()) {
	defer done()
	searchUrl := searchBaseUrl + jav.License
	// 获取 jav 所有需要的信息
	javInfo, err := javkit.GetJavInfo(searchUrl, config, arzonRequest)
	if err != nil {
		log.Println(jav.Path, " 获取信息失败，原因：", err)
		return
	}

	// 创建归类文件夹
	newFolderPath := javkit.CreateNewFolder(jav, javInfo, config)

	// 移动影片
	newVideoPath, err := javkit.RenameAndMoveVideo(jav, javInfo, config, newFolderPath)
	if err != nil {
		os.RemoveAll(newFolderPath)
		log.Println(jav.Path, " 移动影片失败，原因：", err)
		return
	}
	log.Println(newVideoPath)

	// 创建 nfo
	if config.IfNfo == "是" {
		javkit.CreateNfo(newFolderPath, javInfo, config)
	}

	// 下载图片
	if config.IfJpg == "是" {
		fanartRules := strings.Split(config.CustomFanart, "+")
		var fanartName string
		for _, rule := range fanartRules {
			switch rule {
			case "视频":
				videoName := strings.Split(filepath.Base(newVideoPath), ".")[0]
				fanartName += videoName
			case "-fanart.jpg":
				fanartName += rule
			}
		}
		fanartPath := filepath.Join(newFolderPath, fanartName)
		err = javkit.DownloadPic(10, javInfo.CoverUrl, fanartPath, &config)
		if err != nil {
			log.Println(fanartPath, " 下载图片失败，原因：", err)
			return
		}

		err = javkit.MakePoster(fanartPath)
		if err != nil {
			log.Println(fanartPath, " 生成海报失败，原因：", err)
			return
		}
	}

	// 创建软链接
	if config.CreateSymboliclink == "是" && config.SymboliclinkDirectory != "" {
		fullPath := filepath.Join(filepath.Dir(config.ClassifyRoot), config.SymboliclinkDirectory)
		javkit.CreateSymlink(fullPath, newFolderPath, javInfo)
	}

	// 删除旧文件夹
	oldPath := filepath.Dir(jav.Path)
	err = os.RemoveAll(oldPath)
	if err != nil {
		log.Println(oldPath, " 删除旧文件夹失败，原因：", err)
		return
	}
}
