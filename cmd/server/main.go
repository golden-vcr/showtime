package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codingconcepts/env"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/server-common/db"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/admin"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/golden-vcr/showtime/internal/broadcast"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/golden-vcr/showtime/internal/health"
	"github.com/golden-vcr/showtime/internal/history"
	"github.com/golden-vcr/showtime/internal/imagegen"
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

	OpenaiApiKey string `env:"OPENAI_API_KEY" required:"true"`

	SpacesBucketName     string `env:"SPACES_BUCKET_NAME" required:"true"`
	SpacesRegionName     string `env:"SPACES_REGION_NAME" required:"true"`
	SpacesEndpointOrigin string `env:"SPACES_ENDPOINT_URL" required:"true"`
	SpacesAccessKeyId    string `env:"SPACES_ACCESS_KEY_ID" required:"true"`
	SpacesSecretKey      string `env:"SPACES_SECRET_KEY" required:"true"`

	AuthURL   string `env:"AUTH_URL" default:"http://localhost:5002"`
	LedgerURL string `env:"LEDGER_URL" default:"http://localhost:5003"`

	DatabaseHost     string `env:"PGHOST" required:"true"`
	DatabasePort     int    `env:"PGPORT" required:"true"`
	DatabaseName     string `env:"PGDATABASE" required:"true"`
	DatabaseUser     string `env:"PGUSER" required:"true"`
	DatabasePassword string `env:"PGPASSWORD" required:"true"`
	DatabaseSslMode  string `env:"PGSSLMODE"`

	LedgerShowtimeSecretKey string `env:"LEDGER_SHOWTIME_SECRET_KEY"`
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
	connectionString := db.FormatConnectionString(
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

	// Using the same connection string, prepare a broadcast.ChangeListener, which will
	// maintain a dedicated connection to the postgres server and use LISTEN to receive
	// asynchronous notifications (via the 'showtime' NOTIFY channel) whenever broadcast
	// or screening records are inserted/updated
	pqListener := pq.NewListener(connectionString, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		switch ev {
		case pq.ListenerEventConnected:
			fmt.Printf("pq listener connected (err: %v)\n", err)
		case pq.ListenerEventDisconnected:
			fmt.Printf("pq listener disconnected (err: %v)\n", err)
		case pq.ListenerEventReconnected:
			fmt.Printf("pq listener reconnected (err: %v)\n", err)
		case pq.ListenerEventConnectionAttemptFailed:
			fmt.Printf("pq listener connection attempt failed (err: %v)\n", err)
		}
		if err != nil {
			log.Fatalf("pq.Listener failed: %v", err)
		}
	})
	changeListener, err := broadcast.NewChangeListener(ctx, pqListener, q)
	if err != nil {
		log.Fatalf("failed to initialize ChangeListener: %v", err)
	}
	go func() {
		err := changeListener.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("ChangeListener got an error: %v", err)
		}
	}()

	// We need an auth client in order to validate access tokens, and we need a ledger
	// client in order to perform operations that require deducting Golden VCR Fun
	// Points from the auth'd user's balance
	authClient := auth.NewClient(config.AuthURL)
	ledgerClient := ledger.NewClient(config.LedgerURL, config.LedgerShowtimeSecretKey)

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
	alertsChan := make(chan *alerts.Alert, 32)
	{
		// events.Handler gets called in response to EventSub notifications, and
		// whenever it decides that we should broadcast an alert, it write a new
		// alert.Alert into alertsChan
		eventsHandler := events.NewHandler(ctx, q, alertsChan, ledgerClient)

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

	// POST /admin/tape/:id etc. allow the broadcaster to update the state of streams
	{
		adminServer := admin.NewServer(q)
		adminServer.RegisterRoutes(authClient, r.PathPrefix("/admin").Subrouter())
	}

	// GET /state provides clients with real-time information about the current state of
	// the broadcast: whether we've live, what tape is being screened, etc.
	{
		stateHandler := sse.NewHandler(ctx, changeListener.GetStateChanges())
		stateHandler.OnConnectEventFunc = func() broadcast.State {
			return changeListener.GetState()
		}
		r.Path("/state").Methods("GET").Handler(stateHandler)
	}

	// GET /history exposes endpoints that provide information about past broadcasts
	{
		historyServer := history.NewServer(q)
		historyServer.RegisterRoutes(r.PathPrefix("/history").Subrouter())
	}

	// POST /image-gen allows requests to be submitted for image generation
	{
		imageGeneration := imagegen.NewGenerationClient(config.OpenaiApiKey)
		imageStorage, err := imagegen.NewStorageClient(
			config.SpacesAccessKeyId,
			config.SpacesSecretKey,
			config.SpacesEndpointOrigin,
			config.SpacesRegionName,
			config.SpacesBucketName,
		)
		if err != nil {
			log.Fatalf("Failed to initialize storage client for image generation: %v", err)
		}
		imagegenServer := imagegen.NewServer(q, ledgerClient, imageGeneration, imageStorage, alertsChan)
		imagegenServer.RegisterRoutes(authClient, r.PathPrefix("/image-gen").Subrouter())
	}

	// Inject CORS support, since some of these APIs need to be called from the Golden
	// VCR extension, which is hosted by Twitch
	withCors := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://localhost:5180",
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
