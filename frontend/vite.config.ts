import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import path from "path";

export default defineConfig({
  envDir: path.resolve(__dirname, ".."),
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    tailwindcss(),
    react(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": "http://localhost:8080",
    },
    allowedHosts:
      process.env.VITE_DEV_ALLOWED_HOSTS?.split(",")
        .map((h) => h.trim())
        .filter(Boolean) ?? true,
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
