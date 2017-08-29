package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/lets-go-go/logger"
	"github.com/robot4s/wechat/appconf"
	"github.com/robot4s/wechat/httpserver/controller"
)

var (
	// R Instantiate a new router
	R    *httprouter.Router
	port string
	srv  *http.Server
)

func init() {
	R = httprouter.New()

	// 获取天气信息
	R.GET("/wechat-login", controller.LoginQR)
}

// NewHTTPServer 创建Http服务
func NewHTTPServer() {

	port = fmt.Sprintf(":%d", appconf.HTTPSrvPort)

	srv = &http.Server{Addr: port, Handler: R}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil {
			logger.Errorf("[HttpSrv] quit. listen error: %+v", err)
			os.Exit(200)
		}

		logger.Infof("[HttpSrv] listening at: %s", srv.Addr)
	}()
}

// CloseHTTPServer 关闭HTTP服务
func CloseHTTPServer(ctx context.Context) {
	srv.Shutdown(ctx)
}
