import { useEffect, type ReactNode } from 'react';
import { animate, motion, useMotionValue, useTransform } from 'framer-motion';
import { classNames, formatNumber } from '../utils/format';

export type Tone = 'neutral' | 'good' | 'warn' | 'bad' | 'info';

const toneText: Record<Tone, string> = {
  neutral: 'text-slate-100',
  good: 'text-status-good',
  warn: 'text-status-warn',
  bad: 'text-status-danger',
  info: 'text-info',
};

const toneBorder: Record<Tone, string> = {
  neutral: 'border-line',
  good: 'border-status-good/40',
  warn: 'border-status-warn/50',
  bad: 'border-status-danger/70',
  info: 'border-info/40',
};

const toneStroke: Record<Tone, string> = {
  neutral: '#94a3b8',
  good: '#22d3a5',
  warn: '#fbbf24',
  bad: '#ff4d6a',
  info: '#38bdf8',
};

export function toneColor(tone: Tone): string {
  return toneStroke[tone];
}

interface CounterProps {
  value: number;
  format?: (value: number) => string;
  className?: string;
}

export function AnimatedCounter({ value, format = formatNumber, className }: CounterProps) {
  const motionValue = useMotionValue(value);
  const text = useTransform(motionValue, (latest) => format(latest));

  useEffect(() => {
    const controls = animate(motionValue, value, { duration: 0.6, ease: 'easeOut' });
    return () => controls.stop();
  }, [motionValue, value]);

  return <motion.span className={className}>{text}</motion.span>;
}

interface RingProps {
  value: number;
  max: number;
  label: string;
  caption?: string;
  tone?: Tone;
  size?: number;
}

export function ProgressRing({ value, max, label, caption, tone = 'good', size = 104 }: RingProps) {
  const radius = size / 2 - 9;
  const circumference = 2 * Math.PI * radius;
  const fraction = max > 0 ? Math.min(Math.max(value / max, 0), 1) : 0;
  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90">
          <circle cx={size / 2} cy={size / 2} r={radius} stroke="#1c2740" strokeWidth={8} fill="none" />
          <motion.circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={toneStroke[tone]}
            strokeWidth={8}
            strokeLinecap="round"
            fill="none"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset: circumference * (1 - fraction) }}
            transition={{ duration: 0.7, ease: 'easeOut' }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={classNames('text-lg font-bold tabular-nums', toneText[tone])}>{Math.round(fraction * 100)}%</span>
          {caption ? <span className="text-[10px] text-slate-500">{caption}</span> : null}
        </div>
      </div>
      <span className="text-xs font-medium text-slate-400">{label}</span>
    </div>
  );
}

export function StatusPill({ tone, children }: { tone: Tone; children: ReactNode }) {
  const palette: Record<Tone, string> = {
    neutral: 'bg-ink-700 text-slate-300 border-line',
    good: 'bg-status-good/10 text-status-good border-status-good/40',
    warn: 'bg-status-warn/10 text-status-warn border-status-warn/40',
    bad: 'bg-status-danger/10 text-status-danger border-status-danger/40',
    info: 'bg-accent-muted text-brand border-line',
  };
  return (
    <span className={classNames('inline-flex items-center gap-1.5 rounded-sm border px-2 py-0.5 text-xs font-medium', palette[tone])}>
      <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: toneStroke[tone] }} />
      {children}
    </span>
  );
}

interface MetricCardProps {
  label: string;
  value: number;
  format?: (value: number) => string;
  unit?: string;
  hint?: string;
  tone?: Tone;
}

export function MetricCard({ label, value, format, unit, hint, tone = 'neutral' }: MetricCardProps) {
  return (
    <div className={classNames('panel border-t-2 p-4', toneBorder[tone])}>
      <div className="text-xs font-medium text-slate-400">{label}</div>
      <div className="mt-2 flex items-baseline gap-1.5">
        <AnimatedCounter value={value} format={format} className={classNames('text-3xl font-bold tabular-nums', toneText[tone])} />
        {unit ? <span className="text-sm text-slate-500">{unit}</span> : null}
      </div>
      {hint ? <div className="mt-1 text-xs text-slate-500">{hint}</div> : null}
    </div>
  );
}
