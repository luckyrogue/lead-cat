import { type RouteConfig, index, layout } from "@react-router/dev/routes"

export default [
  layout("routes/_app.tsx", [index("routes/_app._index.tsx")]),
] satisfies RouteConfig
