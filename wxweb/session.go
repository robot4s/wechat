package wxweb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/config"
)

var (
	// DefaultCommon default session config
	DefaultCommon = &Common{
		AppId:      "wx782c26e4c19acffb",
		LoginUrl:   "https://login.weixin.qq.com",
		Lang:       "zh_CN",
		DeviceID:   "e" + GetRandomStringFromNum(15),
		SyncSrv:    "webpush.wx.qq.com",
		UploadUrl:  "https://file.wx.qq.com/cgi-bin/mmwebwx-bin/webwxuploadmedia?f=json",
		MediaCount: 0,
	}
	// URLPool url group
	URLPool = []UrlGroup{
		{"wx2.qq.com", "file.wx2.qq.com", "webpush.wx2.qq.com"},
		{"wx8.qq.com", "file.wx8.qq.com", "webpush.wx8.qq.com"},
		{"qq.com", "file.wx.qq.com", "webpush.wx.qq.com"},
		{"web2.wechat.com", "file.web2.wechat.com", "webpush.web2.wechat.com"},
		{"wechat.com", "file.web.wechat.com", "webpush.web.wechat.com"},
	}
)

// Session wechat bot session
type Session struct {
	WxWebCommon     *Common
	WxWebXcg        *XmlConfig
	Cookies         []*http.Cookie
	SynKeyList      *SyncKeyList
	Bot             *User
	Cm              *ContactManager
	QrcodePath      string //qrcode path
	QrcodeUUID      string //uuid
	HandlerRegister *HandlerRegister
	CreateTime      int64
}

// CreateSession create wechat bot session
// if common is nil, session will be created with default config
// if handlerRegister is nil,  session will create a new HandlerRegister
func CreateSession(common *Common, handlerRegister *HandlerRegister) (*Session, error) {
	if common == nil {
		common = DefaultCommon
	}

	wxWebXcg := &XmlConfig{}

	// get qrcode
	uuid, err := JsLogin(common)
	if err != nil {
		return nil, err
	}
	logger.Infof("JsLogin uuid=%v", uuid)
	session := &Session{
		WxWebCommon: common,
		WxWebXcg:    wxWebXcg,
		QrcodeUUID:  uuid,
		CreateTime:  time.Now().Unix(),
	}

	if handlerRegister != nil {
		session.HandlerRegister = handlerRegister
	} else {
		session.HandlerRegister = CreateHandlerRegister()
	}

	if err := QrCodeToFile(common, uuid); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Session) analizeVersion(uri string) {
	u, _ := url.Parse(uri)

	// version may change
	s.WxWebCommon.CgiDomain = u.Scheme + "://" + u.Host
	s.WxWebCommon.CgiUrl = s.WxWebCommon.CgiDomain + "/cgi-bin/mmwebwx-bin"

	for _, urlGroup := range URLPool {
		if strings.Contains(u.Host, urlGroup.IndexUrl) {
			s.WxWebCommon.SyncSrv = urlGroup.SyncUrl
			s.WxWebCommon.UploadUrl = fmt.Sprintf("https://%s/cgi-bin/mmwebwx-bin/webwxuploadmedia?f=json", urlGroup.UploadUrl)
			return
		}
	}
}

func (s *Session) scanWaiter() error {
loop1:
	for {
		select {
		case <-time.After(3 * time.Second):
			redirectURI, err := Login(s.WxWebCommon, s.QrcodeUUID, "0")
			if err != nil {
				logger.Warnf("Login error:%v", err)
				if strings.Contains(err.Error(), "window.code=408") {
					return err
				}
			} else {
				s.WxWebCommon.RedirectUri = redirectURI
				s.analizeVersion(s.WxWebCommon.RedirectUri)
				break loop1
			}
		}
	}
	return nil
}

// LoginAndServe login wechat web and enter message receiving loop
func (s *Session) LoginAndServe(useCache bool) error {

	var (
		err error
	)

	if !useCache {
		if s.Cookies != nil {
			// confirmWaiter
		}

		if err := s.scanWaiter(); err != nil {
			return err
		}

		// update cookies
		if s.Cookies, err = WebNewLoginPage(s.WxWebCommon, s.WxWebXcg, s.WxWebCommon.RedirectUri); err != nil {
			return err
		}
	}

	jb, err := WebWxInit(s.WxWebCommon, s.WxWebXcg)
	if err != nil {
		return err
	}

	jc, err := config.LoadJsonConfigFromBytes(jb)
	if err != nil {
		return err
	}

	s.SynKeyList, err = GetSyncKeyListFromJc(jc)
	if err != nil {
		return err
	}

	s.Bot, _ = GetUserInfoFromJc(jc)
	logger.Infof("Current UserInfo=%+v", s.Bot)

	ret, err := WebWxStatusNotify(s.WxWebCommon, s.WxWebXcg, s.Bot)
	if err != nil {
		return err
	}
	if ret != 0 {
		return fmt.Errorf("WebWxStatusNotify fail, %d", ret)
	}

	cb, err := WebWxGetContact(s.WxWebCommon, s.WxWebXcg, s.Cookies)
	if err != nil {
		return err
	}

	s.Cm, err = CreateContactManagerFromBytes(cb)
	if err != nil {
		return err
	}

	// for v2
	s.Cm.AddUser(s.Bot)

	if err := s.serve(); err != nil {
		return err
	}
	return nil
}

