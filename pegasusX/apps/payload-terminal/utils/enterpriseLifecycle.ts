export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";

type Listener = () => void;

class EnterpriseNetworkLifecycleEngine {
  private _isOnline = typeof navigator === "undefined" ? true : navigator.onLine;
  private _state: NetworkState = this._isOnline ? "CONNECTED" : "OFFLINE";
  private _syncEpoch = 0;
  private _listeners = new Set<Listener>();
  private _transitionTimestamps: number[] = [];
  private _flappingTimer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("online", this.handleOnline);
      window.addEventListener("offline", this.handleOffline);
    }
  }

  public get isOnline(): boolean {
    return this._isOnline && this._state !== "FLAPPING" && this._state !== "CAPTIVE_BLOCKED";
  }

  public get state(): NetworkState {
    return this._state;
  }

  public get syncEpoch(): number {
    return this._syncEpoch;
  }

  public subscribe = (listener: Listener): (() => void) => {
    this._listeners.add(listener);
    return () => {
      this._listeners.delete(listener);
    };
  };

  private notify() {
    for (const listener of this._listeners) {
      listener();
    }
  }

  private recordTransition() {
    const now = Date.now();
    this._transitionTimestamps = this._transitionTimestamps.filter((t) => now - t < 30_000);
    this._transitionTimestamps.push(now);

    if (this._transitionTimestamps.length >= 3) {
      this._state = "FLAPPING";
      this.notify();

      if (this._flappingTimer) clearTimeout(this._flappingTimer);
      this._flappingTimer = setTimeout(() => {
        this._state = this._isOnline ? "CONNECTED" : "OFFLINE";
        this.notify();
        if (this._isOnline) {
          void this.triggerSyncEpoch();
        }
      }, 15_000);
    }
  }

  private handleOnline = () => {
    this._isOnline = true;
    this.recordTransition();
    if (this._state !== "FLAPPING") {
      this._state = "CONNECTED";
      this.notify();
      const jitter = Math.floor(Math.random() * 800) + 200;
      setTimeout(() => {
        void this.triggerSyncEpoch();
      }, jitter);
    }
  };

  private handleOffline = () => {
    this._isOnline = false;
    this.recordTransition();
    if (this._state !== "FLAPPING") {
      this._state = "OFFLINE";
      this.notify();
    }
  };

  public setOnline(online: boolean) {
    if (online) {
      this.handleOnline();
    } else {
      this.handleOffline();
    }
  }

  public async triggerSyncEpoch() {
    this._syncEpoch += 1;
    this.notify();
  }

  public reset() {
    this._isOnline = typeof navigator === "undefined" ? true : navigator.onLine;
    this._state = this._isOnline ? "CONNECTED" : "OFFLINE";
    this._syncEpoch = 0;
    this._transitionTimestamps = [];
    if (this._flappingTimer) {
      clearTimeout(this._flappingTimer);
      this._flappingTimer = null;
    }
    this.notify();
  }
}

export const NetworkLifecycleEngine = new EnterpriseNetworkLifecycleEngine();
