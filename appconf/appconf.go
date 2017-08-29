package appconf

import (
	"fmt"
	"os"

	"github.com/lets-go-go/httpclient"
	"github.com/lets-go-go/logger"
)

var (
	// CorpName 组织名
	CorpName string
	// AppName 程序名
	AppName string
	// AppdataDir 程序数据目录
	AppdataDir string

	// LogDir 日志目录
	LogDir string

	// HTTPSrvPort HTTP服务端口
	HTTPSrvPort int

	// LogLevel 日志级别
	LogLevel logger.LEVEL

	// UserAgent UserAgent
	UserAgent string

	// ProxyType ProxyType
	ProxyType httpclient.ProxyType

	// ProxyURL ProxyURL
	ProxyURL string
)

func init() {
	CorpName = "robot4s"
	AppName = "wechat"

	HTTPSrvPort = 11223

	appdata := os.Getenv("APPDATA")

	AppdataDir = fmt.Sprintf("%s/%s/%s", appdata, CorpName, AppName)

	LogDir = fmt.Sprintf("%s/logs", AppdataDir)

	LogLevel = logger.DEBUG

	if FileExists(LogDir) == false {
		os.MkdirAll(LogDir, os.ModeDir)
	}

	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.109 Safari/537.36"

	ProxyType = httpclient.CustomProxy
	ProxyURL = "http://192.168.16.189:8080"
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
