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
	"time"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/server"
	"github.com/golden-vcr/showtime/internal/twitch"
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
	// Parse config from environment variables
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("error loading .env file: %v", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// TODO: Embed in top-level Config struct
	twitchConfig := twitch.Config{
		ChannelName:        config.TwitchChannelName,
		ClientId:           config.TwitchClientId,
		ClientSecret:       config.TwitchClientSecret,
		ExtensionClientId:  config.TwitchExtensionClientId,
		WebhookCallbackUrl: config.TwitchWebhookCallbackUrl,
		WebhookSecret:      config.TwitchWebhookSecret,
	}

	// Shut down cleanly on
	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer close()

	// Configure our database connection and initialize a Queries struct, so we can read
	// and write to the 'showtime' schema in response to HTTP requests, EventSub
	// notifications, etc.
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

	//
	twitchClient, err := twitch.NewClientWithAppToken(twitchConfig.ClientId, twitchConfig.ClientSecret)
	if err != nil {
		log.Fatalf("error initializing Twitch API client: %v", err)
	}
	channelUserId, err := twitch.GetChannelUserId(twitchClient, twitchConfig.ChannelName)
	if err != nil {
		log.Fatalf("error getting Twitch channel user ID: %v", err)
	}

	// Create a chat.Agent which will listen to IRC chat and expose a stream of
	// chat.LogEvent via a channel
	chatAgent, err := chat.NewAgent(ctx, chat.NewLog(64), config.TwitchChannelName, time.Second)
	if err != nil {
		log.Fatalf("error initializing chat agent: %v", err)
	}
	defer chatAgent.Disconnect()

	// Initialize our HTTP server, which glues all of the above into a coherent set of
	// endpoints for clients to call
	srv := server.New(
		ctx,
		twitchConfig,
		twitchClient,
		channelUserId,
		q,
		chatAgent,
	)
	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.ListenPort)
	server := &http.Server{Addr: addr, Handler: srv}

	// Handle incoming HTTP connections until our top-level context is canceled, at
	// which point shut down cleanyl
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
