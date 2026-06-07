import { createRouter } from "@tanstack/react-router"

import type { RouterContext } from "@/app/router-context"
import { setSession } from "@/shared/auth/session"
import { setAuthToken } from "@/shared/api/client"
import { getQueryClient } from "@/shared/api/query-client"
import { routeTree } from "../routeTree.gen"

export const queryClient = getQueryClient()

export const router = createRouter({
  routeTree,
  context: { queryClient } satisfies RouterContext,
})

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

if (import.meta.hot) {
  import.meta.hot.accept("../routeTree.gen.ts", async () => {
    const { routeTree: nextTree } = await import("../routeTree.gen.ts")
    router.update({ routeTree: nextTree, context: { queryClient } })
  })
}

if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
  setSession({
    accessToken: "dev",
    user: {
      telegramId: 1,
      name: "Dev User",
      email: "dev@local",
      role: "admin",
    },
  })
  setAuthToken("dev")
}
