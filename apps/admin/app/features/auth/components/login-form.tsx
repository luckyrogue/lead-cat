import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CheckCircle2,
  GoogleMark,
  Input,
  Label,
  Loader2,
  Mail,
  MicrosoftMark,
  toastError,
} from "@leadcat/ui"
import { useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"

import { requestMagicLink, ssoStartUrl } from "~/shared/auth/api"
import { useLocale, useT } from "~/shared/i18n/context"

export function LoginForm() {
  const t = useT()
  const locale = useLocale()
  const [sentTo, setSentTo] = useState<string | null>(null)
  const schema = useMemo(
    () =>
      z.object({
        email: z.string().email(t("auth.login.errors.emailInvalid")),
      }),
    [t]
  )
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: { email: "" },
  })

  async function onSubmit(values: z.infer<typeof schema>) {
    try {
      await requestMagicLink(values.email, locale)
      setSentTo(values.email)
    } catch (error) {
      toastError(error, t, "auth.login.magicLinkFailed")
    }
  }

  if (sentTo) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center gap-3 py-8 text-center">
          <span className="grid size-12 place-items-center rounded-full border border-primary/15 bg-primary/10 text-primary">
            <CheckCircle2 className="size-6" />
          </span>
          <div className="space-y-1">
            <p className="text-base font-semibold text-foreground">
              {t("auth.login.inboxTitle")}
            </p>
            <p className="text-sm text-muted-foreground">
              {t("auth.login.inboxDescription", { email: sentTo })}
            </p>
          </div>
          <Button
            variant="ghost"
            className="mt-1"
            onClick={() => setSentTo(null)}
          >
            {t("auth.login.useDifferentEmail")}
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("auth.login.title")}</CardTitle>
        <CardDescription>{t("auth.login.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-2.5">
          <Button
            variant="outline"
            className="w-full"
            onClick={() => {
              window.location.href = ssoStartUrl("google")
            }}
          >
            <GoogleMark className="size-4" />
            {t("auth.login.continueGoogle")}
          </Button>
          <Button
            variant="outline"
            className="w-full"
            onClick={() => {
              window.location.href = ssoStartUrl("microsoft")
            }}
          >
            <MicrosoftMark className="size-4" />
            {t("auth.login.continueMicrosoft")}
          </Button>
        </div>

        <div className="flex items-center gap-3">
          <span className="h-px flex-1 bg-border/70" />
          <span className="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
            {t("auth.login.divider")}
          </span>
          <span className="h-px flex-1 bg-border/70" />
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">{t("auth.login.emailLabel")}</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder={t("auth.login.emailPlaceholder")}
              {...register("email")}
            />
            {errors.email ? (
              <p className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            ) : null}
          </div>
          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Mail className="size-4" />
            )}
            {t("auth.login.submit")}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
