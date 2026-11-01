import type { ReactNode } from 'react';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { MetricPoint } from '../types';

const axisStyle = { fill: '#9aa5b1', fontSize: 11 };
const tooltipStyle = {
  background: '#171c22',
  border: '1px solid #2a313a',
  borderRadius: 4,
  fontSize: 12,
};
const series = { primary: '#73a7ff', secondary: '#d3dae2', tertiary: '#8d99a7', error: '#ef7379' };

function clock(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('en-GB');
}

interface ChartProps { data: MetricPoint[] }

function Frame({ title, data, children }: { title: string; data: MetricPoint[]; children: ReactNode }) {
  return (
    <div className="panel p-4">
      <div className="panel-title mb-3">{title}</div>
      {data.length > 0 ? <div className="h-56">{children}</div> : <div className="flex h-56 items-center justify-center text-sm text-slate-500">Waiting for live metrics.</div>}
    </div>
  );
}

export function LatencyChart({ data }: ChartProps) {
  return (
    <Frame title="Latency (ms)" data={data}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#283039" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 11, paddingTop: 4 }} />
          <Line type="monotone" dataKey="p50" name="p50" stroke={series.primary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
          <Line type="monotone" dataKey="p95" name="p95" stroke={series.secondary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
          <Line type="monotone" dataKey="p99" name="p99" stroke={series.tertiary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function ThroughputChart({ data }: ChartProps) {
  return (
    <Frame title="Throughput (requests / second)" data={data}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#283039" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Line type="monotone" dataKey="rps" name="req/s" stroke={series.primary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function UsersErrorsChart({ data }: ChartProps) {
  return (
    <Frame title="Virtual users and failed requests" data={data}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#283039" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis yAxisId="users" tick={axisStyle} />
          <YAxis yAxisId="errors" orientation="right" tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 11, paddingTop: 4 }} />
          <Line yAxisId="users" type="monotone" dataKey="activeUsers" name="virtual users" stroke={series.primary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
          <Line yAxisId="errors" type="monotone" dataKey="errors" name="failed / s" stroke={series.error} strokeWidth={1.5} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function MongoOpsChart({ data }: ChartProps) {
  return (
    <Frame title="MongoDB read / write throughput (ops / second)" data={data}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#283039" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 11, paddingTop: 4 }} />
          <Line type="monotone" dataKey="mongoReads" name="reads" stroke={series.primary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
          <Line type="monotone" dataKey="mongoWrites" name="writes" stroke={series.secondary} strokeWidth={1.5} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}
