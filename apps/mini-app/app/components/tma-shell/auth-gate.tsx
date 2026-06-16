import { Button, CatFace, Loader2, Send } from "@leadcat/ui"

import { useAuth } from "~/shared/auth/auth-context"
import { useT } from "~/shared/i18n/context"
import { LocaleGate } from "~/components/tma-shell/locale-gate"
import { getBotStartUrl } from "~/shared/tma/telegram-env"

export function AuthGate({ children }: { children: React.ReactNode }) {
  const { status, error, retry } = useAuth()
  const t = useT()

  if (status === "authed") {
    return <LocaleGate>{children}</LocaleGate>
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-5 px-8 text-center">
      <CatFace className="animate-float size-20" />
      {status === "loading" && (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
          <span>{t("auth.signingIn")}</span>
        </div>
      )}

      {status === "not_registered" && <NotRegistered />}

      {status === "error" && (
        <div className="flex flex-col items-center gap-4">
          <div>
            <h1 className="text-lg font-bold text-foreground">
              {t("auth.errorTitle")}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {error === "missing_init_data"
                ? t("auth.errorMissingInitData")
                : t("auth.errorGeneric")}
            </p>
          </div>
          <Button onClick={retry}>{t("common.retry")}</Button>
        </div>
      )}
    </div>
  )
}

function NotRegistered() {
  const t = useT()
  const startUrl = getBotStartUrl()
  return (
    <div className="flex flex-col items-center gap-4">
      <div>
        <h1 className="text-lg font-bold text-foreground">
          {t("auth.notRegisteredTitle")}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("auth.notRegisteredHint")}
        </p>
      </div>
      {startUrl ? (
        <Button asChild>
          <a href={startUrl} target="_blank" rel="noreferrer">
            <Send className="size-4" />
            {t("auth.openBot")}
          </a>
        </Button>
      ) : (
        <p className="text-sm text-muted-foreground">
          {t("auth.contactAdmin")}
        </p>
      )}
    </div>
  )
}
