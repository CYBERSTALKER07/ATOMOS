"use client";

import { useCallback, useEffect, useState } from "react";

export type DesktopConnectivity = {
  isOffline: boolean;
  /** Fires when the browser reports online again. */
  onOnline: (listener: () => void) => () => void;
};

function readOnline(): boolean {
  return typeof navigator === "undefined" ? true : navigator.onLine;
}

/** Tracks `navigator.onLine` with online/offline window events. */
export function useDesktopConnectivity(): DesktopConnectivity {
  const [isOffline, setIsOffline] = useState(() => !readOnline());

  useEffect(() => {
    const sync = () => setIsOffline(!readOnline());
    sync();
    window.addEventListener("online", sync);
    window.addEventListener("offline", sync);
    return () => {
      window.removeEventListener("online", sync);
      window.removeEventListener("offline", sync);
    };
  }, []);

  const onOnline = useCallback((listener: () => void) => {
    window.addEventListener("online", listener);
    return () => window.removeEventListener("online", listener);
  }, []);

  return { isOffline, onOnline };
}
