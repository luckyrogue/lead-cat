import { cpSync, mkdirSync, readdirSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const pkgRoot = dirname(fileURLToPath(import.meta.url))
const brandRoot = join(pkgRoot, "..")
const repoRoot = join(brandRoot, "../..")
const brandPublic = join(brandRoot, "public")
const apps = ["apps/landing", "apps/admin", "apps/mini-app"]

for (const app of apps) {
  const dest = join(repoRoot, app, "public")
  mkdirSync(dest, { recursive: true })
  for (const file of readdirSync(brandPublic)) {
    cpSync(join(brandPublic, file), join(dest, file), { force: true })
  }
  console.log(`synced brand public -> ${app}/public`)
}
