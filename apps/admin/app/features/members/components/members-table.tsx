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
import { useT } from "~/shared/i18n/context"

const ROLES: OrgRole[] = ["owner", "admin", "member"]

function memberLabel(member: OrgMember, fallback: string) {
  return member.name || member.email || member.telegram_username || fallback
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
  const t = useT()

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("members.table.colMember")}</TableHead>
          <TableHead>{t("members.table.colRole")}</TableHead>
          <TableHead>{t("members.table.colStatus")}</TableHead>
          <TableHead className="text-right">
            {t("members.table.colActions")}
          </TableHead>
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
                  {memberLabel(member, t("members.table.unnamedMember"))}
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
                          {t(`members.role.${role}`)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Badge tone="muted">{t(`members.role.${member.role}`)}</Badge>
                )}
              </TableCell>
              <TableCell>
                <Badge tone={member.status === "active" ? "sunny" : "muted"}>
                  {t(`members.status.${member.status}`)}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                {canManage && isLinked ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isPending}
                    onClick={() => onRemove(member)}
                    aria-label={t("members.table.removeAriaLabel")}
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
