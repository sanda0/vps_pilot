import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    // Dev only: forward API and WebSocket calls to the hub (make dev sets HUB_PORT).
    proxy: {
      "/api": {
        target: `http://127.0.0.1:${process.env.HUB_PORT ?? "8080"}`,
        ws: true,
      },
    },
  },
  build: {
    outDir: "../server/cmd/app/dist",
    emptyOutDir: true,
  },
});
