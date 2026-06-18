import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Input,
  Label,
  Loader2,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  UserPlus,
} from "@leadcat/ui"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { OrgRole } from "~/entities/org/types"
import { useT } from "~/shared/i18n/context"

const schema = z.object({
  email: z.string().email("Enter a valid email address"),
  role: z.enum(["admin", "member"]),
})

export type InviteFormValues = z.infer<typeof schema>

type InviteFormProps = {
  pending: boolean
  onSubmit: (values: { email: string; role: OrgRole }) => void
}

export function InviteForm({ pending, onSubmit }: InviteFormProps) {
  const t = useT()
  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<InviteFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", role: "member" },
  })

  return (
    <form
      onSubmit={handleSubmit((values) => {
        onSubmit(values)
        reset({ email: "", role: values.role })
      })}
      className="flex items-end justify-end gap-3"
    >
      <div className="min-w-0 flex-1 space-y-2">
        <Label htmlFor="invite-email">{t("invites.form.emailLabel")}</Label>
        <Input
          id="invite-email"
          type="email"
          placeholder={t("invites.form.emailPlaceholder")}
          {...register("email")}
        />
        {errors.email ? (
          <p className="text-sm text-destructive" role="alert">
            {errors.email.message}
          </p>
        ) : null}
      </div>
      <div className="shrink-0 space-y-2">
        <Label htmlFor="invite-role">{t("invites.form.roleLabel")}</Label>
        <Controller
          control={control}
          name="role"
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange}>
              <SelectTrigger id="invite-role" className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">
                  {t("members.role.member")}
                </SelectItem>
                <SelectItem value="admin">{t("members.role.admin")}</SelectItem>
              </SelectContent>
            </Select>
          )}
        />
      </div>
      <Button type="submit" disabled={pending} className="shrink-0">
        {pending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <UserPlus className="size-4" />
        )}
        {t("invites.form.submitButton")}
      </Button>
    </form>
  )
}
