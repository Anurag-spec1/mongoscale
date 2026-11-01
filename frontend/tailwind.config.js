export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: { 950: '#05070d', 900: '#080c15', 800: '#0d1320', 700: '#131b2d' },
        line: '#1c2740',
        brand: { DEFAULT: '#22d3a5', dim: '#0f8a6c' },
        danger: { DEFAULT: '#ff4d6a', dim: '#7a1d2e' },
        warn: { DEFAULT: '#fbbf24', dim: '#7a5a0a' },
        info: { DEFAULT: '#38bdf8', dim: '#0c4a6e' },
        violet: { DEFAULT: '#a78bfa' },
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      boxShadow: {
        glowGreen: '0 0 40px -10px rgba(34, 211, 165, 0.45)',
        glowRed: '0 0 40px -8px rgba(255, 77, 106, 0.55)',
      },
      keyframes: {
        sweep: { '0%': { transform: 'translateX(-100%)' }, '100%': { transform: 'translateX(100%)' } },
        heartbeat: { '0%, 100%': { transform: 'scale(1)', opacity: '1' }, '50%': { transform: 'scale(1.35)', opacity: '0.55' } },
      },
      animation: {
        sweep: 'sweep 2.2s linear infinite',
        heartbeat: 'heartbeat 1.4s ease-in-out infinite',
      },
    },
  },
  plugins: [],
};
