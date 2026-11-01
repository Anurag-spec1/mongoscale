import { useEffect, useRef, useState } from 'react';

interface PollingState<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
}

export function usePolling<T>(fetcher: (signal: AbortSignal) => Promise<T>, intervalMs: number, enabled = true): PollingState<T> {
  const [state, setState] = useState<PollingState<T>>({ data: null, error: null, loading: true });
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  useEffect(() => {
    if (!enabled) {
      return undefined;
    }
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    let cancelled = false;

    const tick = async () => {
      try {
        const data = await fetcherRef.current(controller.signal);
        if (!cancelled) {
          setState({ data, error: null, loading: false });
        }
      } catch (error) {
        if (!cancelled && !(error instanceof DOMException && error.name === 'AbortError')) {
          setState((previous) => ({
            data: previous.data,
            error: error instanceof Error ? error.message : 'Request failed',
            loading: false,
          }));
        }
      }
      if (!cancelled) {
        timer = setTimeout(tick, intervalMs);
      }
    };
    void tick();

    return () => {
      cancelled = true;
      controller.abort();
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [intervalMs, enabled]);

  return state;
}
