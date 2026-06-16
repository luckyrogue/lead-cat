import type { Config } from "@react-router/dev/config"
import { reactRouterFuture } from "@leadcat/config/react-router"

export default {
  ssr: false,
  future: reactRouterFuture,
} satisfies Config
