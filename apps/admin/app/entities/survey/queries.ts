import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createSurvey,
  deleteSurvey,
  getSurvey,
  listResponses,
  listSurveys,
  updateSurvey,
} from "~/entities/survey/api"
import type { ResponseFilter, SurveyInput } from "~/entities/survey/types"

export const surveyKeys = {
  list: (orgId: string) => ["orgs", orgId, "surveys"] as const,
  detail: (orgId: string, id: string) =>
    ["orgs", orgId, "surveys", id] as const,
  responses: (orgId: string, id: string, filter: ResponseFilter) =>
    ["orgs", orgId, "surveys", id, "responses", filter] as const,
}

export function useSurveys(orgId: string | null) {
  return useQuery({
    queryKey: surveyKeys.list(orgId ?? ""),
    queryFn: listSurveys,
    enabled: Boolean(orgId),
  })
}

export function useSurvey(orgId: string | null, id: string) {
  return useQuery({
    queryKey: surveyKeys.detail(orgId ?? "", id),
    queryFn: () => getSurvey(id),
    enabled: Boolean(orgId && id),
  })
}

export function useResponses(
  orgId: string | null,
  id: string,
  filter: ResponseFilter
) {
  return useQuery({
    queryKey: surveyKeys.responses(orgId ?? "", id, filter),
    queryFn: () => listResponses(id, filter),
    enabled: Boolean(orgId && id),
  })
}

export function useCreateSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SurveyInput) => createSurvey(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}

export function useUpdateSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; input: SurveyInput }) =>
      updateSurvey(args.id, args.input),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}

export function useDeleteSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSurvey(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}
