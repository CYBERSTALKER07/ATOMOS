import Foundation
import Testing
@testable import retailerapp

@Suite("Retailer W3 Credit & AR Self-Service")
struct RetailerW3Tests {

    @Test func w3Helpers_sortInvoicesOldestFirst_andConvertSomToMinorSafely() {
        let sorted = RetailerW3ContractHelper.sortInvoicesOldestFirst([
            ArInvoiceWire(invoiceId: "inv-new", supplierId: "sup-1", status: "OPEN", balanceMinor: 40_000_000, dueAt: "2026-12-01T00:00:00Z"),
            ArInvoiceWire(invoiceId: "inv-old", supplierId: "sup-1", status: "PARTIAL", balanceMinor: 20_000_000, dueAt: "2026-10-05T00:00:00Z")
        ])
        #expect(sorted.map(\.invoiceId) == ["inv-old", "inv-new"])

        #expect(RetailerMoney.somInputToMinor("500000") == 50_000_000)
        #expect(RetailerMoney.minorToSomInput(50_000_000) == "500000")
        #expect(RetailerMoney.formatWholeCurrency(50_000_000) == RetailerMoney.formatWholeSom(50_000_000))
    }

    @Test func w3Decoder_parsesStatementAnd409QuarterlyPromiseConflict() throws {
        let json = """
        {
          "supplier_id": "sup-1",
          "currency": "UZS",
          "total_open_minor": 60000000,
          "total_disputed_minor": 10000000,
          "total_undisputed_minor": 50000000,
          "advance_balance_minor": 5000000,
          "current_quarter_key": "2026-Q4",
          "self_serve_promise_used": true,
          "max_extension_days": 14,
          "invoices": [
            {
              "invoice_id": "inv-1",
              "supplier_id": "sup-1",
              "order_id": "ORD-1",
              "status": "PARTIAL",
              "balance_minor": 20000000,
              "disputed_minor": 10000000,
              "undisputed_minor": 10000000,
              "currency": "UZS",
              "due_at": "2026-10-05T00:00:00Z",
              "aging_bucket": "15_30_DAYS",
              "dunning_step": 1
            }
          ],
          "promises": [],
          "ledger_entries": []
        }
        """.data(using: .utf8)!
        let statement = try JSONDecoder().decode(ArStatementWire.self, from: json)
        #expect(statement.advanceBalanceMinor == 5_000_000)
        #expect(statement.totalDisputedMinor == 10_000_000)
        #expect(statement.selfServePromiseUsed == true)
        #expect(statement.invoices.first?.effectiveUndisputedMinor == 10_000_000)

        let conflictErr = APIError.serverError(
            statusCode: 409,
            message: #"{"code":"quarterly_promise_limit_reached","error":"quarterly_promise_limit_reached","quarter_key":"2026-Q4","message":"Limit reached"}"#
        )
        let parsed = RetailerW3ContractHelper.parseQuarterlyPromiseLimitError(from: conflictErr)
        #expect(parsed?.code == "quarterly_promise_limit_reached")
        #expect(parsed?.quarterKey == "2026-Q4")
    }

