import { StatusScreen } from "@/components/status-screen"

type MaintenanceScreenProps = {
  onReload?: () => void
}

export function MaintenanceScreen({
  onReload = () => window.location.reload(),
}: MaintenanceScreenProps) {
  return (
    <StatusScreen
      emoji="🐱"
      title="Кот уронил сервер"
      action={
        <button
          type="button"
          onClick={onReload}
          className="mt-2 cursor-pointer rounded-[14px] border-none bg-primary px-5 py-3 text-[15px] font-bold text-white"
        >
          Обновить
        </button>
      }
    >
      <p className="m-0 leading-normal text-cat-secondary">
        Сервис временно недоступен. Попробуйте обновить через минуту — статус
        проверяется автоматически.
      </p>
    </StatusScreen>
  )
}
