import {
  createRootRouteWithContext,
  Outlet,
  useRouter,
} from "@tanstack/react-router"
import type { RouterContext } from "@/app/router-context"
import { RouteErrorPage } from "@/components/route-error-page"

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => <Outlet />,
  errorComponent: RootError,
})

function RootError({ error }: { error: unknown }) {
  const router = useRouter()
  return <RouteErrorPage error={error} reset={() => router.invalidate()} />
}
