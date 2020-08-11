package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/thoas/go-funk"

	"github.com/KushNee/javkit-golang/javkit"
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
			manage(config)
		case "remanage":
			reManage(config)
		case "delete":
			filmDelete(config)
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

func filmDelete(config javkit.IniConfig) {
	for {
		name := prompt.Input("name > ", emptyCompleter)
		switch name {
		case "back":
			return
		default:
			videoName, folderPath, prefixPath, err := findManagedFilm(config, name)
			if err != nil {
				prefixPath.Close()
				return
			}
			doDelete(config, prefixPath, videoName, folderPath)

		}
	}

}

func doDelete(config javkit.IniConfig, prefixPath *os.File, videoName string, folderPath string) {
	const reqLimit = 1024

FindFolder:
	for {
		folderList, err := prefixPath.Readdirnames(reqLimit)
		if err != nil {
			if io.EOF == err {
				break
			} else {
				fmt.Printf("读取文件失败，%s\n", err)
				return
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

func manage(config javkit.IniConfig) {

	fmt.Println("输入后回车选择路径")
	waitTime := 5
	for {
		t := prompt.Input("path > ", javkit.PathCompleter)
		switch t {
		case "back":
			return
		}
		remainNumber, deleteParent := doManage(t, config)

		if remainNumber == 0 {
			if deleteParent {
				// 删除旧文件夹
				err := os.RemoveAll(t)
				if err != nil {
					fmt.Println(t, " 删除旧文件夹失败，原因：", err.Error())
					return
				}
				fmt.Println("删除 ", t)
			}
		} else {
			fmt.Printf("有 %d 部影片未归类完成，等待 %d 秒后重试\n", remainNumber, waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
			remainNumber, deleteParent = doManage(t, config)
			if remainNumber == 0 {
				if deleteParent {
					// 删除旧文件夹
					err := os.RemoveAll(t)
					if err != nil {
						fmt.Println(t, " 删除旧文件夹失败，原因：", err.Error())
						return
					}
					fmt.Println("删除 ", t)
				}
			} else {
				fmt.Printf("存在 %d 部影片未完成归类\n", remainNumber)
			}
		}

	}
}

func doManage(path string, config javkit.IniConfig) (int, bool) {

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
		fmt.Println("目录内不存在影片")
		return 0, false
	} else {
		fmt.Printf("共找到 %d 部影片\n\n", len(javList))
	}

	var busUrl string
	if strings.HasSuffix(config.BusUrl, "/") {
		busUrl = config.BusUrl
	} else {
		busUrl = config.BusUrl + "/"
	}
	searchBaseUrl := busUrl + "search/"
	var infoMap = make(map[string]javkit.JavInfo, len(javList))
	for _, jav := range javList {
		title := []string{"[", filepath.Base(jav.Path), "]", " "}
		standPrint := func(messages ...string) {
			messages = append(title, messages...)
			javkit.PrintWithTime(messages...)
		}

		collectAllInfo(config, jav, searchBaseUrl, standPrint, infoMap)
	}
	collectJav(config, infoMap)

	javEmptyList := javkit.GetJavFromFolder(path, config)
	return len(javEmptyList), deleteParent

}

func reManage(config javkit.IniConfig) {
	for {
		name := prompt.Input("name > ", emptyCompleter)
		switch name {
		case "back":
			return
		default:
			// 1. 移动视频文件到临时文件夹（新建于归类根目录下）
			tmpFolder := filepath.Join(config.ClassifyRoot, "tmp")
			if !javkit.Exists(tmpFolder) {
				os.Mkdir(tmpFolder, 0776)
			}
			const reqLimit = 1024
			videoName, folderPath, prefixPath, err := findManagedFilm(config, name)
			if err != nil {
				prefixPath.Close()
				return
			}
			folderList, err := prefixPath.Readdirnames(reqLimit)
			if err != nil {
				if io.EOF == err {
					break
				} else {
					fmt.Printf("读取文件失败，%s\n", err)
					return
				}
			}
			for _, folderName := range folderList {
				// 找到番号对应的文件夹
				if strings.Contains(folderName, videoName) {
					oldFilmPath := filepath.Join(folderPath, folderName)
					videoTypeList := strings.Split(config.FileType, "、")
					oldFilmParentInfo, err := os.Open(oldFilmPath)
					if err != nil {
						fmt.Printf("打开文件夹失败，%s\n", err)
						return
					}
					oldFiles, err := oldFilmParentInfo.Readdirnames(reqLimit)
					if err != nil {
						if io.EOF == err {
							break
						} else {
							fmt.Printf("读取文件失败，%s\n", err)
							return
						}
					}
					for _, file := range oldFiles {
						suffix := path.Ext(file)[1:]
						// 找到番号对应的视频文件
						if funk.Contains(videoTypeList, suffix) {
							oldFullPath := filepath.Join(oldFilmPath, file)
							newFullPath := filepath.Join(tmpFolder, file)

							err := os.Rename(oldFullPath, newFullPath)
							if err != nil {
								fmt.Printf("移动影片失败，%s\n", err)
								return
							}

							// 2. 调用 filmDelete 方法删除旧文件夹
							doDelete(config, prefixPath, videoName, folderPath)

							// 3. 调用 manage 方法重新整理
							waitTime := 5
							remainNumber, deleteParent := doManage(tmpFolder, config)

							if remainNumber == 0 {
								if deleteParent {
									// 删除旧文件夹
									err := os.RemoveAll(tmpFolder)
									if err != nil {
										fmt.Println(tmpFolder, " 删除旧文件夹失败，原因：", err.Error())
										return
									}
									fmt.Println("删除 ", tmpFolder)
								}
							} else {
								fmt.Printf("有 %d 部影片未归类完成，等待 %d 秒后重试\n", remainNumber, waitTime)
								time.Sleep(time.Duration(waitTime) * time.Second)
								remainNumber, deleteParent = doManage(tmpFolder, config)
								if remainNumber == 0 {
									if deleteParent {
										// 删除旧文件夹
										err := os.RemoveAll(tmpFolder)
										if err != nil {
											fmt.Println(tmpFolder, " 删除旧文件夹失败，原因：", err.Error())
											return
										}
										fmt.Println("删除 ", tmpFolder)
										return
									}
								} else {
									fmt.Printf("存在 %d 部影片未完成归类\n", remainNumber)
									return
								}
							}
						}
					}
				}
			}
		}
	}

}

func findManagedFilm(config javkit.IniConfig, name string) (string, string, *os.File, error) {
	videoName, err := javkit.GetVideoTitle(name)
	if err != nil {
		fmt.Printf("番号不正确，%s\n", err.Error())
		return "", "", nil, err
	}

	prefix := strings.Split(videoName, `-`)[0]

	folderPath := filepath.Join(config.ClassifyRoot, prefix)

	prefixPath, err := os.Open(folderPath)
	if err != nil {
		fmt.Printf("无法打开归档目录，%s\n", err.Error())
		return "", "", nil, err
	}
	return videoName, folderPath, prefixPath, nil
}

func collectAllInfo(config javkit.IniConfig, jav javkit.JavFile, searchBaseUrl string, log func(messages ...string), infoMap map[string]javkit.JavInfo) {
	javInfo, err := getInfo(config, jav, searchBaseUrl)
	if err != nil {
		log(jav.Path, " 无法获取影片信息，跳过：", err.Error())
	} else {
		infoMap[jav.Path] = javInfo
	}
}

func getInfo(config javkit.IniConfig, jav javkit.JavFile, searchBaseUrl string) (javkit.JavInfo, error) {
	searchUrl := searchBaseUrl + jav.License
	// 获取 jav 所有需要的信息
	javInfo, err := javkit.GetJavInfo(searchUrl, config)
	return javInfo, err
}

func collectJav(config javkit.IniConfig, infoMap map[string]javkit.JavInfo) {

	for videoPath, javInfo := range infoMap {
		title := []string{"[", filepath.Base(videoPath), "]", " "}
		log := func(messages ...string) {
			messages = append(title, messages...)
			javkit.PrintWithTime(messages...)
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
		newFolderPath := javkit.CreateNewFolder(videoPath, javInfo, config)

		// 移动影片
		newVideoPath, err := javkit.RenameAndMoveVideo(videoPath, javInfo, config, newFolderPath)
		if err != nil {
			log(videoPath, " 移动影片失败，原因：", err.Error())
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
		delete(infoMap, videoPath)
	}

}
