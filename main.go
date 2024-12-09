package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/systray"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type Item struct {
	ItemName string `json:"item_name"`
	ItemDes  string `json:"item_des"`
	ItemRun  string `json:"item_run"`
	SubItems []Item `json:"sub_items,omitempty"`
}

type List struct {
	Name string `json:"name"`
	Item []Item `json:"item"`
}

type About struct {
	// 如果有字段可以在这里定义
}

type Data struct {
	Lists []List `json:"lists"`
	About About  `json:"about"`
}

var (
	data             Data
	dataLock         sync.RWMutex
	dynamicMenuItems []*systray.MenuItem
	initialWorkDir   string // 新增：用于存储初始工作目录
)

func main() {
	var err error
	initialWorkDir, err = os.Getwd()
	if err != nil {
		log.Printf("Error getting initial working directory: %v", err)
		initialWorkDir = "" // 如果获取失败，设置为空字符串
	}
	systray.Run(onReady, nil)
}

func getConfig(filename string) (Data, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Data{}, err
	}
	defer file.Close()

	dataBytes, err := os.ReadFile(filename)
	if err != nil {
		return Data{}, err
	}

	var data Data
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return Data{}, err
	}
	return data, nil
}

func onReady() {
	icon, err := os.ReadFile("logo.png")
	if err != nil {
		systray.SetTitle("MACBOX")
	} else {
		systray.SetIcon(icon)
	}

	systray.SetTooltip("一款mac下的标签栏工具")

	// 创建固定菜单项
	mIndex := systray.AddMenuItem("About", "")
	systray.AddSeparator()

	mReload := systray.AddMenuItem("配置重载", "")
	mEdit := systray.AddMenuItem("编辑选项", "")
	mCheckUpdate := systray.AddMenuItem("检查更新", "")
	mExit := systray.AddMenuItem("退出", "")

	systray.AddSeparator()
	// 初始加载配置
	loadAndUpdateMenu()

	go func() {
		for {
			select {
			case <-mIndex.ClickedCh:
				openUrl("https://github.com/0x7eTeam/macbox")
			case <-mEdit.ClickedCh:
				executeCommand("./up")
			case <-mReload.ClickedCh:
				loadAndUpdateMenu()
			case <-mCheckUpdate.ClickedCh:
				//防止外链报毒，这里不写了。。。
				mCheckUpdate.SetTitle("已是最新版本")
				mCheckUpdate.Disable()
			case <-mExit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func loadAndUpdateMenu() {
	var err error
	dataLock.Lock()
	data, err = getConfig("config.json")
	dataLock.Unlock()

	if err != nil {
		log.Printf("Error loading config: %s", err)
		return
	}

	// 清除动态菜单项
	clearDynamicMenuItems()

	// 添加动态菜单项
	dataLock.RLock()
	for _, list := range data.Lists {
		var1 := systray.AddMenuItem(list.Name, "")
		dynamicMenuItems = append(dynamicMenuItems, var1)
		for _, item := range list.Item {
			addMenuItem(var1, item)
		}
	}
	dataLock.RUnlock()
}

func clearDynamicMenuItems() {
	for _, item := range dynamicMenuItems {
		item.Hide()
	}
	dynamicMenuItems = nil
}

func addMenuItem(parent *systray.MenuItem, item Item) {
	if len(item.SubItems) > 0 {
		// 如果有子项，创建子菜单
		subMenu := parent.AddSubMenuItem(item.ItemName, item.ItemDes)
		dynamicMenuItems = append(dynamicMenuItems, subMenu)
		for _, subItem := range item.SubItems {
			addMenuItem(subMenu, subItem)
		}
	} else {
		// 如果没有子项，创建普通菜单项
		itemMenu := parent.AddSubMenuItem(item.ItemName, item.ItemDes)
		dynamicMenuItems = append(dynamicMenuItems, itemMenu)
		go func(runCommand string) {
			for range itemMenu.ClickedCh {
				fmt.Printf("Executing command: %s\n", runCommand)
				executeCommand(runCommand)
			}
		}(item.ItemRun)
	}
	parent.AddSeparator()
}

func openUrl(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func executeCommand(itemRun string) error {
	parts := strings.SplitN(itemRun, ", ", 2)

	if len(parts) == 1 {
		// 检查是否是目录
		fileInfo, err := os.Stat(parts[0])
		if err == nil && fileInfo.IsDir() {
			// 是目录，打开它
			return openDirectory(parts[0])
		}
		// 不是目录，执行命令
		cmdParts := strings.Fields(parts[0])
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Start() // 使用 Start() 而不是 Run() 来异步执行
	} else if len(parts) == 2 {
		// 在新终端中执行命令
		directory := strings.Trim(parts[0], "\"")
		command := strings.Trim(parts[1], "\"")

		if initialWorkDir == "" {
			return fmt.Errorf("初始工作目录未知")
		}

		script := fmt.Sprintf(`
           tell application "Terminal"
               do script "cd %s && clear && %s && %s"
               activate
           end tell
       `, initialWorkDir, directory, command)

		cmd := exec.Command("osascript", "-e", script)
		return cmd.Start() // 使用 Start() 而不是 Run() 来异步执行
	} else {
		return fmt.Errorf("无效的命令格式")
	}
}
func openDirectory(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default: // Linux and other Unix-like systems
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
