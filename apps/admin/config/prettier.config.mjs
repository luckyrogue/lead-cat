import path from "node:path"
import { fileURLToPath } from "node:url"

import base from "@leadcat/config/prettier"

const rootDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "..")

export default {
  ...base,
  tailwindStylesheet: path.join(rootDir, "app/app.css"),
}