    @Test @MainActor func creditPartnersViewModel_loadsStatementSubmitsFifoAndHandles409() async {
        final class CaptureBox {
            var recordedStatus: String?
            var fifoPayloads: [(PayArFifoRequest, String)] = []
            var invoicePayloads: [(String, PayArInvoiceRequest, String)] = []
        }
        let box = CaptureBox()

        let service = CreditPartnersViewModel.Service(
            fetchRelationships: {
                [
                    CreditRelationshipWire(
                        supplierId: "sup-1",
                        supplierName: "Atlas Dist",
                        creditEnabled: true,
                        termsDays: 14,
                        gracePeriodDays: 3,
                        creditLimitMinor: 100_000_000,
                        availableCreditMinor: 40_000_000,
                        currentBalanceMinor: 60_000_000,
                        onHold: false,
                        prepayOnlyLocked: false
                    )
                ]
            },
            fetchStatement: { supplierId, _ in
                ArStatementWire(
                    supplierId: supplierId ?? "sup-1",
                    currency: "UZS",
                    totalOpenMinor: 60_000_000,
                    totalDisputedMinor: 10_000_000,
                    totalUndisputedMinor: 50_000_000,
                    advanceBalanceMinor: 5_000_000,
                    currentQuarterKey: "2026-Q4",
                    selfServePromiseUsed: false,
                    maxExtensionDays: 14,
                    invoices: [
                        ArInvoiceWire(invoiceId: "inv-2", supplierId: "sup-1", orderId: "ORD-2", status: "OPEN", balanceMinor: 40_000_000, dueAt: "2026-11-25T00:00:00Z"),
                        ArInvoiceWire(invoiceId: "inv-1", supplierId: "sup-1", orderId: "ORD-1", status: "PARTIAL", balanceMinor: 20_000_000, disputedMinor: 10_000_000, undisputedMinor: 10_000_000, dueAt: "2026-10-05T00:00:00Z")
                    ],
                    promises: [],
                    ledgerEntries: []
                )
            },
            fetchInvoices: { status, _ in
                box.recordedStatus = status
                return []
            },
            fetchPromises: { _ in
                ArPromisesResponse(promises: [], currentQuarterKey: "2026-Q4", selfServePromiseUsed: false, maxExtensionDays: 14)
            },
            submitPayFifo: { req, key in
                box.fifoPayloads.append((req, key))
                return PayArFifoResponse(paidMinor: req.amountMinor, advanceAddedMinor: 2_000_000, advanceBalanceMinor: 7_000_000, allocations: [])
            },
            submitPayInvoice: { invId, req, key in
                box.invoicePayloads.append((invId, req, key))
            },
            submitPromise: { _, _ in
                throw APIError.serverError(
                    statusCode: 409,
                    message: #"{"code":"quarterly_promise_limit_reached","error":"quarterly_promise_limit_reached","quarter_key":"2026-Q4","message":"Already used"}"#
                )
            }
        )

        let vm = CreditPartnersViewModel(service: service)
        await vm.load()

        #expect(vm.selectedSupplierId == "sup-1")
        #expect(box.recordedStatus == "OPEN,PARTIAL")
        #expect(vm.invoices.map(\.invoiceId) == ["inv-1", "inv-2"])
        #expect(vm.advanceBalanceMinor == 5_000_000)
        #expect(vm.totalDisputedMinor == 10_000_000)
        #expect(vm.totalUndisputedMinor == 50_000_000)

        let fifoOk = await vm.payFifo(supplierId: "sup-1", amountMinor: 30_000_000, paymentMethod: "WALLET", idempotencyKey: "11111111-2222-4333-8444-555555555555")
        #expect(fifoOk == true)
        #expect(box.fifoPayloads.count == 1)
        #expect(box.fifoPayloads[0].0.amountMinor == 30_000_000)
        #expect(box.fifoPayloads[0].1 == "11111111-2222-4333-8444-555555555555")

        let invOk = await vm.payInvoice(invoiceId: "inv-1", amountMinor: 10_000_000, paymentMethod: "CARD", idempotencyKey: "22222222-3333-4444-8555-666666666666")
        #expect(invOk == true)
        #expect(box.invoicePayloads.count == 1)
        #expect(box.invoicePayloads[0].0 == "inv-1")

        let promOk = await vm.createPromiseExtension(supplierId: "sup-1", extensionDays: 7)
        #expect(promOk == false)
        #expect(vm.quarterlyLimitError?.code == "quarterly_promise_limit_reached")
        #expect(vm.quarterlyLimitError?.quarterKey == "2026-Q4")
    }

    @Test func migrationOrderOriginAndJoinConflictAndIdempotencyKeys() throws {
        let migrationOrderJSON = """
        {
          "order_id": "ord-mig-1",
          "retailer_id": "ret-1",
          "supplier_id": "sup-1",
          "supplier_name": "Navruz Food",
          "state": "CONFIRMED",
          "items": [],
          "amount": 150000,
          "currency": "UZS",
          "order_source": "MIGRATION",
          "version": 1
        }
        """.data(using: .utf8)!
        let decodedOrder = try JSONDecoder().decode(Order.self, from: migrationOrderJSON)
        #expect(decodedOrder.orderSource == "MIGRATION")
        #expect(decodedOrder.isMigrationOrigin == true)

        let conflictErr = APIError.serverError(
            statusCode: 409,
            message: #"{"error":"join_request_pending","request_id":"jr-42","target_tenant_id":"ret-1","legal_name":"Baraka Market","normalized_inn":"301234567"}"#
        )
        let parsedConflict = RetailerW3ContractHelper.parseJoinRequestConflict(from: conflictErr)
        #expect(parsedConflict?.requestId == "jr-42")
        #expect(parsedConflict?.targetTenantId == "ret-1")
        #expect(parsedConflict?.legalName == "Baraka Market")
        #expect(parsedConflict?.normalizedInn == "301234567")

        #expect(RetailerIdempotency.nasiyaAcknowledge(invoiceId: "inv-9") == "nasiya-opening-ack:inv-9")
        #expect(RetailerIdempotency.nasiyaDispute(invoiceId: "inv-9", reason: "wrong qty") == RetailerIdempotency.nasiyaDispute(invoiceId: "inv-9", reason: "wrong qty"))
        #expect(RetailerIdempotency.nasiyaDispute(invoiceId: "inv-9", reason: "wrong qty") != RetailerIdempotency.nasiyaDispute(invoiceId: "inv-9", reason: "other"))
    }

