package robot

import (
	"time"

	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/plugins/wxweb/cleaner"
	"github.com/robot4s/wechat/plugins/wxweb/forwarder"
	"github.com/robot4s/wechat/plugins/wxweb/replier"
	"github.com/robot4s/wechat/plugins/wxweb/revoker"
	"github.com/robot4s/wechat/plugins/wxweb/switcher"
	"github.com/robot4s/wechat/plugins/wxweb/system"
	"github.com/robot4s/wechat/plugins/wxweb/verify"
	"github.com/robot4s/wechat/wxweb"
)

var (
	robots map[string]*wxweb.Session
)

func init() {
	robots = make(map[string]*wxweb.Session)
}

// StartRobot 启动一个
func StartRobot() string {
	// create session
	session, err := wxweb.CreateSession(nil, nil)
	if err != nil {
		logger.Errorf("CreateSession:%+v", err)
		return ""
	}

	go initRobot(session)

	robots[session.QrcodeUUID] = session

	return session.QrcodeUUID
}

func initRobot(session *wxweb.Session) {
	// load plugins for this session
	replier.Register(session)
	switcher.Register(session)
	cleaner.Register(session)
	revoker.Register(session)
	forwarder.Register(session)
	system.Register(session)
	verify.Register(session)

	for {
		if err := session.LoginAndServe(false); err != nil {
			logger.Errorf("session exit, %v", err)
			for i := 0; i < 3; i++ {
				logger.Info("trying re-login with cache")
				if err := session.LoginAndServe(true); err != nil {
					logger.Errorf("re-login error, %v", err)
				}
				time.Sleep(10 * time.Second)
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

	delete(robots, session.QrcodeUUID)
}
