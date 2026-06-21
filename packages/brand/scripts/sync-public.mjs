import { cpSync, mkdirSync, readdirSync, renameSync, rmSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const pkgRoot = dirname(fileURLToPath(import.meta.url))
const brandRoot = join(pkgRoot, "..")
const repoRoot = join(brandRoot, "../..")
const brandPublic = join(brandRoot, "public")
const apps = ["apps/landing", "apps/admin", "apps/mini-app"]

// turbo runs this sync concurrently — one `prebuild` per app — and every run
// copies into all three public/ dirs. Two processes writing the same dest with
// a plain cpSync race and throw EEXIST mid-copy. Copy to a process-unique temp
// file, then renameSync it into place: rename is atomic and overwrites, so
// concurrent runs are safe (last writer wins, identical bytes).
for (const app of apps) {
  const dest = join(repoRoot, app, "public")
  mkdirSync(dest, { recursive: true })
  for (const file of readdirSync(brandPublic)) {
    const finalPath = join(dest, file)
    const tmpPath = `${finalPath}.tmp.${process.pid}`
    try {
      cpSync(join(brandPublic, file), tmpPath, { force: true })
      renameSync(tmpPath, finalPath)
    } finally {
      rmSync(tmpPath, { force: true })
    }
  }
  console.log(`synced brand public -> ${app}/public`)
}
