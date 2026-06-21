const ALLOWED_OAUTH_HOSTS = [
  /^accounts\.google\.com$/i,
  /^login\.microsoftonline\.com$/i,
]

export class OAuthUrlError extends Error {
  constructor(message = "invalid_oauth_url") {
    super(message)
    this.name = "OAuthUrlError"
  }
}

export function assertAllowedOAuthUrl(url: string): string {
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    throw new OAuthUrlError()
  }
  if (parsed.protocol !== "https:") {
    throw new OAuthUrlError()
  }
  const host = parsed.hostname.toLowerCase()
  const allowed = ALLOWED_OAUTH_HOSTS.some((pattern) => pattern.test(host))
  if (!allowed) {
    throw new OAuthUrlError()
  }
  return url
}