func (s *Session) serve() error {
	msg := make(chan []byte, 1000)
	// syncheck
	errChan := make(chan error)
	go s.producer(msg, errChan)
	for {
		select {
		case m := <-msg:
			go s.consumer(m)
		case err := <-errChan:
			// TODO maybe not all consumed messages have not return yet
			logger.Errorf("errChan err:%v", err)
			return err
		}
	}
}
func (s *Session) producer(msg chan []byte, errChan chan error) {
	logger.Info("entering synccheck loop")
loop1:
	for {
		ret, sel, err := SyncCheck(s.WxWebCommon, s.WxWebXcg, s.Cookies, s.WxWebCommon.SyncSrv, s.SynKeyList)
		logger.Tracef("sync server:%v, ret=%v, sel:%v", s.WxWebCommon.SyncSrv, ret, sel)
		if err != nil {
			logger.Errorf("SyncCheck err:%v", err)
			continue
		}
		if ret == 0 {
			// check success
			if sel == 2 {
				// new message
				err := WebWxSync(s.WxWebCommon, s.WxWebXcg, s.Cookies, msg, s.SynKeyList)
				if err != nil {
					logger.Errorf("WebWxSync err:%v", err)
				}
			} else if sel != 0 && sel != 7 {
				errChan <- fmt.Errorf("session down, sel %d", sel)
				break loop1
			}
		} else if ret == 1101 {
			errChan <- nil
			break loop1
		} else if ret == 1205 {
			errChan <- fmt.Errorf("api blocked, ret:%d", 1205)
			break loop1
		} else {
			errChan <- fmt.Errorf("unhandled exception ret %d", ret)
			break loop1
		}
	}

}

func (s *Session) consumer(msg []byte) {
	// analize message
	jc, _ := config.LoadJsonConfigFromBytes(msg)
	msgCount, _ := jc.GetInt("AddMsgCount")
	if msgCount < 1 {
		// no msg details
		return
	}
	msgis, _ := jc.GetInterfaceSlice("AddMsgList")
	for _, v := range msgis {
		rmsg := s.analize(v.(map[string]interface{}))
		err, handles := s.HandlerRegister.Get(rmsg.MsgType)
		if err != nil {
			logger.Warnf("AddMsgList analize err:%v.data=%v", err, string(msg))
			continue
		}

		logger.Debugf("recvd:%+v", msg)
		for _, v := range handles {
			go v.Run(s, rmsg)
		}
	}
}

func (s *Session) analize(msg map[string]interface{}) *ReceivedMessage {
	rmsg := &ReceivedMessage{
		MsgId:         msg["MsgId"].(string),
		OriginContent: msg["Content"].(string),
		FromUserName:  msg["FromUserName"].(string),
		ToUserName:    msg["ToUserName"].(string),
		MsgType:       int(msg["MsgType"].(float64)),
		SubType:       int(msg["SubMsgType"].(float64)),
		Url:           msg["Url"].(string),
	}

	// friend verify message
	if rmsg.MsgType == MSG_FV {
		riif := msg["RecommendInfo"].(map[string]interface{})
		rmsg.RecommendInfo = &RecommendInfo{
			Ticket:   riif["Ticket"].(string),
			UserName: riif["UserName"].(string),
			NickName: riif["NickName"].(string),
			Content:  riif["Content"].(string),
			Sex:      int(riif["Sex"].(float64)),
		}
	}

	if strings.Contains(rmsg.FromUserName, "@@") ||
		strings.Contains(rmsg.ToUserName, "@@") {
		rmsg.IsGroup = true
		// group message
		ss := strings.Split(rmsg.OriginContent, ":<br/>")
		if len(ss) > 1 {
			rmsg.Who = ss[0]
			rmsg.Content = ss[1]
		} else {
			rmsg.Who = s.Bot.UserName
			rmsg.Content = rmsg.OriginContent
		}
	} else {
		// none group message
		rmsg.Who = rmsg.FromUserName
		rmsg.Content = rmsg.OriginContent
	}

	if rmsg.MsgType == MSG_TEXT &&
		len(rmsg.Content) > 1 &&
		strings.HasPrefix(rmsg.Content, "@") {
		// @someone
		ss := strings.Split(rmsg.Content, "\u2005")
		if len(ss) == 2 {
			rmsg.At = ss[0] + "\u2005"
			rmsg.Content = ss[1]
		}
	}
	return rmsg
}

