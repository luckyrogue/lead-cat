type PermissionDeniedProps = {
  title?: string
  message?: string
}

export function PermissionDenied({
  title = "Нет доступа",
  message = "У вашей роли нет прав на этот раздел.",
}: PermissionDeniedProps) {
  return (
    <div
      style={{
        padding: 24,
        textAlign: "center",
        display: "flex",
        flexDirection: "column",
        gap: 12,
        alignItems: "center",
      }}
    >
      <div style={{ fontSize: 40 }}>🙀</div>
      <h2
        style={{
          margin: 0,
          fontFamily: "var(--font-display)",
          fontSize: 20,
          fontWeight: 800,
        }}
      >
        {title}
      </h2>
      <p style={{ margin: 0, color: "var(--cat-secondary, #5B6B7A)" }}>
        {message}
      </p>
    </div>
  )
}
