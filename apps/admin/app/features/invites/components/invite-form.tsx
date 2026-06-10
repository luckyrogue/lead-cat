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
      className="flex flex-col gap-4 sm:flex-row sm:items-end"
    >
      <div className="flex-1 space-y-2">
        <Label htmlFor="invite-email">Email</Label>
        <Input
          id="invite-email"
          type="email"
          placeholder="teammate@company.com"
          {...register("email")}
        />
        {errors.email ? (
          <p className="text-sm text-destructive" role="alert">
            {errors.email.message}
          </p>
        ) : null}
      </div>
      <div className="space-y-2">
        <Label htmlFor="invite-role">Role</Label>
        <Controller
          control={control}
          name="role"
          render={({ field }) => (
            <Select value={field.value} onValueChange={field.onChange}>
              <SelectTrigger id="invite-role" className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">member</SelectItem>
                <SelectItem value="admin">admin</SelectItem>
              </SelectContent>
            </Select>
          )}
        />
      </div>
      <Button type="submit" disabled={pending}>
        {pending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <UserPlus className="size-4" />
        )}
        Invite
      </Button>
    </form>
  )
}
