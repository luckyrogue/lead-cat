import path from "node:path"
import { fileURLToPath } from "node:url"

import { createConfig } from "@leadcat/config/eslint"

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")

export default createConfig({
  rootDir,
  tsconfigPath: path.join(rootDir, "tsconfig.json"),
  featureModules: [
    "dashboard",
    "meetings",
    "members",
    "invites",
    "auth",
    "booking",
    "settings",
    "calendar-connections",
  ],
  crossFeatureExceptions: {
    settings: ["calendar-connections"],
  },
})
