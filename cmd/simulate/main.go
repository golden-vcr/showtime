package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codingconcepts/env"
	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/golden-vcr/showtime/internal/twitch"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/nicklaw5/helix/v2"
)

const (
	TwitchHeaderMessageId        = "twitch-eventsub-message-id"
	TwitchHeaderMessageTimestamp = "twitch-eventsub-message-timestamp"
	TwitchHeaderMessageSignature = "twitch-eventsub-message-signature"
)

type Config struct {
	TwitchChannelName   string `env:"TWITCH_CHANNEL_NAME" required:"true"`
	TwitchClientId      string `env:"TWITCH_CLIENT_ID" required:"true"`
	TwitchClientSecret  string `env:"TWITCH_CLIENT_SECRET" required:"true"`
	TwitchWebhookSecret string `env:"TWITCH_WEBHOOK_SECRET" required:"true"`
}

func main() {
	// We only want to simulate events locally for now; events that can be recorded in
	// the production DB and affect the state of the actual, deployed webapp should only
	// come from Twitch itself
	url := "http://localhost:5001/callback"

	// Parse config from environment variables
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("error loading .env file: %v", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// Parse the name of the subscription type we want to emulate, passed as a CLI arg
	if len(os.Args) <= 1 {
		log.Fatalf("Usage: simulate [channel.follow|...]")
	}
	subscriptionType := os.Args[1]

	// Get the user ID for the configured Twitch channel
	channelUserId, err := getChannelUserId(config.TwitchChannelName, config.TwitchClientId, config.TwitchClientSecret)
	if err != nil {
		log.Fatalf("error getting channel user ID: %v", err)
	}

	// Prepare the JSON-encoded message that we want to send
	payload := buildMessagePayload(subscriptionType, channelUserId, config.TwitchChannelName, url)
	messageBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("failed to encode message payload: %v", err)
	}
	message := string(messageBytes)

	// Prepare the HTTP request that will carry that message in its body
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		log.Fatalf("error initializing HTTP request: %v", err)
	}

	// Set Twitch-Eventsub-* headers to identify the message and cryptographically sign
	// it, using the webhook secret, in a way that helix.VerifyEventSubNotification can
	// verify
	req.Header.Set(TwitchHeaderMessageId, uuid.New().String())
	req.Header.Set(TwitchHeaderMessageTimestamp, time.Now().Format(time.RFC3339))
	req.Header.Set(TwitchHeaderMessageSignature, computeSignature(config.TwitchWebhookSecret, req.Header, message))

	// Print the details of the request to stdout
	fmt.Printf("%s %s\n", req.Method, req.URL)
	for k, values := range req.Header {
		for _, v := range values {
			fmt.Printf("> %s: %s\n", k, v)
		}
	}
	pretty, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		log.Fatalf("failed to pretty-print JSON payload: %v", err)
	}
	fmt.Printf("\n%s\n\n", pretty)

	// Send the request and verify that we get an OK response
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("error sending HTTP request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		log.Fatalf("got response %d", res.StatusCode)
	}
	fmt.Printf("< %d\n", res.StatusCode)
}

func getChannelUserId(channelName string, clientId string, clientSecret string) (string, error) {
	client, err := twitch.NewClientWithAppToken(clientId, clientSecret)
	if err != nil {
		return "", err
	}
	return twitch.GetChannelUserId(client, channelName)
}

type MessagePayload struct {
	Subscription helix.EventSubSubscription `json:"subscription"`
	Challenge    string                     `json:"challenge"`
	Event        json.RawMessage            `json:"event"`
}

func buildMessagePayload(subscriptionType string, channelUserId string, channelName string, webhookUrl string) MessagePayload {
	params := events.RequiredSubscriptionConditionParams{
		ChannelUserId: channelUserId,
	}

	p := MessagePayload{}
	for _, required := range showtime.RequiredSubscriptions {
		if required.Type == subscriptionType {
			p.Subscription.Type = subscriptionType
			p.Subscription.Version = required.Version
			cond, err := params.Format(&required.TemplatedCondition)
			if err != nil {
				log.Fatalf("failed to format subscription condition from template: %v", err)
			}
			p.Subscription.Condition = *cond
			break
		}
	}
	if p.Subscription.Type == "" {
		log.Fatalf("no subscription of type %s is required in events.go", subscriptionType)
	}
	p.Subscription.ID = uuid.New().String()
	p.Subscription.Status = helix.EventSubStatusEnabled
	p.Subscription.Transport.Method = "webhook"
	p.Subscription.Transport.Callback = webhookUrl
	p.Subscription.CreatedAt = helix.Time{Time: time.Now().Add(-5 * time.Minute)}

	p.Event = buildEventPayload(subscriptionType, channelUserId, channelName)

	return p
}

func buildEventPayload(subscriptionType string, channelUserId string, channelName string) json.RawMessage {
	var p any = nil
	switch subscriptionType {
	case helix.EventSubTypeStreamOnline:
		p = &helix.EventSubStreamOnlineEvent{
			ID:                   "9001",
			BroadcasterUserID:    channelUserId,
			BroadcasterUserLogin: channelName,
			BroadcasterUserName:  channelName,
			Type:                 "live",
			StartedAt:            helix.Time{Time: time.Now()},
		}
	case helix.EventSubTypeStreamOffline:
		p = &helix.EventSubStreamOfflineEvent{
			BroadcasterUserID:    channelUserId,
			BroadcasterUserLogin: channelName,
			BroadcasterUserName:  channelName,
		}
	case helix.EventSubTypeChannelFollow:
		p = &helix.EventSubChannelFollowEvent{
			UserID:               "1337",
			UserLogin:            "bigjim",
			UserName:             "BigJim",
			BroadcasterUserID:    channelUserId,
			BroadcasterUserLogin: channelName,
			BroadcasterUserName:  channelName,
			FollowedAt:           helix.Time{Time: time.Now()},
		}
	case helix.EventSubTypeChannelCheer:
		p = &helix.EventSubChannelCheerEvent{
			IsAnonymous:          false,
			UserID:               "1337",
			UserLogin:            "bigjim",
			UserName:             "BigJim",
			BroadcasterUserID:    channelUserId,
			BroadcasterUserLogin: channelName,
			BroadcasterUserName:  channelName,
			Message:              "ghost of a baby seal",
			Bits:                 200,
		}
	default:
		log.Fatalf("subscription type %s is not supported in buildEventPayload", subscriptionType)
	}

	data, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("failed to encode event for subscription type %s: %v", subscriptionType, err)
	}
	return data
}

func computeSignature(secret string, h http.Header, message string) string {
	hmacMessage := []byte(fmt.Sprintf("%s%s%s", h.Get(TwitchHeaderMessageId), h.Get(TwitchHeaderMessageTimestamp), message))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(hmacMessage)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}
