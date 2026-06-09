import type { ReactNode } from "react"
import { BottomSheet } from "./bottom-sheet"

export function Overlay({
  open,
  onClose,
  title,
  children,
  footer,
  onBack,
}: {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  footer?: ReactNode
  onBack?: () => void
}) {
  return (
    <BottomSheet
      open={open}
      onClose={onClose}
      onBack={onBack}
      title={title}
      footer={footer}
      maxH="94%"
      zIndex={75}
      bodyClassName="p-0"
    >
      {children}
    </BottomSheet>
  )
}
