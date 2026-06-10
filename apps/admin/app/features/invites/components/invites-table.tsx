import {
  Badge,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Trash2,
} from "@leadcat/ui"

import type { OrgInvite } from "~/entities/org/types"

function formatDate(value?: string) {
  if (!value) {
    return "—"
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "—"
  }
  return date.toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  })
}

type InvitesTableProps = {
  invites: OrgInvite[]
  pendingId: string | null
  onDelete: (invite: OrgInvite) => void
}

export function InvitesTable({
  invites,
  pendingId,
  onDelete,
}: InvitesTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Email</TableHead>
          <TableHead>Role</TableHead>
          <TableHead>Expires</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {invites.map((invite) => (
          <TableRow key={invite.id}>
            <TableCell className="font-medium text-foreground">
              {invite.email}
            </TableCell>
            <TableCell>
              <Badge tone="muted">{invite.role}</Badge>
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatDate(invite.expires_at)}
            </TableCell>
            <TableCell className="text-right">
              <Button
                variant="ghost"
                size="sm"
                disabled={pendingId === invite.id}
                onClick={() => onDelete(invite)}
                aria-label="Revoke invite"
              >
                <Trash2 className="size-4" />
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
