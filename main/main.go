package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/httpserver"

	"github.com/lets-go-go/httpclient"
	"github.com/lets-go-go/logger"
)

func main() {

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	config := logger.DefalutConfig()
	config.Level = logger.LEVEL(appconf.LogLevel)
	config.LogFileRollingType = logger.RollingDaily
	config.LogFileOutputDir = appconf.LogDir
	config.LogFileName = "wechat"
	config.LogFileMaxCount = 5
	logger.Init(config)

	httpclient.Settings().SetUserAgent(appconf.UserAgent).SetProxy(appconf.ProxyType, appconf.ProxyURL)

	httpserver.NewHTTPServer()

	logger.Infoln("wechat robot service started...")

	<-stopChan // wait for SIGINT
	logger.Infoln("Shutting down server...")

	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	httpserver.CloseHTTPServer(ctx)

	logger.Infoln("gracefully stopped stopped")
}
