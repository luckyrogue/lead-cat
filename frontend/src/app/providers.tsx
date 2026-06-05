import { ThemeProvider } from "next-themes"
import { QueryClientProvider } from "@tanstack/react-query"

import { AppContent } from "@/app/app-content"
import { queryClient } from "@/app/router"

export function AppProviders() {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem>
      <QueryClientProvider client={queryClient}>
        <AppContent />
      </QueryClientProvider>
    </ThemeProvider>
  )
}
