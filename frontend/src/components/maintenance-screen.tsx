import { StatusScreen } from "@/components/status-screen"
import { translate } from "@/shared/miniapp/i18n"
import { readStoredLang } from "@/shared/miniapp/stored-lang"

type MaintenanceScreenProps = {
  onReload?: () => void
}

export function MaintenanceScreen({
  onReload = () => window.location.reload(),
}: MaintenanceScreenProps) {
  const t = (key: Parameters<typeof translate>[1]) =>
    translate(readStoredLang(), key)

  return (
    <StatusScreen
      emoji="🐱"
      title={t("maint_title")}
      action={
        <button
          type="button"
          onClick={onReload}
          className="bg-primary mt-2 cursor-pointer rounded-[14px] border-none px-5 py-3 text-[15px] font-bold text-white"
        >
          {t("maint_reload")}
        </button>
      }
    >
      <p className="text-cat-secondary m-0 leading-normal">{t("maint_body")}</p>
    </StatusScreen>
  )
}
