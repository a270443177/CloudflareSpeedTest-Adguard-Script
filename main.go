package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/asmcos/requests"
	v2 "gopkg.in/yaml.v2"
)

var (
	cfgFile = "config.yaml"
)

type Config struct {
	AdguardUrl string `yaml:"url"`
	UserName   string `yaml:"username"`
	PassWord   string `yaml:"password"`
	Domain     string `yaml:"domain"`
}

func fileExists(filePath string) bool {
	_, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func main() {
	var ip string

	// 获取当前操作系统的文件路径分隔符
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "CloudflareSpeedTest.exe"
	} else {
		binaryName = "CloudflareSpeedTest"
	}

	file_Exists := fileExists(binaryName)
	if !file_Exists {
		fmt.Printf("当前目录下不存在可执行文件：CloudflareSpeedTest")
		return
	}

	// 获取当前程序执行的工作目录（当前路径）
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("无法获取当前路径:", err)
		return
	}

	// 构建文件路径
	binaryPath := filepath.Join(currentDir, binaryName)

	//执行CloudSpeed Test
	// 想要执行的命令，例如："echo" 和 "Hello, World!"
	cmd := exec.Command(binaryPath, "-o", "result.txt")

	// 运行命令，并获取输出
	err = cmd.Run()
	if err != nil {
		fmt.Println("命令执行失败:", err)
		return
	}

	result_Exists := fileExists("result.txt")
	if !result_Exists {
		fmt.Printf("当前目录下没有生成result.txt")
		return
	}

	// 打开文件
	file, err := os.Open("result.txt")
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	// 创建一个 Scanner 对象来逐行读取文件内容
	scanner := bufio.NewScanner(file)

	// 读取第二行数据
	lineCount := 0
	for scanner.Scan() {
		// 逐行读取数据
		line := scanner.Text()
		lineCount++

		// 如果是第二行，则提取其中的 IP 地址信息
		if lineCount == 2 {
			ipInfo := strings.Split(line, ",")[0]
			ip = ipInfo
			break
		}
	}

	if lineCount < 2 {
		fmt.Println("result.txt文件中没有可用的IP")
		return
	}

	cfg, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	conf := new(Config)
	if err := v2.Unmarshal(cfg, conf); err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}

	jsonStr := `{"name": "` + conf.UserName + `","password":"` + conf.PassWord + `"}`

	req := requests.Requests()

	_, err = req.PostJson(conf.AdguardUrl+"/control/login", jsonStr)

	if err != nil {
		fmt.Printf("自动登录AdGuard失败，" + err.Error())
		return
	}

	resp, err := req.Get(conf.AdguardUrl + "/control/filtering/status")

	if err != nil {
		fmt.Printf("获取自定义DNS失败，" + err.Error())
		return
	}

	var jsonData map[string]interface{}

	resp.Json(&jsonData)

	// 提取 user_rules 数组
	userRules, ok := jsonData["user_rules"].([]interface{})

	if !ok {
		fmt.Println("user_rules not found or not an array")
		return
	}

	var ExistDomain = false
	for i, rule := range userRules {
		ruleStr, ok := rule.(string)
		if !ok {
			fmt.Println("Invalid rule format")
			continue
		}

		parts := strings.Split(ruleStr, " ")
		if len(parts) == 2 && parts[1] == conf.Domain {
			parts[0] = ip
			userRules[i] = strings.Join(parts, " ")
			ExistDomain = true
		}
	}
	if !ExistDomain {
		newRule := ip + " " + conf.Domain
		userRules = append(userRules, newRule)
	}

	// 构建更新后的 JSON 数据
	updatedJSON, err := json.Marshal(map[string]interface{}{
		"rules": userRules,
	})
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	_, err = req.PostJson(conf.AdguardUrl+"/control/filtering/set_rules", string(updatedJSON))
	if err != nil {
		fmt.Printf("自动登录AdGuard失败，" + err.Error())
		return
	}

}
