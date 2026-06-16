import type { Meeting } from "~/entities/meeting/types"

type MeetingIdentityProps = {
  meeting: Meeting
  title: string
  indented?: boolean
}

export function MeetingIdentity({
  meeting,
  title,
  indented = false,
}: MeetingIdentityProps) {
  const subtitle =
    [meeting.dept?.trim(), meeting.host?.trim()].filter(Boolean).join(" · ") ||
    null

  return (
    <div className={indented ? "pl-6" : undefined}>
      <p className="font-medium text-foreground">{title}</p>
      {subtitle ? (
        <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
          {subtitle}
        </p>
      ) : null}
    </div>
  )
}
