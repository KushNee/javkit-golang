package main

import (
	"encoding/xml"
	"fmt"
	"github.com/KushNee/javkit-golang/javkit"
	"github.com/c-bata/go-prompt"
	"io"
	"io/ioutil"
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
			homePath, _ := os.UserHomeDir()
			config, err = javkit.GetIniConfig("javlibrary", homePath+"/.config/self-config.ini")
			if err != nil {
				javkit.PrintWithTime("未找到配置文件，请提供配置路径。输入后回车")
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
						javkit.PrintWithTime("加载配置成功")
						if "" == config.Interpreter {
							javkit.PrintWithTime("未提供 Python 解释器，将尝试调用默认 Python")
						}
						notFound = false
					}
				}
			}

		}
	}

	for {
		command := prompt.Input("command > ", javkit.CommandCompleter)
		switch command {
		case "manage":
			Manage(config)
		case "delete":
			Delete(config)
		case "exit":
			return
		}
	}

}

func emptyCompleter(d prompt.Document) []prompt.Suggest {
	input := d.TextBeforeCursor()
	commandList := []prompt.Suggest{
		{Text: "back", Description: "返回"},
	}

	return prompt.FilterContains(commandList, input, true)
}

func Delete(config javkit.IniConfig) {
	for {
		name := prompt.Input("name > ", emptyCompleter)
		switch name {
		case "back":
			return
		default:
			videoName, err := javkit.GetVideoTitle(name)
			if err != nil {
				fmt.Printf("番号不正确，%s\n", err.Error())
			}

			prefix := strings.Split(videoName, `-`)[0]

			folderPath := filepath.Join(config.ClassifyRoot, prefix)

			prefixPath, err := os.Open(folderPath)
			if err != nil {
				fmt.Printf("无法打开归档目录，%s\n", err.Error())
			}
			defer prefixPath.Close()

			const reqLimit = 1024

		FindFolder:
			for {
				folderList, err := prefixPath.Readdirnames(reqLimit)
				if err != nil {
					if io.EOF == err {
						break
					} else {
						fmt.Printf("读取文件失败，%s\n", err)
					}
				}
				for _, folderName := range folderList {
					if strings.Contains(folderName, videoName) {
						nfo, err := ioutil.ReadFile(filepath.Join(folderPath, folderName, videoName+".nfo"))
						if err != nil {
							fmt.Println(err)
						}

						nfoInfo := javkit.MovieInfo{}

						err = xml.Unmarshal(nfo, &nfoInfo)
						if err != nil {
							fmt.Printf("读取信息失败，%s\n", err)
						}

						for _, actor := range nfoInfo.ActorList {
							symRootPath := filepath.Join(filepath.Dir(config.ClassifyRoot), config.SymboliclinkDirectory)
							fullPath := filepath.Join(symRootPath, actor.Name, folderName)
							err = os.Remove(fullPath)
							if err != nil {
								fmt.Printf("删除软链接 %s 失败，%s\n", fullPath, err)
							}
							fmt.Printf("删除软链接 %s\n", fullPath)
						}

						videoPath := filepath.Join(folderPath, folderName)
						err = os.RemoveAll(videoPath)
						if err != nil {
							fmt.Printf("删除影片 %s 失败，%s\n", videoPath, err)
						}
						fmt.Printf("删除影片 %s\n", videoPath)

						break FindFolder
					}
				}
			}

		}
	}

}

func Manage(config javkit.IniConfig) {
	//javkit.PrintWithTime("正在进行 arzon 成人认证。。。")
	//
	//// 获取 arzon cookie
	//arzonRequest, err := javkit.GetArzonCookie(&config)
	//if err != nil {
	//	log.Fatalln("无法完成 arzon 成人验证，请检查网络连接。原因：", err)
	//}
	//
	//javkit.PrintWithTime("完成 arzon 成人认证。。。")

	fmt.Println("输入后回车选择路径")
	for {
		t := prompt.Input("path > ", javkit.PathCompleter)
		switch t {
		case "back":
			return
		}
		mainLogic(t, config)

	}
}

func mainLogic(path string, config javkit.IniConfig) {

	rootInfo, err := os.Stat(path)
	if err != nil {
		fmt.Println("读取路径失败：", err)
	}

	var deleteParent bool
	if rootInfo.IsDir() {
		deleteParent = true
	} else {
		deleteParent = false
	}

	javList := javkit.GetJavFromFolder(path, config)

	if len(javList) == 0 {
		javkit.PrintWithTime("目录内不存在影片")
		return
	}

	searchBaseUrl := "https://javbus.com/search/"
	var wg sync.WaitGroup
	wg.Add(len(javList))
	for _, jav := range javList {
		title := []string{"[", filepath.Base(jav.Path), "]", " "}
		standPrint := func(messages ...string) {
			messages = append(title, messages...)
			javkit.PrintWithTime(messages...)
		}
		go processJav(config, jav, searchBaseUrl, wg.Done, standPrint)
	}
	wg.Wait()

	javEmptyList := javkit.GetJavFromFolder(path, config)
	if len(javEmptyList) == 0 && deleteParent {
		// 删除旧文件夹
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Println(path, " 删除旧文件夹失败，原因：", err.Error())
			return
		}
		fmt.Println("删除 ", path)
	}

}

func processJav(config javkit.IniConfig, jav javkit.JavFile, searchBaseUrl string, done func(), log func(messages ...string)) {
	defer done()
	searchUrl := searchBaseUrl + jav.License
	// 获取 jav 所有需要的信息
	javInfo, err := javkit.GetJavInfo(searchUrl, config, log)
	if err != nil {
		log(jav.Path, " 获取信息失败，原因：", err.Error())
		log("可能与 Python 有关，请使用 Python3.7，并确保安装了所需依赖")
		return
	}

	if javInfo.License == "ABC-123" {
		log("无法获得影片信息，跳过")
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

	log(" 完成归类")
}
