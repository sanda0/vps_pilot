# Embedded UI Implementation Summary

## ✅ Changes Made

### 1. Server Updates (`server/cmd/app/app.go`)
- Added `embed.FS` to embed the frontend dist folder
- Created `serveEmbeddedFiles()` function to serve static assets
- Added SPA routing support (all non-API routes serve index.html)
- Implemented proper content-type handling for different file types
- Added graceful fallback if UI files are missing

### 2. Build System
**Updated `build.sh`:**
- Copies `client/dist` to `server/cmd/app/dist`
- Embeds UI during Go build
- Produces single `vps_pilot` binary

**Created `build_embedded.sh`:**
- Alternative simpler build script

**Updated CLI (`server/cmd/cli/cli.go`):**
- Added `build-full` make target
- Added `clean` to remove embedded dist

### 3. Configuration
**Updated `.gitignore`:**
- Added `cmd/app/dist` to ignore embedded files
- Keeps repository clean

**Updated CORS:**
- Embedded UI is served same-origin, so only the Vite dev server (`http://localhost:5173`) is allowed

### 4. Documentation
**Created `docs/BUILDING.md`:**
- Complete build guide
- Development vs production workflows
- Troubleshooting tips
- CI/CD examples

## 📦 How to Build

```bash
# From project root
./build.sh
```

This creates: `server/vps_pilot` (single executable with embedded UI)

## 🚀 How to Run

```bash
cd server
./vps_pilot
```

Visit: **http://localhost:8080**

## 🔧 Architecture

```
┌─────────────────────────────────────┐
│    vps_pilot (single binary)        │
├─────────────────────────────────────┤
│                                     │
│  ┌────────────────────────────┐    │
│  │   Embedded UI (React)      │    │
│  │   - index.html             │    │
│  │   - /assets/*.js, *.css    │    │
│  │   - Static files           │    │
│  └────────────────────────────┘    │
│                                     │
│  ┌────────────────────────────┐    │
│  │   Go Backend               │    │
│  │   - API Routes (/api/v1/*) │    │
│  │   - WebSocket              │    │
│  │   - Database               │    │
│  │   - TCP Server             │    │
│  └────────────────────────────┘    │
│                                     │
└─────────────────────────────────────┘
```

## 📋 Routing Strategy

1. **API Routes** → Go handlers
   - `/api/v1/auth/login`
   - `/api/v1/nodes`
   - `/api/v1/alerts`

2. **Static Assets** → Embedded files
   - `/assets/*.js`
   - `/assets/*.css`
   - `/favicon.ico`

3. **Everything Else** → `index.html` (SPA)
   - `/`
   - `/dashboard`
   - `/nodes/123`
   - etc.

## 🔍 Testing the Build

```bash
# 1. Build
./build.sh

# 2. Run
cd server
./vps_pilot

# 3. Test endpoints
curl http://localhost:8080/api/v1/auth/login  # API
curl http://localhost:8080                     # UI (HTML)
curl http://localhost:8080/assets/index.js     # Assets
```

## 🎯 Benefits

✅ **Single Binary Deployment**
- No external dependencies
- Easy distribution
- Simple updates

✅ **Simplified Architecture**
- One server port
- No CORS issues in production
- Unified logging

✅ **Better Performance**
- No network overhead for static files
- Files served from memory
- Fast startup time

✅ **Easy Development**
- Separate dev servers (hot reload)
- Single command production build
- Clear separation of concerns

## 🔄 Development Workflow

### Development (Hot Reload)
```bash
# Terminal 1: Backend
cd server && go run main.go

# Terminal 2: Frontend
cd client && npm run dev
```

### Production Build
```bash
./build.sh
cd server && ./vps_pilot
```

## 📝 Next Steps

1. **Test the build:**
   ```bash
   ./build.sh
   cd server
   ./vps_pilot
   ```

2. **Verify UI loads at http://localhost:8080**

3. **Test API endpoints work**

4. **Create release builds for different platforms:**
   ```bash
   GOOS=linux GOARCH=amd64 go build -o vps_pilot-linux-amd64
   GOOS=darwin GOARCH=arm64 go build -o vps_pilot-darwin-arm64
   ```

## 🐛 Troubleshooting

**UI doesn't load:**
- Check `server/cmd/app/dist` exists
- Rebuild with `./build.sh`
- Check console for errors

**API doesn't work:**
- Verify backend is running
- Check `/api/v1` routes
- Review CORS settings

**Build fails:**
- Frontend build: Check `client/dist` created
- Backend build: Check Go errors
- Dependencies: Run `npm install` and `go mod download`
