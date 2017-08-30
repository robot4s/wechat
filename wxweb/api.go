package wxweb

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lets-go-go/httpclient"
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/config"
)

// JsLogin jslogin api
func JsLogin(common *Common) (string, error) {
	km := url.Values{
		"appid":        []string{common.AppId},
		"fun":          []string{"new"},
		"lang":         []string{common.Lang},
		"redirect_uri": []string{common.RedirectUri},
		"_":            []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}

	uri := common.LoginUrl + "/jslogin?" + km.Encode()

	body, _ := httpclient.Get(uri).Text()
	logger.Debugln("JsLogin body=%v", body)

	ss := strings.Split(string(body), "\"")
	if len(ss) < 2 {
		return "", fmt.Errorf("jslogin response invalid, %s", string(body))
	}
	return ss[1], nil
}

// QrCodeToFile get qrcode
func QrCodeToFile(common *Common, uuid string) error {
	km := url.Values{
		"t": []string{"webwx"},
		"_": []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}

	uri := common.LoginUrl + "/qrcode/" + uuid + "?" + km.Encode()

	fileName := fmt.Sprintf("%s.jpg", uuid)
	err := httpclient.Post(uri).SetContentType("application/octet-stream").ToFile(appconf.QRDir, fileName)

	if err != nil {
		logger.Errorf("QrCode error=%v", err)
		return err
	}

	return nil
}

func parse(body string, key string) string {
	vals := strings.Split(body, ";")
	for _, kv := range vals {

		kvs := strings.Split(kv, "=")

		if len(kvs) == 2 && strings.TrimSpace(kvs[0]) == key {
			return strings.TrimSpace(kvs[1])
		}
	}
	return ""
}

// Login login api
func Login(common *Common, uuid, tip string) (string, error) {
	km := url.Values{
		"tip":  []string{tip},
		"uuid": []string{uuid},
		"r":    []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"_":    []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}
	uri := common.LoginUrl + "/cgi-bin/mmwebwx-bin/login?" + km.Encode()

	body, _ := httpclient.Get(uri).Text()
	logger.Debugf("Login body=%v", body)

	rets := regexp.MustCompile("window.code=\"(\\d+)\";window.redirect_uri=\"(\\S+?)\";").FindStringSubmatch(body)

	if len(rets) == 2 {

		code, _ := strconv.Atoi(rets[0])
		uri := rets[1]

		logger.Debugf("Login response code=%v, redirect_uri=%v", code, uri)
	}

	if strings.Contains(body, "window.code=200") &&
		strings.Contains(body, "window.redirect_uri") {
		ss := strings.Split(body, "\"")
		if len(ss) < 2 {
			return "", fmt.Errorf("parse redirect_uri fail, %s", body)
		}
		return ss[1], nil
	}

	return "", fmt.Errorf("login response, %s", body)
}

// WebNewLoginPage webwxnewloginpage api
func WebNewLoginPage(common *Common, xc *XmlConfig, uri string) ([]*http.Cookie, error) {
	u, _ := url.Parse(uri)
	km := u.Query()
	km.Add("fun", "new")
	uri = common.CgiUrl + "/webwxnewloginpage?" + km.Encode()

	client := httpclient.Get(uri)
	body, _ := client.Text()
	logger.Debugln("WebNewLoginPage body=%v", body)

	if err := xml.Unmarshal([]byte(body), xc); err != nil {
		return nil, err
	}
	if xc.Ret != 0 {
		return nil, fmt.Errorf("xc.Ret != 0, %s", string(body))
	}
	return client.Cookies(), nil
}

// WebWxInit webwxinit api
func WebWxInit(common *Common, ce *XmlConfig) ([]byte, error) {
	km := url.Values{
		"pass_ticket": []string{ce.PassTicket},
		"skey":        []string{ce.Skey},
		"r":           []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}

	uri := common.CgiUrl + "/webwxinit?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
	}

	body, _ := httpclient.Post(uri).SendBody(js).Bytes()
	logger.Tracef("WebWxInit body=%v", string(body))

	return body, nil
}

