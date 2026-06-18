import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Loader2,
} from "@leadcat/ui"
import { useMemo } from "react"
import { useForm } from "react-hook-form"
import { Navigate, useNavigate } from "react-router"
import { z } from "zod"

import { AuthLocaleShell } from "~/components/auth-locale-shell"
import { BrandLogo } from "~/components/brand-logo"
import { PageLoading } from "~/components/page-loading"
import { useAcceptInvite, useDeclineInvite, useMyInvites } from "~/entities/invite/queries"
import { useCreateOrg } from "~/entities/org/queries"
import { setActiveOrgId } from "~/shared/api/active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"
import { toastError } from "~/shared/lib/toast"

export default function OnboardingPage() {
  const { data: me, isPending } = useMe()

  return (
    <AuthLocaleShell>
      <OnboardingBody me={me} isPending={isPending} />
    </AuthLocaleShell>
  )
}

function OnboardingBody({
  me,
  isPending,
}: {
  me: ReturnType<typeof useMe>["data"]
  isPending: boolean
}) {
  const t = useT()
  const navigate = useNavigate()
  const createOrg = useCreateOrg()
  const { data: invites = [] } = useMyInvites()
  const accept = useAcceptInvite()
  const decline = useDeclineInvite()
  const schema = useMemo(
    () =>
      z.object({
        name: z.string().min(2, t("auth.onboarding.errors.nameMin")),
      }),
    [t]
  )
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  })

  if (isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <PageLoading>{t("auth.loading")}</PageLoading>
      </div>
    )
  }

  if (!me) {
    return <Navigate to="/login" replace />
  }

  if (me.organizations.length > 0) {
    return <Navigate to="/" replace />
  }

  function onSubmit(values: z.infer<typeof schema>) {
    createOrg.mutate(values.name, {
      onSuccess: (org) => {
        setActiveOrgId(org.id)
        navigate("/", { replace: true })
      },
      onError: (error) => toastError(error, t("auth.onboarding.createFailed")),
    })
  }

  return (
    <div className="flex min-h-svh items-start justify-center overflow-hidden px-4 pt-16 pb-10 sm:items-center sm:px-6 sm:py-10">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,oklch(0.9_0.07_55/0.45),transparent_28%),radial-gradient(circle_at_bottom_right,oklch(0.93_0.08_92/0.6),transparent_36%)]" />
      <div className="relative flex w-full max-w-md flex-col gap-6">
        <BrandLogo subtitle={t("auth.subtitle")} />
        {invites.length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>{t("onboarding.invites.title")}</CardTitle>
              <CardDescription>{t("onboarding.invites.description")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {invites.map((inv) => (
                <div key={inv.invite_id} className="flex items-center justify-between gap-3">
                  <span className="font-medium">{inv.org_name}</span>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      disabled={accept.isPending}
                      onClick={() =>
                        accept.mutate(inv.invite_id, {
                          onSuccess: () => {
                            setActiveOrgId(inv.organization_id)
                            navigate("/", { replace: true })
                          },
                          onError: (e) => toastError(e, t("onboarding.invites.acceptFailed")),
                        })
                      }
                    >
                      {t("onboarding.invites.accept")}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={decline.isPending}
                      onClick={() => decline.mutate(inv.invite_id)}
                    >
                      {t("onboarding.invites.decline")}
                    </Button>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        ) : null}
        <Card>
          <CardHeader>
            <CardTitle>{t("auth.onboarding.title")}</CardTitle>
            <CardDescription>{t("auth.onboarding.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="org-name">{t("auth.onboarding.nameLabel")}</Label>
                <Input
                  id="org-name"
                  placeholder={t("auth.onboarding.namePlaceholder")}
                  autoFocus
                  {...register("name")}
                />
                {errors.name ? (
                  <p className="text-sm text-destructive" role="alert">
                    {errors.name.message}
                  </p>
                ) : null}
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={createOrg.isPending}
              >
                {createOrg.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                {t("auth.onboarding.submit")}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
