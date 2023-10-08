package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/codingconcepts/env"
	"github.com/golden-vcr/server-common/db"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Config struct {
	DatabaseHost     string `env:"PGHOST" required:"true"`
	DatabasePort     int    `env:"PGPORT" required:"true"`
	DatabaseName     string `env:"PGDATABASE" required:"true"`
	DatabaseUser     string `env:"PGUSER" required:"true"`
	DatabasePassword string `env:"PGPASSWORD" required:"true"`
	DatabaseSslMode  string `env:"PGSSLMODE"`
}

func main() {
	// Initialize config from environment vars
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("error loading .env file: %v", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		log.Fatalf("error parsing config: %v", err)
	}

	// Construct a postgres connection string from our config
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

	// Verify that we can connect to the database
	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}

	// Print the current tape ID
	q := queries.New(db)
	currentTapeId, err := q.GetCurrentTapeId(context.Background())
	if err == sql.ErrNoRows {
		currentTapeId = ""
	} else if err != nil {
		log.Fatalf("error getting current tape ID: %v", err)
	}
	if currentTapeId == "" {
		fmt.Printf("current tape: <n/a>\n")
	} else {
		fmt.Printf("current tape: %s\n", currentTapeId)
	}

	// If called with 'set-tape <tape-id>', update the current tape
	if len(os.Args) >= 3 && os.Args[1] == "set-tape" {
		newTapeId := os.Args[2]
		if newTapeId != currentTapeId {
			fmt.Printf("setting current tape to: %s\n", newTapeId)
			if err := q.SetTapeId(context.Background(), newTapeId); err != nil {
				log.Fatalf("error setting tape ID: %v", err)
			}
		}
	} else if len(os.Args) >= 2 && os.Args[1] == "clear-tape" {
		if currentTapeId != "" {
			fmt.Printf("clearing current tape\n")
			if err := q.ClearTapeId(context.Background()); err != nil {
				log.Fatalf("error clearing tape ID: %v", err)
			}
		}
	}
}
