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
