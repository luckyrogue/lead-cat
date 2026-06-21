import { QueryClient } from "@tanstack/react-query"

let browserQueryClient: QueryClient | undefined

export function getQueryClient() {
  if (!browserQueryClient) {
    browserQueryClient = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 30_000,
          retry: 1,
          refetchOnWindowFocus: false,
        },
      },
    })
  }
  return browserQueryClient
}
