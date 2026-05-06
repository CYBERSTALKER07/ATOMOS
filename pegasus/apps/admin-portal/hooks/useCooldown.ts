'use client';

/**
 * useCooldown — Listens for backend rate-limit jail signals (X-Jail-Until)
 * and exposes a live countdown for surface-level UX. Surfaces can render a
 * toast/banner with a ticking "retry in Xs" rather than a generic 429 error.
 *
 * Wire once at the app shell level; emits via the `cooldown` window event
 * dispatched from lib/auth.ts when the backend returns 429 + X-Jail-Until.
 */

import { useEffect, useState } from 'react';

export interface CooldownState {
  active: boolean;
  jailUntil: number;       // unix seconds
  remainingSec: number;
  priority: string;
  lastPath: string;
}

const initial: CooldownState = {
  active: false,
  jailUntil: 0,
  remainingSec: 0,
  priority: '',
  lastPath: '',
};

interface CooldownEventDetail {
  jailUntil: number;
  priority: string;
  path: string;
}

export function useCooldown(): CooldownState {
  const [state, setState] = useState<CooldownState>(initial);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const onCooldown = (ev: Event) => {
      const detail = (ev as CustomEvent<CooldownEventDetail>).detail;
      if (!detail || !detail.jailUntil) return;
      const now = Math.floor(Date.now() / 1000);
      const remaining = Math.max(0, detail.jailUntil - now);
      setState({
        active: remaining > 0,
        jailUntil: detail.jailUntil,
        remainingSec: remaining,
        priority: detail.priority,
        lastPath: detail.path,
      });
    };

    window.addEventListener('cooldown', onCooldown);

    const tick = window.setInterval(() => {
      setState((prev) => {
        if (!prev.active) return prev;
        const now = Math.floor(Date.now() / 1000);
        const remaining = Math.max(0, prev.jailUntil - now);
        return { ...prev, remainingSec: remaining, active: remaining > 0 };
      });
    }, 1000);

    return () => {
      window.removeEventListener('cooldown', onCooldown);
      window.clearInterval(tick);
    };
  }, []);

  return state;
}
