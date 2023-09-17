package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/eventsub"
	"github.com/golden-vcr/showtime/internal/server"
)

type Config struct {
	BindAddr   string `env:"BIND_ADDR"`
	ListenPort uint16 `env:"LISTEN_PORT" default:"5001"`

	TwitchChannelName        string `env:"TWITCH_CHANNEL_NAME" required:"true"`
	TwitchClientId           string `env:"TWITCH_CLIENT_ID" required:"true"`
	TwitchClientSecret       string `env:"TWITCH_CLIENT_SECRET" required:"true"`
	TwitchWebhookCallbackUrl string `env:"TWITCH_WEBHOOK_CALLBACK_URL" default:"https://goldenvcr.com/api/showtime/callback"`
	TwitchWebhookSecret      string `env:"TWITCH_WEBHOOK_SECRET" required:"true"`
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("error loading .env file: %v", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer close()

	eventsubClient, err := eventsub.NewClient(
		config.TwitchChannelName,
		config.TwitchClientId,
		config.TwitchClientSecret,
		config.TwitchWebhookCallbackUrl,
		config.TwitchWebhookSecret,
		showtime.RequiredSubscriptions)
	if err != nil {
		log.Fatalf("error initializing Twitch EventSub API client: %v", err)
	}

	chatClient, err := chat.NewClient(ctx, config.TwitchChannelName)
	if err != nil {
		log.Fatalf("error initializing Twitch IRC chat client: %v", err)
	}

	srv := server.New(eventsubClient, chatClient, config.TwitchWebhookSecret)
	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.ListenPort)
	server := &http.Server{Addr: addr, Handler: srv}

	fmt.Printf("Listening on %s...\n", addr)
	var wg errgroup.Group
	wg.Go(server.ListenAndServe)

	select {
	case <-ctx.Done():
		fmt.Printf("Received signal; closing server...\n")
		server.Shutdown(context.Background())
	}

	err = wg.Wait()
	if err == http.ErrServerClosed {
		fmt.Printf("Server closed.\n")
	} else {
		log.Fatalf("error running server: %v", err)
	}
}
