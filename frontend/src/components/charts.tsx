import type { ReactNode } from 'react';
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
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

const axisStyle = { fill: '#64748b', fontSize: 11 };
const tooltipStyle = {
  background: '#0d1320',
  border: '1px solid #1c2740',
  borderRadius: 12,
  fontSize: 12,
};

function clock(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('en-GB');
}

interface ChartProps {
  data: MetricPoint[];
  stressed: boolean;
}

function Frame({ title, children, stressed }: { title: string; children: ReactNode; stressed: boolean }) {
  return (
    <div className={`panel p-4 transition-colors ${stressed ? 'border-danger/50' : ''}`}>
      <div className="panel-title mb-3">{title}</div>
      <div className="h-56">{children}</div>
    </div>
  );
}

export function LatencyChart({ data, stressed }: ChartProps) {
  return (
    <Frame title="Latency (ms)" stressed={stressed}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#1c2740" strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Line type="monotone" dataKey="p50" name="p50" stroke="#38bdf8" strokeWidth={2} dot={false} isAnimationActive={false} />
          <Line type="monotone" dataKey="p95" name="p95" stroke={stressed ? '#ff4d6a' : '#22d3a5'} strokeWidth={2.5} dot={false} isAnimationActive={false} />
          <Line type="monotone" dataKey="p99" name="p99" stroke="#a78bfa" strokeWidth={2} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function ThroughputChart({ data, stressed }: ChartProps) {
  return (
    <Frame title="Throughput (requests / second)" stressed={stressed}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <defs>
            <linearGradient id="rpsFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={stressed ? '#ff4d6a' : '#22d3a5'} stopOpacity={0.55} />
              <stop offset="100%" stopColor={stressed ? '#ff4d6a' : '#22d3a5'} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="#1c2740" strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Area type="monotone" dataKey="rps" name="req/s" stroke={stressed ? '#ff4d6a' : '#22d3a5'} strokeWidth={2.5} fill="url(#rpsFill)" isAnimationActive={false} />
        </AreaChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function UsersErrorsChart({ data, stressed }: ChartProps) {
  return (
    <Frame title="Virtual users and failed requests" stressed={stressed}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#1c2740" strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis yAxisId="users" tick={axisStyle} />
          <YAxis yAxisId="errors" orientation="right" tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Line yAxisId="users" type="monotone" dataKey="activeUsers" name="virtual users" stroke="#38bdf8" strokeWidth={2.5} dot={false} isAnimationActive={false} />
          <Line yAxisId="errors" type="monotone" dataKey="errors" name="failed / s" stroke="#ff4d6a" strokeWidth={2} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </Frame>
  );
}

export function MongoOpsChart({ data, stressed }: ChartProps) {
  return (
    <Frame title="MongoDB read / write throughput (ops / second)" stressed={stressed}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data.slice(-40)} margin={{ left: -12, right: 8, top: 4 }}>
          <CartesianGrid stroke="#1c2740" strokeDasharray="3 3" />
          <XAxis dataKey="timestamp" tickFormatter={clock} tick={axisStyle} minTickGap={48} />
          <YAxis tick={axisStyle} />
          <Tooltip contentStyle={tooltipStyle} labelFormatter={(value) => clock(Number(value))} />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Bar dataKey="mongoReads" name="reads" stackId="ops" fill="#38bdf8" isAnimationActive={false} />
          <Bar dataKey="mongoWrites" name="writes" stackId="ops" fill="#a78bfa" isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </Frame>
  );
}
