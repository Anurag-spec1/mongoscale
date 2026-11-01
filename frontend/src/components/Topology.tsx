import { motion } from 'framer-motion';
import type { MemberStatus, MongoStatus, Tuning, Topology as TopologyInfo } from '../types';
import { classNames, formatBytes, formatNumber } from '../utils/format';
import { StatusPill, type Tone } from './primitives';

function stateTone(member: MemberStatus): Tone {
  if (!member.healthy) return 'bad';
  if (member.state === 'PRIMARY') return 'good';
  if (member.state === 'SECONDARY') return 'info';
  return 'warn';
}

function stateColor(tone: Tone): string {
  if (tone === 'good') return '#22d3a5';
  if (tone === 'info') return '#38bdf8';
  if (tone === 'warn') return '#fbbf24';
  return '#ff4d6a';
}

interface ReplicaProps {
  name: string;
  members: MemberStatus[];
  onStepDown?: () => void;
  busy?: boolean;
}

export function ReplicaTopology({ name, members, onStepDown, busy = false }: ReplicaProps) {
  const primary = members.find((member) => member.state === 'PRIMARY');
  const secondaries = members.filter((member) => member !== primary);
  const width = 480;
  const height = 230;
  const primaryPoint = { x: width / 2, y: 52 };
  const secondaryPoints = secondaries.map((_, index) => ({
    x: ((index + 1) * width) / (secondaries.length + 1),
    y: 176,
  }));

  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <div>
          <span className="panel-title">Replica set</span>
          <span className="ml-2 font-mono text-xs text-slate-500">{name}</span>
        </div>
        {onStepDown ? (
          <button type="button" className="btn-ghost !px-3 !py-1.5 text-xs" disabled={busy} onClick={onStepDown}>
            {busy ? 'Electing…' : 'Trigger failover'}
          </button>
        ) : null}
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="mt-3 w-full">
        {secondaries.map((member, index) => (
          <motion.line
            key={`line-${member.name}`}
            x1={primaryPoint.x}
            y1={primaryPoint.y}
            x2={secondaryPoints[index].x}
            y2={secondaryPoints[index].y}
            stroke={member.healthy ? '#38bdf8' : '#ff4d6a'}
            strokeWidth={2}
            strokeDasharray="6 6"
            animate={{ strokeDashoffset: member.healthy ? [0, -24] : 0 }}
            transition={{ duration: 1.2, repeat: Infinity, ease: 'linear' }}
          />
        ))}
        {primary ? <Node member={primary} x={primaryPoint.x} y={primaryPoint.y} big /> : null}
        {secondaries.map((member, index) => (
          <Node key={member.name} member={member} x={secondaryPoints[index].x} y={secondaryPoints[index].y} />
        ))}
        {!primary ? (
          <text x={width / 2} y={52} textAnchor="middle" fill="#ff4d6a" fontSize={14} fontWeight={700}>
            ELECTION IN PROGRESS
          </text>
        ) : null}
      </svg>
      <div className="mt-2 grid gap-2 sm:grid-cols-3">
        {members.map((member) => (
          <div key={member.name} className="rounded-xl border border-line bg-ink-900 px-3 py-2">
            <div className="flex items-center justify-between">
              <span className="font-mono text-xs text-slate-300">{member.name}</span>
              <StatusPill tone={stateTone(member)}>{member.state}</StatusPill>
            </div>
            <div className="mt-1 text-[11px] text-slate-500">
              up {Math.round(member.uptimeSeconds / 60)} min · lag {member.lagSeconds.toFixed(0)} s
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function Node({ member, x, y, big = false }: { member: MemberStatus; x: number; y: number; big?: boolean }) {
  const tone = stateTone(member);
  const color = stateColor(tone);
  const radius = big ? 30 : 24;
  return (
    <g>
      <motion.circle
        cx={x}
        cy={y}
        r={radius + 8}
        fill={color}
        opacity={0.12}
        animate={{ r: [radius + 4, radius + 12, radius + 4], opacity: [0.2, 0.05, 0.2] }}
        transition={{ duration: 1.8, repeat: Infinity }}
      />
      <circle cx={x} cy={y} r={radius} fill="#0d1320" stroke={color} strokeWidth={3} />
      <text x={x} y={y + 4} textAnchor="middle" fill={color} fontSize={big ? 12 : 10} fontWeight={800}>
        {member.state === 'PRIMARY' ? 'PRIMARY' : member.state === 'SECONDARY' ? 'SECOND.' : member.state.slice(0, 7)}
      </text>
      <text x={x} y={y + radius + 16} textAnchor="middle" fill="#94a3b8" fontSize={11} className="font-mono">
        {member.name.split(':')[0]}
      </text>
    </g>
  );
}

interface ShardProps {
  distribution: MongoStatus['shards'];
}

export function ShardView({ distribution }: ShardProps) {
  const total = distribution.shards.reduce((sum, shard) => sum + shard.documents, 0);
  const palette = ['#22d3a5', '#38bdf8', '#a78bfa', '#fbbf24'];
  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <span className="panel-title">Sharding distribution - orders</span>
        <StatusPill tone={distribution.sharded ? 'good' : 'warn'}>
          {distribution.sharded ? `sharded on { ${distribution.shardKey} }` : 'not sharded'}
        </StatusPill>
      </div>
      {distribution.shards.length === 0 ? (
        <div className="mt-6 rounded-xl border border-dashed border-line px-4 py-8 text-center text-sm text-slate-500">
          No sharded cluster detected. Start <span className="font-mono text-slate-300">docker-compose.sharded.yml</span> to see orders spread across shards.
        </div>
      ) : (
        <>
          <div className="mt-5 flex h-4 overflow-hidden rounded-full bg-ink-700">
            {distribution.shards.map((shard, index) => (
              <motion.div
                key={shard.name}
                className="h-full"
                style={{ backgroundColor: palette[index % palette.length] }}
                animate={{ width: total > 0 ? `${(shard.documents / total) * 100}%` : '0%' }}
                transition={{ duration: 0.8 }}
              />
            ))}
          </div>
          <div className="mt-5 grid gap-4 md:grid-cols-2">
            {distribution.shards.map((shard, index) => {
              const share = total > 0 ? (shard.documents / total) * 100 : 0;
              return (
                <motion.div
                  key={shard.name}
                  layout
                  className="rounded-2xl border bg-ink-900 p-4"
                  style={{ borderColor: `${palette[index % palette.length]}66` }}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-semibold" style={{ color: palette[index % palette.length] }}>
                      {shard.name}
                    </span>
                    <span className="text-xs text-slate-500">{share.toFixed(1)}% of orders</span>
                  </div>
                  <div className="mt-2 text-2xl font-bold tabular-nums text-slate-100">{formatNumber(shard.documents)}</div>
                  <div className="text-xs text-slate-500">documents · {formatBytes(shard.sizeBytes)}</div>
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {(shard.members ?? []).map((member) => (
                      <StatusPill key={member.name} tone={stateTone(member)} pulse={member.state === 'PRIMARY'}>
                        {member.name.split(':')[0]} · {member.state === 'PRIMARY' ? 'P' : 'S'}
                      </StatusPill>
                    ))}
                  </div>
                </motion.div>
              );
            })}
          </div>
        </>
      )}
      <div className="mt-5 rounded-xl border border-line bg-ink-900/70 p-4 text-sm leading-relaxed text-slate-400">
        <span className="font-semibold text-slate-200">Why a hashed userId shard key?</span> User ids are ObjectIds that grow monotonically, so a ranged key would send every new order to the
        newest chunk and create a write hotspot. Hashing the id scatters inserts evenly over all shards, while lookups by <span className="font-mono text-slate-300">userId</span> still target a
        single shard.
      </div>
    </div>
  );
}

interface DiagramProps {
  tuning: Tuning | undefined;
  topology: TopologyInfo | undefined;
  rps: number;
}

export function ArchitectureDiagram({ tuning, topology, rps }: DiagramProps) {
  const kind = topology?.kind ?? 'standalone';
  const flowing = rps > 1;
  return (
    <div className="panel p-5">
      <span className="panel-title">Request path</span>
      <div className="mt-5 flex flex-col items-stretch gap-3 lg:flex-row lg:items-center">
        <Box title="Clients" lines={['React app', 'k6 / built-in load test']} tone="neutral" />
        <Arrow active={flowing} />
        <Box
          title="Go API"
          lines={[`pool ≤ ${tuning?.maxPoolSize ?? '-'} connections`, tuning?.rateLimit ? 'rate limiting on' : 'rate limiting off', tuning?.projections ? 'projections + keyset paging' : 'full documents + skip paging']}
          tone={tuning?.mode === 'unoptimized' ? 'bad' : 'good'}
        />
        <Arrow active={flowing && Boolean(tuning?.cache)} />
        <Box title="Redis cache" lines={[tuning?.cache ? 'listings + review summaries' : 'bypassed']} tone={tuning?.cache ? 'good' : 'neutral'} muted={!tuning?.cache} />
        <Arrow active={flowing} />
        <Box
          title="MongoDB"
          lines={[
            kind === 'sharded' ? 'mongos → 2 shard replica sets' : kind === 'replica-set' ? '1 primary + 2 secondaries' : 'single standalone node',
            tuning?.indexes ? 'indexes on' : 'no secondary indexes',
            tuning?.secondaryReads ? 'secondaryPreferred reads' : 'primary reads',
          ]}
          tone={tuning?.indexes ? 'good' : 'bad'}
        />
      </div>
    </div>
  );
}

function Box({ title, lines, tone, muted = false }: { title: string; lines: string[]; tone: Tone; muted?: boolean }) {
  const border = tone === 'good' ? 'border-brand/50' : tone === 'bad' ? 'border-danger/60' : 'border-line';
  return (
    <div className={classNames('flex-1 rounded-2xl border bg-ink-900 p-4 transition-opacity', border, muted && 'opacity-50')}>
      <div className="text-sm font-bold text-slate-100">{title}</div>
      <ul className="mt-2 space-y-1 text-xs text-slate-400">
        {lines.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
    </div>
  );
}

function Arrow({ active }: { active: boolean }) {
  return (
    <div className="relative hidden h-1 w-14 shrink-0 overflow-hidden rounded-full bg-ink-700 lg:block">
      {active ? (
        <motion.span
          className="absolute inset-y-0 w-4 rounded-full bg-brand"
          animate={{ x: [-16, 56] }}
          transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
        />
      ) : null}
    </div>
  );
}
