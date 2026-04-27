/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"]
      },
      colors: {
        ink: "#101828",
        brand: {
          50: "#eef8ff",
          100: "#d8efff",
          500: "#0ea5e9",
          600: "#0284c7",
          700: "#0369a1"
        },
        mint: "#10b981"
      }
    }
  },
  plugins: []
};
