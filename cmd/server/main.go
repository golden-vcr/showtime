package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/gen/queries"
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
	TwitchExtensionClientId  string `env:"TWITCH_EXTENSION_CLIENT_ID" required:"true"`
	TwitchWebhookCallbackUrl string `env:"TWITCH_WEBHOOK_CALLBACK_URL" default:"https://goldenvcr.com/api/showtime/callback"`
	TwitchWebhookSecret      string `env:"TWITCH_WEBHOOK_SECRET" required:"true"`

	DatabaseHost     string `env:"PGHOST" required:"true"`
	DatabasePort     int    `env:"PGPORT" required:"true"`
	DatabaseName     string `env:"PGDATABASE" required:"true"`
	DatabaseUser     string `env:"PGUSER" required:"true"`
	DatabasePassword string `env:"PGPASSWORD" required:"true"`
	DatabaseSslMode  string `env:"PGSSLMODE"`
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

	connectionString := formatConnectionString(
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseName,
		config.DatabaseUser,
		config.DatabasePassword,
		config.DatabaseSslMode,
	)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}
	q := queries.New(db)

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

	eventsChan := make(chan *chat.Event, 32)
	chatClient, err := chat.NewClient(ctx, config.TwitchChannelName, eventsChan)
	if err != nil {
		log.Fatalf("error initializing Twitch IRC chat client: %v", err)
	}
	srv := server.New(ctx, config.TwitchExtensionClientId, config.TwitchWebhookSecret, q, eventsubClient, chatClient, eventsChan)

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

func formatConnectionString(host string, port int, dbname string, user string, password string, sslmode string) string {
	urlencodedPassword := url.QueryEscape(password)
	s := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, urlencodedPassword, host, port, dbname)
	if sslmode != "" {
		s += fmt.Sprintf("?sslmode=%s", sslmode)
	}
	return s
}
