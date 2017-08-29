package forwarder

import (
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/wxweb"
)

var (
	// 需要消息互通的群
	groups = map[string]bool{
		"jianshujiaojingdadui": true,
		"forwarder":            true, //for test
	}
)

func Register(session *wxweb.Session) {
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(forward), "text-forwarder")
	session.HandlerRegister.Add(wxweb.MSG_IMG, wxweb.Handler(forward), "img-forwarder")

	if err := session.HandlerRegister.EnableByName("text-forwarder"); err != nil {
		logger.Error(err)
	}

	if err := session.HandlerRegister.EnableByName("img-forwarder"); err != nil {
		logger.Error(err)
	}
}

func forward(session *wxweb.Session, msg *wxweb.ReceivedMessage) {
	if !msg.IsGroup {
		return
	}
	var contact *wxweb.User
	if msg.FromUserName == session.Bot.UserName {
		contact = session.Cm.GetContactByUserName(msg.ToUserName)
	} else {
		contact = session.Cm.GetContactByUserName(msg.FromUserName)
	}
	if contact == nil {
		return
	}
	if _, ok := groups[contact.PYQuanPin]; !ok {
		return
	}
	mm, err := wxweb.CreateMemberManagerFromGroupContact(session, contact)
	if err != nil {
		logger.Debug(err)
		return
	}
	who := mm.GetContactByUserName(msg.Who)
	if who == nil {
		who = session.Bot
	}

	for k, v := range groups {
		if !v {
			continue
		}
		c := session.Cm.GetContactByPYQuanPin(k)
		if c == nil {
			logger.Error("cannot find group contact %s", k)
			continue
		}
		if c.UserName == contact.UserName {
			// ignore
			continue
		}
		if msg.MsgType == wxweb.MSG_TEXT {
			session.SendText("@"+who.NickName+" "+msg.Content, session.Bot.UserName, c.UserName)
		}
		if msg.MsgType == wxweb.MSG_IMG {
			b, err := session.GetImg(msg.MsgId)
			if err == nil {
				session.SendImgFromBytes(b, msg.MsgId+".jpg", session.Bot.UserName, c.UserName)
			} else {
				logger.Error(err)
			}
		}
	}

	//mm, err := wxweb.CreateMemberManagerFromGroupContact(contact)
}
