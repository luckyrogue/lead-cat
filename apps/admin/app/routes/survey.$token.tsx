import { Card, CardContent, CardHeader, CardTitle } from "@leadcat/ui"
import { useEffect, useState } from "react"
import { useParams } from "react-router"

import { AuthLocaleShell } from "~/components/auth-locale-shell"
import { useT } from "~/shared/i18n/context"
import {
  getPublicSurvey,
  type PublicSurvey,
} from "~/features/public-survey/api"
import { SurveyForm } from "~/features/public-survey/components/survey-form"

type State =
  | { kind: "loading" }
  | { kind: "form"; survey: PublicSurvey }
  | { kind: "unavailable" }
  | { kind: "done" }

export default function Route() {
  return (
    <AuthLocaleShell>
      <SurveyScreen />
    </AuthLocaleShell>
  )
}

function SurveyScreen() {
  const t = useT()
  const { token = "" } = useParams()
  const [state, setState] = useState<State>({ kind: "loading" })

  useEffect(() => {
    let active = true
    getPublicSurvey(token).then((r) => {
      if (!active) return
      if (r.status === "ok") setState({ kind: "form", survey: r.survey })
      else if (r.status === "completed") setState({ kind: "done" })
      else setState({ kind: "unavailable" })
    })
    return () => {
      active = false
    }
  }, [token])

  return (
    <div className="mx-auto flex min-h-svh max-w-md items-center px-4 py-10">
      <Card className="w-full">
        {state.kind === "loading" && (
          <CardContent className="py-10 text-center">
            {t("common.loading")}
          </CardContent>
        )}
        {state.kind === "unavailable" && (
          <CardContent className="py-10 text-center text-muted-foreground">
            {t("publicSurvey.unavailable")}
          </CardContent>
        )}
        {state.kind === "done" && (
          <CardContent className="py-10 text-center">
            {t("publicSurvey.thanks")}
          </CardContent>
        )}
        {state.kind === "form" && (
          <>
            <CardHeader>
              <CardTitle className="text-xl">
                {state.survey.survey_name}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <SurveyForm
                token={token}
                questions={state.survey.questions}
                onResult={(r) =>
                  setState(
                    r === "ok" || r === "completed"
                      ? { kind: "done" }
                      : { kind: "form", survey: state.survey }
                  )
                }
              />
            </CardContent>
          </>
        )}
      </Card>
    </div>
  )
}
