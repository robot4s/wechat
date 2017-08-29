package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/robot"
)

// Login 登录
func Login(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	uuid := robot.StartRobot()

	if uuid != "" {
		filePath := fmt.Sprintf("%s/%s.jpg", appconf.QRDir, uuid)
		data, err := ioutil.ReadFile(filePath)

		if err != nil {
			// Write status
			w.WriteHeader(404)
			logger.Errorf("[HttpSrv] Login error: %+v", err)
		} else {
			// Write status
			w.WriteHeader(200)
			w.Write(data)
		}
	} else {
		// Write status
		w.WriteHeader(200)
		w.Write([]byte("login failed"))
	}

}
