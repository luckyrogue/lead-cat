import { StatusScreen } from "@/components/status-screen"

type RouteErrorPageProps = {
  error: unknown
  reset?: () => void
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return "Что-то пошло не так"
}

export function RouteErrorPage({ error, reset }: RouteErrorPageProps) {
  return (
    <StatusScreen
      emoji="😿"
      title="Кот уронил страницу"
      action={
        reset ? (
          <button
            type="button"
            onClick={reset}
            className="cursor-pointer rounded-xl border-none bg-primary px-[18px] py-2.5 font-bold text-white"
          >
            Попробовать снова
          </button>
        ) : null
      }
    >
      <p className="m-0 mb-4 text-cat-secondary">{errorMessage(error)}</p>
    </StatusScreen>
  )
}
