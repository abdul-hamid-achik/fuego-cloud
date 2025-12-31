#!/bin/bash
set -e

echo "Setting up local development environment..."

# Check for required tools
command -v docker >/dev/null 2>&1 || { echo "docker is required but not installed. Aborting." >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo "go is required but not installed. Aborting." >&2; exit 1; }

# Install sqlc if not present
if ! command -v sqlc &> /dev/null; then
    echo "Installing sqlc..."
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
fi

# Install golang-migrate if not present
if ! command -v migrate &> /dev/null; then
    echo "Installing golang-migrate..."
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

# Install task if not present
if ! command -v task &> /dev/null; then
    echo "Installing task..."
    go install github.com/go-task/task/v3/cmd/task@latest
fi

# Copy .env.example to .env if .env doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo "Please update .env with your credentials"
fi

# Generate sqlc code
echo "Generating sqlc code..."
sqlc generate

# Start database
echo "Starting database..."
docker compose up -d db

# Wait for database to be ready
echo "Waiting for database to be ready..."
sleep 10

# Run migrations
echo "Running migrations..."
migrate -path db/migrations -database "$DATABASE_URL" up || true

echo ""
echo "Local environment ready!"
echo "Run 'task dev' to start the development server"
