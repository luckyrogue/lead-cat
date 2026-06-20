import {
  type RouteConfig,
  index,
  layout,
  route,
} from "@react-router/dev/routes"

export default [
  layout("routes/_tma.tsx", [
    index("routes/_tma._index.tsx"),
    route("meetings", "routes/_tma.meetings._index.tsx"),
    route("meetings/create", "routes/_tma.meetings.create.tsx"),
    route("meetings/:meetingId", "routes/_tma.meetings.$meetingId.tsx"),
    route("checker", "routes/_tma.checker.tsx"),
    route("profile", "routes/_tma.profile._index.tsx"),
    route("profile/colleague", "routes/_tma.profile.colleague.tsx"),
  ]),
] satisfies RouteConfig
