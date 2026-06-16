import type { Config } from "@react-router/dev/config"
import { reactRouterFuture } from "@leadcat/config/react-router"

export default {
  ssr: true,
  future: reactRouterFuture,
} satisfies Config
