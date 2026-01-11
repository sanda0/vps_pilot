# Quick Start: Embedded UI Build

## 🚀 Build Single Executable

```bash
# From project root
./build.sh
```

Output: `server/vps_pilot` (single executable with UI)

## 📦 Run

```bash
cd server
./vps_pilot
```

**Access:** http://localhost:8000

## 📋 What Gets Embedded?

- ✅ React UI (built with Vite)
- ✅ All JavaScript bundles
- ✅ CSS stylesheets  
- ✅ Images and fonts
- ✅ Static assets

## 🔧 Build Process

1. **Frontend Build** (`client/`)
   ```bash
   npm run build
   # Creates: client/dist/
   ```

2. **Copy to Server**
   ```bash
   cp -r client/dist server/cmd/app/
   ```

3. **Embed & Build**
   ```bash
   go build -o vps_pilot .
   # Embeds: cmd/app/dist/ into binary
   ```

## 🎯 Result

Single `vps_pilot` binary that serves:
- Frontend UI on `/`
- API endpoints on `/api/v1/*`
- WebSocket on `/api/v1/nodes/ws/system-stat`

## 📍 URLs

| What | URL |
|------|-----|
| **UI** | http://localhost:8000 |
| **API** | http://localhost:8000/api/v1 |
| **Login** | http://localhost:8000/api/v1/auth/login |

## ⚙️ Environment

Create `server/.env`:
```env
DB_PATH=./data
TOKEN_LIFESPAN=60
TOKEN_SECRET=your-secret-key
TCP_SERVER_PORT=55001
```

## 🧪 Test

```bash
# Build
./build.sh

# Run
cd server && ./vps_pilot

# Test API
curl http://localhost:8000/api/v1/auth/login

# Test UI (in browser)
open http://localhost:8000
```

## 📊 Binary Size

- **Without UI:** ~15MB
- **With UI:** ~20-25MB
- **Compressed (upx):** ~8-10MB

## 🔄 Development vs Production

### Development (Hot Reload)
```bash
# Backend
cd server && go run main.go

# Frontend (separate terminal)
cd client && npm run dev
```
- Backend: http://localhost:8000
- Frontend: http://localhost:5173

### Production (Embedded)
```bash
./build.sh
cd server && ./vps_pilot
```
- Everything: http://localhost:8000

## 🛠️ Makefile Commands

```bash
cd server

make migrate          # Run migrations
make build            # Build server only
make build-full       # Build with UI (runs ../build.sh)
make run              # Run server
make create-superuser # Create admin user
make clean            # Remove build artifacts
```

## ✅ Success Checklist

- [ ] Build completes without errors
- [ ] Binary created: `server/vps_pilot`
- [ ] Server starts: `./vps_pilot`
- [ ] UI loads at http://localhost:8000
- [ ] Login page appears
- [ ] API responds at http://localhost:8000/api/v1

## 🐛 Troubleshooting

**Build fails:**
```bash
# Check frontend builds
cd client && npm run build

# Check Go compiles
cd server && go build .
```

**UI doesn't load:**
```bash
# Verify dist was copied
ls server/cmd/app/dist/

# Rebuild
./build.sh
```

**Binary size too large:**
```bash
# Strip and compress
cd server
go build -ldflags "-s -w" -o vps_pilot .
upx --best vps_pilot
```

Done! 🎉
