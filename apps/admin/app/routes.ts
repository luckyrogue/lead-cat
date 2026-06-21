import {
  type RouteConfig,
  index,
  layout,
  route,
} from "@react-router/dev/routes"

export default [
  layout("routes/_auth.tsx", [
    route("login", "routes/_auth.login.tsx"),
    route("auth/magic", "routes/_auth.magic.tsx"),
  ]),
  layout("routes/_app.tsx", [
    index("routes/_app._index.tsx"),
    route("members", "routes/_app.members._index.tsx"),
    route("invites", "routes/_app.invites._index.tsx"),
    route("meetings", "routes/_app.meetings._index.tsx"),
    route("booking", "routes/_app.booking._index.tsx"),
    route("settings", "routes/_app.settings.tsx"),
  ]),
  route("onboarding", "routes/onboarding.tsx"),
  route("logout", "routes/logout.tsx"),
  route("book/:slug", "routes/book.$slug.tsx"),
] satisfies RouteConfig
