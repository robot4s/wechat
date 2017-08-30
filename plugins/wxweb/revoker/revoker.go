package revoker // 以插件名命令包名

import (
	"time"

	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/wxweb"
)

// Register plugin
// 必须有的插件注册函数
// 指定session, 可以对不同用户注册不同插件
func Register(session *wxweb.Session) {
	// 将插件注册到session
	// 第一个参数: 指定消息类型, 所有该类型的消息都会被转发到此插件
	// 第二个参数: 指定消息处理函数, 消息会进入此函数
	// 第三个参数: 自定义插件名，不能重名，switcher插件会用到此名称
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(revoker), "revoker")

	if err := session.HandlerRegister.EnableByName("revoker"); err != nil {
		logger.Errorln(err)
	}
}

func revoker(session *wxweb.Session, msg *wxweb.ReceivedMessage) {
	if msg.FromUserName != session.Bot.UserName {
		return
	}

	time.Sleep(time.Second * 3)
	session.RevokeMsg(msg.MsgId, msg.MsgId, msg.ToUserName)

}
