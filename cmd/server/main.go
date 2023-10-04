package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codingconcepts/env"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/admin"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/golden-vcr/showtime/internal/health"
	"github.com/golden-vcr/showtime/internal/sse"
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

	AuthURL string `env:"AUTH_URL" default:"http://localhost:5002"`

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

	// Shut down cleanly on signal
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

	// Prepare a Twitch client and use it to get the user ID for the configured channel,
	// so we can identify the broadcaster
	twitchClient, err := twitch.NewClientWithAppToken(config.TwitchClientId, config.TwitchClientSecret)
	if err != nil {
		log.Fatalf("error initializing Twitch API client: %v", err)
	}
	channelUserId, err := twitch.GetChannelUserId(twitchClient, config.TwitchChannelName)
	if err != nil {
		log.Fatalf("error getting Twitch channel user ID: %v", err)
	}

	// Start setting up our HTTP handlers, using gorilla/mux for routing
	r := mux.NewRouter()

	// Clients can hit GET /alerts to receive notifications in response to follows,
	// raids, etc.: these are largely initiated in response to Twitch EventSub callbacks
	{
		// events.Handler gets called in response to EventSub notifications, and
		// whenever it decides that we should broadcast an alert, it write a new
		// alert.Alert into alertsChan
		alertsChan := make(chan *alerts.Alert, 32)
		eventsHandler := events.NewHandler(ctx, q, alertsChan)

		// events.Server implements the POST callback that Twitch hits (once we've run
		// cmd/init/main.go to create all EventSub notifications mandated by events.go)
		// in order to let us know when relevant events occur on Twitch: it responds by
		// sending those events to the events.Handler
		eventsServer := events.NewServer(config.TwitchWebhookSecret, eventsHandler)
		r.Path("/callback").Methods("POST").Handler(eventsServer)

		// The sse.Handler exposes our Alert channel via an SSE endpoint, notifying HTTP
		// clients whenever a Twitch-initiated event results in a new alert
		alertsHandler := sse.NewHandler[*alerts.Alert](ctx, alertsChan)
		r.Path("/alerts").Methods("GET").Handler(alertsHandler)
	}

	// Clients can hit GET /chat to open an SSE connection into which we'll write chat
	// log events
	var getChatStatus health.GetChatStatusFunc
	{
		// The chat.Agent sits in IRC chat and interprets messages, writing to our
		// logEventsChan whenever the chat log UI should be updated
		logEventsChan := make(chan *chat.LogEvent, 32)
		chatAgent, err := chat.NewAgent(ctx, 64, logEventsChan, config.TwitchChannelName, time.Second)
		if err != nil {
			log.Fatalf("error initializing chat agent: %v", err)
		}
		getChatStatus = func() error {
			return chatAgent.GetStatus()
		}
		defer chatAgent.Disconnect()

		// The sse.Handler exposes that LogEvent channel via an SSE endpoint
		chatHandler := sse.NewHandler[*chat.LogEvent](ctx, logEventsChan)
		r.Path("/chat").Methods("GET").Handler(chatHandler)
	}

	// Clients can hit GET / to get the health of the Golden VCR Twitch integration,
	// with the response certifying whether all EventSub subscriptions are enabled and
	// the chat agent is connected to IRC
	{
		healthServer := health.NewServer(twitchClient, channelUserId, config.TwitchWebhookCallbackUrl, getChatStatus)
		r.Path("/").Methods("GET").Handler(healthServer)
	}

	// GET /admin/secrets is a temporary test endpoint that we can use to verify that
	// our new access control code (using the auth API) is working as intended: only the
	// broadcaster should be permitted to get data from this endpoint
	{
		authClient := auth.NewClient(config.AuthURL)
		adminServer := &admin.Server{}
		adminServer.RegisterRoutes(authClient, r.PathPrefix("/admin").Subrouter())
	}

	// GET /view exposes the currently-selected tape ID (WIP)
	{
		handleView := func(res http.ResponseWriter, req *http.Request) {
			// Look up the current tape ID, defaulting to "" if no tape change has ever been
			// recorded
			tapeId, err := q.GetCurrentTapeId(req.Context())
			if err == sql.ErrNoRows {
				tapeId = ""
			} else if err != nil {
				fmt.Printf("Error getting tape ID: %v\n", err)
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}

			// Respond with our current state
			type State struct {
				TapeId string `json:"tapeId"`
			}
			state := &State{
				TapeId: tapeId,
			}
			if err := json.NewEncoder(res).Encode(state); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
		r.Path("/view").Methods("GET").HandlerFunc(handleView)
	}

	// Inject CORS support, since some of these APIs need to be called from the Golden
	// VCR extension, which is hosted by Twitch
	withCors := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://localhost:8080",
			fmt.Sprintf("https://%s.ext-twitch.tv", config.TwitchExtensionClientId),
		},
		AllowedMethods: []string{http.MethodGet},
	})
	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.ListenPort)
	server := &http.Server{Addr: addr, Handler: withCors.Handler(r)}

	// Handle incoming HTTP connections until our top-level context is canceled, at
	// which point shut down cleanly
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
