import {
  Button,
  ChevronDown,
  ChevronUp,
  Input,
  Label,
  Plus,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Trash2,
} from "@leadcat/ui"
import {
  Controller,
  useFieldArray,
  useFormContext,
  useWatch,
  type Control,
} from "react-hook-form"

import {
  emptyQuestion,
  type SurveyForm,
} from "~/features/surveys/lib/survey-schema"
import { useT } from "~/shared/i18n/context"

const TYPES = ["single", "multi", "rating", "text"] as const

export function QuestionEditor({ control }: { control: Control<SurveyForm> }) {
  const t = useT()
  const { fields, append, remove, move } = useFieldArray({
    control,
    name: "questions",
  })
  return (
    <div className="space-y-3">
      {fields.map((field, i) => (
        <div key={field.id} className="space-y-2 rounded-lg border p-3">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">
              {t("surveys.question")} {i + 1}
            </span>
            <div className="flex gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                disabled={i === 0}
                onClick={() => move(i, i - 1)}
              >
                <ChevronUp className="size-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                disabled={i === fields.length - 1}
                onClick={() => move(i, i + 1)}
              >
                <ChevronDown className="size-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => remove(i)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          </div>

          <Controller
            control={control}
            name={`questions.${i}.prompt`}
            render={({ field }) => (
              <Input {...field} placeholder={t("surveys.questionPrompt")} />
            )}
          />

          <div className="grid grid-cols-2 gap-2">
            <Controller
              control={control}
              name={`questions.${i}.type`}
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TYPES.map((ty) => (
                      <SelectItem key={ty} value={ty}>
                        {t(`surveys.type.${ty}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            <Controller
              control={control}
              name={`questions.${i}.required`}
              render={({ field }) => (
                <label className="flex cursor-pointer items-center gap-2 text-sm select-none">
                  <button
                    type="button"
                    role="switch"
                    aria-checked={field.value}
                    onClick={() => field.onChange(!field.value)}
                    className={
                      field.value
                        ? "relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border-2 border-primary bg-primary transition-colors"
                        : "relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border-2 border-input bg-input transition-colors"
                    }
                  >
                    <span
                      className={
                        field.value
                          ? "inline-block h-3 w-3 translate-x-4 rounded-full bg-primary-foreground transition-transform"
                          : "inline-block h-3 w-3 translate-x-0.5 rounded-full bg-background transition-transform"
                      }
                    />
                  </button>
                  {t("surveys.required")}
                </label>
              )}
            />
          </div>

          <Controller
            control={control}
            name={`questions.${i}.type`}
            render={({ field: typeField }) => (
              <QuestionExtras
                control={control}
                index={i}
                type={typeField.value}
              />
            )}
          />
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        onClick={() => append(emptyQuestion("text"))}
      >
        <Plus className="size-4" /> {t("surveys.addQuestion")}
      </Button>
    </div>
  )
}

function QuestionExtras({
  control,
  index,
  type,
}: {
  control: Control<SurveyForm>
  index: number
  type: string
}) {
  const t = useT()
  const { setValue } = useFormContext<SurveyForm>()
  const options =
    useWatch({ control, name: `questions.${index}.options` }) ?? []

  function addOption() {
    setValue(`questions.${index}.options`, [...options, ""], {
      shouldDirty: true,
    })
  }

  function removeOption(oi: number) {
    setValue(
      `questions.${index}.options`,
      options.filter((_, idx) => idx !== oi),
      { shouldDirty: true }
    )
  }

  if (type === "single" || type === "multi") {
    return (
      <div className="space-y-1.5">
        <Label className="text-xs">{t("surveys.options")}</Label>
        {options.map((_, oi) => (
          <div key={oi} className="flex gap-2">
            <Controller
              control={control}
              name={`questions.${index}.options.${oi}`}
              render={({ field }) => (
                <Input
                  {...field}
                  placeholder={`${t("surveys.option")} ${oi + 1}`}
                />
              )}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => removeOption(oi)}
            >
              <Trash2 className="size-4" />
            </Button>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={addOption}>
          {t("surveys.addOption")}
        </Button>
      </div>
    )
  }

  if (type === "rating") {
    return (
      <Controller
        control={control}
        name={`questions.${index}.rating_max`}
        render={({ field }) => (
          <Select
            value={String(field.value)}
            onValueChange={(v) => field.onChange(Number(v))}
          >
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[2, 3, 4, 5, 6, 7, 8, 9, 10].map((n) => (
                <SelectItem key={n} value={String(n)}>
                  {n}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      />
    )
  }

  return null
}
