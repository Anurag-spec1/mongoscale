import { useCallback, useState } from 'react';
import { HealthGauges, IndexUsageList, QueryPlanTable } from '../components/DatabasePanels';
import { LatencyChart, MongoOpsChart, ThroughputChart, UsersErrorsChart } from '../components/charts';
import { LoadTestPanel } from '../components/LoadTestPanel';
import { ModeSwitcher } from '../components/ModeSwitcher';
import { MetricCard, type Tone } from '../components/primitives';
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

  return (
    <div className="space-y-6">
      <div className="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1fr)_310px]">
        <div className="min-w-0 space-y-5">
          <p className="text-sm text-slate-400">Server metrics refresh every second and include built-in test and k6 traffic.</p>

          {metrics.data ? <StatusBanner status={status} current={current} /> : (
            <div role={metrics.error ? 'alert' : 'status'} className="panel px-4 py-4 text-sm text-slate-300">
              {metrics.error ? 'Live metrics are unavailable.' : 'Loading live metrics…'}
            </div>
          )}

          <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
            <MetricCard label="Virtual users" value={current?.activeUsers ?? null} format={formatNumber} tone="info" hint={current ? `${formatNumber(current.inflight)} in flight` : 'Waiting for data'} />
            <MetricCard label="Requests / s" value={current?.rps ?? null} format={formatNumber} tone={stressed ? 'bad' : 'good'} hint={snapshot ? `${formatCompact(snapshot.totals.requests)} total` : 'Waiting for data'} />
            <MetricCard label="p95 latency" value={current?.p95 ?? null} format={formatMillis} tone={latencyTone(current?.p95 ?? 0)} hint={current ? `p99 ${formatMillis(current.p99)}` : 'Waiting for data'} />
            <MetricCard label="Error rate" value={current?.errorRate ?? null} format={(value) => `${value.toFixed(1)}%`} tone={errorTone(current?.errorRate ?? 0)} hint="5xx and timeouts" />
            <MetricCard label="Failed requests" value={snapshot?.totals.errors ?? null} format={formatNumber} tone={(snapshot?.totals.errors ?? 0) > 0 ? 'bad' : 'neutral'} hint={snapshot ? `${formatNumber(snapshot.totals.throttled)} throttled` : 'Waiting for data'} />
            <MetricCard label="Mongo ops / s" value={current ? current.mongoReads + current.mongoWrites : null} format={formatNumber} tone="neutral" hint={current ? `${formatNumber(current.mongoReads)} reads | ${formatNumber(current.mongoWrites)} writes` : 'Waiting for data'} />
          </div>

          <RequestStream history={history} error={metrics.error} />

          <div className="grid gap-4 2xl:grid-cols-2">
            <LatencyChart data={history} error={metrics.error} />
            <ThroughputChart data={history} error={metrics.error} />
            <UsersErrorsChart data={history} error={metrics.error} />
            <MongoOpsChart data={history} error={metrics.error} />
          </div>

          <div className="grid min-w-0 gap-4 2xl:grid-cols-[minmax(0,2fr)_minmax(260px,1fr)]">
            <QueryPlanTable plans={mongo.data?.queryPlans ?? []} loading={mongo.loading} error={mongo.error} />
            <IndexUsageList indexes={mongo.data?.indexes ?? []} loading={mongo.loading} error={mongo.error} />
          </div>
        </div>

        <aside className="min-w-0 space-y-4">
          <LoadTestPanel status={loadTest.data} loading={loadTest.loading} statusError={loadTest.error} />
          <ModeSwitcher catalog={catalog.data} loading={catalog.loading} loadError={catalog.error} onChanged={handleModeChanged} />
          <HealthGauges metrics={metrics.data} mongo={mongo.data} loading={metrics.loading || mongo.loading} error={metrics.error ?? mongo.error} />
        </aside>
      </div>
    </div>
  );
}
