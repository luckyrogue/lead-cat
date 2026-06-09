import { StatusScreen } from "@/components/status-screen"
import { translate } from "@/shared/miniapp/i18n"
import { readStoredLang } from "@/shared/miniapp/stored-lang"

type RouteErrorPageProps = {
  error: unknown
  reset?: () => void
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return translate(readStoredLang(), "err_page_generic")
}

export function RouteErrorPage({ error, reset }: RouteErrorPageProps) {
  const t = (key: Parameters<typeof translate>[1]) =>
    translate(readStoredLang(), key)

  return (
    <StatusScreen
      emoji="😿"
      title={t("err_page_title")}
      action={
        reset ? (
          <button
            type="button"
            onClick={reset}
            className="bg-primary cursor-pointer rounded-xl border-none px-[18px] py-2.5 font-bold text-white"
          >
            {t("err_page_retry")}
          </button>
        ) : null
      }
    >
      <p className="text-cat-secondary m-0 mb-4">{errorMessage(error)}</p>
    </StatusScreen>
  )
}
