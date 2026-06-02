package service

import (
	"anonymous-chat/internal/chat"
	"context"
	"io"
)

// ChunkedTransferWriter — для исходящего стриминга Chunked transfer encoding
type ChunkedTransferWriter interface {
	WriteAndFlush(data string) error
}

// RoomProvider — управление комнатами (общий для GET/POST)
type RoomProvider interface {
	GetOrCreate(name string) *chat.Room
	Get(name string) (*chat.Room, bool)
}

// MessagePoster — для отправки сообщений (POST)
type MessagePoster interface {
	PostMessage(ctx context.Context, roomName string, msg MessageData) error
}

// ChatStreamer — для стриминга в комнате (GET)
type ChatStreamer interface {
	StreamChat(ctx context.Context, w ChunkedTransferWriter, roomName string) error
}

// AnonymousChat предоставляет методы для управления анонимными чат-комнатами,
// включая регистрацию участников, отправку сообщений и потоковую передачу событий.
// Реализация потокобезопасна и поддерживает корректное освобождение ресурсов через Close()
type AnonymousChat interface {
	RoomProvider
	MessagePoster
	ChatStreamer
	io.Closer
}
