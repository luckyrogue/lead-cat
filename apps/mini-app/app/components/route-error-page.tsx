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

import { LocaleProvider, useT } from "~/shared/i18n/context"
import { DEFAULT_LOCALE } from "~/shared/i18n/types"

type RouteErrorPageProps = {
  error: unknown
}

export function RouteErrorPage({ error }: RouteErrorPageProps) {
  return (
    <LocaleProvider locale={DEFAULT_LOCALE}>
      <RouteErrorPageContent error={error} />
    </LocaleProvider>
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
            {t("errors.reload")}
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
