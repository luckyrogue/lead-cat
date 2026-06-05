type MaintenanceScreenProps = {
  onReload?: () => void
}

export function MaintenanceScreen({
  onReload = () => window.location.reload(),
}: MaintenanceScreenProps) {
  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        background: "var(--cat-bg, #FFF8F0)",
        fontFamily: "var(--font-body, Inter, sans-serif)",
      }}
    >
      <div
        style={{
          maxWidth: 420,
          width: "100%",
          textAlign: "center",
          display: "flex",
          flexDirection: "column",
          gap: 16,
        }}
      >
        <div style={{ fontSize: 48 }}>🐱</div>
        <h1
          style={{
            margin: 0,
            fontFamily: "var(--font-display, Baloo 2, sans-serif)",
            fontSize: 24,
            fontWeight: 800,
            color: "var(--cat-secondary, #5B6B7A)",
          }}
        >
          Кот уронил сервер
        </h1>
        <p style={{ margin: 0, lineHeight: 1.5, color: "var(--cat-secondary, #5B6B7A)" }}>
          Сервис временно недоступен. Попробуйте обновить через минуту — статус
          проверяется автоматически.
        </p>
        <button
          type="button"
          onClick={onReload}
          style={{
            marginTop: 8,
            padding: "12px 20px",
            borderRadius: 14,
            border: "none",
            cursor: "pointer",
            background: "var(--cat-primary, #E87B35)",
            color: "#fff",
            fontWeight: 700,
            fontSize: 15,
          }}
        >
          Обновить
        </button>
      </div>
    </main>
  )
}
