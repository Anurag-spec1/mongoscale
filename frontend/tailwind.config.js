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
        sans: ['Inter Variable', 'sans-serif'],
        mono: ['JetBrains Mono Variable', 'monospace'],
      },
      spacing: {
        1.5: '6px',
        2.5: '10px',
        3.5: '14px',
      },
      borderRadius: {
        sm: '8px',
        md: '10px',
        lg: '12px',
        xl: '16px',
        '2xl': '20px',
        '3xl': '24px',
      },
      fontSize: {
        xxs: ['0.6875rem', { lineHeight: '1rem' }],
        metric: ['1.75rem', { lineHeight: '2rem' }],
      },
    },
  },
  plugins: [],
};
