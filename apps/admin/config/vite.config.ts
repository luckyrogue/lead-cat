import { reactRouter } from "@react-router/dev/vite"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vite"

const API_TARGET = process.env.VITE_API_PROXY ?? "http://localhost:8080"

export default defineConfig({
  plugins: [tailwindcss(), reactRouter()],
  resolve: {
    tsconfigPaths: true,
  },
  server: {
    proxy: {
      "/api": {
        target: API_TARGET,
        changeOrigin: true,
      },
    },
  },
  ssr: {
    noExternal: ["@leadcat/ui"],
  },
  optimizeDeps: {
    include: ["lucide-react"],
  },
})