    @Test @MainActor func onboardingViewModel_failsClosedOnInnUnavailable_allowsInformalShop_andHandles409JoinRequest() async {
        // 1. INN UNAVAILABLE fails closed
        let unavailableVM = OnboardingViewModel(
            lookupINN: { inn in
                RetailerINNLookupResponse(
                    status: "UNAVAILABLE",
                    normalizedInn: inn,
                    entityType: nil,
                    legalName: nil,
                    address: nil,
                    registryState: nil,
                    cached: nil,
                    existingTenantId: nil,
                    existingTenantType: nil,
                    existingLegalName: nil,
                    existingJoinEligible: nil
                )
            },
            setupRetailer: { _, _ in Issue.record("Should not submit when INN unavailable") } as ([String: AnyEncodable], String) async throws -> Void,
            currentRetailerId: { "ret-u1" },
            onConfigured: { Issue.record("Should not mark configured when INN unavailable") }
        )
        unavailableVM.taxId = "301234567"
        let advanced = await unavailableVM.advanceFromTaxAsync()
        #expect(advanced == false)
        #expect(unavailableVM.step == .tax)
        #expect(unavailableVM.innError != nil)

        // 2. Informal corner shop skips INN check and persists informal_shop = true
        final class SetupCapture {
            var submittedPayloads: [(String, Bool)] = []
            var configuredCount = 0
        }
        let capture = SetupCapture()
        let informalVM = OnboardingViewModel(
            lookupINN: { _ in
                Issue.record("Should not lookup INN in informal shop mode")
                throw APIError.invalidResponse
            },
            setupRetailer: { payload, _ in
                let data = try JSONEncoder().encode(payload)
                let obj = try JSONSerialization.jsonObject(with: data) as? [String: Any]
                let taxId = obj?["tax_id"] as? String ?? ""
                let informal = obj?["informal_shop"] as? Bool ?? false
                capture.submittedPayloads.append((taxId, informal))
            },
            currentRetailerId: { "ret-inf-1" },
            onConfigured: { capture.configuredCount += 1 }
        )
        informalVM.isInformalShop = true
        let informalAdvanced = await informalVM.advanceFromTaxAsync()
        #expect(informalAdvanced == true)
        #expect(informalVM.step == .address)
        informalVM.billingAddress = "Chorsu 12, Tashkent"
        informalVM.shippingAddress = "Chorsu 12, Tashkent"
        informalVM.city = "Tashkent"
        let informalOK = await informalVM.submit()
        #expect(informalOK == true)
        #expect(capture.configuredCount == 1)
        #expect(capture.submittedPayloads.count == 1)
        #expect(capture.submittedPayloads[0].0 == "")
        #expect(capture.submittedPayloads[0].1 == true)

        // 3. Duplicate INN 409 join_request_pending blocks markConfigured and shows joinRequestSent
        var joinConfiguredCount = 0
        let joinVM = OnboardingViewModel(
            lookupINN: { inn in
                RetailerINNLookupResponse(
                    status: "FOUND",
                    normalizedInn: inn,
                    entityType: "OOO",
                    legalName: "Baraka Savdo",
                    address: "Tashkent",
                    registryState: "ACTIVE",
                    cached: true,
                    existingTenantId: "ret-existing-1",
                    existingTenantType: "RETAILER",
                    existingLegalName: "Baraka Savdo",
                    existingJoinEligible: true
                )
            },
            setupRetailer: { _, _ in
                throw APIError.serverError(
                    statusCode: 409,
                    message: #"{"error":"join_request_pending","request_id":"jr-77","target_tenant_id":"ret-existing-1","legal_name":"Baraka Savdo","normalized_inn":"301234567"}"#
                )
            },
            currentRetailerId: { "ret-join-1" },
            onConfigured: { joinConfiguredCount += 1 }
        )
        joinVM.taxId = "301234567"
        #expect(await joinVM.advanceFromTaxAsync() == true)
        joinVM.billingAddress = "Amir Temur 10"
        joinVM.shippingAddress = "Amir Temur 10"
        joinVM.city = "Tashkent"
        let submitResult = await joinVM.submit()
        #expect(submitResult == false)
        #expect(joinVM.joinRequestSent == true)
        #expect(joinVM.joinRequestConflict?.requestId == "jr-77")
        #expect(joinConfiguredCount == 0)
    }

