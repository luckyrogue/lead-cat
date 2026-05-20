const STORAGE_KEY = "leadcat_access_token";

export function getAccessToken(): string | null {
  return localStorage.getItem(STORAGE_KEY);
}

export function setAccessToken(token: string | null) {
  if (token) {
    localStorage.setItem(STORAGE_KEY, token);
  } else {
    localStorage.removeItem(STORAGE_KEY);
  }
}

export function isAuthenticated(): boolean {
  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return true;
  }
  return !!getAccessToken();
}
