import { useState } from 'react';
import { motion } from 'framer-motion';
import { api, ApiError } from '../services/api';
import type { ModeChange, OptimizationCatalog, OptimizationMode } from '../types';
import { classNames } from '../utils/format';

const order: OptimizationMode[] = ['unoptimized', 'indexed', 'replica-set', 'sharded', 'fully-optimized'];

interface Props {
  catalog: OptimizationCatalog | null;
  onChanged: () => void;
}

export function ModeSwitcher({ catalog, onChanged }: Props) {
  const [pending, setPending] = useState<OptimizationMode | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  const apply = async (mode: OptimizationMode) => {
    setPending(mode);
    setError(null);
    setWarnings([]);
    try {
      const change = await api.postEmpty<ModeChange>(`/system/optimizations/${mode}/toggle`);
      setWarnings(change.warnings ?? []);
      onChanged();
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not switch mode');
    } finally {
      setPending(null);
    }
  };

  const current = catalog?.current;
  const modes = order.map((mode) => catalog?.modes.find((entry) => entry.mode === mode)).filter((entry) => entry !== undefined);

  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <span className="panel-title">Optimization mode</span>
        <span className="text-xs text-slate-500">switch live, even mid-test</span>
      </div>
      <div className="mt-4 space-y-2">
        {modes.map((tuning, index) => {
          const active = tuning.mode === current;
          const loading = pending === tuning.mode;
          return (
            <motion.button
              key={tuning.mode}
              type="button"
              whileTap={{ scale: 0.99 }}
              disabled={pending !== null}
              onClick={() => apply(tuning.mode)}
              className={classNames(
                'flex w-full items-start gap-3 rounded-xl border px-3 py-2.5 text-left transition disabled:cursor-wait',
                active
                  ? tuning.mode === 'unoptimized'
                    ? 'border-danger/60 bg-danger/10'
                    : 'border-brand/60 bg-brand/10'
                  : 'border-line bg-ink-900 hover:border-slate-500',
              )}
            >
              <span
                className={classNames(
                  'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold',
                  active ? (tuning.mode === 'unoptimized' ? 'bg-danger text-white' : 'bg-brand text-ink-950') : 'bg-ink-700 text-slate-400',
                )}
              >
                {loading ? '…' : index + 1}
              </span>
              <span className="min-w-0">
                <span className="block text-sm font-semibold text-slate-100">{tuning.label}</span>
                <span className="block text-xs leading-snug text-slate-500">{tuning.description}</span>
              </span>
            </motion.button>
          );
        })}
      </div>
      {warnings.length > 0 ? (
        <div className="mt-3 space-y-1 rounded-xl border border-warn/40 bg-warn/10 px-3 py-2 text-xs text-warn">
          {warnings.map((warning) => (
            <div key={warning}>{warning}</div>
          ))}
        </div>
      ) : null}
      {error ? <div className="mt-3 rounded-xl border border-danger/50 bg-danger/10 px-3 py-2 text-xs text-danger">{error}</div> : null}
    </div>
  );
}