// SyncCheck synccheck api
func SyncCheck(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	server string, skl *SyncKeyList) (int, int, error) {
	km := url.Values{
		"r":        []string{strconv.FormatInt(time.Now().Unix()*1000, 10)},
		"sid":      []string{ce.Wxsid},
		"uin":      []string{ce.Wxuin},
		"skey":     []string{ce.Skey},
		"deviceid": []string{common.DeviceID},
		"synckey":  []string{skl.String()},
		"_":        []string{strconv.FormatInt(time.Now().Unix()*1000, 10)},
	}
	uri := "https://" + server + "/cgi-bin/mmwebwx-bin/synccheck?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
	}

	body, _ := httpclient.Get(uri).SetTimeout(time.Duration(30) * time.Second).SetCookies(cookies).SendBody(js).Text()
	logger.Tracef("SyncCheck body=%v", body)

	reg := regexp.MustCompile("window.synccheck={retcode:\"(\\d+)\",selector:\"(\\d+)\"}")
	sub := reg.FindStringSubmatch(body)

	if len(sub) < 2 {
		logger.Errorf("SyncCheck error, body=%v", body)
		return 0, 0, nil
	}

	retcode, _ := strconv.Atoi(sub[1])
	selector, _ := strconv.Atoi(sub[2])
	return retcode, selector, nil
}

// WebWxSync webwxsync api
func WebWxSync(common *Common,
	ce *XmlConfig,
	cookies []*http.Cookie,
	msg chan []byte, skl *SyncKeyList) error {

	km := url.Values{
		"skey":        []string{ce.Skey},
		"sid":         []string{ce.Wxsid},
		"lang":        []string{common.Lang},
		"pass_ticket": []string{ce.PassTicket},
	}

	uri := common.CgiUrl + "/webwxsync?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		SyncKey: skl,
		rr:      ^int(time.Now().Unix()) + 1,
	}

	body, _ := httpclient.Post(uri).SetTimeout(time.Duration(10) * time.Second).SetCookies(cookies).SendBody(js).Bytes()
	logger.Traceln("WebWxSync body=%v", string(body))

	jc, err := config.LoadJsonConfigFromBytes(body)
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

	msg <- body

	skl.List = skl.List[:0]
	skl1, _ := GetSyncKeyListFromJc(jc)
	skl.Count = skl1.Count
	skl.List = append(skl.List, skl1.List...)
	return nil
}

// WebWxStatusNotify webwxstatusnotify api
func WebWxStatusNotify(common *Common, ce *XmlConfig, bot *User) (int, error) {
	km := url.Values{
		"pass_ticket": []string{ce.PassTicket},
		"lang":        []string{common.Lang},
	}
	uri := common.CgiUrl + "/webwxstatusnotify?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Code:         3,
		FromUserName: bot.UserName,
		ToUserName:   bot.UserName,
		ClientMsgId:  int(time.Now().Unix()),
	}

	body, _ := httpclient.Post(uri).SendBody(js).Bytes()
	logger.Traceln("SyncCheck body=%v", string(body))

	jc, _ := config.LoadJsonConfigFromBytes(body)
	ret, _ := jc.GetInt("BaseResponse.Ret")
	return ret, nil
}

// WebWxGetContact webwxgetcontact api
func WebWxGetContact(common *Common, ce *XmlConfig, cookies []*http.Cookie) ([]byte, error) {
	km := url.Values{
		"r":    []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"seq":  []string{"0"},
		"skey": []string{ce.Skey},
	}
	uri := common.CgiUrl + "/webwxgetcontact?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxGetContact body=%v", string(body))

	return body, nil
}

// WebWxSendMsg webwxsendmsg api
func WebWxSendMsg(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	from, to string, msg string) ([]byte, error) {

	km := url.Values{
		"pass_ticket": []string{ce.PassTicket},
	}

	uri := common.CgiUrl + "/webwxsendmsg?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Msg: &TextMessage{
			Type:         1,
			Content:      msg,
			FromUserName: from,
			ToUserName:   to,
			LocalID:      int(time.Now().Unix() * 1e4),
			ClientMsgId:  int(time.Now().Unix() * 1e4),
		},
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Traceln("WebWxSendMsg body=%v", string(body))
	return body, nil
}

