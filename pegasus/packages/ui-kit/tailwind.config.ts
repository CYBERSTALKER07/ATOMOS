import type { Config } from "tailwindcss";

const config: Config = {
    content: [
        "../../apps/**/*.{js,ts,jsx,tsx,mdx}",
        "./src/**/*.{js,ts,jsx,tsx,mdx}",
    ],
    theme: {
        fontFamily: {
            sans: ["Inter", "Geist", "sans-serif"],
        },
        extend: {
            colors: {
                background: "#ffffff",
                foreground: "#000000",
                slate: {
                    50: '#f8fafc',
                    100: '#f1f5f9',
                    200: '#e2e8f0',
                    300: '#cbd5e1',
                    400: '#94a3b8',
                    500: '#64748b',
                    600: '#475569',
                    700: '#334155',
                    800: '#1e293b',
                    900: '#0f172a',
                    950: '#020617',
                },
                primary: {
                    DEFAULT: "#000000",
                    foreground: "#ffffff",
                },
                secondary: {
                    DEFAULT: "#f1f5f9", // slate-100
                    foreground: "#0f172a", // slate-900
                },
                border: "#e2e8f0", // slate-200
            },
            boxShadow: {
                // Enforcing NO soft consumer shadows. Only hard borders or minimal crisp shadows if absolutely necessary
                sm: "none",
                DEFAULT: "none",
                md: "none",
                lg: "none",
                xl: "none",
                "2xl": "none",
            },
            backgroundImage: {
                // Enforcing NO gradients
                'none': 'none',
                'gradient-to-t': 'none',
                'gradient-to-tr': 'none',
                'gradient-to-r': 'none',
                'gradient-to-br': 'none',
                'gradient-to-b': 'none',
                'gradient-to-bl': 'none',
                'gradient-to-l': 'none',
                'gradient-to-tl': 'none',
            }
        },
    },
    plugins: [],
};

export default config;
