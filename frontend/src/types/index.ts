export type Role = 'customer' | 'admin';

export interface User {
  id: string;
  name: string;
  email: string;
  role: Role;
  createdAt: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export type OptimizationMode = 'unoptimized' | 'indexed' | 'replica-set' | 'sharded' | 'fully-optimized';

export interface Tuning {
  mode: OptimizationMode;
  label: string;
  description: string;
  indexes: boolean;
  secondaryReads: boolean;
  majorityWrites: boolean;
  sharding: boolean;
  projections: boolean;
  keysetPagination: boolean;
  cache: boolean;
  rateLimit: boolean;
  maxPoolSize: number;
  minPoolSize: number;
}

export interface Topology {
  kind: 'standalone' | 'replica-set' | 'sharded';
  setName?: string;
}

export interface MetricPoint {
  timestamp: number;
  rps: number;
  p50: number;
  p95: number;
  p99: number;
  errorRate: number;
  errors: number;
  throttled: number;
  activeUsers: number;
  inflight: number;
  mongoReads: number;
  mongoWrites: number;
  slowQueries: number;
  poolInUse: number;
  poolWaiting: number;
  cpuPercent: number;
  memoryMB: number;
  goroutines: number;
}

export type SystemStatus = 'idle' | 'healthy' | 'degraded' | 'critical';

export interface PoolUsage {
  open: number;
  inUse: number;
  waiting: number;
  max: number;
  failed: number;
}

export interface MetricsResponse {
  snapshot: {
    status: SystemStatus;
    current: MetricPoint;
    totals: { requests: number; errors: number; throttled: number };
    history: MetricPoint[];
  };
  mode: OptimizationMode;
  tuning: Tuning;
  topology: Topology;
  pool: PoolUsage;
  cache: { enabled: boolean; hits: number; misses: number };
  slowQueries: number;
  mongoFailures: number;
  uptimeSeconds: number;
}

export interface MemberStatus {
  name: string;
  state: string;
  healthy: boolean;
  uptimeSeconds: number;
  lagSeconds: number;
}

export interface ShardStatus {
  name: string;
  host: string;
  documents: number;
  sizeBytes: number;
  members: MemberStatus[] | null;
}

export interface QueryPlan {
  name: string;
  collection: string;
  filter: string;
  stage: 'COLLSCAN' | 'IXSCAN';
  indexName?: string;
  docsExamined: number;
  keysExamined: number;
  returned: number;
  executionMillis: number;
}

export interface MongoStatus {
  topology: Topology;
  version: string;
  uptimeSeconds: number;
  connections: { current: number; available: number; active: number };
  opCounters: Record<string, number>;
  residentMB: number;
  virtualMB: number;
  pool: PoolUsage;
  slowQueries: number;
  replicaSet?: { name: string; primaryName: string; members: MemberStatus[] };
  shards: { collection: string; sharded: boolean; shardKey?: string; shards: ShardStatus[] };
  indexes: { collection: string; name: string; accesses: number }[];
  queryPlans: QueryPlan[];
  collections: { name: string; count: number }[];
  collectedAt: string;
}

export interface LoadTestStatus {
  running: boolean;
  phase: string;
  mode: OptimizationMode;
  targetUsers: number;
  activeUsers: number;
  peakUsers: number;
  requests: number;
  failures: number;
  throttled: number;
  elapsedSeconds: number;
  totalSeconds: number;
  progressPercent: number;
  message?: string;
}

export interface LoadTestRequest {
  users: number;
  rampUpSeconds: number;
  holdSeconds: number;
  rampDownSeconds: number;
  thinkTimeMinMs?: number;
  thinkTimeMaxMs?: number;
}

export interface LoadTestResult {
  id: string;
  mode: OptimizationMode;
  topology: string;
  targetUsers: number;
  peakUsers: number;
  durationSeconds: number;
  totalRequests: number;
  failedRequests: number;
  throttled: number;
  errorRate: number;
  rps: number;
  p50: number;
  p95: number;
  p99: number;
  finishedAt: string;
}

export interface OptimizationCatalog {
  current: OptimizationMode;
  tuning: Tuning;
  modes: Tuning[];
}

export interface ModeChange {
  mode: OptimizationMode;
  tuning: Tuning;
  topology: Topology;
  warnings: string[] | null;
  durationMs: number;
}
