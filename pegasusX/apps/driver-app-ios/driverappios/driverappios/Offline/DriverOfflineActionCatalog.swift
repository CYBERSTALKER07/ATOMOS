import Foundation
import PegasusKit

enum DriverOfflineActionStatus: String {
    case pending = "PENDING"
    case dead = "DEAD"
}

enum DriverOfflineActionCatalog {
    static let maxAttempts = OfflineHttpSemantics.maxAttemptsDefault
    static let proximityMaxAge: TimeInterval = 120

    static let proximity = "v1/delivery/proximity-unlock"
    static let shopClosed = "v1/delivery/shop-closed"
    static let partial = "v1/delivery/partial-offload"
    static let deliver = "v1/order/deliver"
    static let collectCash = "v1/order/collect-cash"
    static let credit = "v1/delivery/credit-delivery"
    static let arrive = "v1/delivery/arrive"
    static let offload = "v1/order/confirm-offload"
    static let complete = "v1/order/complete"
    static let split = "v1/delivery/split-payment"
    static let bypassOffload = "v1/delivery/bypass-offload"
    static let paymentBypass = "v1/delivery/confirm-payment-bypass"
    static let depart = "v1/fleet/driver/depart"
    static let returnComplete = "v1/fleet/driver/return-complete"
    static let cashRecon = "v1/driver/cash-reconciliations"
    static let routeReorder = "v1/fleet/route/reorder"
    static let availability = "v1/driver/availability"

    static func normalize(_ endpoint: String) -> String {
        OfflineHttpSemantics.normalizeEndpoint(endpoint)
    }

    static func priority(for endpoint: String) -> Int {
        switch normalize(endpoint) {
        case proximity: return 10
        case shopClosed, partial, deliver: return 20
        case collectCash, credit: return 30
        default: return 40
        }
    }

    static func isOfflineEligible(_ endpoint: String) -> Bool {
        let ep = normalize(endpoint)
        let known: Set<String> = [
            proximity, shopClosed, partial, deliver, collectCash, credit,
            arrive, offload, complete, split, bypassOffload, paymentBypass,
            depart, returnComplete, cashRecon, routeReorder, availability,
        ]
        return known.contains(ep) || ep.contains("/fiscal/retry")
    }

    static func isRetryableHTTP(_ code: Int) -> Bool {
        OfflineHttpSemantics.isRetryableHTTP(code)
    }

    static func isSuccessHTTP(_ code: Int) -> Bool {
        OfflineHttpSemantics.isSuccessHTTP(code)
    }

    /// True only for transport / retryable failures. Business 4xx (geofence, forbidden,
    /// invalid transition, etc.) must never enqueue — that caused P0-4 silent "success".
    static func isNetworkEnqueueable(_ error: Error) -> Bool {
        if let apiErr = error as? APIError {
            switch apiErr {
            case .networkError:
                return true
            case .httpError(let code):
                return isRetryableHTTP(code)
            case .problemDetail(let problem):
                if problem.retryable == true { return true }
                return isRetryableHTTP(problem.status)
            case .unauthorized, .forbidden, .decodingError, .invalidURL, .explainError:
                return false
            }
        }
        if let urlErr = error as? URLError {
            switch urlErr.code {
            case .notConnectedToInternet, .timedOut, .networkConnectionLost,
                 .cannotFindHost, .cannotConnectToHost, .dnsLookupFailed,
                 .internationalRoamingOff, .dataNotAllowed:
                return true
            default:
                return false
            }
        }
        let ns = error as NSError
        if ns.domain == NSURLErrorDomain {
            switch URLError.Code(rawValue: ns.code) {
            case .notConnectedToInternet, .timedOut, .networkConnectionLost,
                 .cannotFindHost, .cannotConnectToHost, .dnsLookupFailed,
                 .internationalRoamingOff, .dataNotAllowed:
                return true
            default:
                return false
            }
        }
        return false
    }

    static func nowIso() -> String {
        ISO8601DateFormatter().string(from: Date())
    }
}
