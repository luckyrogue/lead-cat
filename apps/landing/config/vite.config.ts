import { reactRouter } from "@react-router/dev/vite"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [tailwindcss(), reactRouter()],
  resolve: {
    tsconfigPaths: true,
    dedupe: ["three", "@react-three/fiber", "@react-three/drei"],
  },
  ssr: {
    noExternal: ["@leadcat/ui"],
  },
  optimizeDeps: {
    include: ["lucide-react"],
  },
})
