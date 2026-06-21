import { useState } from "react"
import { useParams } from "react-router"
import {
  ArrowUpRight,
  Badge,
  Button,
  ChevronDown,
  ChevronRight,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@leadcat/ui"

import { useResponses, useSurvey } from "~/entities/survey/queries"
import { responsesCsvPath } from "~/entities/survey/api"
import type { ResponseFilter, SurveyResponse } from "~/entities/survey/types"
import { ListPageShell } from "~/components/list-page-shell"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"

function resolveBaseUrl(): string {
  const url = import.meta.env.VITE_API_URL
  return typeof url === "string" && url.length > 0 ? url.replace(/\/$/, "") : ""
}

type AnswerCellProps = {
  response: SurveyResponse
}

function AnswerCell({ response }: AnswerCellProps) {
  const [expanded, setExpanded] = useState(false)

  if (response.answers.length === 0) {
    return <span className="text-muted-foreground">—</span>
  }

  const summary = response.answers
    .map((a) => (Array.isArray(a.value) ? a.value.join("; ") : String(a.value)))
    .join(" · ")
    .slice(0, 60)

  if (!expanded) {
    return (
      <button
        type="button"
        className="flex items-center gap-1 text-left text-xs text-muted-foreground hover:text-foreground"
        onClick={() => setExpanded(true)}
      >
        <ChevronRight className="size-3 shrink-0" />
        <span className="max-w-[180px] truncate">{summary}</span>
      </button>
    )
  }

  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        onClick={() => setExpanded(false)}
      >
        <ChevronDown className="size-3 shrink-0" />
        <span>Collapse</span>
      </button>
      <dl className="flex flex-col gap-0.5">
        {response.answers.map((a) => {
          const displayValue = Array.isArray(a.value)
            ? a.value.join("; ")
            : String(a.value)
          return (
            <div key={a.question_id} className="text-xs">
              <span className="font-medium text-foreground">{a.prompt}:</span>{" "}
              <span className="text-muted-foreground">{displayValue}</span>
            </div>
          )
        })}
      </dl>
    </div>
  )
}

export function ResponsesPage() {
  const t = useT()
  const { id } = useParams<{ id: string }>()
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])

  const [filter, setFilter] = useState<ResponseFilter>({})

  const surveyId = id ?? ""
  const {
    data: surveyData,
    isPending,
    error,
  } = useResponses(activeOrgId, surveyId, filter)
  const { data: survey } = useSurvey(activeOrgId, surveyId)

  const responses = surveyData?.responses ?? []

  function handleExportCsv() {
    const baseUrl = resolveBaseUrl()
    const path = responsesCsvPath(surveyId, filter)
    window.open(`${baseUrl}${path}`, "_blank", "noopener,noreferrer")
  }

  return (
    <ListPageShell
      eyebrow={t("surveys.title")}
      title={survey?.name ?? t("surveys.responsesTitle")}
      description=""
      actions={
        <Button variant="outline" onClick={handleExportCsv}>
          <ArrowUpRight className="size-4" />
          {t("surveys.exportCsv")}
        </Button>
      }
      isLoading={isPending}
      loadingMessage={t("surveys.responsesTitle")}
      error={error}
      isEmpty={responses.length === 0}
      emptyState={
        <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
          {t("surveys.responsesTitle")}
        </div>
      }
    >
      {/* Filter bar */}
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">
            {t("meetings.filter.labelStatus")}
          </label>
          <Select
            value={filter.status ?? ""}
            onValueChange={(v) =>
              setFilter((f) => ({
                ...f,
                status: v === "" ? undefined : (v as ResponseFilter["status"]),
              }))
            }
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder={t("meetings.filter.statusAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">{t("meetings.filter.statusAll")}</SelectItem>
              <SelectItem value="sent">{t("surveys.status.sent")}</SelectItem>
              <SelectItem value="completed">
                {t("surveys.status.completed")}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">
            {t("meetings.filter.labelType") ?? "Service"}
          </label>
          <Select
            value={filter.reason ?? ""}
            onValueChange={(v) =>
              setFilter((f) => ({
                ...f,
                reason: v === "" ? undefined : (v as ResponseFilter["reason"]),
              }))
            }
          >
            <SelectTrigger className="w-44">
              <SelectValue placeholder={t("meetings.filter.statusAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">{t("meetings.filter.statusAll")}</SelectItem>
              <SelectItem value="slot_taken">
                {t("surveys.reason.slot_taken")}
              </SelectItem>
              <SelectItem value="invalid_booking">
                {t("surveys.reason.invalid_booking")}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">
            {t("meetings.filter.labelFrom")}
          </label>
          <input
            type="date"
            value={filter.from ?? ""}
            onChange={(e) =>
              setFilter((f) => ({
                ...f,
                from: e.target.value || undefined,
              }))
            }
            className="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-muted-foreground">
            {t("meetings.filter.labelTo")}
          </label>
          <input
            type="date"
            value={filter.to ?? ""}
            onChange={(e) =>
              setFilter((f) => ({
                ...f,
                to: e.target.value || undefined,
              }))
            }
            className="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          />
        </div>
      </div>

      {/* Responses table */}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="min-w-[9rem]">
              {t("meetings.filter.labelFrom") === "From"
                ? "Date"
                : t("meetings.filter.labelFrom")}
            </TableHead>
            <TableHead className="min-w-[10rem]">
              {t("members.table.colMember")}
            </TableHead>
            <TableHead className="min-w-[8rem]">
              {t("meetings.filter.labelType") ?? "Service"}
            </TableHead>
            <TableHead className="min-w-[7rem]">
              {t("meetings.filter.labelStatus")}
            </TableHead>
            <TableHead className="min-w-[12rem]">
              {t("surveys.responsesTitle")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {responses.map((response) => (
            <TableRow key={response.id}>
              <TableCell className="min-w-[9rem] whitespace-nowrap">
                <span className="text-sm text-foreground">
                  {new Date(response.created_at).toLocaleDateString()}
                </span>
              </TableCell>
              <TableCell className="min-w-[10rem]">
                <p className="text-sm font-medium text-foreground">
                  {response.booker_name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {response.booker_email}
                </p>
              </TableCell>
              <TableCell className="min-w-[8rem] text-sm text-foreground">
                {response.decline_reason === "slot_taken"
                  ? t("surveys.reason.slot_taken")
                  : response.decline_reason === "invalid_booking"
                    ? t("surveys.reason.invalid_booking")
                    : response.decline_reason}
              </TableCell>
              <TableCell className="min-w-[7rem]">
                <Badge
                  tone={response.status === "completed" ? "sunny" : "muted"}
                >
                  {response.status === "completed"
                    ? t("surveys.status.completed")
                    : t("surveys.status.sent")}
                </Badge>
              </TableCell>
              <TableCell className="min-w-[12rem]">
                <AnswerCell response={response} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </ListPageShell>
  )
}
