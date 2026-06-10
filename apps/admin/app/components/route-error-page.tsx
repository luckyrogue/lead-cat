import { isRouteErrorResponse, Link } from "react-router"

import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@leadcat/ui"

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
    <main className="flex min-h-svh items-center justify-center p-6">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent />
        <CardFooter className="gap-2">
          <Button asChild variant="outline">
            <Link to="/">Back home</Link>
          </Button>
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
