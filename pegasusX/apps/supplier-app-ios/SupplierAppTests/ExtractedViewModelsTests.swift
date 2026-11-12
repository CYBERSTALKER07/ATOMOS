import XCTest
@testable import SupplierAppIOS

@MainActor
final class ExtractedViewModelsTests: XCTestCase {
    private struct StubError: LocalizedError {
        let message: String
        var errorDescription: String? { message }
    }

    private final class Box<T>: @unchecked Sendable {
        var value: T
        init(_ value: T) { self.value = value }
    }

    func testCreditAdminDisableViewModelValidationAndRouting() async {
        let capturedPaths = Box<[String]>([])
        let capturedBodies = Box<[CreditAdminDisableBody]>([])

        let vm = CreditAdminDisableViewModel { path, body, _ in
            capturedPaths.value.append(path)
            capturedBodies.value.append(body)
            return CreditAdminDisableResponse(status: "disabled_ok", error: nil)
        }

        XCTAssertFalse(vm.canSubmit)
        vm.mode = "mobile_supplier.chrome.source_program"
        vm.supplierId = "sup_1"
        vm.ticketId = "TCK-100"
        vm.reason = "   "
        XCTAssertFalse(vm.canSubmit)

        vm.reason = "regulatory_freeze"
        XCTAssertTrue(vm.canSubmit)

        await vm.submit()
        XCTAssertEqual(capturedPaths.value, ["v1/admin/credit-program/sup_1/disable"])
        XCTAssertEqual(capturedBodies.value.first?.ticket_id, "TCK-100")
        XCTAssertEqual(capturedBodies.value.first?.reason, "regulatory_freeze")
        XCTAssertEqual(vm.message, "disabled_ok")

        vm.mode = "relationship"
        vm.retailerId = "   "
        XCTAssertFalse(vm.canSubmit)

        vm.retailerId = "ret_42"
        XCTAssertTrue(vm.canSubmit)

        await vm.submit()
        XCTAssertEqual(capturedPaths.value.last, "v1/admin/credit-relationships/sup_1/ret_42/disable")
    }

    func testCreditAdminDisableViewModelErrorHandling() async {
        let vm = CreditAdminDisableViewModel { _, _, _ in
            throw StubError(message: "permission_denied")
        }
        vm.mode = "mobile_supplier.chrome.source_program"
        vm.supplierId = "sup_1"
        vm.ticketId = "TCK-9"
        vm.reason = "audit_lock"
        await vm.submit()
        XCTAssertEqual(vm.message, "permission_denied")
        XCTAssertFalse(vm.busy)
    }

