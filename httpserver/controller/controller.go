package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
)

// LoginQR 登录QR
func LoginQR(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	filePath := fmt.Sprintf("%s/qr.jpg", appconf.AppdataDir)

	data, err := ioutil.ReadFile(filePath)

	if err != nil {
		// Write status
		w.WriteHeader(404)
		logger.Errorf("[HttpSrv] Get Logo error: %v", err)
	} else {
		// Write status
		w.WriteHeader(200)
		w.Write(data)
	}

}