// WebWxUploadMedia webwxuploadmedia api
func WebWxUploadMedia(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	filename string, content []byte) (string, error) {

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, _ := w.CreateFormFile("filename", filename)
	if _, err := io.Copy(fw, bytes.NewReader(content)); err != nil {
		return "", err
	}

	ss := strings.Split(filename, ".")
	if len(ss) != 2 {
		return "", fmt.Errorf("file type suffix not found")
	}
	suffix := ss[1]

	fw, _ = w.CreateFormField("id")
	fw.Write([]byte("WU_FILE_" + strconv.Itoa(int(common.MediaCount))))
	common.MediaCount = atomic.AddUint32(&common.MediaCount, 1)

	fw, _ = w.CreateFormField("name")
	fw.Write([]byte(filename))

	fw, _ = w.CreateFormField("type")
	if suffix == "gif" {
		fw.Write([]byte("image/gif"))
	} else {
		fw.Write([]byte("image/jpeg"))
	}

	fw, _ = w.CreateFormField("lastModifieDate")
	fw.Write([]byte("Mon Feb 13 2017 17:27:23 GMT+0800 (CST)"))

	fw, _ = w.CreateFormField("size")
	fw.Write([]byte(strconv.Itoa(len(content))))

	fw, _ = w.CreateFormField("mediatype")
	if suffix == "gif" {
		fw.Write([]byte("doc"))
	} else {
		fw.Write([]byte("pic"))
	}

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		ClientMediaId: int(time.Now().Unix() * 1e4),
		TotalLen:      len(content),
		StartPos:      0,
		DataLen:       len(content),
		MediaType:     4,
	}

	jb, _ := json.Marshal(js)

	fw, _ = w.CreateFormField("uploadmediarequest")
	fw.Write(jb)

	fw, _ = w.CreateFormField("webwx_data_ticket")
	for _, v := range cookies {
		if strings.Contains(v.String(), "webwx_data_ticket") {
			fw.Write([]byte(strings.Split(v.String(), "=")[1]))
			break
		}
	}

	fw, _ = w.CreateFormField("pass_ticket")
	fw.Write([]byte(ce.PassTicket))
	w.Close()

	req, err := http.NewRequest("POST", common.UploadUrl, &b)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", w.FormDataContentType())
	req.Header.Add("User-Agent", common.UserAgent)

	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(common.UploadUrl)
	jar.SetCookies(u, cookies)
	client := &http.Client{Jar: jar}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	jc, err := config.LoadJsonConfigFromBytes(body)
	if err != nil {
		return "", err
	}
	ret, _ := jc.GetInt("BaseResponse.Ret")
	if ret != 0 {
		return "", fmt.Errorf("BaseResponse.Ret=%d", ret)
	}
	mediaID, _ := jc.GetString("MediaId")
	return mediaID, nil
}

// WebWxSendMsgImg webwxsendmsgimg api
func WebWxSendMsgImg(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	from, to, media string) (int, error) {

	km := url.Values{
		"pass_ticket": []string{ce.PassTicket},
		"fun":         []string{"async"},
		"f":           []string{"json"},
		"lang":        []string{common.Lang},
	}

	uri := common.CgiUrl + "/webwxsendmsgimg?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Msg: &MediaMessage{
			Type:         3,
			Content:      "",
			FromUserName: from,
			ToUserName:   to,
			LocalID:      int(time.Now().Unix() * 1e4),
			ClientMsgId:  int(time.Now().Unix() * 1e4),
			MediaId:      media,
		},
		Scene: 0,
	}
	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxSendMsgImg body=%v", string(body))

	jc, _ := config.LoadJsonConfigFromBytes(body)
	ret, _ := jc.GetInt("BaseResponse.Ret")
	return ret, nil
}

// WebWxGetMsgImg webwxgetmsgimg api
func WebWxGetMsgImg(common *Common, ce *XmlConfig, cookies []*http.Cookie, msgID string) ([]byte, error) {
	km := url.Values{
		"MsgID": []string{msgID},
		"skey":  []string{ce.Skey},
		"type":  []string{"slave"},
	}

	uri := common.CgiUrl + "/webwxgetmsgimg?" + km.Encode()

	body, _ := httpclient.Get(uri).SetCookies(cookies).Bytes()

	return body, nil
}

// WebWxSendEmoticon webwxsendemoticon api
func WebWxSendEmoticon(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	from, to, media string) (int, error) {

	km := url.Values{
		"fun":  []string{"sys"},
		"lang": []string{common.Lang},
	}

	uri := common.CgiUrl + "/webwxsendemoticon?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Msg: &EmotionMessage{
			Type:         47,
			EmojiFlag:    2,
			FromUserName: from,
			ToUserName:   to,
			LocalID:      int(time.Now().Unix() * 1e4),
			ClientMsgId:  int(time.Now().Unix() * 1e4),
			MediaId:      media,
		},
		Scene: 0,
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxSendEmoticon body=%v", string(body))

	jc, _ := config.LoadJsonConfigFromBytes(body)
	ret, _ := jc.GetInt("BaseResponse.Ret")
	return ret, nil
}

