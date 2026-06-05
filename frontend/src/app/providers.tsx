import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { RouterProvider, createRouter } from "@tanstack/react-router"
import { ThemeProvider } from "next-themes"
import { routeTree } from "../routeTree.gen"
import { setAuthToken } from "@/shared/api/client"

const queryClient = new QueryClient()
export const router = createRouter({ routeTree })

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

if (import.meta.hot) {
  import.meta.hot.accept("../routeTree.gen.ts", async () => {
    const { routeTree: nextTree } = await import("../routeTree.gen.ts")
    router.update({ routeTree: nextTree })
  })
}

if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
  setAuthToken("dev")
}

export function AppProviders() {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeProvider>
  )
}
