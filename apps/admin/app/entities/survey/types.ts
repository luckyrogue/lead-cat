import type {
  Survey as ApiSurvey,
  SurveyInput as ApiSurveyInput,
  SurveyQuestion as ApiSurveyQuestion,
  SurveyResponse as ApiSurveyResponse,
} from "@leadcat/api-client"

export type Survey = ApiSurvey
export type SurveyQuestion = ApiSurveyQuestion
export type SurveyResponse = ApiSurveyResponse
export type SurveyInput = ApiSurveyInput

export type QuestionType = "single" | "multi" | "rating" | "text"

export type ResponseFilter = {
  status?: "sent" | "completed"
  reason?: "slot_taken" | "invalid_booking"
  from?: string
  to?: string
}
