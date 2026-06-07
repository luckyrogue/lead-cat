import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"
import { Sheet } from "@/components/tma-shell"

export function MeetingDetailActions({
  isSeries,
  onEdit,
  onDelete,
}: {
  isSeries: boolean
  onEdit: (scope: "this" | "whole") => void
  onDelete: (scope: "this" | "whole") => void
}) {
  const { t } = useTmaApp()
  const [editSheet, setEditSheet] = useState(false)
  const [delSheet, setDelSheet] = useState(false)

  return (
    <>
      <div className="mt-[18px] flex gap-2.5">
        <CatBtn
          variant="outline"
          full
          icon={
            <CatIcon name="pencil" size={18} className="text-tma-text" sw={2} />
          }
          onClick={() => (isSeries ? setEditSheet(true) : onEdit("this"))}
        >
          {t("edit")}
        </CatBtn>
        <CatBtn
          variant="danger"
          icon={
            <CatIcon
              name="trash"
              size={18}
              className="text-tma-danger"
              sw={2}
            />
          }
          onClick={() => (isSeries ? setDelSheet(true) : onDelete("this"))}
          className="shrink-0 grow-0 basis-auto"
        >
          {t("del")}
        </CatBtn>
      </div>
      <Sheet
        open={editSheet}
        onClose={() => setEditSheet(false)}
        maxH="fit-content"
      >
        <div className="flex flex-col gap-2.5 p-4">
          <CatBtn
            variant="outline"
            full
            onClick={() => {
              setEditSheet(false)
              onEdit("this")
            }}
          >
            {t("editThis")}
          </CatBtn>
          <CatBtn
            variant="primary"
            full
            onClick={() => {
              setEditSheet(false)
              onEdit("whole")
            }}
          >
            {t("editSeries")}
          </CatBtn>
        </div>
      </Sheet>
      <Sheet
        open={delSheet}
        onClose={() => setDelSheet(false)}
        maxH="fit-content"
      >
        <div className="flex flex-col gap-2.5 p-4">
          <CatBtn
            variant="outline"
            full
            onClick={() => {
              setDelSheet(false)
              onDelete("this")
            }}
          >
            {t("delThis")}
          </CatBtn>
          <CatBtn
            variant="danger"
            full
            onClick={() => {
              setDelSheet(false)
              onDelete("whole")
            }}
          >
            {t("delSeries")}
          </CatBtn>
        </div>
      </Sheet>
    </>
  )
}
