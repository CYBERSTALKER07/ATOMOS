import Foundation

enum RetailerRequestIssue {
    case restricted
    case offline
    case degraded
}

enum RetailerErrorSupport {
    static func classify(_ error: Error) -> RetailerRequestIssue {
        if let apiError = error as? APIError {
            switch apiError {
            case .serverError(let statusCode, _):
                if statusCode == 401 || statusCode == 403 {
                    return .restricted
                }
            case .problemDetail(let problem):
                if problem.status == 401 || problem.status == 403 {
                    return .restricted
                }
            default:
                break
            }
        }

        if let urlError = error as? URLError {
            switch urlError.code {
            case .notConnectedToInternet,
                    .networkConnectionLost,
                    .timedOut,
                    .cannotFindHost,
                    .cannotConnectToHost,
                    .dnsLookupFailed,
                    .internationalRoamingOff,
                    .dataNotAllowed,
                    .callIsActive:
                return .offline
            default:
                break
            }
        }

        let nsError = error as NSError
        if nsError.domain == NSURLErrorDomain {
            return .offline
        }

        return .degraded
    }

    static func message(
        for error: Error,
        restricted: String,
        offline: String,
        fallback: String
    ) -> String {
        switch classify(error) {
        case .restricted:
            return restricted
        case .offline:
            return offline
        case .degraded:
            return fallback
        }
    }

    static func retryQueuedMessage(
        for error: Error,
        fallback: String = "Saved for retry. We will retry shortly."
    ) -> String {
        switch classify(error) {
        case .restricted:
            return "Saved for retry. Account access is restricted right now."
        case .offline:
            return "Saved for retry. Offline mode active; reconnect to complete payment."
        case .degraded:
            return fallback
        }
    }
}
