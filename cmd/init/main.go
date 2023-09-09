package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/codingconcepts/env"
	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/internal/eventsub"
	"github.com/joho/godotenv"
	"github.com/nicklaw5/helix/v2"
)

type Config struct {
	TwitchChannelName        string `env:"TWITCH_CHANNEL_NAME" required:"true"`
	TwitchClientId           string `env:"TWITCH_CLIENT_ID" required:"true"`
	TwitchClientSecret       string `env:"TWITCH_CLIENT_SECRET" required:"true"`
	TwitchWebhookCallbackUrl string `env:"TWITCH_WEBHOOK_CALLBACK_URL" default:"https://goldenvcr.com/api/showtime/callback"`
	TwitchWebhookSecret      string `env:"TWITCH_WEBHOOK_SECRET" required:"true"`
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
	fmt.Printf("Configured for Twitch channel: %s\n", config.TwitchChannelName)

	// Open a web browser and prompt the user to log into their Twitch account and
	// grant access to the Golden VCR app with the requested scopes. This initiates an
	// OAuth code grant flow, giving us an authorization code that can be exchanged for
	// a user access token.
	//
	// Note that our Twitch app MUST be configured with a redirect URL matching the
	// supplied port (e.g. 'http://localhost:3033/auth'), and that port must be free
	// for us to run a small HTTP server on for the duration of this function call
	//
	// Note also that we don't actually need to exchange the authoriztaion code for a
	// user access token, because all of the EventSub API operations used by this
	// program (i.e. creating subscriptions and querying the status of existing
	// subscriptions) use an application access token. However, the individual events
	// that we're subscribing to still require that the user (i.e. the Twitch channel)
	// we're getting events for has authorized our app. Until the user explicitly
	// grants authorization via an in-browser OAuth flow, the Twitch API will respond
	// with 403 errors when we attempt to create EventSub subscriptions.
	_, err = eventsub.PromptForCodeGrant(context.Background(), config.TwitchClientId, showtime.GetRequiredUserScopes(), 3033)
	if err != nil {
		log.Fatalf("failed to get user authorization: %v", err)
	}

	// Initialize a Twitch API client so we can use EventSub API endpoints to manage
	// event subscriptions
	client, err := eventsub.NewClient(
		config.TwitchChannelName,
		config.TwitchClientId,
		config.TwitchClientSecret,
		config.TwitchWebhookCallbackUrl,
		config.TwitchWebhookSecret,
		showtime.RequiredSubscriptions)
	if err != nil {
		log.Fatalf("failed to initialize Twitch API client: %v", err)
	}

	// Query the API to get a list of all current subscriptions that are relevant to
	// our app
	subscriptions, err := client.GetOwnedSubscriptions()
	if err != nil {
		log.Fatalf("failed to get list of subscriptions from Twitch API: %v", err)
	}
	fmt.Printf("\nFound %d existing subscriptions:\n", len(subscriptions))
	for _, subscription := range subscriptions {
		fmt.Printf("- [%s] %s v%s (%s)\n", subscription.Status, subscription.Type, subscription.Version, subscription.ID)
	}

	// Reconcile that list against the declared set of subscriptions that we require
	reconciled, err := client.ReconcileRequiredSubscriptions(subscriptions)
	if err != nil {
		log.Fatalf("failed to reconcile required subscriptions: %v", err)
	}

	// First iterate through any required subscriptions that already exist: if their
	// status is pending or enabled, then we don't need to do anything; otherwise we
	// want to delete them
	nonEnabledSubscriptions := make([]eventsub.ExistingSubscription, 0, len(reconciled.Existing))
	for _, existing := range reconciled.Existing {
		if existing.Value.Status != helix.EventSubStatusPending && existing.Value.Status != helix.EventSubStatusEnabled {
			nonEnabledSubscriptions = append(nonEnabledSubscriptions, existing)
		}
	}

	// If we have subscriptions that we need to delete, get rid of them before doing
	// anything else, but require that the user has confirmed by passing a flag
	allowDelete := len(os.Args) > 1 && (os.Args[1] == "-f" || os.Args[1] == "--force")
	if len(nonEnabledSubscriptions) > 0 || len(reconciled.ToDelete) > 0 {
		fmt.Printf("\nThe following subscriptions need to be deleted:\n")
		for _, existing := range nonEnabledSubscriptions {
			fmt.Printf("- %s (%s): status is '%s'\n", existing.Value.ID, existing.Value.Type, existing.Value.Status)
		}
		for _, subscription := range reconciled.ToDelete {
			fmt.Printf("- %s (%s): notifies %s but is not declared in RequiredSubscriptions\n", subscription.ID, subscription.Type, config.TwitchWebhookCallbackUrl)
		}
		if !allowDelete {
			fmt.Printf("Please re-run with -f/--force if you wish to delete these subscriptions.\n")
			fmt.Printf("New subscriptions will not be created until deletion occurs.\n")
			os.Exit(1)
		}
		for _, existing := range nonEnabledSubscriptions {
			fmt.Printf("Deleting subscription %s...\n", existing.Value.ID)
			if err := client.DeleteSubscription(existing.Value.ID); err != nil {
				log.Fatalf("Failed to delete subscription %s: %v\n", existing.Value.ID, err)
			}

			// Ensure that we recreate this subscription (in the hopes that it will
			// become successfully enabled)
			fmt.Printf("A new '%s' subscription will be created to replace subscription %s.\n", existing.Required.Type, existing.Value.ID)
			reconciled.ToCreate = append(reconciled.ToCreate, existing.Required)
		}
		for _, subscription := range reconciled.ToDelete {
			fmt.Printf("Deleting subscription %s...\n", subscription.ID)
			if err := client.DeleteSubscription(subscription.ID); err != nil {
				log.Fatalf("Failed to delete subscription %s\n", subscription.ID)
			}
		}
		fmt.Printf("Deletion complete.\n")
	}

	// Create any new subscriptions that are required according to events.go but that
	// don't currently exist
	for _, required := range reconciled.ToCreate {
		fmt.Printf("\nCreating a new '%s' v%s subscription...\n", required.Type, required.Version)
		if err := client.CreateSubscription(required); err != nil {
			log.Fatalf("Failed to create subscription: %v", err)
		}
		fmt.Printf("Relevant %s events will now notify: %s\n", required.Type, config.TwitchWebhookCallbackUrl)
	}

	fmt.Printf("\nAll required subscriptions to %s (as declared in events.go) exist.\n", config.TwitchWebhookCallbackUrl)
}
