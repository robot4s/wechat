package share

import (
	"strings"

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
	session.HandlerRegister.Add(wxweb.MSG_TEXT, wxweb.Handler(share), "share")

	if err := session.HandlerRegister.EnableByName("share"); err != nil {
		logger.Errorln(err)
	}
}

// 消息处理函数
func share(session *wxweb.Session, msg *wxweb.ReceivedMessage) {

	// 取出收到的内容
	// 取text
	if strings.Contains(msg.Content, "纸牌屋") {
		text := "https://pan.baidu.com/s/1sl4S0nr#list/path=%2F"
		session.SendText("纸牌屋第五季在线观看 无毒无广\n"+text, session.Bot.UserName, wxweb.RealTargetUserName(session, msg))
	}

	// for issue#13 debug
	//who := session.Cm.GetContactByUserName(msg.FromUserName)
	//logger.Debug("who send", who)
	//if msg.IsGroup {
	//	mm, err := wxweb.CreateMemberManagerFromGroupContact(session, who)
	//	if err != nil {
	//		logger.Error(err)
	//		return
	//	}
	//	info := mm.GetContactByUserName(msg.Who)
	//	logger.Debug(info)
	//}
}
