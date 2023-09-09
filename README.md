# showtime

The **showtime**  API handles backend functionality required during live events. It
uses the Twitch API to register [event subscriptions](https://dev.twitch.tv/docs/eventsub/),
so that Twitch will call webhook handlers via `POST /callback` whenever stream-related
events occur.

## Prerequisites

Install [Go 1.21](https://go.dev/doc/install). If successful, you should be able to run:

```
> go version
go version go1.21.0 windows/amd64
```

## Initial setup

Create a file in the root of this repo called `.env` that contains the environment
variables required in [`main.go`](./cmd/server/main.go). If you have the
[`terraform`](https://github.com/golden-vcr/terraform) repo cloned alongside this one,
simply open a shell there and run:

- `terraform output -raw twitch_api_env > ../showtime/.env`

The Golden VCR Twitch App (identified by the Client ID etc. configured in Terraform) is
intended for use with a single Twitch channel: [GoldenVCR](https://www.twitch.tv/goldenvcr).

In order for this server to be notified when events occur on that channel, we need to
use the [EventSub API](https://dev.twitch.tv/docs/eventsub/) to register webhooks that
Twitch will call in response to those events.

[`server_callback.go`](./internal/server/server_callback.go) implements the request
handler for those webhooks. This server must be running, with that route accessible via
`https://goldenvcr.com/api/showtime/callback`, before the EventSub API will allow event
subscriptions to be created.

The required set of event subscriptions is defined in [`events.go`](./events.go). A
helper program defined in [`cmd/init/main.go`](./cmd/init/main.go) helps to automate
the process of ensuring that the requisite subscriptions are created through the Twitch
API.

Once your `.env` file is populated and this server is deployed to `goldenvcr.com`, run:

- `go run cmd/init/main.go`

If the command exits successfully, then all required event subscriptions exist, and the
server _should_ receive all the events it needs to make Twitch magic happen.

Note that while creating webhook subscriptions via the EventSub API requires an
application access token to authorize the requests, the API will only allow
subscriptions to be established for a given Twitch channel if the user (i.e.
[GoldenVCR](https://www.twitch.tv/goldenvcr)) has previoulsy granted access to the app
with the requisite scopes, via one of the OAuth flows described here:

- https://dev.twitch.tv/docs/authentication/

When you run the `init` program, it will open a browser window and prompt you for
access. The code in [`authflow.go`](./internal/eventsub/authflow.go) implements the
client-side logic for this auth flow.

## Running

Once your `.env` file is populated, you should be able to build and run the server:

- `go run cmd/server/main.go`

If successful, you should be able to run `curl http://localhost:5001/status` and
receive a response.
