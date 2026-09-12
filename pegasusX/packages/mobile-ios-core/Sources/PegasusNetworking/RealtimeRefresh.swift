import SwiftUI

/// Stale-while-revalidate reload hooks driven by WebSocket hub epochs.
///
/// View models should implement `load(silent:)` — skip `loading = true` when `silent` is true
/// and data is already populated.
public struct SilentRefreshModifier: ViewModifier {
    private let refreshEpoch: Int
    private let reconnectEpoch: Int
    private let reload: (_ silent: Bool) -> Void

    public init(
        refreshEpoch: Int,
        reconnectEpoch: Int = 0,
        reload: @escaping (_ silent: Bool) -> Void
    ) {
        self.refreshEpoch = refreshEpoch
        self.reconnectEpoch = reconnectEpoch
        self.reload = reload
    }

    public func body(content: Content) -> some View {
        content
            .onChange(of: refreshEpoch) { _, _ in reload(true) }
            .onChange(of: reconnectEpoch) { _, _ in reload(true) }
    }
}

public extension View {
    func silentRealtimeRefresh(
        refreshEpoch: Int,
        reconnectEpoch: Int = 0,
        reload: @escaping (_ silent: Bool) -> Void
    ) -> some View {
        modifier(
            SilentRefreshModifier(
                refreshEpoch: refreshEpoch,
                reconnectEpoch: reconnectEpoch,
                reload: reload
            )
        )
    }
}

public enum RealtimeLoadState {
    public static func showFullScreenLoading(loading: Bool, hasData: Bool) -> Bool {
        loading && !hasData
    }
}

/// Fail-closed code for native pulse GET. Do not treat HTTP failure as an empty timeline.
public enum PulseHonesty {
    public static let failed = "pulse_failed"
    public static let commandFailed = "control_tower_pulse_failed"

    public struct Result<T> {
        public let events: [T]
        public let error: String?
    }

    public struct ObjectResult<T> {
        public let value: T?
        public let error: String?
    }

    public static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
        if ok, let incoming {
            return Result(events: incoming, error: nil)
        }
        return Result(events: previous, error: failed)
    }

    public static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
        if ok, let incoming {
            return ObjectResult(value: incoming, error: nil)
        }
        return ObjectResult(value: previous, error: commandFailed)
    }
}
