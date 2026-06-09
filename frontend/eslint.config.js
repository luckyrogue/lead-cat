import js from "@eslint/js"
import globals from "globals"
import reactHooks from "eslint-plugin-react-hooks"
import importPlugin from "eslint-plugin-import"
import tseslint from "typescript-eslint"

const featureSlices = [
  "auth",
  "home",
  "meetings",
  "meeting-create",
  "checker",
  "auto",
  "profile",
]

const featureZones = featureSlices.flatMap((slice) =>
  featureSlices
    .filter((other) => other !== slice)
    .map((other) => ({
      target: `./src/features/${slice}`,
      from: `./src/features/${other}`,
    }))
)

export default tseslint.config(
  {
    ignores: [
      "dist/**",
      "src/routeTree.gen.ts",
      "src/shared/api/generated/**",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      import: importPlugin,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-hooks/refs": "off",
      "react-hooks/set-state-in-effect": "off",
      "import/no-restricted-paths": [
        "error",
        {
          zones: [
            ...featureZones,
            { target: "./src/entities", from: "./src/features" },
            { target: "./src/entities", from: "./src/components" },
            { target: "./src/shared", from: "./src/features" },
          ],
        },
      ],
    },
    settings: {
      "import/resolver": {
        typescript: {
          project: "./tsconfig.json",
        },
      },
    },
  }
)
