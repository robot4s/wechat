package replier

import (
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/wxweb"
)

// register plugin
func Register(session *wxweb.Session) {
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(autoReply), "text-replier")
	if err := session.HandlerRegister.Add(wxweb.MSG_IMG, wxweb.Handler(autoReply), "img-replier"); err != nil {
		logger.Error(err)
	}

	if err := session.HandlerRegister.EnableByName("text-replier"); err != nil {
		logger.Error(err)
	}

	if err := session.HandlerRegister.EnableByName("img-replier"); err != nil {
		logger.Error(err)
	}

}
func autoReply(session *wxweb.Session, msg *wxweb.ReceivedMessage) {
	if !msg.IsGroup {
		session.SendText("暂时不在，稍后回复", session.Bot.UserName, msg.FromUserName)
	}
}
