import { Card, CardContent, Video } from "@leadcat/ui"

import { PageHeader } from "~/components/page-header"

export default function Meetings() {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Organization"
        title="Meetings"
        description="Schedule and manage Google Meet sessions for your team."
      />
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-14 text-center">
          <span className="grid size-14 place-items-center rounded-[calc(var(--radius)*1.1)] border border-primary/15 bg-primary/10 text-primary">
            <Video className="size-7" />
          </span>
          <div className="max-w-md space-y-2">
            <p className="text-lg font-semibold text-foreground">
              Meetings management is coming
            </p>
            <p className="text-sm leading-6 text-muted-foreground">
              The meetings API is currently available through the Telegram Mini
              App only. Web-auth meeting endpoints are needed before this
              dashboard can list and manage meetings here.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
