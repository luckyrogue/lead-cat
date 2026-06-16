import { Button, Label } from "@leadcat/ui"

import type { MeetingMutationScope } from "~/entities/meeting/types"
import { useT } from "~/shared/i18n/context"

type Props = {
  value: MeetingMutationScope
  onChange: (scope: MeetingMutationScope) => void
}

export function ScopeToggle({ value, onChange }: Props) {
  const t = useT()
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{t("meetings.scope.applyTo")}</Label>
      <div className="grid grid-cols-2 gap-2">
        <Button
          type="button"
          variant={value === "this" ? "default" : "outline"}
          onClick={() => onChange("this")}
        >
          {t("meetings.scope.thisMeeting")}
        </Button>
        <Button
          type="button"
          variant={value === "whole" ? "default" : "outline"}
          onClick={() => onChange("whole")}
        >
          {t("meetings.scope.wholeSeries")}
        </Button>
      </div>
    </div>
  )
}
