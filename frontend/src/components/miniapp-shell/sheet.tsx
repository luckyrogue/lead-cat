import type { ReactNode } from "react"
import { BottomSheet, type BottomSheetProps } from "./bottom-sheet"

export function Sheet({
  open,
  onClose,
  children,
  maxH = "88%",
}: {
  open: boolean
  onClose: () => void
  children: ReactNode
  maxH?: BottomSheetProps["maxH"]
}) {
  return (
    <BottomSheet open={open} onClose={onClose} maxH={maxH}>
      {children}
    </BottomSheet>
  )
}
