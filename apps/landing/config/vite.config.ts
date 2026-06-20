import path from "node:path"
import { fileURLToPath } from "node:url"

import { reactRouter } from "@react-router/dev/vite"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vite"

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../.."
)

export default defineConfig({
  envDir: repoRoot,
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
