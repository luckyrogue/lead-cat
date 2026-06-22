import { useState } from "react"
import { Link } from "react-router"
import {
  Badge,
  Button,
  ClipboardList,
  Loader2,
  Pencil,
  Trash2,
  toastError,
  toastSuccess,
} from "@leadcat/ui"

import { useSurveys, useDeleteSurvey } from "~/entities/survey/queries"
import type { Survey } from "~/entities/survey/types"
import { SurveyDialog } from "~/features/surveys/components/survey-dialog"
import { ListPageShell } from "~/components/list-page-shell"
import { toApiError } from "~/shared/api/client"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"

export function SurveysPage() {
  const t = useT()
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const { data: surveys, isPending, error } = useSurveys(activeOrgId)
  const deleteSurvey = useDeleteSurvey(activeOrgId ?? "")

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<Survey | null>(null)

  function openCreate() {
    setEditing(null)
    setDialogOpen(true)
  }

  function openEdit(survey: Survey) {
    setEditing(survey)
    setDialogOpen(true)
  }

  function handleDelete(survey: Survey) {
    deleteSurvey.mutate(survey.id, {
      onSuccess: () => toastSuccess(t("surveys.toast.deleted")),
      onError: (err) => {
        const apiErr = toApiError(err)
        if (
          apiErr.code === "survey_has_responses" ||
          apiErr.message === "survey_has_responses"
        ) {
          toastError(err, t, "surveys.deactivateInstead")
        } else {
          toastError(err, t, "surveys.toast.deleteFailed")
        }
      },
    })
  }

  return (
    <>
      <ListPageShell
        eyebrow={t("surveys.title")}
        title={t("surveys.title")}
        description=""
        actions={
          <Button onClick={openCreate}>
            <ClipboardList className="size-4" />
            {t("surveys.create")}
          </Button>
        }
        isLoading={isPending}
        loadingMessage={t("surveys.title")}
        error={error}
        isEmpty={(surveys?.length ?? 0) === 0}
        emptyState={
          <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
            {t("surveys.empty")}
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          {(surveys ?? []).map((survey) => {
            const isPendingDelete =
              deleteSurvey.isPending && deleteSurvey.variables === survey.id

            return (
              <div
                key={survey.id}
                className="flex flex-col gap-3 rounded-[calc(var(--radius)*1.15)] border border-border bg-background p-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground">
                      {survey.name}
                    </span>
                    <Badge tone={survey.is_active ? "sunny" : "muted"}>
                      {survey.is_active
                        ? t("surveys.active")
                        : t("surveys.inactive")}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {t("surveys.questionsCount", {
                        count: survey.questions.length,
                      })}
                    </span>
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-2">
                  <Button variant="outline" size="sm" asChild>
                    <Link to={`/surveys/${survey.id}/responses`}>
                      {t("surveys.responses")}
                    </Link>
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openEdit(survey)}
                  >
                    <Pencil className="size-4" />
                    {t("common.edit")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={isPendingDelete}
                    onClick={() => handleDelete(survey)}
                  >
                    {isPendingDelete ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Trash2 className="size-4" />
                    )}
                    {t("common.delete")}
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      </ListPageShell>

      {activeOrgId && (
        <SurveyDialog
          open={dialogOpen}
          onOpenChange={(next) => {
            setDialogOpen(next)
            if (!next) setEditing(null)
          }}
          survey={editing ?? undefined}
          orgId={activeOrgId}
        />
      )}
    </>
  )
}
