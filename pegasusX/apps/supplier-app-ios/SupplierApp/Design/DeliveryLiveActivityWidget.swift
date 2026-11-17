import ActivityKit
import SwiftUI
import WidgetKit

public struct DeliveryLiveActivityWidget: Widget {
    public init() {}

    public var body: some WidgetConfiguration {
        ActivityConfiguration(for: DeliveryActivityAttributes.self) { context in
            LockScreenLiveActivityView(context: context)
                .activityBackgroundTint(Color.black.opacity(0.88))
                .activitySystemActionForegroundColor(Color.white)
        } dynamicIsland: { context in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    HStack(spacing: 6) {
                        Image(systemName: context.attributes.isDriverPerspective ? "location.north.line.fill" : "shippingbox.fill")
                            .font(.callout.bold())
                            .foregroundStyle(Color.blue)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(context.attributes.isDriverPerspective ? formatDistance(context.state.remainingDistanceMeters) : context.attributes.supplierName)
                                .font(.system(.footnote, design: .rounded, weight: .bold))
                                .foregroundStyle(.white)
                                .lineLimit(1)
                            Text(context.state.status)
                                .font(.caption2.weight(.medium))
                                .foregroundStyle(.white.opacity(0.7))
                        }
                    }
                    .padding(.leading, 8)
                }

                DynamicIslandExpandedRegion(.trailing) {
                    VStack(alignment: .trailing, spacing: 2) {
                        HStack(spacing: 4) {
                            Image(systemName: "clock.fill")
                                .font(.caption2)
                            Text("\(context.state.etaMinutes) min")
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        }
                        .foregroundStyle(Color.green)

                        if let token = context.state.deliveryToken, !token.isEmpty {
                            Text("PIN: \(token)")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(Color.yellow)
                        }
                    }
                    .padding(.trailing, 8)
                }

                DynamicIslandExpandedRegion(.center) {
                    Text(context.attributes.isDriverPerspective ? context.state.nextInstruction : "Approaching \(context.attributes.destinationName)")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(.white)
                        .lineLimit(1)
                }

                DynamicIslandExpandedRegion(.bottom) {
                    VStack(spacing: 4) {
                        ProgressView(value: min(max(context.state.progress, 0.0), 1.0))
                            .tint(Color.blue)
                        HStack {
                            Text(context.attributes.isDriverPerspective
                                ? "Step \(context.state.currentStepIndex)/\(max(context.state.totalSteps, 1))"
                                : "\(context.attributes.totalItemCount) items")
                                .font(.caption2)
                                .foregroundStyle(.white.opacity(0.6))
                            Spacer()
                            Text(context.attributes.orderId)
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundStyle(.white.opacity(0.6))
                        }
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 6)
                }
            } compactLeading: {
                Image(systemName: context.attributes.isDriverPerspective ? "location.fill" : "truck.box.fill")
                    .font(.caption.bold())
                    .foregroundStyle(Color.blue)
            } compactTrailing: {
                Text("\(context.state.etaMinutes)m")
                    .font(.system(.caption, design: .monospaced, weight: .bold))
                    .foregroundStyle(Color.green)
            } minimal: {
                Image(systemName: context.attributes.isDriverPerspective ? "location.fill" : "shippingbox.fill")
                    .font(.caption2)
                    .foregroundStyle(Color.blue)
            }
        }
    }

    private func formatDistance(_ meters: Double) -> String {
        if meters < 1000 {
            return "\(Int(meters.rounded())) m"
        }
        return String(format: "%.1f km", meters / 1000.0)
    }
}

private struct LockScreenLiveActivityView: View {
    let context: ActivityViewContext<DeliveryActivityAttributes>

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                HStack(spacing: 6) {
                    Circle()
                        .fill(Color.green)
                        .frame(width: 8, height: 8)
                    Text(context.attributes.isDriverPerspective ? "LIVE ROUTE NAVIGATION" : "INBOUND DELIVERY")
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(.white.opacity(0.8))
                }
                Spacer()
                Text("\(context.state.etaMinutes) MIN ETA")
                    .font(.system(.caption, design: .monospaced, weight: .bold))
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Color.green.opacity(0.2), in: Capsule())
                    .foregroundStyle(Color.green)
            }

            HStack(alignment: .center, spacing: 12) {
                Image(systemName: context.attributes.isDriverPerspective ? "arrow.triangle.turn.up.right.diamond.fill" : "truck.box.fill")
                    .font(.title.weight(.semibold))
                    .foregroundStyle(Color.blue)

                VStack(alignment: .leading, spacing: 2) {
                    Text(context.attributes.isDriverPerspective ? context.state.nextInstruction : context.attributes.supplierName)
                        .font(.subheadline.bold())
                        .foregroundStyle(Color.white)
                        .lineLimit(2)

                    Text(context.attributes.isDriverPerspective ? "Distance: \(Int(context.state.remainingDistanceMeters))m" : "To: \(context.attributes.destinationName)")
                        .font(.caption)
                        .foregroundStyle(Color.white.opacity(0.7))
                }
                Spacer()

                if let token = context.state.deliveryToken, !token.isEmpty {
                    VStack(spacing: 2) {
                        Text("VERIFY PIN")
                            .font(.caption2.bold())
                            .foregroundStyle(Color.white.opacity(0.6))
                        Text(token)
                            .font(.system(.callout, design: .monospaced, weight: .bold))
                            .foregroundStyle(Color.yellow)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.white.opacity(0.12), in: RoundedRectangle(cornerRadius: 8))
                }
            }

            ProgressView(value: min(max(context.state.progress, 0.0), 1.0))
                .tint(Color.blue)
        }
        .padding(16)
        .background(Color.black.opacity(0.92))
    }
}
