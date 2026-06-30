import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/chat": "http://localhost:8741",
      "/api": "http://localhost:8741",
      "/v1": "http://localhost:8741",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
