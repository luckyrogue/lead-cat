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
import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { Navigate, useNavigate } from "react-router"
import { z } from "zod"

import { BrandLogo } from "~/components/brand-logo"
import { PageLoading } from "~/components/page-loading"
import { useCreateOrg } from "~/entities/org/queries"
import { setActiveOrgId } from "~/shared/api/active-org"
import { useMe } from "~/shared/auth/use-me"
import { toastError } from "~/shared/lib/toast"

const schema = z.object({
  name: z.string().min(2, "Give your organization a name"),
})

type FormValues = z.infer<typeof schema>

export default function OnboardingPage() {
  const navigate = useNavigate()
  const { data: me, isPending } = useMe()
  const createOrg = useCreateOrg()
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  })

  useEffect(() => {
    if (me && me.organizations.length > 0) {
      navigate("/", { replace: true })
    }
  }, [me, navigate])

  if (isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <PageLoading>Loading…</PageLoading>
      </div>
    )
  }

  if (!me) {
    return <Navigate to="/login" replace />
  }

  function onSubmit(values: FormValues) {
    createOrg.mutate(values.name, {
      onSuccess: (org) => {
        setActiveOrgId(org.id)
        navigate("/", { replace: true })
      },
      onError: (error) => toastError(error, "Could not create organization."),
    })
  }

  return (
    <div className="relative flex min-h-svh items-center justify-center overflow-hidden px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,oklch(0.9_0.07_55/0.45),transparent_28%),radial-gradient(circle_at_bottom_right,oklch(0.93_0.08_92/0.6),transparent_36%)]" />
      <div className="relative flex w-full max-w-md flex-col gap-6">
        <BrandLogo subtitle="Admin Panel" />
        <Card>
          <CardHeader>
            <CardTitle>Create your organization</CardTitle>
            <CardDescription>
              Set up a workspace to manage members, invites, and meetings.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="org-name">Organization name</Label>
                <Input
                  id="org-name"
                  placeholder="Acme Inc."
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
                Create organization
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
