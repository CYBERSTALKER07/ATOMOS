import Foundation

struct RemainingStop: Equatable, Identifiable {
    let id: String
    let title: String
    let state: OrderState
    let sequenceIndex: Int
    var firstClass: Bool { RemainingStops.isFirstClass(state) }
}

enum RemainingStops {
    static func isFirstClass(_ state: OrderState) -> Bool {
        state == .ARRIVED_SHOP_CLOSED || state == .FISCAL_FAILED
    }

    static func remaining(_ orders: [Order]) -> [RemainingStop] {
        orders
            .filter { $0.state != .COMPLETED && $0.state != .CANCELLED }
            .sorted {
                let a = $0.sequenceIndex ?? Int.max
                let b = $1.sequenceIndex ?? Int.max
                if a != b { return a < b }
                return $0.id < $1.id
            }
            .map {
                RemainingStop(
                    id: $0.id,
                    title: $0.retailerName,
                    state: $0.state,
                    sequenceIndex: $0.sequenceIndex ?? 0
                )
            }
    }

    static func remaining(from inputs: [RemainingStop]) -> [RemainingStop] {
        inputs
            .filter { $0.state != .COMPLETED && $0.state != .CANCELLED }
            .sorted {
                if $0.sequenceIndex != $1.sequenceIndex { return $0.sequenceIndex < $1.sequenceIndex }
                return $0.id < $1.id
            }
    }
}

struct MoneyHealthCounts: Equatable {
    var pendingCash: Int
    var openFiscal: Int
    var creditLeave: Int

    static func from(orders: [Order]) -> MoneyHealthCounts {
        MoneyHealthCounts(
            pendingCash: orders.filter { $0.state == .PENDING_CASH_COLLECTION || $0.state == .AWAITING_PAYMENT }.count,
            openFiscal: orders.filter { $0.state == .FISCALIZING || $0.state == .FISCAL_FAILED }.count,
            creditLeave: orders.filter { $0.state == .DELIVERED_ON_CREDIT }.count
        )
    }
}
