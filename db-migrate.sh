#!/usr/bin/env bash
set -e

GOLANG_MIGRATE_VERSION=v4.16.2

function require_env() {
    NAME="$1"
    VALUE="$2"
    if [ "$VALUE" == "" ]; then
        echo "ERROR: $NAME must be set"
        exit 1
    fi
}

function url_encode() {
    echo "$@" | sed -e 's/%/%25/g' -e 's/ /%20/g' -e 's/!/%21/g' -e 's/"/%22/g' -e "s/'/%27/g" -e 's/#/%23/g' -e 's/(/%28/g' -e 's/)/%29/g' -e 's/+/%2b/g' -e 's/,/%2c/g' -e 's/-/%2d/g' -e 's/:/%3a/g' -e 's/;/%3b/g' -e 's/?/%3f/g' -e 's/@/%40/g' -e 's/\$/%24/g' -e 's/\&/%26/g' -e 's/\*/%2a/g' -e 's/\./%2e/g' -e 's/\//%2f/g' -e 's/\[/%5b/g' -e 's/\\/%5c/g' -e 's/\]/%5d/g' -e 's/\^/%5e/g' -e 's/_/%5f/g' -e 's/`/%60/g' -e 's/{/%7b/g' -e 's/|/%7c/g' -e 's/}/%7d/g' -e 's/~/%7e/g'
}

if [ -f .env ]; then
    source .env
fi
if [ -f bin/.env ]; then
    source bin/.env
fi
require_env 'PGHOST' "$PGHOST"
require_env 'PGPORT' "$PGPORT"
require_env 'PGDATABASE' "$PGDATABASE"
require_env 'PGUSER' "$PGUSER"
require_env 'PGPASSWORD' "$PGPASSWORD"

SCHEMA_NAME=$(basename $(pwd))
POSTGRES_URI="postgres://$PGUSER:$(url_encode "$PGPASSWORD")@$PGHOST:$PGPORT/$PGDATABASE?search_path=public&x-migrations-table=${SCHEMA_NAME}_migrations"
if [ "$PGSSLMODE" != "" ]; then
    POSTGRES_URI="$POSTGRES_URI&sslmode=$PGSSLMODE"
fi

MIGRATE_COMMAND="$@"
if [ "$MIGRATE_COMMAND" == "" ]; then
    MIGRATE_COMMAND="up"
fi

go run \
    -tags 'postgres' \
    "github.com/golang-migrate/migrate/v4/cmd/migrate@$GOLANG_MIGRATE_VERSION" \
    -source file://db/migrations \
    -database "$POSTGRES_URI" \
    $MIGRATE_COMMAND
