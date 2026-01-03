# Fuego Cloud Deployment Guide

This guide covers everything needed to deploy fuego-cloud to production.

## Prerequisites

- A Kubernetes cluster (k3s recommended)
- Neon database account
- GitHub account (for OAuth and Container Registry)
- Domain name with DNS control
- (Optional) Cloudflare account for DNS management

## 1. GitHub Repository Secrets

Add these secrets in **Settings > Secrets and variables > Actions**:

### Required Secrets

| Secret | Description | Example |
|--------|-------------|---------|
| `DATABASE_URL` | Neon production database URL | `postgres://user:pass@ep-xxx.us-east-2.aws.neon.tech/neondb?sslmode=require` |
| `JWT_SECRET` | JWT signing key (min 32 chars) | Generate with: `openssl rand -base64 32` |
| `ENCRYPTION_KEY` | AES-256 key (exactly 32 bytes) | Generate with: `openssl rand -base64 24` (gives 32 chars) |
| `GITHUB_CLIENT_ID` | GitHub OAuth App Client ID | `Iv1.xxxxxxxxxx` |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App Client Secret | `xxxxxxxxxxxxxxxx` |

### Optional Secrets (for full functionality)

| Secret | Description |
|--------|-------------|
| `NEON_API_KEY` | Neon API key for branch management |
| `NEON_PROJECT_ID` | Neon project ID |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token for DNS |
| `CLOUDFLARE_ZONE_ID` | Cloudflare zone ID |
| `KUBECONFIG_BASE64` | Base64-encoded kubeconfig for deployments |

## 2. Neon Database Setup

### Create Production Database

