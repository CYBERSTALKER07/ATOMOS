import Foundation

struct AnimatedDriverCoordinate: Identifiable {
    let id: String
    let lat: Double
    let lng: Double
    let stale: Bool
}

/// Smoothly interpolates driver GPS markers between poll / websocket refreshes.
final class FleetDriverMarkerAnimator {
    private struct AnimState {
        let driverId: String
        var fromLat: Double
        var fromLng: Double
        var toLat: Double
        var toLng: Double
        var startMs: TimeInterval
        let stale: Bool
    }

    private var states: [String: AnimState] = [:]
    private let durationMs: TimeInterval = 1_200

    func updateTargets(_ routes: [WarehouseFleetLiveRoute]) {
        var active = Set<String>()
        for route in routes {
            guard route.liveLocationAvailable, let location = route.driverLocation else { continue }
            let lat = location.resolvedLatitude
            let lng = location.resolvedLongitude
            guard lat.isFinite, lng.isFinite else { continue }
            active.insert(route.driverId)
            let stale = route.locationStale == true
            if var existing = states[route.driverId] {
                if existing.toLat != lat || existing.toLng != lng {
                    let now = Date().timeIntervalSince1970 * 1_000
                    let progress = progressAt(nowMs: now, state: existing)
                    let currentLat = lerp(existing.fromLat, existing.toLat, progress)
                    let currentLng = lerp(existing.fromLng, existing.toLng, progress)
                    existing.fromLat = currentLat
                    existing.fromLng = currentLng
                    existing.toLat = lat
                    existing.toLng = lng
                    existing.startMs = now
                }
                states[route.driverId] = existing
            } else {
                let now = Date().timeIntervalSince1970 * 1_000
                states[route.driverId] = AnimState(
                    driverId: route.driverId,
                    fromLat: lat,
                    fromLng: lng,
                    toLat: lat,
                    toLng: lng,
                    startMs: now,
                    stale: stale
                )
            }
        }
        states = states.filter { active.contains($0.key) }
    }

    func snapshot(now: Date = .now) -> [AnimatedDriverCoordinate] {
        let nowMs = now.timeIntervalSince1970 * 1_000
        return states.values.map { state in
            let t = easeOut(progressAt(nowMs: nowMs, state: state))
            return AnimatedDriverCoordinate(
                id: state.driverId,
                lat: lerp(state.fromLat, state.toLat, t),
                lng: lerp(state.fromLng, state.toLng, t),
                stale: state.stale
            )
        }
    }

    private func progressAt(nowMs: TimeInterval, state: AnimState) -> Double {
        guard durationMs > 0 else { return 1 }
        return min(1, max(0, (nowMs - state.startMs) / durationMs))
    }

    private func lerp(_ a: Double, _ b: Double, _ t: Double) -> Double {
        a + (b - a) * t
    }

    private func easeOut(_ t: Double) -> Double {
        t * (2 - t)
    }
}
