import { useCallback, useEffect, useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import {
  getChatStatus,
  getIntegrations,
} from "@/entities/admin/api"
import {
  linkChat,
  patchIntegrations,
  syncChatMembers,
  verifyIntegrations,
} from "@/entities/admin/write-api"
import { Overlay } from "@/components/tma-shell"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { CatBtn, CatCard } from "@/shared/ui/cat/primitives"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useTmaApp } from "@/shared/tma/context"
import { translate } from "@/shared/tma/i18n"

export function AdminSetupPage() {
  const p = useTmaApp()
  const navigate = useNavigate()
  const t = (key: Parameters<typeof translate>[1]) => translate(p.lang, key)
  const goBack = () => void navigate({ to: "/profile" })

  const [loading, setLoading] = useState(true)
  const [saJson, setSaJson] = useState("")
  const [subject, setSubject] = useState("")
  const [calendarId, setCalendarId] = useState("primary")
  const [meetLink, setMeetLink] = useState("")
  const [tz, setTz] = useState("Asia/Almaty")
  const [chatLinked, setChatLinked] = useState(false)
  const [chatId, setChatId] = useState("")
  const [newChatId, setNewChatId] = useState("")
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [integrations, chat] = await Promise.all([
        getIntegrations(),
        getChatStatus(),
      ])
      setSubject(integrations.googleSubject ?? "")
      setCalendarId(integrations.googleCalendarID || "primary")
      setMeetLink(integrations.meetLink ?? "")
      setTz(integrations.tz || "Asia/Almaty")
      setChatLinked(chat.linked)
      setChatId(chat.chatId ? String(chat.chatId) : "")
    } catch (err) {
      toastError(err, t("auth_login_failed"))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  const saveGoogle = async () => {
    setBusy(true)
    try {
      await patchIntegrations({
        googleSAJson: saJson || undefined,
        googleSubject: subject || undefined,
        googleCalendarID: calendarId || undefined,
        meetLink: meetLink || undefined,
        tz: tz || undefined,
      })
      setSaJson("")
      toastSuccess("Google сохранён")
      await load()
    } catch (err) {
      toastError(err, t("auth_login_failed"))
    } finally {
      setBusy(false)
    }
  }

  const verifyGoogle = async () => {
    setBusy(true)
    try {
      const res = await verifyIntegrations()
      if (res.ok) {
        toastSuccess(res.calendarSummary ?? "Подключение OK")
      } else {
        toastError(new Error("verify failed"), t("auth_login_failed"))
      }
    } catch (err) {
      toastError(err, t("auth_login_failed"))
    } finally {
      setBusy(false)
    }
  }

  const doLinkChat = async () => {
    const id = Number.parseInt(newChatId.trim(), 10)
    if (!Number.isFinite(id) || id === 0) {
      toastError(new Error("invalid chat_id"), "Укажите chat_id")
      return
    }
    setBusy(true)
    try {
      await linkChat(id)
      toastSuccess("Чат привязан")
      setNewChatId("")
      await load()
    } catch (err) {
      toastError(err, t("auth_login_failed"))
    } finally {
      setBusy(false)
    }
  }

  const syncMembers = async () => {
    setBusy(true)
    try {
      const res = await syncChatMembers()
      toastSuccess(`Синхронизировано: ${res.added}`)
    } catch (err) {
      toastError(err, t("auth_login_failed"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Overlay open onClose={goBack} onBack={goBack} title={t("admin")}>
      <div className="flex flex-col gap-4 px-4 pb-8">
        {loading ? (
          <p className="text-tma-muted text-center text-sm">{t("loading")}</p>
        ) : (
          <>
            <SettingsGroup title="Google Calendar">
              <CatCard className="flex flex-col gap-3 p-4">
                <label className="text-tma-muted text-xs font-semibold">
                  Service Account JSON
                </label>
                <textarea
                  value={saJson}
                  onChange={(e) => setSaJson(e.target.value)}
                  placeholder='{"type":"service_account",...}'
                  rows={4}
                  className="tma-input font-mono text-xs"
                />
                <input
                  value={subject}
                  onChange={(e) => setSubject(e.target.value)}
                  placeholder="subject@company.com"
                  className="tma-input"
                />
                <input
                  value={calendarId}
                  onChange={(e) => setCalendarId(e.target.value)}
                  placeholder="calendar id (primary)"
                  className="tma-input"
                />
                <input
                  value={meetLink}
                  onChange={(e) => setMeetLink(e.target.value)}
                  placeholder="meet link template (optional)"
                  className="tma-input"
                />
                <input
                  value={tz}
                  onChange={(e) => setTz(e.target.value)}
                  placeholder="Asia/Almaty"
                  className="tma-input"
                />
                <div className="flex gap-2">
                  <CatBtn
                    variant="primary"
                    className="flex-1"
                    disabled={busy}
                    onClick={() => void saveGoogle()}
                  >
                    Сохранить
                  </CatBtn>
                  <CatBtn
                    variant="outline"
                    className="flex-1"
                    disabled={busy}
                    onClick={() => void verifyGoogle()}
                  >
                    Проверить
                  </CatBtn>
                </div>
              </CatCard>
            </SettingsGroup>

            <SettingsGroup title="Telegram чат">
              <CatCard className="flex flex-col gap-3 p-4">
                <p className="text-tma-muted text-sm">
                  {chatLinked
                    ? `Привязан: ${chatId}`
                    : "Чат не привязан. Отправьте /chatid в группе."}
                </p>
                <input
                  value={newChatId}
                  onChange={(e) => setNewChatId(e.target.value)}
                  placeholder="chat_id"
                  inputMode="numeric"
                  className="tma-input"
                />
                <CatBtn
                  variant="primary"
                  disabled={busy}
                  onClick={() => void doLinkChat()}
                >
                  Привязать чат
                </CatBtn>
              </CatCard>
            </SettingsGroup>

            <SettingsGroup title="Участники">
              <CatCard className="p-4">
                <CatBtn
                  variant="outline"
                  full
                  disabled={busy || !chatLinked}
                  onClick={() => void syncMembers()}
                >
                  Синхронизировать из чата
                </CatBtn>
              </CatCard>
            </SettingsGroup>
          </>
        )}
      </div>
    </Overlay>
  )
}
