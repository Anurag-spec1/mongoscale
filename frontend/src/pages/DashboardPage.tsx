import { useCallback, useState } from 'react';
import { HealthGauges, IndexUsageList, QueryPlanTable } from '../components/DatabasePanels';
import { LatencyChart, MongoOpsChart, ThroughputChart, UsersErrorsChart } from '../components/charts';
import { LoadTestPanel } from '../components/LoadTestPanel';
import { ModeSwitcher } from '../components/ModeSwitcher';
import { MetricCard, StatusPill, type Tone } from '../components/primitives';
import { RequestStream, StatusBanner } from '../components/StatusBanner';
import { usePolling } from '../hooks/usePolling';
import { api } from '../services/api';
import type { LoadTestStatus, MetricsResponse, MongoStatus, OptimizationCatalog } from '../types';
import { formatCompact, formatMillis, formatNumber } from '../utils/format';

function latencyTone(p95: number): Tone {
  if (p95 >= 2000) return 'bad';
  if (p95 >= 500) return 'warn';
  return 'good';
}

function errorTone(rate: number): Tone {
  if (rate >= 10) return 'bad';
  if (rate >= 2) return 'warn';
  return 'good';
}

export default function DashboardPage() {
  const [refreshToken, setRefreshToken] = useState(0);
  const metrics = usePolling((signal) => api.get<MetricsResponse>('/system/metrics?history=120', signal), 1000);
  const loadTest = usePolling((signal) => api.get<LoadTestStatus>('/system/load-test/status', signal), 1000);
  const mongo = usePolling((signal) => api.get<MongoStatus>('/system/mongodb-status', signal), 3000);
  const catalog = usePolling(
    (signal) => api.get<OptimizationCatalog>(`/system/optimizations?r=${refreshToken}`, signal),
    4000,
  );

  const handleModeChanged = useCallback(() => setRefreshToken((value) => value + 1), []);

  const snapshot = metrics.data?.snapshot;
  const current = snapshot?.current;
  const history = snapshot?.history ?? [];
  const status = snapshot?.status ?? 'idle';
  const stressed = status === 'critical';
  const tuning = metrics.data?.tuning;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-extrabold text-slate-100">Live performance dashboard</h1>
          <p className="mt-1 text-sm text-slate-500">Server-side metrics, refreshed every second. They include traffic from the built-in test and from k6.</p>
        </div>
        <div className="flex items-center gap-2">
          {tuning ? <StatusPill tone={tuning.mode === 'unoptimized' ? 'bad' : 'good'}>{tuning.label}</StatusPill> : null}
          {metrics.data ? <StatusPill tone="info">{metrics.data.topology.kind}</StatusPill> : null}
          {metrics.error ? <StatusPill tone="bad">API unreachable</StatusPill> : null}
        </div>
      </div>

      <StatusBanner status={status} current={current} />

      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-6">
        <MetricCard label="Virtual users" value={current?.activeUsers ?? 0} format={formatNumber} tone="info" hint={`${formatNumber(current?.inflight ?? 0)} in flight`} />
        <MetricCard label="Requests / s" value={current?.rps ?? 0} format={formatNumber} tone={stressed ? 'bad' : 'good'} hint={`${formatCompact(snapshot?.totals.requests ?? 0)} total`} />
        <MetricCard label="p95 latency" value={current?.p95 ?? 0} format={formatMillis} tone={latencyTone(current?.p95 ?? 0)} hint={`p99 ${formatMillis(current?.p99 ?? 0)}`} />
        <MetricCard label="Error rate" value={current?.errorRate ?? 0} format={(value) => `${value.toFixed(1)}%`} tone={errorTone(current?.errorRate ?? 0)} hint="5xx and timeouts" />
        <MetricCard label="Failed requests" value={snapshot?.totals.errors ?? 0} format={formatNumber} tone={(snapshot?.totals.errors ?? 0) > 0 ? 'bad' : 'neutral'} hint={`${formatNumber(snapshot?.totals.throttled ?? 0)} throttled`} />
        <MetricCard
          label="Mongo ops / s"
          value={(current?.mongoReads ?? 0) + (current?.mongoWrites ?? 0)}
          format={formatNumber}
          tone="neutral"
          hint={`${formatNumber(current?.mongoReads ?? 0)} reads · ${formatNumber(current?.mongoWrites ?? 0)} writes`}
        />
      </div>

      <RequestStream history={history} />

      <div className="grid gap-4 lg:grid-cols-2">
        <LatencyChart data={history} />
        <ThroughputChart data={history} />
        <UsersErrorsChart data={history} />
        <MongoOpsChart data={history} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[1fr_1fr_1fr]">
        <LoadTestPanel status={loadTest.data} />
        <ModeSwitcher catalog={catalog.data} onChanged={handleModeChanged} />
        <HealthGauges metrics={metrics.data} mongo={mongo.data} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[2fr_1fr]">
        <QueryPlanTable plans={mongo.data?.queryPlans ?? []} />
        <IndexUsageList indexes={mongo.data?.indexes ?? []} />
      </div>
    </div>
  );
}