    @Test @MainActor func creditPartnersViewModel_loadsNasiyaOpeningBalances_confirmsAndDisputesWithIdempotency() async {
        final class OpeningCapture {
            var ackCalls: [(String, String)] = []
            var disputeCalls: [(String, String, String)] = []
        }
        let capture = OpeningCapture()

        let service = CreditPartnersViewModel.Service(
            fetchRelationships: { [] },
            fetchStatement: { _, _ in
                ArStatementWire(
                    supplierId: "",
                    currency: "UZS",
                    totalOpenMinor: 0,
                    totalDisputedMinor: 0,
                    totalUndisputedMinor: 0,
                    advanceBalanceMinor: 0,
                    currentQuarterKey: "",
                    selfServePromiseUsed: false,
                    maxExtensionDays: 14,
                    invoices: [],
                    promises: [],
                    ledgerEntries: []
                )
            },
            fetchInvoices: { _, _ in [] },
            fetchPromises: { _ in
                ArPromisesResponse(promises: [], currentQuarterKey: "", selfServePromiseUsed: false, maxExtensionDays: 14)
            },
            submitPayFifo: { req, _ in
                PayArFifoResponse(paidMinor: req.amountMinor, advanceAddedMinor: 0, advanceBalanceMinor: 0, allocations: [])
            },
            submitPayInvoice: { _, _, _ in },
            submitPromise: { _, _ in
                throw APIError.invalidResponse
            },
            fetchOpeningBalances: {
                [
                    MigrationOpeningInvoiceWire(
                        invoiceId: "inv-open-1",
                        supplierId: "sup-1",
                        supplierName: "Navruz Food",
                        orderId: nil,
                        externalId: "1C-AR-100",
                        principalMinor: 250_000_000,
                        balanceMinor: 250_000_000,
                        currency: "UZS",
                        status: "PENDING_ACK",
                        source: "MIGRATION",
                        dunningFrozen: true,
                        dueAt: "2026-12-01T00:00:00Z"
                    ),
                    MigrationOpeningInvoiceWire(
                        invoiceId: "inv-open-2",
                        supplierId: "sup-2",
                        supplierName: "Orient Trade",
                        externalId: "1C-AR-200",
                        principalMinor: 120_000_000,
                        balanceMinor: 120_000_000,
                        currency: "UZS",
                        status: "PENDING_ACK",
                        source: "MIGRATION",
                        dunningFrozen: true
                    )
                ]
            },
            submitAcknowledgeOpeningBalance: { id, key in
                capture.ackCalls.append((id, key))
                return MigrationOpeningInvoiceWire(
                    invoiceId: id,
                    supplierId: "sup-1",
                    supplierName: "Navruz Food",
                    externalId: "1C-AR-100",
                    principalMinor: 250_000_000,
                    balanceMinor: 250_000_000,
                    currency: "UZS",
                    status: "ACKNOWLEDGED",
                    source: "MIGRATION",
                    dunningFrozen: false
                )
            },
            submitDisputeOpeningBalance: { id, reason, key in
                capture.disputeCalls.append((id, reason, key))
                return MigrationOpeningInvoiceWire(
                    invoiceId: id,
                    supplierId: "sup-2",
                    supplierName: "Orient Trade",
                    externalId: "1C-AR-200",
                    principalMinor: 120_000_000,
                    balanceMinor: 120_000_000,
                    currency: "UZS",
                    status: "DISPUTED",
                    source: "MIGRATION",
                    dunningFrozen: true,
                    disputeReason: reason
                )
            }
        )

        let vm = CreditPartnersViewModel(service: service)
        await vm.load()

        #expect(vm.openingBalances.count == 2)
        #expect(vm.openingBalances[0].dunningFrozen == true)

        let ackOK = await vm.confirmOpeningBalance(invoiceId: "inv-open-1")
        #expect(ackOK == true)
        #expect(capture.ackCalls.count == 1)
        #expect(capture.ackCalls[0].1 == "nasiya-opening-ack:inv-open-1")
        #expect(vm.openingBalances.first(where: { $0.invoiceId == "inv-open-1" })?.isAcknowledged == true)
        #expect(vm.openingBalances.first(where: { $0.invoiceId == "inv-open-1" })?.dunningFrozen == false)

        let emptyDisputeOK = await vm.disputeOpeningBalance(invoiceId: "inv-open-2", reason: "   ")
        #expect(emptyDisputeOK == false)
        #expect(capture.disputeCalls.isEmpty)

        let disputeOK = await vm.disputeOpeningBalance(invoiceId: "inv-open-2", reason: "Paid cash in October")
        #expect(disputeOK == true)
        #expect(capture.disputeCalls.count == 1)
        #expect(capture.disputeCalls[0].1 == "Paid cash in October")
        #expect(capture.disputeCalls[0].2 == RetailerIdempotency.nasiyaDispute(invoiceId: "inv-open-2", reason: "Paid cash in October"))
        #expect(vm.openingBalances.first(where: { $0.invoiceId == "inv-open-2" })?.isDisputed == true)
        #expect(vm.openingBalances.first(where: { $0.invoiceId == "inv-open-2" })?.disputeReason == "Paid cash in October")
    }

