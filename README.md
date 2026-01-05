# Nexo Cloud

The cloud platform built specifically for [Fuego](https://github.com/abdul-hamid-achik/fuego) applications. Deploy, scale, and manage your Go apps with zero configuration.

## Features

- **One-Command Deploy** - Push your Fuego app to production in seconds
- **Auto-Scaling** - Scale from zero to thousands of requests automatically
- **Custom Domains** - Bring your own domain with automatic SSL
- **Environment Variables** - Securely manage secrets with AES-256 encryption
- **Real-time Logs** - Stream logs from your running containers
- **GitHub OAuth** - Authenticate with your GitHub account
- **Multi-tenant** - Isolated Kubernetes namespaces per user

## Quick Start

```bash
# Clone the repo
git clone https://github.com/abdul-hamid-achik/nexo-cloud.git
cd nexo-cloud

# Copy environment variables
cp .env.example .env

# Start the database
docker compose up -d

# Run migrations
task migrate:up

# Generate code (sqlc + templ)
task generate

# Run the server
task dev
```

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Node.js 18+ (for Tailwind CSS)
- kubectl (for K8s deployments)

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `JWT_SECRET` | Secret for signing JWTs (min 32 chars) | Yes |
| `ENCRYPTION_KEY` | AES-256 key for env var encryption (32 bytes) | Yes |
| `GITHUB_CLIENT_ID` | GitHub OAuth App client ID | Yes |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App client secret | Yes |
| `GITHUB_CALLBACK_URL` | OAuth callback URL | Yes |
| `KUBECONFIG` | Path to kubeconfig file | For deploys |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token | For custom domains |
| `CLOUDFLARE_ZONE_ID` | Cloudflare zone ID | For custom domains |

See [.env.example](.env.example) for all available options.

## API Endpoints

### Authentication
- `GET /api/auth` - Start GitHub OAuth flow
- `GET /api/auth/callback` - OAuth callback
- `POST /api/auth/token` - Generate API token

### Apps
- `GET /api/apps` - List apps
- `POST /api/apps` - Create app
- `GET /api/apps/:name` - Get app details
- `DELETE /api/apps/:name` - Delete app
- `POST /api/apps/:name/restart` - Restart app
- `POST /api/apps/:name/scale` - Scale app

### Deployments
- `GET /api/apps/:name/deployments` - List deployments
- `POST /api/apps/:name/deployments` - Create deployment
- `GET /api/apps/:name/deployments/:id` - Get deployment

### Environment Variables
- `GET /api/apps/:name/env` - Get env vars
- `PUT /api/apps/:name/env` - Update env vars

### Domains
- `GET /api/apps/:name/domains` - List domains
- `POST /api/apps/:name/domains` - Add domain
- `DELETE /api/apps/:name/domains/:domain` - Remove domain
- `POST /api/apps/:name/domains/:domain/verify` - Verify domain

### Metrics & Logs
- `GET /api/apps/:name/metrics` - Get app metrics
- `GET /api/apps/:name/activity` - Get activity logs

## Architecture

```
nexo-cloud/
├── app/                    # HTTP handlers (Fuego routes)
│   ├── api/               # REST API endpoints
│   ├── dashboard/         # Web dashboard (templ)
│   └── components/        # Shared UI components
├── db/
│   ├── migrations/        # SQL migrations
│   └── queries/           # sqlc queries
├── internal/
│   ├── auth/             # JWT & OAuth
│   ├── config/           # Configuration
│   ├── crypto/           # Encryption utilities
│   └── k8s/              # Kubernetes client
├── infrastructure/
│   ├── ansible/          # Server provisioning
│   └── terraform/        # Infrastructure as code
└── k8s/                  # Kubernetes manifests
```

## Development

```bash
# Run all tests
task test

# Run with hot reload
task dev

# Build for production
task build

# Generate sqlc + templ
task generate

# Build CSS
task css
```

## Deployment

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for production deployment instructions.

### Quick Deploy with Docker

```bash
docker build -t nexo-cloud .
docker run -p 3000:3000 --env-file .env nexo-cloud
```

### Deploy to Kubernetes

```bash
kubectl apply -f k8s/
```

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- [Fuego Framework](https://github.com/abdul-hamid-achik/fuego)
- [Documentation](https://nexo.build/docs)
- [Discord](https://discord.gg/nexo)
