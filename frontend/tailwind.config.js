export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: { 950: 'var(--canvas)', 900: 'var(--surface)', 800: 'var(--surface-raised)', 700: 'var(--surface-hover)' },
        line: 'var(--border)',
        brand: { DEFAULT: 'var(--accent)', dim: 'var(--accent-muted)' },
        danger: { DEFAULT: 'var(--status-danger)', dim: 'var(--status-danger-muted)' },
        warn: { DEFAULT: 'var(--status-warning)', dim: 'var(--status-warning-muted)' },
        info: { DEFAULT: 'var(--accent)', dim: 'var(--accent-muted)' },
        violet: { DEFAULT: 'var(--accent)' },
        status: {
          good: 'var(--status-good)',
          warn: 'var(--status-warning)',
          danger: 'var(--status-danger)',
        },
      },
      fontFamily: {
        sans: ['InterVariable', 'Inter', 'sans-serif'],
        mono: ['JetBrainsMonoVariable', 'JetBrains Mono', 'monospace'],
      },
      spacing: {
        1.5: '6px',
        2.5: '10px',
        3.5: '14px',
      },
      borderRadius: {
        sm: '6px',
        md: '8px',
        lg: '8px',
        xl: '8px',
        '2xl': '8px',
        '3xl': '8px',
      },
      fontSize: {
        xxs: ['0.6875rem', { lineHeight: '1rem' }],
        metric: ['1.75rem', { lineHeight: '2rem' }],
      },
    },
  },
  plugins: [],
};
