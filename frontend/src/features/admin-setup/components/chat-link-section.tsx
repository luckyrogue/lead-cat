import { useState } from "react"
import { useMiniApp } from "@/shared/miniapp/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useChatStatus } from "@/entities/admin/queries"
import { useLinkChat } from "@/entities/admin/mutations"
import { SettingsGroup } from "@/features/profile/components/settings-group"

export function ChatLinkSection() {
  const { t } = useMiniApp()
  const { data: chat } = useChatStatus()
  const linkMut = useLinkChat()
  const [chatId, setChatId] = useState("")
  const [chatTitle, setChatTitle] = useState("")

  async function onLink() {
    const id = Number.parseInt(chatId.trim(), 10)
    if (!Number.isFinite(id) || id === 0) {
      toastError(new Error("invalid chat_id"), t("chatIdInvalid" as never))
      return
    }
    try {
      await linkMut.mutateAsync({ chatId: id, chatTitle: chatTitle.trim() || undefined })
      toastSuccess(t("chatLinked" as never))
      setChatId("")
      setChatTitle("")
    } catch (err) {
      toastError(err, t("chatLinkFailed" as never))
    }
  }

  return (
    <SettingsGroup title={t("adminChatLink" as never)}>
      <div className="flex flex-col gap-3 px-3 pb-3">
        <div className="text-miniapp-muted text-sm">
          {chat?.linked
            ? `${t("chatLinkedStatus" as never)}: ${chat.chatTitle ?? chat.chatId}`
            : t("chatNotLinked" as never)}
        </div>
        <label className="flex flex-col gap-1">
          <span className="text-miniapp-muted text-xs font-bold">{t("chatIdLabel" as never)}</span>
          <input
            value={chatId}
            onChange={(e) => setChatId(e.target.value)}
            inputMode="numeric"
            placeholder="-100123456789"
            className="border-miniapp-border bg-miniapp-card text-miniapp-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
          />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-miniapp-muted text-xs font-bold">{t("chatTitleLabel" as never)}</span>
          <input
            value={chatTitle}
            onChange={(e) => setChatTitle(e.target.value)}
            placeholder={t("chatTitlePlaceholder" as never)}
            className="border-miniapp-border bg-miniapp-card text-miniapp-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
          />
        </label>
        <button
          type="button"
          disabled={!chatId.trim() || linkMut.isPending}
          onClick={() => void onLink()}
          className="border-miniapp-accent bg-miniapp-accent self-start rounded-[12px] px-4 py-2 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          {linkMut.isPending ? t("saving" as never) : t("linkChatButton" as never)}
        </button>
      </div>
    </SettingsGroup>
  )
}
