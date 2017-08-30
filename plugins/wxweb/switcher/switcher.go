package switcher

import (
	"strings"

	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/wxweb"
)

// Register plugin
func Register(session *wxweb.Session) {
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(switcher), "switcher")
	if err := session.HandlerRegister.EnableByName("switcher"); err != nil {
		logger.Errorln(err)
	}
}

func switcher(session *wxweb.Session, msg *wxweb.ReceivedMessage) {
	// contact filter
	contact := session.Cm.GetContactByUserName(msg.FromUserName)
	if contact == nil {
		logger.Errorf("no this contact:%v, ignore", msg.FromUserName)
		return
	}

	if strings.ToLower(msg.Content) == "dump" {
		session.SendText(session.HandlerRegister.Dump(), session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		return
	}

	if !strings.Contains(strings.ToLower(msg.Content), "enable") &&
		!strings.Contains(strings.ToLower(msg.Content), "disable") {
		return
	}

	ss := strings.Split(msg.Content, " ")
	if len(ss) < 2 {
		return
	}
	if strings.ToLower(ss[1]) == "switcher" {
		session.SendText("hehe, you think too much", session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		return
	}

	var (
		err error
	)
	if strings.ToLower(ss[0]) == "enable" {
		if err = session.HandlerRegister.EnableByName(ss[1]); err == nil {
			session.SendText(msg.Content+" [DONE]", session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		} else {
			session.SendText(err.Error(), session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		}
	} else if strings.ToLower(ss[0]) == "disable" {
		if err = session.HandlerRegister.DisableByName(ss[1]); err == nil {
			session.SendText(msg.Content+" [DONE]", session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		} else {
			session.SendText(err.Error(), session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
		}
	}
}
