#!/bin/bash
set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | xargs)
fi

if [ -z "$DATABASE_URL" ]; then
    echo "DATABASE_URL is not set"
    exit 1
fi

case "$1" in
    up)
        echo "Running migrations..."
        migrate -path db/migrations -database "$DATABASE_URL" up
        ;;
    down)
        echo "Rolling back last migration..."
        migrate -path db/migrations -database "$DATABASE_URL" down 1
        ;;
    drop)
        echo "Dropping all migrations..."
        migrate -path db/migrations -database "$DATABASE_URL" drop -f
        ;;
    version)
        echo "Current migration version:"
        migrate -path db/migrations -database "$DATABASE_URL" version
        ;;
    *)
        echo "Usage: $0 {up|down|drop|version}"
        exit 1
        ;;
esac
