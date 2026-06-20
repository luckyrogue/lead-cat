import {
  Button,
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  CatFace,
} from "@leadcat/ui"
import { isRouteErrorResponse } from "react-router"

type RouteErrorPageProps = {
  error: unknown
}

export function RouteErrorPage({ error }: RouteErrorPageProps) {
  let title = "Something went wrong"
  let description = "An unexpected error occurred."

  if (isRouteErrorResponse(error)) {
    title = error.status === 404 ? "404" : String(error.status)
    description =
      error.status === 404
        ? "This page could not be found."
        : error.statusText || description
  } else if (import.meta.env.DEV && error instanceof Error) {
    description = error.message
  }

  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-4 px-6 text-center">
      <CatFace className="animate-float size-16" />
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardFooter className="justify-center">
          <Button
            type="button"
            variant="secondary"
            onClick={() => window.location.reload()}
          >
            Reload
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
