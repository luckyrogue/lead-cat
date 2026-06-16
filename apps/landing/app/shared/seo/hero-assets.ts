const LEAD_CAT_MODEL_URL = "/models/lead-cat.glb"

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
