import SwiftUI

struct TomorrowBoardView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var date = Calendar.current.date(byAdding: .day, value: 1, to: Date()) ?? Date()
    @State private var preorders: [WarehouseOpsBoardOrder] = []
    @State private var deliverBefore: [WarehouseOpsBoardOrder] = []
    @State private var loading = true
    @State private var error: String?

    private var dateString: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter.string(from: date)
    }

    var body: some View {
        Group {
            if loading && preorders.isEmpty && deliverBefore.isEmpty {
                WarehouseLoadingView(
                    title: "Loading tomorrow board",
                    message: "Fetching orders and manifests grouped by delivery date."
                )
            } else if let error, preorders.isEmpty && deliverBefore.isEmpty {
                WarehouseErrorView(message: error) { load() }
            } else {
                List {
                    Section {
                        DatePicker("Date", selection: $date, displayedComponents: .date)
                            .onChange(of: date) { _, _ in load() }
                    }
                    if preorders.isEmpty && deliverBefore.isEmpty {
                        Section {
                            Text("No orders scheduled for this date.")
                                .foregroundStyle(.secondary)
                        }
                    }
                    if !preorders.isEmpty {
                        Section("Pre-orders") {
                            ForEach(preorders) { row in
                                boardRow(row, lane: "Pre-order")
                            }
                        }
                    }
                    if !deliverBefore.isEmpty {
                        Section("Deliver by") {
                            ForEach(deliverBefore) { row in
                                boardRow(row, lane: "Deliver by")
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("Tomorrow board")
        .task { load() }
        .refreshable { load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            load(silent: silent)
        }
    }

    @ViewBuilder
    private func boardRow(_ row: WarehouseOpsBoardOrder, lane: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(lane)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(row.orderId)
                .font(.headline)
            Text(row.deliveryExpectation?.targetLabel ?? row.status)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private func load(silent: Bool = false) {
        Task {
            if !silent { loading = true }
            error = nil
            do {
                let body = try await WarehouseService.opsBoard(date: dateString)
                preorders = body.preorders
                deliverBefore = body.deliverBefore
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }
}
