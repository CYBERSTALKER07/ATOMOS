import SwiftUI

enum PulseHonesty {
    static let failed = "pulse_failed"
    static let commandFailed = "control_tower_pulse_failed"

    struct Result<T> {
        let events: [T]
        let error: String?
    }

    struct ObjectResult<T> {
        let value: T?
        let error: String?
    }

    static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
        if ok, let incoming {
            return Result(events: incoming, error: nil)
        }
        return Result(events: previous, error: failed)
    }

    static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
        if ok, let incoming {
            return ObjectResult(value: incoming, error: nil)
        }
        return ObjectResult(value: previous, error: commandFailed)
    }
}

struct PulseStripView: View {
    let events: [RetailerPulseEvent]
    let loading: Bool
    var error: String? = nil

    var body: some View {
        Group {
            if loading && events.isEmpty && (error ?? "").isEmpty {
                Text("Loading network pulse…")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textTertiary)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if let error, !error.isEmpty {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(AppTheme.destructive)
                    .frame(maxWidth: .infinity, alignment: .leading)
            } else if !events.isEmpty {
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Text("Network pulse")
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textTertiary)

                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: AppTheme.spacingMD) {
                            ForEach(events.prefix(12)) { event in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(event.title.isEmpty ? (event.kind ?? "Event") : event.title)
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                        .foregroundStyle(AppTheme.textPrimary)
                                        .lineLimit(2)
                                    if let description = event.description, !description.isEmpty {
                                        Text(description)
                                            .font(.caption)
                                            .foregroundStyle(AppTheme.textTertiary)
                                            .lineLimit(2)
                                    }
                                }
                                .padding(.horizontal, 12)
                                .padding(.vertical, 10)
                                .frame(minWidth: 180, maxWidth: 240, alignment: .leading)
                                .background(AppTheme.cardBackground)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                                .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
                            }
                        }
                    }
                }
            }
        }
    }
}
