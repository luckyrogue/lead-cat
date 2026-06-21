import { api } from "~/shared/api/client"
import type {
  ResponseFilter,
  Survey,
  SurveyInput,
  SurveyResponse,
} from "~/entities/survey/types"

type SurveysResponse = { surveys: Survey[] }
type ResponsesResponse = { survey: Survey; responses: SurveyResponse[] }

export async function listSurveys(): Promise<Survey[]> {
  const { data } = await api.get<SurveysResponse>("/api/surveys")
  return data.surveys ?? []
}

export async function getSurvey(id: string): Promise<Survey> {
  const { data } = await api.get<Survey>(`/api/surveys/${id}`)
  return data
}

export async function createSurvey(input: SurveyInput): Promise<Survey> {
  const { data } = await api.post<Survey>("/api/surveys", input)
  return data
}

export async function updateSurvey(
  id: string,
  input: SurveyInput
): Promise<Survey> {
  const { data } = await api.patch<Survey>(`/api/surveys/${id}`, input)
  return data
}

export async function deleteSurvey(id: string): Promise<void> {
  await api.delete(`/api/surveys/${id}`)
}

function filterQuery(filter: ResponseFilter): string {
  const params = new URLSearchParams()
  if (filter.status) params.set("status", filter.status)
  if (filter.reason) params.set("reason", filter.reason)
  if (filter.from) params.set("from", filter.from)
  if (filter.to) params.set("to", filter.to)
  const q = params.toString()
  return q ? `?${q}` : ""
}

export async function listResponses(
  id: string,
  filter: ResponseFilter = {}
): Promise<ResponsesResponse> {
  const { data } = await api.get<ResponsesResponse>(
    `/api/surveys/${id}/responses${filterQuery(filter)}`
  )
  return data
}

export function responsesCsvPath(
  id: string,
  filter: ResponseFilter = {}
): string {
  return `/api/surveys/${id}/responses.csv${filterQuery(filter)}`
}
