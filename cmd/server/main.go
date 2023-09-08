package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"

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

	eventsubClient, err := eventsub.NewClient(config.TwitchChannelName, config.TwitchClientId, config.TwitchClientSecret, config.TwitchWebhookCallbackUrl, config.TwitchWebhookSecret)
	if err != nil {
		log.Fatalf("error initializing Twitch EventSub API client: %v", err)
	}

	srv := server.New(eventsubClient, config.TwitchWebhookSecret)

	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.ListenPort)
	fmt.Printf("Listening on %s...\n", addr)
	err = http.ListenAndServe(addr, srv)
	if err == http.ErrServerClosed {
		fmt.Printf("Server closed.\n")
	} else {
		log.Fatalf("error running server: %v", err)
	}
}
