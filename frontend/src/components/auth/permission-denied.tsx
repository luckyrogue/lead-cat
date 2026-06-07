type PermissionDeniedProps = {
  title?: string
  message?: string
}

export function PermissionDenied({
  title = "Нет доступа",
  message = "У вашей роли нет прав на этот раздел.",
}: PermissionDeniedProps) {
  return (
    <div className="flex flex-col items-center gap-3 p-6 text-center">
      <div className="text-[40px]">🙀</div>
      <h2 className="font-display m-0 text-xl font-extrabold">{title}</h2>
      <p className="m-0 text-cat-secondary">{message}</p>
    </div>
  )
}
