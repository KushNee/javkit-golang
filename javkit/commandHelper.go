package javkit

import (
	"github.com/c-bata/go-prompt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func PathCompleter(d prompt.Document) []prompt.Suggest {
	var suggestionList []prompt.Suggest
	var path string

	input := d.TextBeforeCursor()
	commandList := []prompt.Suggest{
		{Text: "exit", Description: "退出"},
	}

	path = fixDir(input)

	pathSuggestions := filterInCurrentPath(path)
	if len(pathSuggestions) > 0 {
		suggestionList = pathSuggestions
	} else {
		suggestionList = commandList
	}

	return suggestionList
}

func fixDir(input string) string {
	var path string
	if strings.HasPrefix(input, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			PrintWithTime("无法获取家目录，原因：", err.Error())
		}
		path = home + input[1:]
	} else if strings.HasPrefix(input, ".") {
		current, err := os.Getwd()
		if err != nil {
			PrintWithTime("无法获取当前目录，原因：", err.Error())
		}
		path = current + input[1:]
	} else {
		path = input
	}
	return filepath.Clean(path)
}

func detectType(dir os.FileInfo) string {
	var description string
	if dir.IsDir() {
		description = "目录"
	} else {
		description = "文件"
	}
	return description
}

func filterInCurrentPath(path string) []prompt.Suggest {
	var sgs []prompt.Suggest
	if !strings.HasPrefix(path, "/") {
		return sgs
	}
	upperPath := filepath.Dir(path)
	var search string
	if !strings.HasSuffix(path, "/") {
		search = filepath.Base(path)
	}

	dirs, err := ioutil.ReadDir(upperPath)
	if err != nil {
		log.Println("遍历当前目录失败，原因：", err)
	}
	for _, dir := range dirs {
		if strings.HasPrefix(dir.Name(), ".") {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(dir.Name()), strings.ToLower(search)) {
			continue
		}
		suggestion := prompt.Suggest{
			Text:        filepath.Join(upperPath, dir.Name()),
			Description: detectType(dir),
		}
		sgs = append(sgs, suggestion)
	}
	if search != "" {
		for _, dir := range dirs {
			if strings.HasPrefix(dir.Name(), ".") || !strings.Contains(dir.Name(), search) {
				continue
			}
			intoSuggestions := filterIntoPath(filepath.Join(upperPath, dir.Name()))
			sgs = append(sgs, intoSuggestions...)
		}
	}
	return sgs
}

func filterIntoPath(path string) []prompt.Suggest {
	var sgs []prompt.Suggest

	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		log.Println("遍历当前目录失败，原因：", err)
	}
	for _, dir := range dirs {
		if strings.HasPrefix(dir.Name(), ".") {
			continue
		}

		suggestion := prompt.Suggest{
			Text:        filepath.Join(path, dir.Name()),
			Description: detectType(dir),
		}
		sgs = append(sgs, suggestion)
	}
	return sgs
}
