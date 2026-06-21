export const ACKEE_BASE_PATH = "/ackee" as const

export const ACKEE_UPSTREAM_URL = "https://analytics.rysdavletov.org"

export function ackeeViteProxy(): Record<
  string,
  {
    target: string
    changeOrigin: boolean
    secure: boolean
    rewrite: (path: string) => string
  }
> {
  return {
    [ACKEE_BASE_PATH]: {
      target: ACKEE_UPSTREAM_URL,
      changeOrigin: true,
      secure: true,
      rewrite: (path) => path.replace(new RegExp(`^${ACKEE_BASE_PATH}`), ""),
    },
  }
}
