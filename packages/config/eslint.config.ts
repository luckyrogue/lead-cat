import js from "@eslint/js"
import pluginImport from "eslint-plugin-import"
import pluginReact from "eslint-plugin-react"
import globals from "globals"
import tseslint from "typescript-eslint"

type LayerZone = {
  target: string
  from: string
  except?: readonly string[]
}

type CreateConfigOptions = {
  rootDir: string
  tsconfigPath: string
  featureModules?: readonly string[]
  crossFeatureExceptions?: Record<string, readonly string[]>
}

const fsdLayerZones: readonly LayerZone[] = [
  { target: "./app/shared", from: "./app/features" },
  { target: "./app/shared", from: "./app/entities" },
  { target: "./app/entities", from: "./app/features" },
]

export function createConfig({
  rootDir,
  tsconfigPath,
  featureModules = [],
  crossFeatureExceptions = {},
}: CreateConfigOptions) {
  const featuresDir = "./app/features"

  const featureBoundaryZones = featureModules.map((feature) => ({
    target: `${featuresDir}/${feature}`,
    from: featuresDir,
    except: [
      `./${feature}`,
      ...(crossFeatureExceptions[feature] ?? []).map(
        (allowed) => `./${allowed}`
      ),
    ],
  }))

  return tseslint.config(
    {
      ignores: [
        "build/**",
        "dist/**",
        "node_modules/**",
        ".react-router/**",
        "app/shared/api/generated/**",
      ],
    },
    {
      languageOptions: {
        parserOptions: {
          tsconfigRootDir: rootDir,
        },
      },
    },
    js.configs.recommended,
    ...tseslint.configs.recommended,
    {
      files: ["config/**/*.{ts,mjs,cjs}"],
      extends: [tseslint.configs.disableTypeChecked],
    },
    {
      files: ["**/*.{js,mjs,cjs,ts,mts,cts,jsx,tsx}"],
      languageOptions: {
        globals: globals.browser,
      },
      plugins: {
        import: pluginImport,
      },
      settings: {
        "import/resolver": {
          typescript: {
            project: tsconfigPath,
          },
        },
      },
      rules: {
        "import/no-restricted-paths": [
          "error",
          {
            zones: [...fsdLayerZones, ...featureBoundaryZones],
          },
        ],
      },
    },
    {
      files: ["**/*.{jsx,tsx}"],
      ...pluginReact.configs.flat.recommended,
      settings: {
        react: {
          version: "detect",
        },
      },
      rules: {
        "react/react-in-jsx-scope": "off",
        "react/prop-types": "off",
      },
    }
  )
}
