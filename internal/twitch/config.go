package twitch

type Config struct {
	ChannelName        string `env:"TWITCH_CHANNEL_NAME" required:"true"`
	ClientId           string `env:"TWITCH_CLIENT_ID" required:"true"`
	ClientSecret       string `env:"TWITCH_CLIENT_SECRET" required:"true"`
	ExtensionClientId  string `env:"TWITCH_EXTENSION_CLIENT_ID" required:"true"`
	WebhookCallbackUrl string `env:"TWITCH_WEBHOOK_CALLBACK_URL" default:"https://goldenvcr.com/api/showtime/callback"`
	WebhookSecret      string `env:"TWITCH_WEBHOOK_SECRET" required:"true"`
}
