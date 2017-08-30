package verify // auto verify friend request

import (
	"fmt"

	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/wxweb" // 导入协议包
	// 导入日志包
)

// Register plugin
// 必须有的插件注册函数
// 指定session, 可以对不同用户注册不同插件
func Register(session *wxweb.Session) {
	// 将插件注册到session
	// 第一个参数: 指定消息类型, 所有该类型的消息都会被转发到此插件
	// 第二个参数: 指定消息处理函数, 消息会进入此函数
	// 第三个参数: 自定义插件名，不能重名，switcher插件会用到此名称
	session.HandlerRegister.Add(wxweb.MSG_FV, wxweb.Handler(verify), "verify")

	if err := session.HandlerRegister.EnableByName("verify"); err != nil {
		logger.Error(err)
	}
}

// 消息处理函数
func verify(session *wxweb.Session, msg *wxweb.ReceivedMessage) {

	logger.Infof("receive add friend notify:%v", msg.Content)

	master := session.Cm.GetContactByPYQuanPin(appconf.RobotMaster)
	if err := session.AcceptFriend("", []*wxweb.VerifyUser{{Value: msg.RecommendInfo.UserName, VerifyUserTicket: msg.RecommendInfo.Ticket}}); err != nil {
		errMsg := fmt.Sprintf("accept %s's friend request error, %s", msg.RecommendInfo.NickName, err.Error())
		logger.Errorln(errMsg)

		if master != nil {
			// 接受好友失败发送微信消息给指定好友
			session.SendText(errMsg, session.Bot.UserName, master.UserName)
		}
		return
	}

	session.SendText(appconf.WelcomeMsg, session.Bot.UserName, msg.RecommendInfo.UserName)

	// 回复消息
	// 第一个参数: 回复的内容
	// 第二个参数: 机器人ID
	// 第三个参数: 联系人/群组/特殊账号ID
	if master != nil {
		session.SendText(fmt.Sprintf("%s accepted", msg.RecommendInfo.NickName), session.Bot.UserName, master.UserName)
	}

}
