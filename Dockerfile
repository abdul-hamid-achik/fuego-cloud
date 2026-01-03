FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git curl nodejs npm

# Install fuego CLI
RUN go install github.com/abdul-hamid-achik/fuego/cmd/fuego@latest

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build Tailwind CSS if styles exist
RUN if [ -f "styles/input.css" ]; then fuego tailwind build; fi

# Build with fuego CLI (handles bracket notation directories)
RUN fuego build

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
