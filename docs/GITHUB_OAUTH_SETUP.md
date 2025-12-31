# GitHub OAuth Setup

This guide walks you through setting up GitHub OAuth for Fuego Cloud authentication.

## Prerequisites

- A GitHub account
- Access to your GitHub organization settings (if using an organization)

## Step 1: Create a GitHub OAuth App

1. Go to GitHub Settings:
   - For personal account: https://github.com/settings/developers
   - For organization: https://github.com/organizations/YOUR_ORG/settings/applications

2. Click **"New OAuth App"**

3. Fill in the application details:

   | Field | Development Value | Production Value |
   |-------|-------------------|------------------|
   | Application name | Fuego Cloud (Dev) | Fuego Cloud |
   | Homepage URL | http://localhost:3000 | https://cloud.fuego.build |
   | Application description | Deployment platform for Fuego apps | Deployment platform for Fuego apps |
   | Authorization callback URL | http://localhost:3000/api/auth/callback | https://cloud.fuego.build/api/auth/callback |

4. Click **"Register application"**

## Step 2: Get Your Credentials

After creating the app:

1. Copy the **Client ID** - this is public and safe to share
2. Click **"Generate a new client secret"**
3. Copy the **Client Secret** immediately - it won't be shown again

## Step 3: Configure Environment Variables

Add these to your `.env` file:

```bash
GITHUB_CLIENT_ID=your_client_id_here
GITHUB_CLIENT_SECRET=your_client_secret_here
GITHUB_CALLBACK_URL=http://localhost:3000/api/auth/callback
```

## Step 4: Verify Setup

1. Start the development server:
   ```bash
   task dev
   ```

2. Navigate to http://localhost:3000/login

3. Click "Login with GitHub"

4. You should be redirected to GitHub for authorization

5. After authorizing, you should be redirected back to the dashboard

## Security Best Practices

1. **Never commit secrets**: Keep `.env` in `.gitignore`
2. **Use separate apps**: Create different OAuth apps for development and production
3. **Rotate secrets**: Periodically regenerate client secrets
4. **Limit scopes**: Only request necessary permissions

## Troubleshooting

### "The redirect_uri MUST match the registered callback URL"

- Ensure `GITHUB_CALLBACK_URL` exactly matches the callback URL in your OAuth app settings
- Check for trailing slashes
- Verify protocol (http vs https)

### "Bad credentials"

- Verify your `GITHUB_CLIENT_SECRET` is correct
- Regenerate the secret if needed

### "OAuth App access restricted"

- If using an organization, ensure the OAuth app is approved
- Organization owners may need to grant access

## OAuth Scopes

Fuego Cloud requests the following scopes:

| Scope | Purpose |
|-------|---------|
| `user:email` | Access user's email address |
| `read:user` | Access user's profile information |

These are the minimum scopes needed for authentication.
