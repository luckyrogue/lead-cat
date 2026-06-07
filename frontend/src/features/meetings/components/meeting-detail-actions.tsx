import { useTmaApp } from "@/shared/tma/context"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingDetailActions({
  onEdit,
  onDelete,
}: {
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useTmaApp()

  return (
    <div className="mt-[18px] flex gap-2.5">
      <CatBtn
        variant="outline"
        full
        icon={<CatIcon name="pencil" size={18} className="text-tma-text" sw={2} />}
        onClick={onEdit}
      >
        {t("edit")}
      </CatBtn>
      <CatBtn
        variant="danger"
        icon={<CatIcon name="trash" size={18} className="text-tma-danger" sw={2} />}
        onClick={onDelete}
        className="shrink-0 grow-0 basis-auto"
      >
        {t("del")}
      </CatBtn>
    </div>
  )
}
