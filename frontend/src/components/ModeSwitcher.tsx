import { useState } from 'react';
import { api, ApiError } from '../services/api';
import type { ModeChange, OptimizationCatalog, OptimizationMode } from '../types';

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
  const selectedMode = modes.find((entry) => entry.mode === current);

  return (
    <div className="panel p-5">
      <label className="mt-4 block">
        <span className="label">Optimization mode</span>
        <select
          className="input"
          value={current ?? ''}
          disabled={pending !== null || modes.length === 0}
          onChange={(event) => void apply(event.target.value as OptimizationMode)}
        >
          {!current ? <option value="">Loading modes…</option> : null}
          {modes.map((tuning) => <option key={tuning.mode} value={tuning.mode}>{tuning.label}</option>)}
        </select>
      </label>
      <p className="mt-3 min-h-10 text-xs leading-relaxed text-slate-400">
        {pending ? `Switching to ${modes.find((entry) => entry.mode === pending)?.label ?? 'selected mode'}…` : selectedMode?.description ?? 'Optimization modes are unavailable.'}
      </p>
      {warnings.length > 0 ? (
        <div role="status" className="mt-3 space-y-1 border-l-2 border-warn bg-status-warn/10 px-3 py-2 text-xs text-warn">
          {warnings.map((warning) => (
            <div key={warning}>{warning}</div>
          ))}
        </div>
      ) : null}
      {error ? <div role="alert" className="mt-3 border-l-2 border-danger bg-status-danger/10 px-3 py-2 text-xs text-danger">{error}</div> : null}
    </div>
  );
}