1. Go to [Neon Console](https://console.neon.tech)
2. Create a new project or use existing
3. Copy the connection string from the dashboard

### Create CI Test Branch

For running tests in CI with a real database:

```bash
# Install Neon CLI
npm install -g neonctl

# Authenticate
neonctl auth

# Create a test branch (branched from main)
neonctl branches create --name ci-test --project-id YOUR_PROJECT_ID

# Get the connection string
neonctl connection-string ci-test --project-id YOUR_PROJECT_ID
```

Add this as `DATABASE_URL` secret in GitHub.

### Run Migrations

```bash
# Using migrate CLI
migrate -path db/migrations -database "$DATABASE_URL" up

# Or using the provided script
./scripts/migrate.sh up
```

## 3. GitHub OAuth App Setup

1. Go to **GitHub Settings > Developer settings > OAuth Apps**
2. Click **New OAuth App**
3. Fill in:
   - **Application name**: `Fuego Cloud`
   - **Homepage URL**: `https://cloud.fuego.build` (your domain)
   - **Authorization callback URL**: `https://cloud.fuego.build/api/auth/callback`
4. Save and copy Client ID and Client Secret

## 4. Kubernetes Cluster Setup

### Option A: Using Terraform (Recommended for Hetzner)

```bash
cd infrastructure/terraform

# Initialize
terraform init

# Create variables file
cat > terraform.tfvars <<EOF
hcloud_token = "your-hetzner-token"
ssh_public_key = "~/.ssh/id_rsa.pub"
server_count = 3
server_type = "cx21"
EOF

# Plan and apply
terraform plan
terraform apply
```

### Option B: Using Existing Cluster

Ensure your cluster has:
- Traefik or another ingress controller
- cert-manager for TLS
- Storage class for persistent volumes

### Configure Cluster with Ansible

```bash
cd infrastructure/ansible

# Update inventory with your server IPs
# (Terraform generates this automatically)

# Run playbook
ansible-playbook -i inventory.yml site.yml
```

## 5. Application Secrets in Kubernetes

Create secrets in the cluster:

```bash
# Create namespace
kubectl create namespace fuego-cloud

# Create secrets
kubectl create secret generic fuego-cloud-secrets \
  --namespace fuego-cloud \
  --from-literal=DATABASE_URL="$DATABASE_URL" \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --from-literal=ENCRYPTION_KEY="$ENCRYPTION_KEY" \
  --from-literal=GITHUB_CLIENT_ID="$GITHUB_CLIENT_ID" \
  --from-literal=GITHUB_CLIENT_SECRET="$GITHUB_CLIENT_SECRET"
```

Or use Sealed Secrets for GitOps:

```bash
# Install kubeseal
brew install kubeseal

# Create sealed secret
kubectl create secret generic fuego-cloud-secrets \
  --namespace fuego-cloud \
  --from-literal=DATABASE_URL="$DATABASE_URL" \
  --dry-run=client -o yaml | \
  kubeseal --format yaml > sealed-secret.yaml

# Apply
kubectl apply -f sealed-secret.yaml
```

## 6. Deploy the Application

### Manual Deployment

```bash
# Build and push image
docker build -t ghcr.io/abdul-hamid-achik/fuego-cloud:latest .
docker push ghcr.io/abdul-hamid-achik/fuego-cloud:latest

# Apply Kubernetes manifests
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### Automated Deployment (via GitHub Actions)

Push a tag to trigger the release workflow:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will:
1. Run tests
2. Build Docker image
3. Push to GitHub Container Registry
4. Create GitHub Release

## 7. DNS Configuration

### Cloudflare (Recommended)

1. Add A record: `cloud.fuego.build` → Cluster IP
2. Add wildcard A record: `*.fuego.build` → Cluster IP
3. Enable proxy for DDoS protection

### Manual DNS

Add these records:
```
cloud.fuego.build.     A     <CLUSTER_IP>
*.fuego.build.         A     <CLUSTER_IP>
```

## 8. CI/CD Pipeline Configuration

Update `.github/workflows/ci.yml` to run tests with database:

```yaml
- name: Run tests
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
  run: |
    go test -v -race -coverprofile=coverage.out ./...
```

## 9. Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | No | Server port (default: 3000) |
| `HOST` | No | Server host (default: 0.0.0.0) |
| `ENVIRONMENT` | No | `development`, `staging`, `production` |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `JWT_SECRET` | Yes | JWT signing secret (min 32 chars) |
| `ENCRYPTION_KEY` | Yes | AES-256 encryption key (32 bytes) |
| `GITHUB_CLIENT_ID` | Yes | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | Yes | GitHub OAuth client secret |
| `GITHUB_CALLBACK_URL` | No | OAuth callback URL |
| `KUBECONFIG` | No | Path to kubeconfig file |
| `K8S_NAMESPACE_PREFIX` | No | Namespace prefix for tenant apps |
| `PLATFORM_DOMAIN` | No | Platform domain (default: cloud.fuego.build) |
| `APPS_DOMAIN_SUFFIX` | No | Apps domain suffix (default: fuego.build) |

## 10. Monitoring & Logging

### Prometheus Metrics

The app exposes metrics at `/api/metrics`. Configure Prometheus to scrape:

```yaml
scrape_configs:
  - job_name: 'fuego-cloud'
    static_configs:
      - targets: ['fuego-cloud.fuego-cloud.svc:3000']
```

### Logging

Logs are output to stdout in JSON format. Use a log aggregator like:
- Loki + Grafana
- Elasticsearch + Kibana
- Datadog

## 11. Backup Strategy

### Database Backups

Neon provides automatic point-in-time recovery. Additionally:

```bash
# Manual backup
pg_dump "$DATABASE_URL" > backup-$(date +%Y%m%d).sql

# Restore
psql "$DATABASE_URL" < backup-20240101.sql
```

### Kubernetes Backups

Use Velero for cluster backups:

```bash
velero backup create fuego-cloud-backup \
  --include-namespaces fuego-cloud
```

## Quick Start Checklist

- [ ] Create Neon database and get connection string
- [ ] Create GitHub OAuth App
- [ ] Add GitHub repository secrets:
  - [ ] `DATABASE_URL`
  - [ ] `JWT_SECRET`
  - [ ] `ENCRYPTION_KEY`
  - [ ] `GITHUB_CLIENT_ID`
  - [ ] `GITHUB_CLIENT_SECRET`
- [ ] Run database migrations
- [ ] Set up Kubernetes cluster
- [ ] Configure DNS records
- [ ] Deploy application
- [ ] Verify health check: `curl https://cloud.fuego.build/api/health`

## Troubleshooting

### Database Connection Issues

```bash
# Test connection
psql "$DATABASE_URL" -c "SELECT 1"

# Check if migrations ran
psql "$DATABASE_URL" -c "SELECT * FROM schema_migrations"
```

### Pod Not Starting

```bash
# Check logs
kubectl logs -n fuego-cloud deployment/fuego-cloud

# Check events
kubectl describe pod -n fuego-cloud -l app=fuego-cloud
```

### OAuth Callback Errors

1. Verify callback URL matches exactly in GitHub OAuth settings
2. Check `GITHUB_CALLBACK_URL` environment variable
3. Ensure HTTPS is working correctly
