# showtime

The **showtime**  API handles backend functionality required during live events. It
uses the Twitch API to register [event subscriptions](https://dev.twitch.tv/docs/eventsub/),
so that Twitch will call webhook handlers via `POST /callback` whenever stream-related
events occur.

- **OpenAPI specification:** https://golden-vcr.github.io/showtime/

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
- `terraform output -raw openai_api_env >> ../showtime/.env`
- `terraform output -raw discord_env >> ../showtime/.env`
- `terraform output -raw user_images_s3_env >> ../showtime/.env`
- `terraform output -raw ledger_s2s_auth_env >> ../showtime/.env`
- `./local-db.sh env >> ../showtime/.env`

### Running the database

This API stores persistent data in a PostgreSQL database. When running in a live
environment, each API has its own database, and connection details are configured from
Terraform secrets via .env files.

For local development, we run a self-contained postgres database in Docker, and all
server-side applications share the same set of throwaway credentials.

We use a script in the [`terraform`](https://github.com/golden-vcr/terraform) repo,
called `./local-db.sh`, to manage this local database. To start up a fresh database and
apply migrations, run:

- _(from `terraform`:)_ `./local-db.sh up`
- _(from `showtime`:)_ `./db-migrate.sh`

If you need to blow away your local database and start over, just run
`./local-db.sh down` and repeat these steps.

### Generating database queries

If you modify the SQL code in [`db/queries`](./db/queries/), you'll need to generate
new Go code to [`gen/queries`](./gen/queries/). To do so, simply run:

- `./db-generate-queries.sh`

### Creating Twitch webhooks

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
access. The code in [`authflow.go`](./internal/twitch/authflow.go) implements the
client-side logic for this auth flow.

## Running

Once your `.env` file is populated, you should be able to build and run the server:

- `go run cmd/server/main.go`

If successful, you should be able to run `curl http://localhost:5001/status` and
receive a response.

To simulate Twitch EventSub events when running locally, run the `simulate` command,
e.g.:

- `go run cmd/simulate/main.go stream.online`
- `go run cmd/simulate/main.go channel.follow`
- `go run cmd/simulate/main.go stream.offline`

## Auth dependency

Note that in order to call endpoints that require authorization, you'll need to be
running the [auth](https://github.com/golden-vcr/auth) API locally as well. By default,
**showtime** is configured to reach the auth server via its default URL of
`http://localhost:5002`, so it's sufficient to simply `go run cmd/server/main.go` from
both repos.

In production, **showtime** and **auth** currently run alongside each other on a single
DigitalOcean droplet, with auth listening on 5002, so no further configuration is
necessary. If we scale up beyond a single host, then showtime should be configured with
an appropriate `AUTH_URL` value to hit the production auth server.
