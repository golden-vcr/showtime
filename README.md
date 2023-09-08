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

## Running

Create a file in the root of this repo called `.env` that contains the environment
variables required in [`main.go`](./cmd/server/main.go). If you have the
[`terraform`](https://github.com/golden-vcr/terraform) repo cloned alongside this one,
simply open a shell there and run:

- `terraform output -raw twitch_api_env > ../showtime/.env`

Once your `.env` file is populated, you should be able to build and run the server:

- `go run cmd/server/main.go`

If successful, you should be able to run `curl -X POST http://localhost:5001/callback`
and receive a 400 error, and the server should print
`Failed to verify signature from callback request`.
