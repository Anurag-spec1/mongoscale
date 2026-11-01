import { useState } from 'react';
import { api, ApiError } from '../services/api';
import type { LoadTestRequest, LoadTestStatus } from '../types';
import { classNames, formatDuration, formatNumber } from '../utils/format';
import { ProgressMeter, StatusPill } from './primitives';

interface Preset {
  label: string;
  caption: string;
  request: LoadTestRequest;
}

const think = { thinkTimeMinMs: 6000, thinkTimeMaxMs: 14000 };

const presets: Preset[] = [
  { label: '1,000 users', caption: 'ramp 30s, hold 60s', request: { users: 1000, rampUpSeconds: 30, holdSeconds: 60, rampDownSeconds: 15, ...think } },
  { label: '5,000 users', caption: 'ramp 60s, hold 60s', request: { users: 5000, rampUpSeconds: 60, holdSeconds: 60, rampDownSeconds: 15, ...think } },
  { label: '10,000 users', caption: 'ramp 90s, hold 90s', request: { users: 10000, rampUpSeconds: 90, holdSeconds: 90, rampDownSeconds: 15, ...think } },
];

interface Props {
  status: LoadTestStatus | null;
  loading: boolean;
  statusError: string | null;
}

export function LoadTestPanel({ status, loading, statusError }: Props) {
  const [selected, setSelected] = useState(0);
  const [custom, setCustom] = useState<LoadTestRequest>({ users: 2000, rampUpSeconds: 30, holdSeconds: 60, rampDownSeconds: 15, ...think });
  const [useCustom, setUseCustom] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const running = status?.running ?? false;
  const request = useCustom ? custom : presets[selected].request;

  const start = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.post<LoadTestStatus>('/system/load-test/start', request);
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not start the load test');
    } finally {
      setBusy(false);
    }
  };

  const stop = async () => {
    setBusy(true);
    try {
      await api.postEmpty<LoadTestStatus>('/system/load-test/stop');
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not stop the load test');
    } finally {
      setBusy(false);
    }
  };

  const failureRate = status && status.requests > 0 ? (status.failures / status.requests) * 100 : 0;

  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <span className="panel-title">Live load test</span>
        <StatusPill tone={running ? 'info' : 'neutral'}>
          {status?.phase ?? (loading ? 'Loading' : 'Unavailable')}
        </StatusPill>
      </div>

      <div role="group" aria-label="Load test preset" className="mt-4 grid grid-cols-3 border border-line">
        {presets.map((preset, index) => (
          <button
            key={preset.label}
            type="button"
            disabled={running}
            aria-pressed={!useCustom && selected === index}
            onClick={() => {
              setSelected(index);
              setUseCustom(false);
            }}
            className={classNames(
              'min-w-0 border-r border-line px-2 py-2.5 text-left transition-colors last:border-r-0 disabled:opacity-50 sm:px-3',
              !useCustom && selected === index ? 'bg-accent-muted text-slate-100' : 'bg-ink-900 text-slate-300 hover:bg-ink-800',
            )}
          >
            <div className="text-xs font-semibold sm:text-sm">{preset.label}</div>
            <div className="mt-1 text-[10px] leading-tight text-slate-500">{preset.caption}</div>
          </button>
        ))}
      </div>

      <button type="button" disabled={running} aria-expanded={useCustom} aria-controls="advanced-load-options" onClick={() => setUseCustom((value) => !value)} className="mt-4 text-sm text-brand hover:underline disabled:opacity-50">
        {useCustom ? 'Hide advanced options' : 'Advanced options'}
      </button>

      {useCustom ? (
        <div id="advanced-load-options" className="mt-3 grid grid-cols-2 gap-3 border-t border-line pt-4">
          <NumberField label="Virtual users" value={custom.users} max={20000} disabled={running} onChange={(users) => setCustom({ ...custom, users })} />
          <NumberField label="Ramp up (s)" value={custom.rampUpSeconds} max={600} disabled={running} onChange={(rampUpSeconds) => setCustom({ ...custom, rampUpSeconds })} />
          <NumberField label="Hold (s)" value={custom.holdSeconds} max={600} disabled={running} onChange={(holdSeconds) => setCustom({ ...custom, holdSeconds })} />
          <NumberField label="Ramp down (s)" value={custom.rampDownSeconds} max={600} disabled={running} onChange={(rampDownSeconds) => setCustom({ ...custom, rampDownSeconds })} />
          <NumberField
            label="Think time min (s)"
            value={(custom.thinkTimeMinMs ?? 6000) / 1000}
            max={60}
            disabled={running}
            onChange={(seconds) => setCustom({ ...custom, thinkTimeMinMs: seconds * 1000, thinkTimeMaxMs: Math.max(custom.thinkTimeMaxMs ?? 14000, seconds * 1000) })}
          />
          <NumberField
            label="Think time max (s)"
            value={(custom.thinkTimeMaxMs ?? 14000) / 1000}
            max={60}
            disabled={running}
            onChange={(seconds) => setCustom({ ...custom, thinkTimeMaxMs: Math.max(seconds * 1000, custom.thinkTimeMinMs ?? 6000) })}
          />
        </div>
      ) : null}

      {status ? <div className="mt-5 space-y-4">
        <ProgressMeter
          value={status.progressPercent}
          max={100}
          label="Test progress"
          valueLabel={`${Math.round(status.progressPercent)}% | ${formatDuration(status.elapsedSeconds)} / ${formatDuration(status.totalSeconds)}`}
          tone={running ? 'info' : 'neutral'}
        />
        <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <Readout label="Virtual users" value={formatNumber(status.activeUsers)} />
          <Readout label="Peak users" value={formatNumber(status.peakUsers)} />
          <Readout label="Requests" value={formatNumber(status.requests)} />
          <Readout label="Failed" value={formatNumber(status.failures)} danger={status.failures > 0} />
          <Readout label="Throttled" value={formatNumber(status.throttled)} />
          <Readout label="Failure rate" value={`${failureRate.toFixed(1)}%`} danger={failureRate >= 5} />
        </div>
      </div> : <div role={statusError ? 'alert' : 'status'} className="mt-5 border-t border-line pt-4 text-sm text-slate-400">{statusError ? 'Load test status is unavailable.' : loading ? 'Loading load test status…' : 'No load test status reported.'}</div>}

      {statusError ? <div role="alert" className="mt-3 text-xs text-danger">{statusError}</div> : null}

      {error ? (
        <div role="alert" className="mt-4 border-l-2 border-danger bg-status-danger/10 px-3 py-2 text-sm text-danger">
          {error}
        </div>
      ) : null}

      <div className="mt-5 flex gap-3">
        {running ? (
          <button type="button" className="btn-danger flex-1" disabled={busy} onClick={stop}>
            Stop test
          </button>
        ) : (
          <button type="button" className="btn-primary flex-1" disabled={busy} onClick={start}>
            {busy ? 'Starting…' : `Start ${formatNumber(request.users)} user test`}
          </button>
        )}
      </div>
    </div>
  );
}

function NumberField({ label, value, max, onChange, disabled }: { label: string; value: number; max: number; onChange: (value: number) => void; disabled: boolean }) {
  return (
    <label className="block">
      <span className="label">{label}</span>
      <input
        className="input"
        type="number"
        min={1}
        max={max}
        disabled={disabled}
        value={value}
        onChange={(event) => onChange(Math.max(1, Math.min(max, Number(event.target.value) || 1)))}
      />
    </label>
  );
}

function Readout({ label, value, danger = false }: { label: string; value: string; danger?: boolean }) {
  return (
    <div>
      <div className="text-xs text-slate-400">{label}</div>
      <div className={classNames('font-semibold tabular-nums', danger ? 'text-danger' : 'text-slate-100')}>{value}</div>
    </div>
  );
}
