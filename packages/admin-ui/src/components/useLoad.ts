import { useEffect, useState } from "react";
export type LoadState<T> = { data: T | null; loading: boolean; error: string | null };

/** Runs `load` when `deps` change (skipped while `enabled` is false); cancels stale results. */
export function useLoad<T>(load: () => Promise<T>, deps: unknown[], enabled = true): LoadState<T> {
  const [state, setState] = useState<LoadState<T>>({ data: null, loading: false, error: null });
  useEffect(() => {
    if (!enabled) return;
    let cancelled = false;
    setState((s) => ({ ...s, loading: true, error: null }));
    load()
      .then((data) => !cancelled && setState({ data, loading: false, error: null }))
      .catch((e: unknown) =>
        !cancelled && setState({ data: null, loading: false, error: e instanceof Error ? e.message : String(e) }),
      );
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, ...deps]);
  return state;
}
