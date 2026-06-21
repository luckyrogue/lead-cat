import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Loader2,
} from "@leadcat/ui"
import { Controller, FormProvider, useForm } from "react-hook-form"

import type { Survey } from "~/entities/survey/types"
import { useCreateSurvey, useUpdateSurvey } from "~/entities/survey/queries"
import { useT } from "~/shared/i18n/context"
import { toastApiError, toastSuccess } from "~/shared/lib/toast"

import {
  emptyQuestion,
  surveySchema,
  toSurveyInput,
  type SurveyForm,
} from "~/features/surveys/lib/survey-schema"
import { QuestionEditor } from "./question-editor"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  survey?: Survey
  orgId: string
}

function surveyToForm(survey: Survey): SurveyForm {
  return {
    name: survey.name,
    is_active: survey.is_active,
    questions: survey.questions.map((q) => ({
      prompt: q.prompt,
      type: q.type,
      options: q.options ?? [],
      rating_max: q.rating_max,
      required: q.required,
    })),
  }
}

const CREATE_DEFAULTS: SurveyForm = {
  name: "",
  is_active: true,
  questions: [emptyQuestion("text")],
}

export function SurveyDialog({ open, onOpenChange, survey, orgId }: Props) {
  const t = useT()
  const createSurvey = useCreateSurvey(orgId)
  const updateSurvey = useUpdateSurvey(orgId)

  const defaultValues: SurveyForm = survey
    ? surveyToForm(survey)
    : CREATE_DEFAULTS

  const methods = useForm<SurveyForm>({
    resolver: zodResolver(surveySchema),
    defaultValues,
  })
  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = methods

  function handleOpenChange(next: boolean) {
    if (!next) reset(defaultValues)
    onOpenChange(next)
  }

  const isPending = createSurvey.isPending || updateSurvey.isPending

  function submit(values: SurveyForm) {
    const input = toSurveyInput(values)
    if (survey) {
      updateSurvey.mutate(
        { id: survey.id, input },
        {
          onSuccess: () => {
            toastSuccess(t("surveys.toast.updated"))
            handleOpenChange(false)
          },
          onError: (err) => toastApiError(err, "surveys.toast.updateFailed"),
        }
      )
    } else {
      createSurvey.mutate(input, {
        onSuccess: () => {
          toastSuccess(t("surveys.toast.created"))
          handleOpenChange(false)
        },
        onError: (err) => toastApiError(err, "surveys.toast.createFailed"),
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {survey
              ? t("surveys.dialog.editTitle")
              : t("surveys.dialog.createTitle")}
          </DialogTitle>
          <DialogDescription>
            {survey
              ? t("surveys.dialog.editDescription")
              : t("surveys.dialog.createDescription")}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form
            id="survey-form"
            onSubmit={handleSubmit(submit)}
            className="flex flex-col gap-4"
          >
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="survey-name">{t("surveys.name")}</Label>
              <Input
                id="survey-name"
                placeholder={t("surveys.name")}
                {...register("name")}
                aria-invalid={Boolean(errors.name)}
              />
              {errors.name && (
                <p className="text-xs text-destructive">
                  {errors.name.message}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <Label>{t("surveys.active")}</Label>
              <Controller
                control={control}
                name="is_active"
                render={({ field }) => (
                  <button
                    type="button"
                    role="switch"
                    aria-checked={field.value}
                    aria-label={t("surveys.active")}
                    onClick={() => field.onChange(!field.value)}
                    className={
                      field.value
                        ? "relative inline-flex h-6 w-11 items-center rounded-full border-2 border-primary bg-primary transition-colors"
                        : "relative inline-flex h-6 w-11 items-center rounded-full border-2 border-input bg-input transition-colors"
                    }
                  >
                    <span
                      className={
                        field.value
                          ? "inline-block h-4 w-4 translate-x-5 rounded-full bg-primary-foreground transition-transform"
                          : "inline-block h-4 w-4 translate-x-0.5 rounded-full bg-background transition-transform"
                      }
                    />
                  </button>
                )}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <QuestionEditor control={control} />
              {errors.questions && (
                <p className="text-xs text-destructive">
                  {errors.questions.message ?? errors.questions.root?.message}
                </p>
              )}
            </div>
          </form>
        </FormProvider>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button type="submit" form="survey-form" disabled={isPending}>
            {isPending ? <Loader2 className="size-4 animate-spin" /> : null}
            {survey ? t("surveys.dialog.save") : t("surveys.dialog.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
