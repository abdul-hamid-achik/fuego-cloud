# Fuego CLI Extension for Fuego Cloud

This document outlines the planned CLI extension for the Fuego framework to support deployment to Fuego Cloud.

## Overview

The Fuego CLI will be extended with deployment commands that integrate seamlessly with Fuego Cloud. This will enable developers to deploy their Fuego applications with a single command.

## Proposed Commands

### `fuego login`

Authenticate with Fuego Cloud.

```bash
fuego login
```

**Flow:**
1. Opens browser to `https://cloud.fuego.build/api/auth/cli`
2. User authorizes the CLI
3. CLI receives a short-lived code
4. CLI exchanges code for an API token
5. Token is stored in `~/.fuego/credentials`

**Options:**
- `--token <token>` - Use an existing API token instead of browser auth

### `fuego deploy`

Deploy the current project to Fuego Cloud.

```bash
fuego deploy
```

**Flow:**
1. Read `fuego.yaml` for app configuration
2. Build the application (`fuego build`)
3. Build and push Docker image to GHCR
4. Trigger deployment via Fuego Cloud API
5. Stream deployment logs
6. Display deployment URL on success

**Options:**
- `--prod` - Deploy to production (default is preview)
- `--no-build` - Skip build step (use existing image)
- `--env <key=value>` - Set environment variable
- `--region <region>` - Deploy to specific region

### `fuego apps`

List all applications.

```bash
fuego apps
```

**Output:**
```
NAME           STATUS    DEPLOYMENTS  LAST DEPLOYED
my-app         running   12           2 hours ago
api-service    stopped   5            3 days ago
```

### `fuego apps create <name>`

Create a new application.

```bash
fuego apps create my-new-app
```

**Options:**
- `--region <region>` - Specify region (default: gdl)
- `--size <size>` - Instance size (starter, pro, enterprise)

### `fuego logs <app>`

Stream application logs.

```bash
fuego logs my-app
```

**Options:**
- `-f, --follow` - Stream logs in real-time
- `--since <duration>` - Show logs since duration (e.g., 1h, 30m)
- `--tail <n>` - Number of lines to show (default: 100)

### `fuego env`

Manage environment variables.

```bash
# List env vars (values redacted)
fuego env list my-app

# Set env var
fuego env set my-app DATABASE_URL=postgres://...

# Remove env var
fuego env unset my-app DATABASE_URL

# Pull env vars to local .env
fuego env pull my-app
```

### `fuego domains`

Manage custom domains.

```bash
# List domains
fuego domains list my-app

# Add domain
fuego domains add my-app example.com

# Remove domain
fuego domains remove my-app example.com

# Verify domain
fuego domains verify my-app example.com
```

### `fuego rollback <app>`

Rollback to a previous deployment.

```bash
# Rollback to previous deployment
fuego rollback my-app

# Rollback to specific version
fuego rollback my-app --version 5
```

## Configuration

### `fuego.yaml` Extensions

```yaml
# Existing fuego config
port: 3000
app_dir: "app"

# New cloud deployment config
cloud:
  name: my-app
  region: gdl
  size: starter
  
  # Build settings
  build:
    dockerfile: Dockerfile
    context: .
    
  # Health check
  healthcheck:
    path: /api/health
    interval: 30s
    timeout: 5s
    
  # Scaling
  scaling:
    min: 1
    max: 3
    
  # Environment variables (non-sensitive)
  env:
    GO_ENV: production
```

## Authentication Flow

### Browser-based (Interactive)

```
┌─────────┐          ┌──────────────┐          ┌─────────────┐
│   CLI   │          │  Fuego Cloud │          │   GitHub    │
└────┬────┘          └──────┬───────┘          └──────┬──────┘
     │                      │                         │
     │  Open browser        │                         │
     │─────────────────────>│                         │
     │                      │                         │
     │                      │  OAuth redirect         │
     │                      │────────────────────────>│
     │                      │                         │
     │                      │  Auth code              │
     │                      │<────────────────────────│
     │                      │                         │
     │  CLI code            │                         │
     │<─────────────────────│                         │
     │                      │                         │
     │  Exchange for token  │                         │
     │─────────────────────>│                         │
     │                      │                         │
     │  API token           │                         │
     │<─────────────────────│                         │
     │                      │                         │
```

### Token-based (CI/CD)

```bash
# Generate token in dashboard
# Set in CI environment
export FUEGO_API_TOKEN=fgt_xxxxx

# Deploy in CI
fuego deploy --prod
```

## Implementation Notes

### API Endpoints Required

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/auth/cli` | GET | Initiate CLI auth flow |
| `/api/auth/token` | POST | Exchange code for token |
| `/api/apps` | GET, POST | List/create apps |
| `/api/apps/:name` | GET, PUT, DELETE | App operations |
| `/api/apps/:name/deployments` | GET, POST | Deployments |
| `/api/apps/:name/logs` | GET (SSE) | Stream logs |
| `/api/apps/:name/env` | GET, PUT | Environment vars |
| `/api/apps/:name/domains` | GET, POST | Domains |

### Credential Storage

Credentials stored in `~/.fuego/credentials`:

```json
{
  "token": "fgt_xxxxx",
  "created_at": "2024-01-15T10:00:00Z",
  "expires_at": "2025-01-15T10:00:00Z"
}
```

### Error Handling

- Clear error messages with actionable next steps
- Suggest `fuego login` when authentication fails
- Show deployment logs on failure

## Future Considerations

1. **Team collaboration** - Invite team members, role-based access
2. **Preview deployments** - Automatic deployments for PRs
3. **Secrets management** - Encrypted secrets with rotation
4. **Metrics & monitoring** - CPU, memory, request metrics
5. **Database provisioning** - Managed Neon databases per app
