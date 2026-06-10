import {
  Badge,
  Button,
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
  Trash2,
} from "@leadcat/ui"

import type { OrgMember, OrgRole } from "~/entities/org/types"

const ROLES: OrgRole[] = ["owner", "admin", "member"]

function memberLabel(member: OrgMember) {
  if (member.telegram_username) {
    return `@${member.telegram_username}`
  }
  if (member.invited_email) {
    return member.invited_email
  }
  return member.user_id
}

type MembersTableProps = {
  members: OrgMember[]
  canManage: boolean
  pendingUserId: string | null
  onRoleChange: (userId: string, role: OrgRole) => void
  onRemove: (member: OrgMember) => void
}

export function MembersTable({
  members,
  canManage,
  pendingUserId,
  onRoleChange,
  onRemove,
}: MembersTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Member</TableHead>
          <TableHead>Role</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {members.map((member) => {
          const isPending = pendingUserId === member.user_id
          return (
            <TableRow key={member.user_id}>
              <TableCell>
                <span className="font-medium text-foreground">
                  {memberLabel(member)}
                </span>
              </TableCell>
              <TableCell>
                {canManage ? (
                  <Select
                    value={member.role}
                    disabled={isPending}
                    onValueChange={(value) =>
                      onRoleChange(member.user_id, value as OrgRole)
                    }
                  >
                    <SelectTrigger className="h-9 w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {ROLES.map((role) => (
                        <SelectItem key={role} value={role}>
                          {role}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Badge tone="muted">{member.role}</Badge>
                )}
              </TableCell>
              <TableCell className="text-right">
                {canManage ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isPending}
                    onClick={() => onRemove(member)}
                    aria-label="Remove member"
                  >
                    <Trash2 className="size-4" />
                  </Button>
                ) : (
                  <span className="text-sm text-muted-foreground">—</span>
                )}
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
