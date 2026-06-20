import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  CheckCircle2,
  Input,
  Label,
  Loader2,
  Mail,
} from "@leadcat/ui"
import { useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"

import { requestMagicLink, ssoStartUrl } from "~/shared/auth/api"
import { useLocale, useT } from "~/shared/i18n/context"
import { toastError } from "~/shared/lib/toast"

function GoogleMark() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" aria-hidden="true">
      <path
        fill="#EA4335"
        d="M12 10.8v3.6h5.1c-.2 1.3-1.6 3.9-5.1 3.9-3.1 0-5.6-2.6-5.6-5.7s2.5-5.7 5.6-5.7c1.8 0 3 .8 3.6 1.4l2.5-2.4C16.6 3.9 14.5 3 12 3 6.9 3 2.8 7.1 2.8 12S6.9 21 12 21c5.2 0 8.7-3.7 8.7-8.8 0-.6-.1-1-.2-1.4H12z"
      />
    </svg>
  )
}

function MicrosoftMark() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" aria-hidden="true">
      <path fill="#F25022" d="M3 3h8.5v8.5H3z" />
      <path fill="#7FBA00" d="M12.5 3H21v8.5h-8.5z" />
      <path fill="#00A4EF" d="M3 12.5h8.5V21H3z" />
      <path fill="#FFB900" d="M12.5 12.5H21V21h-8.5z" />
    </svg>
  )
}

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
      toastError(error, t("auth.login.magicLinkFailed"))
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
            <GoogleMark />
            {t("auth.login.continueGoogle")}
          </Button>
          <Button
            variant="outline"
            className="w-full"
            onClick={() => {
              window.location.href = ssoStartUrl("microsoft")
            }}
          >
            <MicrosoftMark />
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
