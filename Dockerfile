FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git curl nodejs npm

# Install fuego CLI, templ, and sqlc
RUN go install github.com/abdul-hamid-achik/fuego/cmd/fuego@latest && \
    go install github.com/a-h/templ/cmd/templ@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build Tailwind CSS (Tailwind v4)
RUN if [ -f "styles/input.css" ]; then \
    npm install tailwindcss @tailwindcss/cli && \
    npx @tailwindcss/cli -i styles/input.css -o static/css/styles.css --minify; \
    fi

# Generate code (sqlc + templ)
RUN sqlc generate && templ generate

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fuego-cloud .

# Production image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy binary and static assets
COPY --from=builder /app/fuego-cloud .
COPY --from=builder /app/static ./static

# Create non-root user
RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 3000
CMD ["./fuego-cloud"]
