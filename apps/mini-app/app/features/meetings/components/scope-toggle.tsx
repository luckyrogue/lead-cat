import { Button, Label } from "@leadcat/ui"

import type { MeetingMutationScope } from "~/entities/meeting/types"

type Props = {
  value: MeetingMutationScope
  onChange: (scope: MeetingMutationScope) => void
}

export function ScopeToggle({ value, onChange }: Props) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>Apply to</Label>
      <div className="grid grid-cols-2 gap-2">
        <Button
          type="button"
          variant={value === "this" ? "default" : "outline"}
          onClick={() => onChange("this")}
        >
          This meeting
        </Button>
        <Button
          type="button"
          variant={value === "whole" ? "default" : "outline"}
          onClick={() => onChange("whole")}
        >
          Whole series
        </Button>
      </div>
    </div>
  )
}
