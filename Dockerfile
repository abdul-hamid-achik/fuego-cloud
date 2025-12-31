FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git curl nodejs npm

RUN go install github.com/abdul-hamid-achik/fuego/cmd/fuego@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN if [ -f "styles/input.css" ]; then fuego tailwind build; fi
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o platform .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/platform .
COPY --from=builder /app/static ./static

RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 3000
CMD ["./platform"]
