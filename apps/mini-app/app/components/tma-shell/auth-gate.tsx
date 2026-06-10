import { Button, CatFace, Loader2, Send } from "@leadcat/ui"

import { useAuth } from "~/shared/auth/auth-context"
import { getBotStartUrl } from "~/shared/tma/telegram-env"

export function AuthGate({ children }: { children: React.ReactNode }) {
  const { status, error, retry } = useAuth()

  if (status === "authed") {
    return <>{children}</>
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-5 px-8 text-center">
      <CatFace className="size-20 animate-float" />
      {status === "loading" && (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
          <span>Signing you in…</span>
        </div>
      )}

      {status === "not_registered" && <NotRegistered />}

      {status === "error" && (
        <div className="flex flex-col items-center gap-4">
          <div>
            <h1 className="text-lg font-bold text-foreground">Couldn't sign in</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {error === "missing_init_data"
                ? "Open this app from inside Telegram to continue."
                : "Something went wrong while authenticating."}
            </p>
          </div>
          <Button onClick={retry}>Try again</Button>
        </div>
      )}
    </div>
  )
}

function NotRegistered() {
  const startUrl = getBotStartUrl()
  return (
    <div className="flex flex-col items-center gap-4">
      <div>
        <h1 className="text-lg font-bold text-foreground">Not registered yet</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Open the bot and press start to get access to your meetings.
        </p>
      </div>
      {startUrl ? (
        <Button asChild>
          <a href={startUrl} target="_blank" rel="noreferrer">
            <Send className="size-4" />
            Open the bot
          </a>
        </Button>
      ) : (
        <p className="text-sm text-muted-foreground">
          Contact your administrator to get registered.
        </p>
      )}
    </div>
  )
}
