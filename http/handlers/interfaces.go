package handlers

import (
	"net/http"

	"whatsapp-notifier/http/middleware"
)

type Router interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	HandleWithMiddleware(pattern string, handler http.Handler, middlewares ...middleware.Middleware)
	HandleFuncWithMiddleware(pattern string, handler func(http.ResponseWriter, *http.Request), middlewares ...middleware.Middleware)
}
