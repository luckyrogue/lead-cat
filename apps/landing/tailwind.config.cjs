/** @type {import("tailwindcss").Config} */
module.exports = {
  presets: [require("@leadcat/config/tailwind")],
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "../../packages/ui/src/**/*.{ts,tsx}",
  ],
};
