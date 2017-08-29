package system // 以插件名命令包名

import (
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/wxweb" // 导入协议包
)

// Register plugin
// 必须有的插件注册函数
// 指定session, 可以对不同用户注册不同插件
func Register(session *wxweb.Session) {
	// 将插件注册到session
	// 第一个参数: 指定消息类型, 所有该类型的消息都会被转发到此插件
	// 第二个参数: 指定消息处理函数, 消息会进入此函数
	// 第三个参数: 自定义插件名，不能重名，switcher插件会用到此名称
	session.HandlerRegister.Add(wxweb.MSG_SYS, wxweb.Handler(system), "system-sys")
	session.HandlerRegister.Add(wxweb.MSG_WITHDRAW, wxweb.Handler(system), "system-withdraw")

	if err := session.HandlerRegister.EnableByName("system-sys"); err != nil {
		logger.Error(err)
	}

	if err := session.HandlerRegister.EnableByName("system-withdraw"); err != nil {
		logger.Error(err)
	}
}

// 消息处理函数
func system(session *wxweb.Session, msg *wxweb.ReceivedMessage) {

	logger.Debug(msg)
}
