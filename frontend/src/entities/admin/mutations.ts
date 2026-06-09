import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  createWorkspace, linkChat, patchIntegrations,
  syncChatMembers, verifyIntegrations,
  type IntegrationsPatch,
} from "./write-api"
import { adminKeys } from "./queries"

export function useCreateWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createWorkspace,
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.all }) },
  })
}

export function useUpdateIntegrations() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (p: IntegrationsPatch) => patchIntegrations(p),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.all }) },
  })
}

export function useVerifyIntegrations() {
  return useMutation({ mutationFn: verifyIntegrations })
}

export function useLinkChat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ chatId, chatTitle }: { chatId: number; chatTitle?: string }) =>
      linkChat(chatId, chatTitle),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.chat() }) },
  })
}

export function useSyncMembers() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: syncChatMembers,
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.members() }) },
  })
}
