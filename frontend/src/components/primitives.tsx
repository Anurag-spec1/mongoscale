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
    const controls = animate(motionValue, value, { duration: 0.18, ease: 'easeOut' });
    return () => controls.stop();
  }, [motionValue, value]);

  return <motion.span className={className}>{text}</motion.span>;
}

interface MeterProps {
  value: number;
  max: number;
  label: string;
  valueLabel: string;
  tone?: Tone;
}

export function ProgressMeter({ value, max, label, valueLabel, tone = 'good' }: MeterProps) {
  const fraction = max > 0 ? Math.min(Math.max(value / max, 0), 1) : 0;
  const meterColor: Record<Tone, string> = {
    neutral: 'bg-slate-500',
    good: 'bg-status-good',
    warn: 'bg-status-warn',
    bad: 'bg-status-danger',
    info: 'bg-brand',
  };
  return (
    <div className="min-w-0 space-y-2">
      <div className="flex items-baseline justify-between gap-2">
        <span className="truncate text-xs text-slate-400">{label}</span>
        <span className="shrink-0 font-mono text-xs text-slate-200">{valueLabel}</span>
      </div>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={max}
        aria-valuenow={Math.min(Math.max(value, 0), max)}
        className="h-1.5 overflow-hidden rounded-sm bg-ink-700"
      >
        <div className={classNames('h-full transition-[width] duration-200', meterColor[tone])} style={{ width: `${fraction * 100}%` }} />
      </div>
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
  value: number | null;
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
        {value === null ? <span className="text-3xl font-bold text-slate-500">--</span> : <AnimatedCounter value={value} format={format} className={classNames('text-3xl font-bold tabular-nums', toneText[tone])} />}
        {unit ? <span className="text-sm text-slate-500">{unit}</span> : null}
      </div>
      {hint ? <div className="mt-1 text-xs text-slate-500">{hint}</div> : null}
    </div>
  );
}
