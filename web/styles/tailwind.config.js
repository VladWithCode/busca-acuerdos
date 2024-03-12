/** @type {import('tailwindcss').Config} */
module.exports = {
  relative: true,
  content: ["web/templates/**/*.html"],
  theme: {
    extend: {
        colors: {
            "primary-500": "#BE3CC7",
            "primary-600": "#8E3095",
            "primary-700": "#6A266F",
            "primary-800": "#461A49",
            "primary-900": "#220D23",
            "secondary-500": "#FC4A53",
            "secondary-600": "#E0464E",
            "secondary-700": "#A7383D",
            "secondary-800": "#6E272A",
            "secondary-900": "#351415",
            "accent-500": "#684DCD",
            "accent-600": "#523E9D",
            "accent-700": "#3F3075",
            "accent-800": "#2A214D",
            "accent-900": "#151125",
        }
    },
  },
  plugins: [],
}

