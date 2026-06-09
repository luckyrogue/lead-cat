import { toastSuccess } from "@/shared/lib/toast"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingDetailMeetLink() {
  const { t } = useTmaApp()

  return (
    <button
      type="button"
      onClick={() => toastSuccess("🔗 Google Meet")}
      className="bg-tma-accent text-tma-accent-text shadow-tma-sm mb-4 flex w-full cursor-pointer items-center gap-3 rounded-2xl border-none px-[15px] py-[13px]"
    >
      <span className="bg-white/22 flex size-9 items-center justify-center rounded-[11px]">
        <CatIcon
          name="link"
          size={19}
          className="text-tma-accent-text"
          sw={2.2}
        />
      </span>
      <div className="flex-1 text-left">
        <div className="text-xs font-semibold opacity-85">{t("meetLink")}</div>
        <div className="font-display text-base font-extrabold">
          {t("joinMeet")}
        </div>
      </div>
      <CatIcon
        name="arrowR"
        size={20}
        className="text-tma-accent-text"
        sw={2.2}
      />
    </button>
  )
}
