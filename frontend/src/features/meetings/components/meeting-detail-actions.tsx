import { useTmaApp } from "@/shared/tma/context"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"

export function MeetingDetailActions({
  onEdit,
  onDelete,
}: {
  onEdit: () => void
  onDelete: () => void
}) {
  const p = useTmaApp()
  const t = p.t

  return (
    <div style={{ display: "flex", gap: 10, marginTop: 18 }}>
      <CatBtn
        variant="outline"
        full
        icon={<CatIcon name="pencil" size={18} color={p.text} sw={2} />}
        onClick={onEdit}
      >
        {t("edit")}
      </CatBtn>
      <CatBtn
        variant="danger"
        icon={<CatIcon name="trash" size={18} color={p.danger} sw={2} />}
        onClick={onDelete}
        style={{ flex: "0 0 auto" }}
      >
        {t("del")}
      </CatBtn>
    </div>
  )
}
