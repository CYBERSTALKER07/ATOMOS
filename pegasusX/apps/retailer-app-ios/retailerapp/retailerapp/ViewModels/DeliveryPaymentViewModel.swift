import Foundation
import Observation
import UIKit

enum PaymentPhase: Equatable {
    case choose
    case cashConfirm
    case processing
    case cashPending
    case fiscalizing
    case success
    case failed
}

struct CardGatewayOption: Identifiable {
    let gateway: String
    let label: String
    let description: String

    var id: String { gateway }
}

@Observable
@MainActor
final class DeliveryPaymentViewModel {
    var phase: PaymentPhase = .choose
    var errorMessage: String?
    var catalogCodes: [String] = []
    var packDisplayCurrency = packCurrency(MarketPackStore.pack)

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    func cardGatewayOptions(availableCardGateways: [String]) -> [CardGatewayOption] {
        let gateways = filterRetailerCardGateways(
            incoming: availableCardGateways,
            catalog: catalogCodes
        )

        return gateways.compactMap { gateway in
            switch gateway {
            case "GLOBAL_PAY":
                CardGatewayOption(gateway: gateway, label: "GlobalPay", description: "Pay via GlobalPay checkout")
            default:
                nil
            }
        }
    }

    func fetchPaymentCatalog() async {
        do {
            let resp = try await api.getPaymentCatalog()
            catalogCodes = selectableRetailerCatalogCodes(resp.catalog)
            if !resp.currencyCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                packDisplayCurrency = resp.currencyCode.uppercased()
            }
        } catch {
            catalogCodes = []
            packDisplayCurrency = packCurrency(MarketPackStore.pack)
        }
    }

    func initiateCashCheckout(orderId: String) async {
        Haptics.medium()
        do {
            let body = ["order_id": orderId]
            let _: [String: DiscardablePaymentCodable] = try await api.post(
                path: "/v1/order/cash-checkout",
                body: body,
                headers: ["Idempotency-Key": "retailer-cash-checkout:\(orderId)"]
            )
            Haptics.success()
            phase = .cashPending
        } catch {
            Haptics.error()
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry payment.",
                fallback: "Payment could not be completed. Please try again."
            )
            phase = .failed
        }
    }

    func confirmCash(orderId: String) async {
        Haptics.medium()
        phase = .processing
        do {
            let body = ["order_id": orderId]
            let _: [String: DiscardablePaymentCodable] = try await api.post(
                path: "/v1/delivery/confirm-cash",
                body: body,
                headers: ["Idempotency-Key": RetailerIdempotency.confirmCash(orderId: orderId)]
            )
            Haptics.success()
            phase = .cashPending
        } catch {
            Haptics.error()
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry cash confirmation.",
                fallback: "Cash confirmation could not be completed. Please try again."
            )
            phase = .failed
        }
    }

    func initiateCardCheckout(orderId: String, gateway: String) async {
        Haptics.medium()
        phase = .processing
        do {
            let body = ["order_id": orderId, "gateway": gateway]
            let resp: CardCheckoutResponse = try await api.post(
                path: "/v1/order/card-checkout",
                body: body,
                headers: ["Idempotency-Key": "retailer-card-checkout:\(orderId):\(gateway)"]
            )
            if let url = URL(string: resp.paymentUrl), !resp.paymentUrl.isEmpty {
                _ = await UIApplication.shared.open(url)
            } else {
                Haptics.error()
                errorMessage = "Payment gateway is not configured for this supplier."
                phase = .failed
            }
        } catch {
            Haptics.error()
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry payment.",
                fallback: "Payment could not be completed. Please try again."
            )
            phase = .failed
        }
    }

    func listenForCompletion(orderId: String) async {
        for await wsEvent in ws.eventStream() {
            switch wsEvent {
            case .orderCompleted(let completed) where completed.orderId == orderId:
                Haptics.success()
                phase = .success
                return
            case .fiscalSucceeded(let succeeded) where succeeded.orderId == orderId:
                Haptics.success()
                phase = .success
                return
            case .paymentSettled(let settled) where settled.orderId == orderId:
                phase = .fiscalizing
            case .fiscalizing(let fid) where fid == orderId:
                phase = .fiscalizing
            case .orderStatusChanged(let sid, let state) where sid == orderId:
                let st = state.uppercased()
                if st == "FISCALIZING" {
                    phase = .fiscalizing
                } else if st == "COMPLETED" {
                    Haptics.success()
                    phase = .success
                    return
                }
            default:
                continue
            }
        }
    }

    func resetToChoose() {
        Haptics.light()
        phase = .choose
        errorMessage = nil
    }
}

// MARK: - Card Checkout Response

private struct CardCheckoutResponse: Decodable {
    let orderId: String
    let state: String
    let amountUzs: Int64
    let gateway: String
    let resolvedGateway: String?
    let policySource: String?
    let allowedGateways: [String]?
    let policyReason: String?
    let paymentUrl: String
    let invoiceId: String
    let sessionId: String?
    let attemptId: String?
    let attemptNo: Int?
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amountUzs = "amount"
        case gateway
        case resolvedGateway = "resolved_gateway"
        case policySource = "policy_source"
        case allowedGateways = "allowed_gateways"
        case policyReason = "policy_reason"
        case paymentUrl = "payment_url"
        case invoiceId = "invoice_id"
        case sessionId = "session_id"
        case attemptId = "attempt_id"
        case attemptNo = "attempt_no"
        case message
    }
}

// MARK: - DiscardableCodable Helper

private struct DiscardablePaymentCodable: Decodable {
    init(from decoder: Decoder) throws {
        _ = try decoder.singleValueContainer()
    }
}
