package main

import (
	"fmt"
	"github.com/KushNee/javkit-golang/javkit"
	"github.com/c-bata/go-prompt"
	"github.com/imroc/req"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {

	// 获取配置
	config, err := javkit.GetIniConfig("javlibrary", "self-config.ini")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			javkit.PrintWithTime("当前路径未找到配置文件，请提供配置路径。输入后回车")
			notFound := true
			for notFound {
				configPath := prompt.Input("path > ", javkit.PathCompleter)
				switch configPath {
				case "exit":
					return
				default:
					config, err = javkit.GetIniConfig("javlibrary", configPath)
					if err != nil {
						if strings.Contains(err.Error(), "no such file or directory") {
							javkit.PrintWithTime("未找到配置文件，请提供配置路径。输入后回车")
						} else {
							javkit.PrintWithTime("加载配置文件失败，原因：", err.Error())
						}
						continue
					}
					notFound = false
				}
			}

		}
	}

	javkit.PrintWithTime("加载配置成功")

	javkit.PrintWithTime("正在进行 arzon 成人认证。。。")

	// 获取 arzon cookie
	arzonRequest, err := javkit.GetArzonCookie(&config)
	if err != nil {
		log.Fatalln("无法完成 arzon 成人验证，请检查网络连接。原因：", err)
	}

	javkit.PrintWithTime("完成 arzon 成人认证。。。")

	fmt.Println("输入后回车选择路径")
	pathLoop := true
	for pathLoop {
		t := prompt.Input("path > ", javkit.PathCompleter)
		switch t {
		case "exit":
			return
		case "back":
			pathLoop = false
			continue
		}
		mainLogic(t, arzonRequest, config)

	}
}

func mainLogic(path string, arzonRequest *req.Req, config javkit.IniConfig) {

	javList := javkit.GetJavFromFolder(path, config)

	if len(javList) == 0 {
		javkit.PrintWithTime("目录内不存在影片")
		return
	}

	searchBaseUrl := config.LibraryUrl + "cn/" + "vl_searchbyid.php?keyword="
	var wg sync.WaitGroup
	wg.Add(len(javList))
	for _, jav := range javList {
		standPrint := func(messages ...string) {
			title := []string{"[", filepath.Base(jav.Path), "]", " "}
			messages = append(title, messages...)
			javkit.PrintWithTime(messages...)
		}
		go processJav(config, jav, searchBaseUrl, arzonRequest, wg.Done, standPrint)
	}
	wg.Wait()

}

func processJav(config javkit.IniConfig, jav javkit.JavFile, searchBaseUrl string, arzonRequest *req.Req, done func(), log func(messages ...string)) {
	defer done()
	searchUrl := searchBaseUrl + jav.License
	// 获取 jav 所有需要的信息
	javInfo, err := javkit.GetJavInfo(searchUrl, config, arzonRequest, log)
	if err != nil {
		log(jav.Path, " 获取信息失败，原因：", err.Error())
		log("可能与 Python 有关，请使用 Python3.7，并确保安装了所需依赖")
		return
	}

	var picDownloadWg sync.WaitGroup
	picDownloadWg.Add(1)
	var picData []byte
	var picError error

	if config.IfJpg == "是" {
		go javkit.DownloadPicAsync(10, javInfo.CoverUrl, &config, &picData, &picError, picDownloadWg.Done)
	} else {
		picDownloadWg.Done()
	}

	// 创建归类文件夹
	newFolderPath := javkit.CreateNewFolder(jav, javInfo, config)

	// 移动影片
	newVideoPath, err := javkit.RenameAndMoveVideo(jav, javInfo, config, newFolderPath, log)
	if err != nil {
		log(jav.Path, " 移动影片失败，原因：", err.Error())
		err = os.RemoveAll(newFolderPath)
		if err != nil {
			log("删除新建文件夹", newFolderPath, "失败，原因：", err.Error())
		}
		return
	}
	log(newVideoPath)

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

		picDownloadWg.Wait()

		if picError != nil && picError.Error() != "" {
			log("下载图片失败，原因：", picError.Error())
		}
		err = javkit.SavePic(fanartPath, picData)
		if err != nil {
			log(fanartPath, " 保存图片失败，原因：", err.Error())
		}

		err = javkit.MakePoster(fanartPath)
		if err != nil {
			log(fanartPath, " 生成海报失败，原因：", err.Error())
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
		log(oldPath, " 删除旧文件夹失败，原因：", err.Error())
		return
	}

	log(" 完成归类")
}
