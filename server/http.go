package server

import (
	"context"
	"net/http"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/seniorGolang/gokit/logger"
	"github.com/seniorGolang/gokit/utils"
)

var (
	log = logger.Log.WithField("module", "httpServer")
)

func StartFastHttpServer(handler http.Handler, address string) (srv *fasthttp.Server) {

	srv = &fasthttp.Server{
		MaxRequestBodySize: 200 * 1024 * 1024,
		Handler:            NewFastHTTPHandler(handler),
	}

	go func() {
		log.WithField("address", address).Info("listening")
		utils.ExitOnError(log, srv.ListenAndServe(address), "Could not start http server on: "+address)
	}()
	return
}

func ShutdownFastHttpServer(srv *fasthttp.Server) {

	err := srv.Shutdown()

	if err != nil {
		log.Error(err)
	}
	log.Info("shutdown server success")
}

func StartHttpServer(handler http.Handler, address string) (srv *http.Server) {

	srv = &http.Server{
		Addr:        address,
		ReadTimeout: time.Second * 15,
		IdleTimeout: time.Second * 60,
		Handler:     handler,
	}

	go func() {
		log.WithField("address", address).Info("listening")
		utils.ExitOnError(log, srv.ListenAndServe(), "Could not start http server on: "+address)
	}()
	return
}

func ShutdownHttpServer(srv *http.Server) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err := srv.Shutdown(ctx)

	if err != nil {
		log.Error(err)
	}
	log.Info("shutdown server success")
}
