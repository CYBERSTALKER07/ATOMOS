import SwiftUI

enum StatusStackMode: String {
    case empty
    case zero
    case live
    case unavailable
}

struct StatusStackRow: Equatable {
    let key: String
    let count: Int?
    let share: Double
}

struct StatusStackModel: Equatable {
    let mode: StatusStackMode
    let rows: [StatusStackRow]
    let total: Int
}

let orderStatusFunnel = [
    "PENDING", "SCHEDULED", "AUTO_ACCEPTED", "BACKORDERED",
    "LOADED", "IN_TRANSIT", "DELAYED",
    "ARRIVED", "ARRIVED_SHOP_CLOSED",
    "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT",
    "FISCALIZING", "FISCAL_FAILED", "RECONCILIATION_REQUIRED",
    "COMPLETED", "CANCELLED",
]

let manifestStates = [
    "DRAFT", "LOADING", "SEALED", "DISPATCHED", "COMPLETED", "CANCELLED",
]

let truckDutyStatuses = [
    "AVAILABLE",
    "IN_TRANSIT",
    "RETURNING_TO_WAREHOUSE",
    "OFF_SHIFT",
    "UNASSIGNED",
    "VEHICLE_INACTIVE",
    "UNAVAILABLE",
    "INACTIVE",
]

let factoryTransferStates = [
    "CREATED", "APPROVED", "PENDING", "ASSIGNED", "LOADING",
    "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED", "REASSIGNED",
]

let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]

let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]

func canonicalizeOrderStatus(_ status: String) -> String {
    switch status.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() {
    case "DISPATCHED": return "LOADED"
    case "EN_ROUTE": return "IN_TRANSIT"
    case "ARRIVING": return "ARRIVED"
    case "SHOP_CLOSED_PENDING": return "ARRIVED_SHOP_CLOSED"
    default: return status.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
    }
}

func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
    var next = Dictionary(uniqueKeysWithValues: orderStatusFunnel.map { ($0, 0) })
    for (key, value) in counts {
        let normalized = canonicalizeOrderStatus(key)
        if next[normalized] != nil {
            next[normalized] = value
        }
    }
    let key = canonicalizeOrderStatus(status)
    if next[key] != nil {
        next[key, default: 0] += 1
    }
    return next
}

func statusStackModel(
    dictionary: [String] = orderStatusFunnel,
    counts: [String: Int]?,
    available: Bool = true
) -> StatusStackModel {
    if !available {
        return StatusStackModel(
            mode: .unavailable,
            rows: dictionary.map { StatusStackRow(key: $0, count: nil, share: 0) },
            total: 0
        )
    }
    guard let counts else {
        return StatusStackModel(mode: .empty, rows: [], total: 0)
    }
    var rows = dictionary.map { key in
        StatusStackRow(key: key, count: counts[key] ?? 0, share: 0)
    }
    let total = rows.reduce(0) { $0 + ($1.count ?? 0) }
    if total > 0 {
        rows = rows.map { row in
            StatusStackRow(key: row.key, count: row.count, share: Double(row.count ?? 0) / Double(total))
        }
    }
    return StatusStackModel(mode: total == 0 ? .zero : .live, rows: rows, total: total)
}

struct SourceChip: View {
    let source: String

    var body: some View {
        Text(source.uppercased())
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .overlay(Capsule().strokeBorder(.secondary, lineWidth: 1))
            .accessibilityIdentifier("gs-u-source-chip")
    }
}

struct CommandStatusJump: Hashable, Identifiable {
    let id: UUID
    let status: String
    var supplierId: String?

    init(status: String, supplierId: String? = nil) {
        self.id = UUID()
        self.status = canonicalizeOrderStatus(status)
        self.supplierId = supplierId
    }
}

struct StatusStackView: View {
    let dictionary: [String]
    let counts: [String: Int]?
    var available: Bool = true
    var source: String? = nil
    var onSelect: ((String) -> Void)? = nil

    private var model: StatusStackModel {
        statusStackModel(dictionary: dictionary, counts: counts, available: available)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Spacer()
                SourceChip(source: source ?? (model.mode == .unavailable ? "unavailable" : model.mode == .empty ? "empty" : "live"))
            }
            if model.mode == .empty {
                Text("No status counts")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else if model.mode == .unavailable {
                Text("Status counts unavailable")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if model.mode == .live {
                GeometryReader { geo in
                    HStack(spacing: 0) {
                        ForEach(model.rows.filter { $0.share > 0 }, id: \.key) { row in
                            Rectangle()
                                .fill(Color.primary.opacity(0.35 + row.share * 0.65))
                                .frame(width: geo.size.width * row.share)
                        }
                    }
                }
                .frame(height: 10)
                .clipShape(Capsule())
            }
            if model.mode != .empty {
                FlexibleStatusChips(rows: model.rows, onSelect: onSelect)
            }
        }
        .accessibilityIdentifier("gs-u-status-stack")
    }
}

private struct FlexibleStatusChips: View {
    let rows: [StatusStackRow]
    var onSelect: ((String) -> Void)? = nil

    var body: some View {
        FlowLayout {
            ForEach(rows, id: \.key) { row in
                chip(row)
            }
        }
    }

    @ViewBuilder
    private func chip(_ row: StatusStackRow) -> some View {
        let label = HStack(spacing: 6) {
            Text(row.key.replacingOccurrences(of: "_", with: " "))
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(row.count.map(String.init) ?? "—")
                .font(.caption.monospacedDigit())
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .frame(minHeight: 44)
        .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(.secondary.opacity(0.4)))
        .accessibilityIdentifier("gs-u-chip-\(row.key)")

        if let onSelect {
            Button {
                onSelect(row.key)
            } label: {
                label
            }
            .buttonStyle(.plain)
        } else {
            label
        }
    }
}

private struct FlowLayout: Layout {
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let maxWidth = proposal.width ?? .infinity
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowH: CGFloat = 0
        var height: CGFloat = 0
        for view in subviews {
            let size = view.sizeThatFits(.unspecified)
            if x + size.width > maxWidth && x > 0 {
                x = 0
                y += rowH + 8
                rowH = 0
            }
            rowH = max(rowH, size.height)
            x += size.width + 8
            height = y + rowH
        }
        return CGSize(width: maxWidth.isFinite ? maxWidth : x, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX
        var y = bounds.minY
        var rowH: CGFloat = 0
        for view in subviews {
            let size = view.sizeThatFits(.unspecified)
            if x + size.width > bounds.maxX && x > bounds.minX {
                x = bounds.minX
                y += rowH + 8
                rowH = 0
            }
            view.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
            x += size.width + 8
            rowH = max(rowH, size.height)
        }
    }
}
