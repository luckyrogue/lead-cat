import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Input,
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

import {
  InlineFormAction,
  InlineFormField,
  InlineFormRow,
} from "~/components/inline-form-row"
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
    >
      <InlineFormRow>
        <InlineFormField
          col={1}
          htmlFor="invite-email"
          label={t("invites.form.emailLabel")}
          error={errors.email?.message}
        >
          <Input
            id="invite-email"
            type="email"
            placeholder={t("invites.form.emailPlaceholder")}
            {...register("email")}
          />
        </InlineFormField>

        <InlineFormField
          col={2}
          htmlFor="invite-role"
          label={t("invites.form.roleLabel")}
        >
          <Controller
            control={control}
            name="role"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger id="invite-role" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="member">
                    {t("members.role.member")}
                  </SelectItem>
                  <SelectItem value="admin">
                    {t("members.role.admin")}
                  </SelectItem>
                </SelectContent>
              </Select>
            )}
          />
        </InlineFormField>

        <InlineFormAction>
          <Button type="submit" disabled={pending} className="w-full sm:w-auto">
            {pending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <UserPlus className="size-4" />
            )}
            {t("invites.form.submitButton")}
          </Button>
        </InlineFormAction>
      </InlineFormRow>
    </form>
  )
}
