# Building VPS Pilot with Embedded UI

This guide explains how to build VPS Pilot as a single executable with the UI embedded.

## Quick Start

### Option 1: Full Build Script (Recommended)

```bash
# From project root
chmod +x build.sh
./build.sh
```

This will:
1. Build the React frontend
2. Copy it to `server/cmd/app/dist`
3. Build the Go binary with embedded UI
4. Create `server/vps_pilot` executable

### Option 2: Manual Build

```bash
# 1. Build frontend
cd client
npm install
npm run build

# 2. Copy to server
cd ..
rm -rf server/cmd/app/dist
cp -r client/dist server/cmd/app/

# 3. Build Go binary
cd server
go build -o vps_pilot .
```

## How It Works

### Frontend Build
The React app is built using Vite, creating an optimized production bundle in `client/dist/`:
- Minified JavaScript
- Optimized CSS
- Static assets (images, fonts, etc.)

### Embedding in Go
The Go application uses `embed.FS` to include the frontend files at compile time:

```go
//go:embed dist
var staticFiles embed.FS
```

This means:
- ✅ Single binary deployment
- ✅ No external file dependencies
- ✅ Frontend always available
- ✅ Simplified deployment

### Serving Strategy
The server serves files with this priority:
1. **API routes** (`/api/v1/*`) - JSON responses
2. **Static assets** (`/assets/*`) - JS, CSS, images from embedded FS
3. **SPA fallback** - All other routes serve `index.html` for client-side routing

## Running the Binary

```bash
cd server
./vps_pilot
```

Access the application at: **http://localhost:8000**

## Development vs Production

### Development (Separate Frontend)
```bash
# Terminal 1: Run backend
cd server
go run main.go

# Terminal 2: Run frontend dev server
cd client
npm run dev
```
Frontend: http://localhost:5173 (hot reload)
API: http://localhost:8000/api/v1

### Production (Embedded UI)
```bash
./build.sh
cd server
./vps_pilot
```
Everything: http://localhost:8000

## Build Options

### Standard Build
```bash
./build.sh
```

### Optimized Build (Smaller Binary)
```bash
cd server
go build -ldflags "-s -w" -o vps_pilot .
```
- `-s`: Strip debug symbols
- `-w`: Strip DWARF debug info

### Cross-Compilation
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o vps_pilot-linux-amd64 .

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o vps_pilot-linux-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o vps_pilot.exe .
```

## File Structure

```
vps_pilot/
├── build.sh                    # Main build script
├── client/
│   ├── dist/                   # Frontend build output (gitignored)
│   └── src/                    # Frontend source
└── server/
    ├── cmd/
    │   └── app/
    │       ├── app.go          # Server with embed directive
    │       └── dist/           # Embedded UI (gitignored, copied during build)
    ├── main.go
    └── vps_pilot               # Final binary (gitignored)
```

## Troubleshooting

### "Failed to load embedded UI files"
The binary will still run but won't serve the frontend. This means:
- `cmd/app/dist` was missing during build
- Run `./build.sh` to rebuild with UI

### API works but UI doesn't load
Check:
1. Binary was built with UI: `ls -la server/cmd/app/dist`
2. CORS settings in `app.go`
3. Browser console for errors

### Large Binary Size
The embedded UI adds ~5-10MB. To reduce:
```bash
# Optimize React build
cd client
npm run build

# Strip Go binary
cd ../server
go build -ldflags "-s -w" -o vps_pilot .

# Optional: Compress with UPX
upx --best --lzma vps_pilot
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Build Release
on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build
        run: ./build.sh
      
      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: vps_pilot
          path: server/vps_pilot
```

## Environment Variables

Create `server/.env`:
```env
# Database
DB_PATH=./data

# JWT
TOKEN_LIFESPAN=60
TOKEN_SECRET=your-secret-key-min-32-chars

# TCP Server
TCP_SERVER_PORT=55001
```

## Next Steps

- [Deployment Guide](DEPLOYMENT.md)
- [Configuration Options](CONFIGURATION.md)
- [API Documentation](API.md)
