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
                sm: "0 1px 2px 0 rgba(0, 0, 0, 0.05)",
                DEFAULT: "0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px -1px rgba(0, 0, 0, 0.1)",
                md: "0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)",
                lg: "0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)",
                xl: "0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)",
                "2xl": "0 25px 50px -12px rgba(0, 0, 0, 0.25)",
                inner: "inset 0 2px 4px 0 rgba(0, 0, 0, 0.05)",
                none: "none",
                soft: "0 4px 16px rgba(0,0,0,0.04)",
                "soft-hover": "0 12px 32px -12px rgba(0, 0, 0, 0.08)",
            },
        },
    },
    plugins: [],
};

export default config;
