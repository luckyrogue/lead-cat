const MAILPIT = process.env.E2E_MAILPIT_URL ?? "http://localhost:8025"

export async function getLatestMagicLink(email: string): Promise<string> {
  for (let i = 0; i < 30; i++) {
    const res = await fetch(`${MAILPIT}/api/v1/search?query=to:${encodeURIComponent(email)}`)
    if (res.ok) {
      const data = (await res.json()) as { messages?: { ID: string }[] }
      const id = data.messages?.[0]?.ID
      if (id) {
        const msg = await fetch(`${MAILPIT}/api/v1/message/${id}`)
        const body = (await msg.json()) as { Text?: string; HTML?: string }
        const m = (body.Text ?? "" + (body.HTML ?? "")).match(/https?:\/\/[^\s"'<>]*magic\/verify[^\s"'<>]*/)
        if (m) return m[0]
      }
    }
    await new Promise((r) => setTimeout(r, 1000))
  }
  throw new Error(`no magic link for ${email}`)
}
