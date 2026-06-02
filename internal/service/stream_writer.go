package service

import (
	"context"
	"net/http"

	"github.com/pavel/anonymous-chat/internal/adapters/stream"
)

type ChunkedWriter interface {
	Write(context.Context, []byte) error
}

type ChatWriterImp struct {
	stream ChunkedWriter
}

func NewChatWriter(w ChunkedWriter) *ChatWriterImp {
	return &ChatWriterImp{stream: w}
}

func (w ChatWriterImp) Write(ctx context.Context, wm WireMessage) error {
	return w.stream.Write(ctx, wm.Marshal())
}

func NewChatWriterFromHTTP(w http.ResponseWriter) (ChatWriter, error) {
	chunked, err := stream.NewChunkedStreamer(w)
	if err != nil {
		return nil, err
	}
	return NewChatWriter(chunked), nil
}
