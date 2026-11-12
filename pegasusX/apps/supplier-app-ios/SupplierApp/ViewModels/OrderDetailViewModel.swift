import Foundation
import Observation

@Observable
@MainActor
final class OrderDetailViewModel {
    var warehouseDetail: WarehouseOrderDetail?
    var detailError: String?
    var opsReason = ""
    var proposeDate = Date()
    var showProposeSheet = false
    var showRejectDialog = false

    private static let tashkentIsoFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        return formatter
    }()

    func isoDeliveryDate() -> String {
        let tz = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        var components = Calendar.current.dateComponents(in: tz, from: proposeDate)
        components.hour = 12
        components.minute = 0
        components.second = 0
        let noon = Calendar.current.date(from: components) ?? proposeDate
        return Self.tashkentIsoFormatter.string(from: noon)
    }

    func loadWarehouseDetail(for order: SupplierOrder) async {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return }
        do {
            let loaded = try await SupplierOperationsService.getWarehouseOrder(
                orderId: order.orderId,
                warehouseId: warehouseId
            )
            let outcome = PulseHonesty.applyObject(
                ok: true,
                incoming: loaded,
                previous: warehouseDetail,
                failure: "warehouse_detail_failed"
            )
            warehouseDetail = outcome.value
            detailError = nil
        } catch {
            let outcome = PulseHonesty.applyObject(
                ok: false,
                incoming: nil as WarehouseOrderDetail?,
                previous: warehouseDetail,
                failure: error.localizedDescription
            )
            warehouseDetail = outcome.value
            if warehouseDetail == nil {
                detailError = outcome.error
            } else if let notice = outcome.error {
                detailError = nil
                SupplierStatusCenter.shared.notice = notice
            }
        }
    }
}