    @Test @MainActor func teamViewModel_loadsApprovesAndRejectsJoinRequests() async {
        final class Capture {
            var approved: [(String, String?, String)] = []
            var rejected: [(String, String?, String)] = []
            var memberLoads = 0
        }
        let capture = Capture()
        let req1 = RetailerJoinRequestItem(
            requestId: "jr-ios-1",
            homeCell: "uz-01",
            targetTenantType: "RETAILER",
            targetTenantId: "ret-1",
            normalizedInn: "301234567",
            requesterPhone: "+998901112233",
            requesterName: "Dilshod Store",
            requesterEmail: nil,
            roleSought: "RETAILER",
            status: "PENDING",
            decidedBy: nil,
            decidedAt: nil,
            decisionReason: nil,
            createdAt: nil
        )
        let req2 = RetailerJoinRequestItem(
            requestId: "jr-ios-2",
            homeCell: "uz-01",
            targetTenantType: "RETAILER",
            targetTenantId: "ret-1",
            normalizedInn: "301234567",
            requesterPhone: "+998904445566",
            requesterName: "Unknown Caller",
            requesterEmail: nil,
            roleSought: "CASHIER",
            status: "PENDING",
            decidedBy: nil,
            decidedAt: nil,
            decisionReason: nil,
            createdAt: nil
        )

        let vm = TeamViewModel(
            getOrgMembers: {
                capture.memberLoads += 1
                return RetailerOrgMembersResponse(retailerId: "ret-1", items: [], updatedAt: nil)
            },
            listJoinRequests: { _ in [req1, req2] },
            approveJoinRequest: { id, reason, key in
                capture.approved.append((id, reason, key))
                return req1
            },
            rejectJoinRequest: { id, reason, key in
                capture.rejected.append((id, reason, key))
                return req2
            }
        )

        await vm.load()
        #expect(vm.joinRequests.count == 2)
        #expect(capture.memberLoads == 1)

        let okApprove = await vm.approveJoinRequest(requestId: "jr-ios-1")
        #expect(okApprove == true)
        #expect(capture.approved.count == 1)
        #expect(capture.approved[0].2 == "retailer-join-decide:jr-ios-1:approve")
        #expect(capture.memberLoads == 2)
        #expect(vm.joinRequests.map { $0.requestId } == ["jr-ios-2"])

        let okReject = await vm.rejectJoinRequest(requestId: "jr-ios-2", reason: "Not partner")
        #expect(okReject == true)
        #expect(capture.rejected.count == 1)
        #expect(capture.rejected[0].2 == "retailer-join-decide:jr-ios-2:reject")
        #expect(vm.joinRequests.isEmpty)
    }
}
