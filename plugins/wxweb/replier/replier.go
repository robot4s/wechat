package replier

import (
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/wxweb"
)

// Register register plugin
func Register(session *wxweb.Session) {
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(autoReply), "text-replier")
	if err := session.HandlerRegister.Add(wxweb.MSG_IMG, wxweb.Handler(autoReply), "img-replier"); err != nil {
		logger.Errorln(err)
	}

	if err := session.HandlerRegister.EnableByName("text-replier"); err != nil {
		logger.Errorln(err)
	}

	if err := session.HandlerRegister.EnableByName("img-replier"); err != nil {
		logger.Errorln(err)
	}

}
func autoReply(session *wxweb.Session, msg *wxweb.ReceivedMessage) {
	if !msg.IsGroup {
		if msg.Content == "1" {
			// 发送群邀请链接
		} else if msg.Content == "2" {
			// 发送QQ群二维码
		} else if msg.Content == "3" {
			// 发送公众号
		} else {
			session.SendText(appconf.WelcomeMsg, session.Bot.UserName, msg.FromUserName)
		}

	}
}
