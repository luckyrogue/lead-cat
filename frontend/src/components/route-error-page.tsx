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
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        background: "var(--cat-bg, #FFF8F0)",
      }}
    >
      <div style={{ maxWidth: 420, textAlign: "center" }}>
        <div style={{ fontSize: 48, marginBottom: 12 }}>😿</div>
        <h1
          style={{
            margin: "0 0 8px",
            fontFamily: "var(--font-display)",
            fontSize: 22,
          }}
        >
          Кот уронил страницу
        </h1>
        <p style={{ margin: "0 0 16px", color: "var(--cat-secondary)" }}>
          {errorMessage(error)}
        </p>
        {reset ? (
          <button
            type="button"
            onClick={reset}
            style={{
              padding: "10px 18px",
              borderRadius: 12,
              border: "none",
              background: "var(--cat-primary, #E87B35)",
              color: "#fff",
              fontWeight: 700,
              cursor: "pointer",
            }}
          >
            Попробовать снова
          </button>
        ) : null}
      </div>
    </main>
  )
}
