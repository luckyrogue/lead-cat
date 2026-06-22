import { Field as UIField } from "@leadcat/ui"

export function Field(props: React.ComponentProps<typeof UIField>) {
  return <UIField className="space-y-1.5" {...props} />
}
