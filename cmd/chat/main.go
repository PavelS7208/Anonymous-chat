package main

import (
	"anonymous-chat/internal/chat"
	"anonymous-chat/internal/service"
	"context"
	"log"
	"time"
)

func main() {

	app := NewAnonymousChatApp(
		WithServerAddr(":8080"),
		WithTimeouts(5*time.Second, 60*time.Second),
		WithChatConfig(
			//chat.WithInitialHistoryLength(10),  // По дефолту InitialHistoryLength стоит 10 как в ТЗ
			chat.WithMaxHistoryStorageLength(1000),
			//chat.WithMaxMemberID(1000),  // По дефолту как в тз (1<<63)-1
			// chat.WithInitialHistoryCap(3000), // можно переопределить, по дефолту длина*2
			// chat.WithEventChannelBuf(64),    // По дефолту сейчас 64 (без тестов, просто на глаз)
		),
		WithServiceConfig(
			//service.WithMaxMessageBytes(512),   // По дефолту 1024 как в ТЗ, можно поменять
			service.WithRoomNamePattern(`^[a-zA-Z0-9_-]{1,64}$`), // По дефолту длина имени 32
		),
		// WithLogger(customLogger), // при необходимости
	)

	if err := app.Run(); err != nil {
		log.Printf("Server error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Close(ctx); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}
}
