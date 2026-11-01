import { useState } from 'react';
import { motion } from 'framer-motion';
import { ArchitectureDiagram, ReplicaTopology, ShardView } from '../components/Topology';
import { StatusPill } from '../components/primitives';
import { usePolling } from '../hooks/usePolling';
import { api, ApiError, query } from '../services/api';
import type { MetricsResponse, MongoStatus } from '../types';
import { formatNumber } from '../utils/format';

export default function ArchitecturePage() {
  const mongo = usePolling((signal) => api.get<MongoStatus>('/system/mongodb-status', signal), 2000);
  const metrics = usePolling((signal) => api.get<MetricsResponse>('/system/metrics?history=5', signal), 2000);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const status = mongo.data;
  const sharded = status?.topology.kind === 'sharded';

  const stepDown = async (shard?: string) => {
    setBusy(true);
    setNotice(null);
    try {
      await api.postEmpty(`/system/replica-set/step-down${query({ shard })}`);
      setNotice('Primary stepped down. Watch a secondary take over within a few seconds.');
    } catch (caught) {
      setNotice(caught instanceof ApiError ? caught.message : 'Could not trigger the failover');
    } finally {
      setTimeout(() => setBusy(false), 4000);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-extrabold text-slate-100">MongoDB architecture</h1>
        <p className="mt-1 text-sm text-slate-500">Live topology, replica-set health and shard distribution.</p>
      </div>

      <ArchitectureDiagram tuning={metrics.data?.tuning} topology={status?.topology} rps={metrics.data?.snapshot.current.rps ?? 0} />

      {notice ? (
        <motion.div initial={{ opacity: 0, y: -6 }} animate={{ opacity: 1, y: 0 }} className="rounded-xl border border-info/40 bg-info/10 px-4 py-3 text-sm text-info">
          {notice}
        </motion.div>
      ) : null}

      {status?.replicaSet ? (
        <ReplicaTopology name={status.replicaSet.name} members={status.replicaSet.members} onStepDown={() => stepDown()} busy={busy} />
      ) : null}

      {sharded ? (
        <div className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-2">
            {status?.shards.shards.map((shard) =>
              shard.members && shard.members.length > 0 ? (
                <div key={shard.name} className="space-y-2">
                  <ReplicaTopology name={shard.name} members={shard.members} onStepDown={() => stepDown(shard.name)} busy={busy} />
                </div>
              ) : null,
            )}
          </div>
        </div>
      ) : null}

      {status ? <ShardView distribution={status.shards} /> : null}

      {status && !status.replicaSet && !sharded ? (
        <div className="panel p-6 text-sm text-slate-400">
          This deployment is a <StatusPill tone="warn">standalone</StatusPill> MongoDB node. Start{' '}
          <span className="font-mono text-slate-200">docker-compose.replica.yml</span> for the replica-set view or{' '}
          <span className="font-mono text-slate-200">docker-compose.sharded.yml</span> for the sharded cluster.
        </div>
      ) : null}

      {status ? (
        <div className="panel p-5">
          <span className="panel-title">Collections</span>
          <div className="mt-4 grid grid-cols-2 gap-4 md:grid-cols-4">
            {status.collections.map((collection) => (
              <div key={collection.name} className="rounded-xl border border-line bg-ink-900 p-4">
                <div className="font-mono text-xs text-slate-500">{collection.name}</div>
                <div className="mt-1 text-2xl font-bold tabular-nums text-slate-100">{formatNumber(collection.count)}</div>
              </div>
            ))}
          </div>
          <div className="mt-4 text-xs text-slate-500">
            MongoDB {status.version} · uptime {Math.round(status.uptimeSeconds / 60)} min · {formatNumber(status.connections.current)} open connections
          </div>
        </div>
      ) : null}
    </div>
  );
}
