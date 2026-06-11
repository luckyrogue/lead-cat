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
  return member.name || member.email || member.telegram_username || "Member"
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
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {members.map((member) => {
          const isPending = pendingUserId === member.user_id
          const isLinked = Boolean(member.user_id)
          const subline =
            member.name && member.email
              ? member.email
              : member.telegram_username
          return (
            <TableRow key={member.user_id ?? member.email}>
              <TableCell>
                <span className="font-medium text-foreground">
                  {memberLabel(member)}
                </span>
                {subline ? (
                  <span className="block text-xs text-muted-foreground">
                    {subline}
                  </span>
                ) : null}
              </TableCell>
              <TableCell>
                {canManage && isLinked ? (
                  <Select
                    value={member.role}
                    disabled={isPending}
                    onValueChange={(value) =>
                      onRoleChange(member.user_id as string, value as OrgRole)
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
              <TableCell>
                <Badge tone={member.status === "active" ? "sunny" : "muted"}>
                  {member.status}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                {canManage && isLinked ? (
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
