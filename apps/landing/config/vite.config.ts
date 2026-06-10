import { reactRouter } from "@react-router/dev/vite"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vite"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tailwindcss(), reactRouter(), tsconfigPaths()],
  resolve: {
    dedupe: ["three", "@react-three/fiber", "@react-three/drei"],
  },
  ssr: {
    noExternal: ["@leadcat/ui"],
  },
  optimizeDeps: {
    include: ["lucide-react"],
  },
})
