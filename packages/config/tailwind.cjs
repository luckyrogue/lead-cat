/** @type {import("tailwindcss").Config} */
module.exports = {
  theme: {
    extend: {
      colors: {
        brand: {
          50: "#fdf2fb",
          100: "#fce7f8",
          200: "#fbcff1",
          300: "#f9a8e4",
          400: "#f472d0",
          500: "#e84ab7",
          600: "#cc2c98",
          700: "#a81f7a",
          800: "#8a1c64",
          900: "#721c54",
        },
        accent: {
          50: "#eef9ff",
          100: "#d9f1ff",
          200: "#bce8ff",
          300: "#8edaff",
          400: "#59c3ff",
          500: "#33a6fb",
          600: "#1d87f0",
          700: "#1670dc",
          800: "#185bb2",
          900: "#1a4f8c",
        },
        surface: {
          DEFAULT: "#fffdfb",
          soft: "#fdf6ff",
          muted: "#f4eefb",
          sunk: "#ece4f7",
        },
      },
      borderRadius: {
        DEFAULT: "1rem",
        xl: "1.25rem",
        "2xl": "1.5rem",
        "3xl": "2rem",
      },
      boxShadow: {
        soft: "0 8px 30px -12px rgba(168, 31, 122, 0.25)",
        pop: "0 4px 0 0 rgba(204, 44, 152, 0.35)",
        glow: "0 0 24px -4px rgba(232, 74, 183, 0.45)",
      },
      keyframes: {
        float: {
          "0%, 100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-8px)" },
        },
        pop: {
          "0%": { transform: "scale(0.92)", opacity: "0" },
          "60%": { transform: "scale(1.04)", opacity: "1" },
          "100%": { transform: "scale(1)", opacity: "1" },
        },
      },
      animation: {
        float: "float 4s ease-in-out infinite",
        pop: "pop 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) both",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
      },
    },
  },
};
