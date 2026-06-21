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

import { AuthLocaleShell } from "~/components/auth-locale-shell"
import { useT } from "~/shared/i18n/context"

type RouteErrorPageProps = {
  error: unknown
}

export function RouteErrorPage({ error }: RouteErrorPageProps) {
  return (
    <AuthLocaleShell>
      <RouteErrorPageContent error={error} />
    </AuthLocaleShell>
  )
}

function RouteErrorPageContent({ error }: RouteErrorPageProps) {
  const t = useT()
  let title = t("errors.genericTitle")
  let description = t("errors.genericDescription")

  if (isRouteErrorResponse(error)) {
    title = error.status === 404 ? "404" : String(error.status)
    description =
      error.status === 404
        ? t("errors.notFoundDescription")
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
            <Link to="/">{t("errors.backHome")}</Link>
          </Button>
          <Button
            type="button"
            variant="secondary"
            onClick={() => window.location.reload()}
          >
            {t("errors.reload")}
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
