/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
        "./App.{js,jsx,ts,tsx}",
        "./app/**/*.{js,jsx,ts,tsx}",
        "./src/**/*.{js,jsx,ts,tsx}",
    ],
    presets: [require("nativewind/preset")],
    theme: {
        extend: {
            colors: {
                surface: '#F2F2F7',
                surfaceElevated: '#FFFFFF',
                line: 'rgba(60,60,67,0.12)',
                textPrimary: '#000000',
                textSecondary: '#3C3C43',
                textTertiary: '#8E8E93',
                accent: '#6750A4',
            },
        },
    },
    plugins: [],
}