    func testJSONFeedViewModelLoadAndSilentRetryPreservesContent() async {
        let shouldFail = Box(false)
        let vm = JSONFeedViewModel { path, _ in
            if shouldFail.value {
                throw StubError(message: "upstream_502")
            }
            return #"{"path":"\#(path)","ok":true}"#
        }

        await vm.load(path: "v1/supplier/feed", query: ["limit": "10"])
        XCTAssertFalse(vm.loading)
        XCTAssertNil(vm.errorText)
        XCTAssertEqual(vm.bodyText, #"{"path":"v1/supplier/feed","ok":true}"#)

        shouldFail.value = true
        await vm.load(path: "v1/supplier/feed", query: [:], silent: true)
        XCTAssertEqual(vm.bodyText, #"{"path":"v1/supplier/feed","ok":true}"#)
        XCTAssertNil(vm.errorText)
        XCTAssertNotNil(SupplierStatusCenter.shared.notice)
        XCTAssertFalse(vm.loading)

        let freshVM = JSONFeedViewModel { _, _ in
            throw StubError(message: "upstream_502")
        }
        await freshVM.load(path: "v1/supplier/feed", query: [:], silent: false)
        XCTAssertEqual(freshVM.errorText, "upstream_502")
        XCTAssertFalse(freshVM.loading)
    }

    func testNotificationInboxViewModelLoadsAndMarksAllRead() async {
        let markAllReadCount = Box(0)
        let currentUnread = Box(2)

        let vm = NotificationInboxViewModel(
            pageFetcher: {
                let unread = currentUnread.value
                let payload = """
                {
                  "notifications": [
                    {
                      "notification_id": "notif_1",
                      "type": "DISPATCH",
                      "title": "Manifest Sealed",
                      "body": "Route 7 sealed",
                      "payload": "{}",
                      "channel": "IN_APP",
                      "created_at": "2026-11-12T10:00:00Z",
                      "read_at": \(unread == 0 ? "\"2026-11-12T10:05:00Z\"" : "null")
                    }
                  ],
                  "unread_count": \(unread),
                  "has_more": false
                }
                """
                return try JSONDecoder().decode(SupplierNotificationsPage.self, from: Data(payload.utf8))
            },
            markAllHandler: { _ in
                markAllReadCount.value += 1
                currentUnread.value = 0
            }
        )

        await vm.load()
        XCTAssertEqual(vm.items.count, 1)
        XCTAssertEqual(vm.unreadCount, 2)
        XCTAssertNil(vm.items[0].readAt)

        await vm.markAllRead()
        XCTAssertEqual(markAllReadCount.value, 1)
        XCTAssertEqual(vm.unreadCount, 0)
        XCTAssertNotNil(vm.items[0].readAt)
    }

    func testComplianceAuditViewModelDashboardAndReceiptOpening() async {
        var openedURLs: [URL] = []

        let vm = ComplianceAuditViewModel(
            loadDashboard: {
                let payload = """
                {
                  "summary": {
                    "open_fiscal_count": 3,
                    "force_complete_count": 1,
                    "claim_mismatch_count": 0,
                    "credit_freeze_count": 2,
                    "generated_at": "2026-11-12T12:00:00Z"
                  },
                  "open_fiscal": [
                    {"order_id": "ord_101", "status": "DELIVERED", "fiscal_status": "PENDING"}
                  ],
                  "force_completes": [],
                  "claim_mismatches": [],
                  "credit_freezes": []
                }
                """
                return try JSONDecoder().decode(ComplianceDashboardResponse.self, from: Data(payload.utf8))
            },
            fetchReceipt: { orderId in
                if orderId == "ord_404" {
                    throw APIError.httpError(404)
                }
                let payload = """
                {
                  "receipt_id": "rcpt_1",
                  "html_url": "",
                  "pdf_url": "https://ofd.soliq.uz/r/\(orderId).pdf",
                  "qr_url": "https://ofd.soliq.uz/q/\(orderId)",
                  "party_copy": "MERCHANT"
                }
                """
                return try JSONDecoder().decode(OrderReceiptMeta.self, from: Data(payload.utf8))
            },
            openURL: { url in
                openedURLs.append(url)
            }
        )

        await vm.load()
        XCTAssertFalse(vm.loading)
        XCTAssertNil(vm.error)
        XCTAssertEqual(vm.summary?.openFiscalCount, 3)
        XCTAssertEqual(vm.openFiscal.first?.orderId, "ord_101")

        await vm.openReceipt(orderId: "ord_101")
        XCTAssertEqual(openedURLs.map(\.absoluteString), ["https://ofd.soliq.uz/q/ord_101"])

        await vm.openReceipt(orderId: "ord_404")
        XCTAssertNil(vm.error)
        XCTAssertEqual(openedURLs.count, 1)
    }

    func testReturnPolicyBillingGateAndOrdersViewModelStateTransitions() {
        let returnVM = ReturnPolicySettingsViewModel()
        returnVM.sanitizeHoursText("72hrs!")
        XCTAssertEqual(returnVM.hoursText, "72")
        XCTAssertEqual(returnVM.hours, 72)

        returnVM.selectPreset(24)
        XCTAssertEqual(returnVM.hours, 24)
        XCTAssertEqual(returnVM.hoursText, "24")

        returnVM.sanitizeConcealed("12d")
        XCTAssertEqual(returnVM.concealed, "12")

        let billingVM = BillingGateViewModel()
        XCTAssertFalse(billingVM.canSubmit)
        billingVM.bankName = "Ipak Yuli"
        billingVM.accountHolder = "Pegasus LLC"
        billingVM.accountNumber = "20208000900000123456"
        billingVM.swiftBic = "IPKYUZ22"
        billingVM.gatewayOptions[0].isSelected = true
        XCTAssertEqual(billingVM.selectedGateways, ["CASH", "GLOBAL_PAY"])
        XCTAssertTrue(billingVM.canSubmit)

        let ordersVM = OrdersViewModel()
        XCTAssertFalse(ordersVM.showReassignSheet)
        ordersVM.reassignTarget = "ord_99"
        XCTAssertTrue(ordersVM.showReassignSheet)
        ordersVM.showReassignSheet = false
        XCTAssertNil(ordersVM.reassignTarget)

        ordersVM.reassignMessage = "Reassigned to driver_2"
        XCTAssertTrue(ordersVM.showReassignMessageAlert)
        ordersVM.showReassignMessageAlert = false
        XCTAssertNil(ordersVM.reassignMessage)
    }
}
