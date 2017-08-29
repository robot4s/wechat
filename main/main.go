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
	"github.com/robot4s/wechat/plugins/wxweb/cleaner"
	"github.com/robot4s/wechat/plugins/wxweb/forwarder"
	"github.com/robot4s/wechat/plugins/wxweb/replier"
	"github.com/robot4s/wechat/plugins/wxweb/revoker"
	"github.com/robot4s/wechat/plugins/wxweb/switcher"
	"github.com/robot4s/wechat/plugins/wxweb/system"
	"github.com/robot4s/wechat/wxweb"
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

	// create session
	session, err := wxweb.CreateSession(nil, nil)
	if err != nil {
		logger.Errorf("CreateSession:%+v", err)
		return
	}

	// if err := desktop.Open("http://127.0.0.1:11223/wechat-login"); err != nil {
	// 	logger.Errorf("open error:%+v", err)
	// }

	// load plugins for this session
	replier.Register(session)
	switcher.Register(session)
	cleaner.Register(session)
	revoker.Register(session)
	forwarder.Register(session)
	system.Register(session)

	// enable by type example
	if err := session.HandlerRegister.EnableByType(wxweb.MSG_SYS); err != nil {
		logger.Errorf("EnableByType err:%+v", err)
		return
	}

	for {
		if err := session.LoginAndServe(false); err != nil {
			logger.Errorf("session exit, %s", err)
			for i := 0; i < 3; i++ {
				logger.Info("trying re-login with cache")
				if err := session.LoginAndServe(true); err != nil {
					logger.Errorf("re-login error, %v", err)
				}
				time.Sleep(3 * time.Second)
			}
			if session, err = wxweb.CreateSession(nil, session.HandlerRegister); err != nil {
				logger.Errorf("create new sesion failed, %v", err)
				break
			}
		} else {
			logger.Info("closed by user")
			break
		}
	}

	<-stopChan // wait for SIGINT
	logger.Infoln("Shutting down server...")

	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	httpserver.CloseHTTPServer(ctx)

	logger.Infoln("gracefully stopped stopped")
}
