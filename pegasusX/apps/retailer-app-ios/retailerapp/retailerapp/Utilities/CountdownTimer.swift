import SwiftUI

// MARK: - Countdown Timer

@Observable
final class CountdownTimer {
    var remainingString: String = "--:--:--"
    var isExpired: Bool = false

    private var targetDate: Date?
    private var anchorDate: Date?
    private var anchorUptime: TimeInterval?

    private static let isoFormatter = ISO8601DateFormatter()

    func start(from isoString: String, anchorNow: Date = Date(), anchorUptime: TimeInterval = ProcessInfo.processInfo.systemUptime) {
        targetDate = Self.isoFormatter.date(from: isoString)
        anchorDate = anchorNow
        self.anchorUptime = anchorUptime
    }

    func start(from date: Date, anchorNow: Date = Date(), anchorUptime: TimeInterval = ProcessInfo.processInfo.systemUptime) {
        targetDate = date
        anchorDate = anchorNow
        self.anchorUptime = anchorUptime
    }

    /// Monotonic clock update (Invariant I4): immune to user wall-clock rollback.
    func updateMonotonic(currentUptime: TimeInterval = ProcessInfo.processInfo.systemUptime) {
        guard let anchorDate, let anchorUptime else {
            update(now: Date())
            return
        }
        let elapsed = max(0, currentUptime - anchorUptime)
        update(now: anchorDate.addingTimeInterval(elapsed))
    }

    func update(now: Date) {
        guard let target = targetDate else {
            remainingString = "--:--:--"
            return
        }

        let remaining = target.timeIntervalSince(now)
        if remaining <= 0 {
            remainingString = "00:00:00"
            isExpired = true
            return
        }

        isExpired = false
        let hours = Int(remaining) / 3600
        let minutes = (Int(remaining) % 3600) / 60
        let seconds = Int(remaining) % 60
        remainingString = String(format: "%02d:%02d:%02d", hours, minutes, seconds)
    }
}

// MARK: - Countdown View (using TimelineView)

struct CountdownText: View {
    let targetISO: String
    var font: Font = .caption.monospacedDigit()
    var color: Color = AppTheme.textSecondary

    @State private var anchorDate = Date()
    @State private var anchorUptime = ProcessInfo.processInfo.systemUptime

    var body: some View {
        TimelineView(.periodic(from: .now, by: 1.0)) { _ in
            let elapsed = max(0, ProcessInfo.processInfo.systemUptime - anchorUptime)
            let monotonicNow = anchorDate.addingTimeInterval(elapsed)
            Text(countdownString(from: monotonicNow))
                .font(font)
                .foregroundStyle(color)
        }
        .onAppear {
            anchorDate = Date()
            anchorUptime = ProcessInfo.processInfo.systemUptime
        }
    }

    private static let isoFormatter = ISO8601DateFormatter()

    private func countdownString(from now: Date) -> String {
        guard let target = Self.isoFormatter.date(from: targetISO) else { return "--:--:--" }
        let remaining = target.timeIntervalSince(now)
        guard remaining > 0 else { return "Arrived" }
        let hours = Int(remaining) / 3600
        let minutes = (Int(remaining) % 3600) / 60
        let seconds = Int(remaining) % 60
        return String(format: "%02d:%02d:%02d", hours, minutes, seconds)
    }
}

#Preview {
    CountdownText(targetISO: "2026-03-17T18:00:00Z")
}
