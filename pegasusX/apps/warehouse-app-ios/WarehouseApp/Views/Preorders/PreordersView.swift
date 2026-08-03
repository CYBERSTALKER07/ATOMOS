import SwiftUI

private enum PreorderSheet: Identifiable {
    case propose(WarehousePreorderRow)
    case reject(WarehousePreorderRow)

    var id: String {
        switch self {
        case .propose(let row): return "propose-\(row.orderId)"
        case .reject(let row): return "reject-\(row.orderId)"
        }
    }
}

struct PreordersView: View {
    @State private var rows: [WarehousePreorderRow] = []
    @State private var loading = true
    @State private var acting = false
    @State private var activeSheet: PreorderSheet?
    @State private var proposeDate = Date()
    @State private var reasonInput = ""
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading && rows.isEmpty {
                WarehouseLoadingView(title: "Loading pre-orders…", message: "Fetching scheduled deliveries")
            } else if rows.isEmpty {
                if let error = statusMessage, error.contains("network") || error.contains("fail") {
                    WarehouseErrorView(message: error, retry: { Task { await load() } })
                } else {
                    WarehouseEmptyView(title: "No scheduled pre-orders", message: "No future orders waiting for delivery scheduling.")
                }
            } else {
                PreordersList(
                    rows: rows,
                    acting: acting,
                    onProposeDate: { row in
                        reasonInput = ""
                        proposeDate = initialProposeDate(for: row)
                        activeSheet = .propose(row)
                    },
                    onReject: { row in
                        reasonInput = ""
                        activeSheet = .reject(row)
                    }
                )
        }
        .navigationTitle("Pre-orders")
        .onAppear {
            Task { await load() }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .propose(let row):
                NavigationStack {
                    Form {
                        DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                        Section("Reason for date change") {
                            TextField("Reason", text: $reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                        }
                    }
                    .navigationTitle("Propose date")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { activeSheet = nil }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Send") { submitPropose(row) }
                                .disabled(acting || reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        }
                    }
                }
                .presentationDetents([.medium, .large])
            case .reject(let row):
                NavigationStack {
                    Form {
                        Section("Rejection reason") {
                            TextField("Reason", text: $reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                        }
                    }
                    .navigationTitle("Reject pre-order")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { activeSheet = nil }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Reject", role: .destructive) { submitReject(row) }
                                .disabled(acting || reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        }
                    }
                }
                .presentationDetents([.medium])
            }
        }
        .safeAreaInset(edge: .bottom) {
            if let statusMessage {
                Text(statusMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            let data = try await WarehouseService.preorders()
            rows = data.preorders.isEmpty ? data.items : data.preorders
        } catch {
            rows = []
            statusMessage = error.localizedDescription
        }
    }

    private func submitPropose(_ row: WarehousePreorderRow) {
        let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else { return }
        acting = true
        statusMessage = nil
        let iso = isoDeliveryDate(from: proposeDate)
        Task {
            defer { acting = false }
            do {
                let response = try await WarehouseOperationsService.proposePreorderDelivery(
                    orderId: row.orderId,
                    proposedDeliveryDate: iso,
                    reason: reason
                )
                activeSheet = nil
                statusMessage = "Delivery date proposed · \(response.status)"
                await load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }

    private func submitReject(_ row: WarehousePreorderRow) {
        let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else { return }
        acting = true
        statusMessage = nil
        Task {
            defer { acting = false }
            do {
                let response = try await WarehouseOperationsService.rejectPreorder(orderId: row.orderId, reason: reason)
                activeSheet = nil
                statusMessage = "Pre-order rejected · \(response.status)"
                await load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }


    private func initialProposeDate(for row: WarehousePreorderRow) -> Date {
        if let raw = row.requestedDeliveryDate?.prefix(10), !raw.isEmpty {
            let formatter = DateFormatter()
            formatter.calendar = Calendar(identifier: .gregorian)
            formatter.locale = Locale(identifier: "en_US_POSIX")
            formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
            formatter.dateFormat = "yyyy-MM-dd"
            if let parsed = formatter.date(from: String(raw)) {
                return parsed
            }
        }
        return Date()
    }

    private func isoDeliveryDate(from date: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 5 * 3600)!
        let day = calendar.dateComponents([.year, .month, .day], from: date)
        var noon = DateComponents()
        noon.year = day.year
        noon.month = day.month
        noon.day = day.day
        noon.hour = 12
        noon.minute = 0
        noon.second = 0
        noon.timeZone = calendar.timeZone
        let normalized = calendar.date(from: noon) ?? date
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withColonSeparatorInTimeZone]
        formatter.timeZone = calendar.timeZone
        return formatter.string(from: normalized)
    }
}

struct StockCommitmentsView: View {
    @State private var rows: [StockCommitmentRow] = []
    @State private var loading = true

    var body: some View {
        Group {
            if loading && rows.isEmpty {
                WarehouseLoadingView(title: "Loading commitments…", message: "Calculating stock reservations")
            } else if rows.isEmpty {
                WarehouseEmptyView(title: "No commitments", message: "No stock is currently reserved for upcoming orders.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.name ?? row.skuId).font(.headline)
                        Text("Available \(row.availableQty) · ASAP \(row.reservedAsap) · Scheduled \(row.reservedScheduled)")
                            .font(.caption)
                        Text("On hand \(row.onHand)").font(.caption2)
                        if row.deficitQty > 0 {
                            Text("Short \(row.deficitQty)").font(.caption).foregroundStyle(.red)
                        }
                    }
                }
            }
        }
        .navigationTitle("Stock commitments")
        .task { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            let data = try await WarehouseService.stockCommitments()
            rows = data.items.isEmpty ? data.skus : data.items
        } catch {
            rows = []
        }
    }
}
