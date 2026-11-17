// Maestro HTTP helper to check or toggle Toxiproxy toxic state without shell sleep/mocks.
if (typeof http !== "undefined" && env && env.TOXIPROXY_URL) {
  var mode = env.MODE || "status";
  try {
    if (mode === "offline") {
      http.post(env.TOXIPROXY_URL + "/proxies/backend_go/toxics", {
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "timeout_down", type: "timeout", attributes: { timeout: 1 } }),
      });
    } else if (mode === "online") {
      http.delete(env.TOXIPROXY_URL + "/proxies/backend_go/toxics/timeout_down");
    } else {
      http.get(env.TOXIPROXY_URL + "/version");
    }
  } catch (e) {
    // Non-fatal when running standalone without Toxiproxy sidecar
  }
}
