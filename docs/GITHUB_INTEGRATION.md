# GitHub Integration Guide

## Overview

VPS Pilot now supports GitHub integration using Personal Access Tokens. This allows you to easily select repositories from your GitHub account when creating projects.

## Features

- ✅ Connect your GitHub account using Personal Access Token
- ✅ Browse and select repositories from a dropdown
- ✅ Auto-fill repository URL and default branch
- ✅ Secure token storage (encrypted in database)
- ✅ Works with both public and private repositories
- ✅ Optional - can still manually enter repository URLs

## Setup Instructions

### For End Users

1. **Navigate to GitHub Settings**
   - Go to Settings → GitHub in your VPS Pilot dashboard
   - Or visit: `http://your-server:8000/settings/github`

2. **Create a GitHub Personal Access Token**
   - Click the link in the settings page or visit: https://github.com/settings/tokens/new
   - Set a descriptive note (e.g., "VPS Pilot")
   - Select the **`repo`** scope (required to read repository information)
   - Click "Generate token"
   - **Copy the token immediately** (you won't be able to see it again)

3. **Connect to VPS Pilot**
   - Paste the token in the GitHub Settings page
   - Click "Connect GitHub"
   - You should see a success message

4. **Use in Projects**
   - When creating a new project, you'll see a dropdown with all your repositories
   - Select a repository, and the URL and default branch will be auto-filled
   - You can still manually enter repository URLs if preferred

### Token Permissions

The Personal Access Token needs the following scope:
- `repo` - Full control of private repositories (required to list and read repos)

### Security Notes

- ✅ Tokens are stored encrypted in the database
- ✅ Tokens are only accessible by the user who created them
- ✅ You can disconnect GitHub at any time
- ⚠️ Never share your Personal Access Token
- ⚠️ If compromised, revoke it immediately on GitHub

## For Developers

### Backend API Endpoints

All endpoints require authentication (JWT token in Authorization header):

#### Save GitHub Token
```http
POST /api/v1/github/token
Content-Type: application/json

{
  "token": "ghp_xxxxxxxxxxxxxxxxxxxxx"
}
```

#### Get User Repositories
```http
GET /api/v1/github/repos
```

#### Check Connection Status
```http
GET /api/v1/github/status
```

#### Disconnect GitHub
```http
DELETE /api/v1/github/token
```

### Frontend Usage

```typescript
import { githubApi } from '@/lib/api';

// Save token
await githubApi.saveToken('ghp_xxxxx');

// Get repositories
const repos = await githubApi.getRepos();

// Check status
const status = await githubApi.getStatus();

// Disconnect
await githubApi.deleteToken();
```

### Database Schema

The `github_token` column is added to the `users` table:

```sql
ALTER TABLE users ADD COLUMN github_token TEXT;
```

### Implementation Details

1. **Handler**: `server/internal/handlers/github_handler.go`
2. **Service**: `server/internal/services/user_service.go` (GitHub methods)
3. **Repository**: `server/internal/db/repo.go` (Database operations)
4. **Frontend Page**: `client/src/pages/settings/github.tsx`
5. **Project Form**: `client/src/components/project-form.tsx`

## Troubleshooting

### "Invalid GitHub token" error
- Make sure you selected the `repo` scope when creating the token
- Verify the token is not expired
- Create a new token if needed

### No repositories showing
- Ensure the token has the correct permissions
- Check that you have repositories in your GitHub account
- Try disconnecting and reconnecting

### Connection issues
- Verify your server can reach `https://api.github.com`
- Check firewall settings
- Ensure SSL certificates are valid

## Alternative: Manual Repository URLs

If you prefer not to connect GitHub or encounter issues:
- You can still manually enter any Git repository URL
- Supported formats:
  - HTTPS: `https://github.com/username/repo.git`
  - SSH: `git@github.com:username/repo.git`
  - Works with any Git hosting provider (GitLab, Bitbucket, etc.)

## Privacy & Data

- GitHub tokens are stored in your local VPS Pilot database
- Tokens are never sent to any third-party services
- VPS Pilot only reads repository information (names, URLs, branches)
- No write access to your repositories
- You can disconnect at any time to remove the token

## Future Enhancements

Potential features for future releases:
- Webhook support for auto-deploy on push
- Branch selection dropdown
- Commit history viewing
- GitLab and Bitbucket support
- OAuth flow (for hosted instances)
