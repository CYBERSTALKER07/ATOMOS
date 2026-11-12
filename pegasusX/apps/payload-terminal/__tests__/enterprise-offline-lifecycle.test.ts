import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { NetworkLifecycleEngine } from "../utils/enterpriseLifecycle";

describe("Payload Terminal Enterprise Network Lifecycle Engine", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    NetworkLifecycleEngine.reset();
  });

  afterEach(() => {
    NetworkLifecycleEngine.reset();
  });

  it("suppresses network operations when transitioning to OFFLINE", () => {
    NetworkLifecycleEngine.setOnline(false);

    expect(NetworkLifecycleEngine.isOnline).toBe(false);
    expect(NetworkLifecycleEngine.state).toBe("OFFLINE");
  });

  it("detects rapid flapping and enters 15-second cooldown state", () => {
    NetworkLifecycleEngine.setOnline(false);
    NetworkLifecycleEngine.setOnline(true);
    NetworkLifecycleEngine.setOnline(false);

    expect(NetworkLifecycleEngine.state).toBe("FLAPPING");
    expect(NetworkLifecycleEngine.isOnline).toBe(false);
  });

  it("coordinates single epoch refresh across multiple subscribers upon reconnect", async () => {
    NetworkLifecycleEngine.setOnline(false);

    const subscriberA = vi.fn();
    const subscriberB = vi.fn();

    const unsubA = NetworkLifecycleEngine.subscribe(subscriberA);
    const unsubB = NetworkLifecycleEngine.subscribe(subscriberB);

    expect(NetworkLifecycleEngine.syncEpoch).toBe(0);

    NetworkLifecycleEngine.setOnline(true);
    await NetworkLifecycleEngine.triggerSyncEpoch();

    expect(NetworkLifecycleEngine.syncEpoch).toBe(1);
    expect(subscriberA).toHaveBeenCalled();
    expect(subscriberB).toHaveBeenCalled();

    unsubA();
    unsubB();
  });
});
