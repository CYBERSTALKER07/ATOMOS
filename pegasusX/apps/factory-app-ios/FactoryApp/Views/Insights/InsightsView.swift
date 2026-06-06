import SwiftUI

struct InsightsView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var insights: [Insight] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if insights.isEmpty {
                    ContentUnavailableView("No Insights", systemImage: "chart.bar.xaxis", description: Text("No replenishment insights"))
                } else {
                    List {
                        ForEach(Array(insights.enumerated()), id: \.element.id) { index, insight in
                            InsightRow(insight: insight)
                                .staggeredAppear(index: index)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Replenishment Insights")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .task { load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard let eventType = event.eventType else { return }
                        guard eventType == .supplyRequestUpdate || eventType == .transferUpdate else { return }
                        load()
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                insights = try await FactoryService.insights().insights
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

// MARK: - Insight Row
private struct InsightRow: View {
    let insight: Insight

    private var urgencyColor: Color {
        switch insight.urgency.uppercased() {
        case "CRITICAL": .red
        case "HIGH": .orange
        case "MEDIUM": .secondary
        default: .green
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(insight.productName.isEmpty ? String(insight.productId.prefix(8)) : insight.productName)
                        .font(.subheadline.bold())
                    Text(insight.warehouseName.isEmpty ? String(insight.warehouseId.prefix(8)) : insight.warehouseName)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Text(insight.urgency)
                    .font(.caption2.bold())
                    .foregroundStyle(urgencyColor)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(urgencyColor.opacity(0.12))
                    .clipShape(Capsule())
            }

            HStack {
                MetricPill(label: "Stock", value: "\(insight.currentStock)")
                MetricPill(label: "Vel/day", value: String(format: "%.1f", insight.avgDailyVelocity))
                MetricPill(label: "Days", value: "\(insight.daysUntilStockout)")
                MetricPill(label: "Reorder", value: "\(insight.reorderQuantity)")
            }
        }
        .padding(.vertical, 4)
    }
}

private struct MetricPill: View {
    let label: String
    let value: String

    var body: some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.caption.bold())
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
        .background(.quaternary)
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM))
    }
}