//WebWxGetIcon webwxgeticon api
func WebWxGetIcon(common *Common, ce *XmlConfig, cookies []*http.Cookie,
	username, chatroomid string) ([]byte, error) {
	km := url.Values{
		"seq":      []string{"0"},
		"username": []string{username},
		"skey":     []string{ce.Skey},
	}
	if chatroomid != "" {
		km.Add("chatroomid", chatroomid)
	}

	uri := common.CgiUrl + "/webwxgeticon?" + km.Encode()

	body, _ := httpclient.Get(uri).SetCookies(cookies).Bytes()
	logger.Traceln("WebWxGetIcon body=%v", string(body))
	return body, nil
}

// WebWxGetIconByHeadImgURL get head img
func WebWxGetIconByHeadImgURL(common *Common, ce *XmlConfig, cookies []*http.Cookie, headImgURL string) ([]byte, error) {
	uri := common.CgiDomain + headImgURL

	body, _ := httpclient.Get(uri).SetCookies(cookies).Bytes()
	logger.Debugf("WebWxGetIconByHeadImgURL body=%v", string(body))

	return body, nil
}

//WebWxBatchGetContact webwxbatchgetcontact api
func WebWxBatchGetContact(common *Common, ce *XmlConfig, cookies []*http.Cookie, cl []*User) ([]byte, error) {
	km := url.Values{
		"r":    []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"type": []string{"ex"},
	}
	uri := common.CgiUrl + "/webwxbatchgetcontact?" + km.Encode()

	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Count: len(cl),
		List:  cl,
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxBatchGetContact body=%v", string(body))
	return body, nil
}

// WebWxVerifyUser webwxverifyuser api
func WebWxVerifyUser(common *Common, ce *XmlConfig, cookies []*http.Cookie, opcode int, verifyContent string, vul []*VerifyUser) ([]byte, error) {
	km := url.Values{
		"r":           []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"pass_ticket": []string{ce.PassTicket},
	}

	uri := common.CgiUrl + "/webwxverifyuser?" + km.Encode()
	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		Opcode:             opcode,
		SceneList:          []int{33},
		SceneListCount:     1,
		VerifyContent:      verifyContent,
		VerifyUserList:     vul,
		VerifyUserListSize: len(vul),
		skey:               ce.Skey,
	}
	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxVerifyUser body=%v", string(body))
	return body, nil
}

// WebWxCreateChatroom webwxcreatechatroom api
func WebWxCreateChatroom(common *Common, ce *XmlConfig, cookies []*http.Cookie, users []*User, topic string) (interface{}, error) {
	km := url.Values{
		"r":           []string{strconv.FormatInt(time.Now().Unix(), 10)},
		"pass_ticket": []string{ce.PassTicket},
	}

	uri := common.CgiUrl + "/webwxcreatechatroom?" + km.Encode()
	js := InitReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		MemberCount: len(users),
		MemberList:  users,
		Topic:       topic,
	}
	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxCreateChatroom body=%v", string(body))
	return body, nil
}

// WebWxRevokeMsg webwxrevokemsg api
func WebWxRevokeMsg(common *Common, ce *XmlConfig, cookies []*http.Cookie, clientMsgID, svrMsgID, toUserName string) error {
	km := url.Values{
		"lang": []string{common.Lang},
	}

	uri := common.CgiUrl + "/webwxrevokemsg?" + km.Encode()
	js := RevokeReqBody{
		BaseRequest: &BaseRequest{
			ce.Wxuin,
			ce.Wxsid,
			ce.Skey,
			common.DeviceID,
		},
		ClientMsgId: clientMsgID,
		SvrMsgId:    svrMsgID,
		ToUserName:  toUserName,
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxRevokeMsg body=%v", string(body))

	jc, err := config.LoadJsonConfigFromBytes(body)
	if err != nil {
		return err
	}
	retcode, _ := jc.GetInt("BaseResponse.Ret")
	if retcode != 0 {
		return fmt.Errorf("BaseResponse.Ret %d", retcode)
	}
	return nil
}

//WebWxLogout webwxlogout api
func WebWxLogout(common *Common, ce *XmlConfig, cookies []*http.Cookie) error {
	km := url.Values{
		"redirect": []string{"1"},
		"type":     []string{"1"},
		"skey":     []string{ce.Skey},
	}

	uri := common.CgiUrl + "/webwxlogout?" + km.Encode()
	js := LogoutReqBody{
		uin: ce.Wxuin,
		sid: ce.Wxsid,
	}

	body, _ := httpclient.Post(uri).SetCookies(cookies).SendBody(js).Bytes()
	logger.Debugf("WebWxLogout body=%v", string(body))

	return nil
}
