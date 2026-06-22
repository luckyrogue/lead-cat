import { LEAD_CAT_MODEL_URL } from "@leadcat/ui/3d"

export function heroAssetLinks() {
  return [
    {
      rel: "preload",
      href: LEAD_CAT_MODEL_URL,
      as: "fetch",
      crossOrigin: "anonymous",
    },
  ]
}