// After message funcs
func (s *Session) After(duration time.Duration) *Session {
	select {
	case <-time.After(duration):
		return s
	}
}

// At At
func (s *Session) At(d time.Time) *Session {
	return s.After(d.Sub(time.Now()))
}

// SendText send text msg type 1
func (s *Session) SendText(msg, from, to string) (string, string, error) {
	b, err := WebWxSendMsg(s.WxWebCommon, s.WxWebXcg, s.Cookies, from, to, msg)
	if err != nil {
		return "", "", err
	}
	jc, _ := config.LoadJsonConfigFromBytes(b)
	ret, _ := jc.GetInt("BaseResponse.Ret")
	if ret != 0 {
		errMsg, _ := jc.GetString("BaseResponse.ErrMsg")
		return "", "", fmt.Errorf("WebWxSendMsg Ret=%d, ErrMsg=%s", ret, errMsg)
	}
	msgID, _ := jc.GetString("MsgID")
	localID, _ := jc.GetString("LocalID")
	return msgID, localID, nil
}

// SendImg send img, upload then send
func (s *Session) SendImg(path, from, to string) {
	ss := strings.Split(path, "/")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Error(err)
		return
	}
	mediaID, err := WebWxUploadMedia(s.WxWebCommon, s.WxWebXcg, s.Cookies, ss[len(ss)-1], b)
	if err != nil {
		logger.Error(err)
		return
	}
	ret, err := WebWxSendMsgImg(s.WxWebCommon, s.WxWebXcg, s.Cookies, from, to, mediaID)
	if err != nil || ret != 0 {
		logger.Error(ret, err)
		return
	}
}

//SendImgFromBytes send image from mem
func (s *Session) SendImgFromBytes(b []byte, path, from, to string) {
	ss := strings.Split(path, "/")
	mediaId, err := WebWxUploadMedia(s.WxWebCommon, s.WxWebXcg, s.Cookies, ss[len(ss)-1], b)
	if err != nil {
		logger.Error(err)
		return
	}
	ret, err := WebWxSendMsgImg(s.WxWebCommon, s.WxWebXcg, s.Cookies, from, to, mediaId)
	if err != nil || ret != 0 {
		logger.Error(ret, err)
		return
	}
}

//GetImg get img by MsgId
func (s *Session) GetImg(msgID string) ([]byte, error) {
	return WebWxGetMsgImg(s.WxWebCommon, s.WxWebXcg, s.Cookies, msgID)
}

// SendEmotionFromPath send gif, upload then send
func (s *Session) SendEmotionFromPath(path, from, to string) {
	ss := strings.Split(path, "/")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Error(err)
		return
	}
	mediaID, err := WebWxUploadMedia(s.WxWebCommon, s.WxWebXcg, s.Cookies, ss[len(ss)-1], b)
	if err != nil {
		logger.Error(err)
		return
	}
	ret, err := WebWxSendEmoticon(s.WxWebCommon, s.WxWebXcg, s.Cookies, from, to, mediaID)
	if err != nil || ret != 0 {
		logger.Error(ret, err)
	}
}

// SendEmotionFromBytes send gif/emoji from mem
func (s *Session) SendEmotionFromBytes(b []byte, from, to string) {
	mediaID, err := WebWxUploadMedia(s.WxWebCommon, s.WxWebXcg, s.Cookies, from+".gif", b)
	if err != nil {
		logger.Error(err)
		return
	}
	ret, err := WebWxSendEmoticon(s.WxWebCommon, s.WxWebXcg, s.Cookies, from, to, mediaID)
	if err != nil || ret != 0 {
		logger.Error(ret, err)
	}
}

// RevokeMsg revoke message
func (s *Session) RevokeMsg(clientMsgID, svrMsgID, toUserName string) {
	err := WebWxRevokeMsg(s.WxWebCommon, s.WxWebXcg, s.Cookies, clientMsgID, svrMsgID, toUserName)
	if err != nil {
		logger.Errorf("revoke msg %s failed, %s", clientMsgID+":"+svrMsgID, err)
		return
	}
}

// Logout logout web wechat
func (s *Session) Logout() error {
	return WebWxLogout(s.WxWebCommon, s.WxWebXcg, s.Cookies)
}

// AcceptFriend AcceptFriend
func (s *Session) AcceptFriend(verifyContent string, vul []*VerifyUser) error {
	b, err := WebWxVerifyUser(s.WxWebCommon, s.WxWebXcg, s.Cookies, 3, verifyContent, vul)
	if err != nil {
		return err
	}
	jc, err := config.LoadJsonConfigFromBytes(b)
	if err != nil {
		return err
	}
	retcode, err := jc.GetInt("BaseResponse.Ret")
	if err != nil {
		return err
	}
	if retcode != 0 {
		return fmt.Errorf("BaseResponse.Ret %d", retcode)
	}
	return nil
}
