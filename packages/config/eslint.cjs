const js = require("@eslint/js");
const tseslint = require("typescript-eslint");
const react = require("eslint-plugin-react");
const reactHooks = require("eslint-plugin-react-hooks");
const importPlugin = require("eslint-plugin-import");

const fsdZones = [
  { target: "**/src/shared/**", from: ["**/src/app/**", "**/src/routes/**", "**/src/components/**", "**/src/features/**", "**/src/entities/**"] },
  { target: "**/src/entities/**", from: ["**/src/app/**", "**/src/routes/**", "**/src/components/**", "**/src/features/**"] },
  { target: "**/src/features/*/**", from: ["**/src/app/**", "**/src/routes/**", "**/src/components/**"] },
  { target: "**/src/components/**", from: ["**/src/app/**", "**/src/routes/**"] },
  { target: "**/src/routes/**", from: ["**/src/app/**"] },
];

module.exports = [
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: {
      react,
      "react-hooks": reactHooks,
      import: importPlugin,
    },
    languageOptions: {
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
    },
    settings: {
      react: { version: "detect" },
      "import/resolver": {
        typescript: { alwaysTryTypes: true },
        node: { extensions: [".js", ".jsx", ".ts", ".tsx"] },
      },
    },
    rules: {
      "react/react-in-jsx-scope": "off",
      "react/prop-types": "off",
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "import/no-restricted-paths": [
        "error",
        {
          zones: [
            ...fsdZones,
            {
              target: "**/src/features/auth/**",
              from: "**/src/features/!(auth)/**",
            },
            {
              target: "**/src/features/dashboard/**",
              from: "**/src/features/!(dashboard)/**",
            },
            {
              target: "**/src/features/home/**",
              from: "**/src/features/!(home)/**",
            },
            {
              target: "**/src/features/landing/**",
              from: "**/src/features/!(landing)/**",
            },
          ],
        },
      ],
    },
  },
];
