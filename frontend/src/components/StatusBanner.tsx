import { AnimatePresence, motion } from 'framer-motion';
import type { MetricPoint, SystemStatus } from '../types';
import { classNames, formatMillis } from '../utils/format';

interface BannerCopy {
  title: string;
  subtitle: string;
  icon: string;
  box: string;
  text: string;
}

const copy: Record<SystemStatus, BannerCopy> = {
  idle: {
    title: 'SYSTEM IDLE',
    subtitle: 'No traffic yet. Start a load test to see how this MongoDB configuration behaves.',
    icon: '◌',
    box: 'border-line bg-ink-800/70',
    text: 'text-slate-300',
  },
  healthy: {
    title: 'SYSTEM HEALTHY / SCALED',
    subtitle: 'Latency is stable, requests are succeeding and the database is keeping up.',
    icon: '✔',
    box: 'border-brand/50 bg-brand/10 shadow-glowGreen',
    text: 'text-brand',
  },
  degraded: {
    title: 'PERFORMANCE DEGRADED',
    subtitle: 'Latency or error rate is climbing. The database is starting to struggle.',
    icon: '▲',
    box: 'border-warn/60 bg-warn/10',
    text: 'text-warn',
  },
  critical: {
    title: 'SYSTEM UNDER STRESS / CRASH RISK',
    subtitle: 'Queries are timing out and requests are failing. This configuration cannot carry the load.',
    icon: '⚠',
    box: 'border-danger/70 bg-danger/15 shadow-glowRed',
    text: 'text-danger',
  },
};

interface Props {
  status: SystemStatus;
  current: MetricPoint | undefined;
}

export function StatusBanner({ status, current }: Props) {
  const content = copy[status];
  const critical = status === 'critical';
  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={status}
        initial={{ opacity: 0, y: -12, scale: 0.98 }}
        animate={critical ? { opacity: 1, y: 0, scale: 1, x: [0, -5, 5, -4, 4, 0] } : { opacity: 1, y: 0, scale: 1, x: 0 }}
        exit={{ opacity: 0, y: 12 }}
        transition={critical ? { duration: 0.55, repeat: Infinity, repeatDelay: 1.6 } : { duration: 0.35 }}
        className={classNames('relative overflow-hidden rounded-2xl border px-6 py-5', content.box)}
      >
        {critical ? (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <div className="absolute inset-y-0 w-1/3 animate-sweep bg-gradient-to-r from-transparent via-danger/25 to-transparent" />
          </div>
        ) : null}
        {status === 'healthy' ? (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <div className="absolute inset-y-0 w-1/3 animate-sweep bg-gradient-to-r from-transparent via-brand/15 to-transparent" />
          </div>
        ) : null}
        <div className="relative flex flex-wrap items-center gap-5">
          <motion.div
            className={classNames('text-5xl leading-none', content.text)}
            animate={critical ? { scale: [1, 1.2, 1] } : { scale: 1 }}
            transition={critical ? { duration: 0.8, repeat: Infinity } : undefined}
          >
            {content.icon}
          </motion.div>
          <div className="min-w-0 flex-1">
            <div className={classNames('text-2xl font-extrabold tracking-wide md:text-3xl', content.text)}>{content.title}</div>
            <div className="mt-1 text-sm text-slate-300">{content.subtitle}</div>
          </div>
          {current && status !== 'idle' ? (
            <div className="flex gap-6 text-right">
              <Stat label="p95" value={formatMillis(current.p95)} className={content.text} />
              <Stat label="errors" value={`${current.errorRate.toFixed(1)}%`} className={content.text} />
              <Stat label="req/s" value={Math.round(current.rps).toLocaleString('en-IN')} className={content.text} />
            </div>
          ) : null}
        </div>
      </motion.div>
    </AnimatePresence>
  );
}

function Stat({ label, value, className }: { label: string; value: string; className: string }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-widest text-slate-400">{label}</div>
      <div className={classNames('text-xl font-bold tabular-nums', className)}>{value}</div>
    </div>
  );
}

export function RequestStream({ history }: { history: MetricPoint[] }) {
  const tiles = history.slice(-60);
  const failed = tiles.reduce((sum, point) => sum + point.errors, 0);
  return (
    <div className="panel p-4">
      <div className="flex items-center justify-between">
        <span className="panel-title">Request stream (last 60 s)</span>
        <span className={classNames('text-xs tabular-nums', failed > 0 ? 'text-danger' : 'text-slate-500')}>
          {failed > 0 ? `✕ ${failed.toLocaleString('en-IN')} failed` : 'no failures'}
        </span>
      </div>
      <div className="mt-3 grid grid-cols-[repeat(30,minmax(0,1fr))] gap-1">
        {tiles.map((point) => {
          const critical = point.errorRate >= 10 || point.p95 >= 2000;
          const degraded = point.errorRate >= 2 || point.p95 >= 500;
          const idle = point.rps < 1;
          const color = idle ? '#1c2740' : critical ? '#ff4d6a' : degraded ? '#fbbf24' : '#22d3a5';
          return (
            <motion.div
              key={point.timestamp}
              initial={{ scale: 0, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="aspect-square rounded-[4px]"
              style={{ backgroundColor: color }}
              title={`${Math.round(point.rps)} req/s, p95 ${point.p95.toFixed(0)} ms, errors ${point.errorRate.toFixed(1)}%`}
            />
          );
        })}
      </div>
    </div>
  );
}
