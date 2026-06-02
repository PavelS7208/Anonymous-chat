package main

import (
	"anonymous-chat/internal/service"
	"context"
	"io"
	"log"
	"net/http"
)

type AnonymousChatBackender interface {
	service.MessagePoster
	service.ChatStreamer
	io.Closer
}

// compile-time проверка: AnonymousChat реализует наш узкий интерфейс
var _ AnonymousChatBackender = service.AnonymousChat(nil)

type AnonymousChatApp struct {
	chatBackender AnonymousChatBackender // узкий контракт
	httpSrv       *http.Server
	logger        *log.Logger
	addr          string
}

type Runner interface {
	Run() error
	Close(context.Context) error
}

var _ Runner = (*AnonymousChatApp)(nil) // compile-time check
