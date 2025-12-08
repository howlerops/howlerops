import tailwindcssAnimate from "tailwindcss-animate"

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      keyframes: {
        "highlight-fade": {
          "0%": { backgroundColor: "hsl(var(--primary) / 0.15)" },
          "100%": { backgroundColor: "transparent" },
        },
      },
      animation: {
        "highlight-fade": "highlight-fade 2s ease-out forwards",
      },
    },
  },
  plugins: [tailwindcssAnimate],
}