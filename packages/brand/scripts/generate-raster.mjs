import { cpSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

import { Resvg } from "@resvg/resvg-js"

const pkgRoot = dirname(fileURLToPath(import.meta.url))
const brandRoot = join(pkgRoot, "..")
const assetsDir = join(brandRoot, "assets")
const publicDir = join(brandRoot, "public")

mkdirSync(publicDir, { recursive: true })

for (const name of ["favicon.svg", "logo.svg", "logo-mark.svg", "og-image.svg"]) {
  cpSync(join(assetsDir, name), join(publicDir, name))
}

function rasterize(svgName, outName, width) {
  const svg = readFileSync(join(assetsDir, svgName), "utf-8")
  const resvg = new Resvg(svg, { fitTo: { mode: "width", value: width } })
  writeFileSync(join(publicDir, outName), resvg.render().asPng())
}

rasterize("logo-mark.svg", "apple-touch-icon.png", 180)
rasterize("favicon.svg", "favicon-32.png", 32)
rasterize("favicon.svg", "favicon-16.png", 16)
rasterize("og-image.svg", "og-image.png", 1200)

writeFileSync(
  join(publicDir, "site.webmanifest"),
  `${JSON.stringify(
    {
      name: "Lead Cat",
      short_name: "Lead Cat",
      description:
        "Meetings your team will actually love. Google Calendar sync and Telegram Mini App.",
      icons: [
        {
          src: "/favicon.svg",
          sizes: "any",
          type: "image/svg+xml",
          purpose: "any",
        },
        {
          src: "/apple-touch-icon.png",
          sizes: "180x180",
          type: "image/png",
          purpose: "any",
        },
        {
          src: "/og-image.png",
          sizes: "1200x630",
          type: "image/png",
          purpose: "any",
        },
      ],
      theme_color: "#F2603F",
      background_color: "#FFF8EF",
      display: "standalone",
      lang: "ru",
    },
    null,
    2,
  )}\n`,
)

console.log("brand assets generated in packages/brand/public")
