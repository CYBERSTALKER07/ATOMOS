import SwiftUI

// MARK: - Countdown Timer

@Observable
final class CountdownTimer {
    var remainingString: String = "--:--:--"
    var isExpired: Bool = false

    private var targetDate: Date?

    func start(from isoString: String) {
        let formatter = ISO8601DateFormatter()
        targetDate = formatter.date(from: isoString)
    }

    func start(from date: Date) {
        targetDate = date
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

    var body: some View {
        TimelineView(.periodic(from: .now, by: 1.0)) { context in
            Text(countdownString(from: context.date))
                .font(font)
                .foregroundStyle(color)
        }
    }

    private func countdownString(from now: Date) -> String {
        let formatter = ISO8601DateFormatter()
        guard let target = formatter.date(from: targetISO) else { return "--:--:--" }
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
