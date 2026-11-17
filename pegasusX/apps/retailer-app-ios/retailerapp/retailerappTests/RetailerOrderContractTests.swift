import Foundation
import SwiftData
import Testing
@testable import retailerapp

@MainActor
struct RetailerOrderContractTests {

    private func makePackProduct(
        id: String = "prod-1",
        variantId: String = "sku-1",
        name: String = "Whole Milk",
        packCount: Int = 6,
        priceMinor: Int64 = 12_000
    ) -> (Product, Variant) {
        let variant = Variant(
            id: variantId,
            size: "1L",
            pack: "Case \(packCount)",
            packCount: packCount,
            weightPerUnit: "1L",
            priceMinor: priceMinor
        )
        let product = Product(
            id: id,
            name: name,
            description: "",
            nutrition: "",
            imageURL: nil,
            variants: [variant],
            supplierID: "sup-1",
            sellByBlock: false,
            unitsPerBlock: nil,
            priceMinor: priceMinor
        )
        return (product, variant)
    }

    private func makeInMemoryModelContext() throws -> ModelContext {
        let config = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(for: PendingOrder.self, configurations: config)
        return ModelContext(container)
    }

    // MARK: - Pack Snapping & Integer So'm Formatting

    @Test func packSnapping_handlesNearestFloorAndCeil() {
        #expect(RetailerPackHelper.snapToPackMultiple(8, packSize: 6, mode: .nearest) == 6)
        #expect(RetailerPackHelper.snapToPackMultiple(10, packSize: 6, mode: .nearest) == 12)
        #expect(RetailerPackHelper.snapToPackMultiple(9, packSize: 6, mode: .nearest) == 12)
        #expect(RetailerPackHelper.snapToPackMultiple(8, packSize: 6, mode: .floor) == 6)
        #expect(RetailerPackHelper.snapToPackMultiple(2, packSize: 6, mode: .floor) == 6)
        #expect(RetailerPackHelper.snapToPackMultiple(8, packSize: 6, mode: .ceil) == 12)
        #expect(RetailerPackHelper.snapToPackMultiple(12, packSize: 6, mode: .nearest) == 12)
    }

    @Test func priceGapFormatting_formatsRoundedIntegerSomNeverRawTiyinOrMinusZero() {
        #expect(RetailerMoney.formatPriceGapSom(0) == "0 so'm")
        #expect(RetailerMoney.formatPriceGapSom(15_000) == "+150 so'm")
        #expect(RetailerMoney.formatPriceGapSom(-25_000) == "-250 so'm")
        #expect(RetailerMoney.formatPriceGapSom(150) == "+2 so'm")
        #expect(RetailerMoney.formatPriceGapSom(-40) == "0 so'm")
        #expect(RetailerMoney.formatWholeSom(125_000) == "1 250 so'm")
    }

    @Test func orderAndPayloadCoding_preservesFlagsSubstituteGapAndAutoConfirm() throws {
        let cart = CartManager()
        let (product, variant) = makePackProduct(packCount: 6)
        cart.add(product: product, variant: variant, quantity: 6, mustHave: true, allowSubstitutes: false)
        cart.toggleAllowSubstitutes(itemId: "\(product.id)-\(variant.id)")

        let payload = cart.buildCheckoutPayload(
            retailerId: "ret-1",
            paymentGateway: "CASH",
            autoConfirmOptIn: true
        )
        let encoded = try JSONEncoder().encode(payload)
        let jsonString = try #require(String(data: encoded, encoding: .utf8))
        #expect(jsonString.contains(#""must_have":true"#))
        #expect(jsonString.contains(#""allow_substitutes":true"#))
        #expect(jsonString.contains(#""auto_confirm_opt_in":true"#))

        let orderJson = Data(#"""
        {
          "order_id": "ord-99",
          "retailer_id": "ret-1",
          "state": "SCHEDULED",
          "amount": 180000,
          "auto_confirm_opt_in": true,
          "items": [
            {
              "id": "li-1",
              "product_id": "prod-1",
              "product_name": "Milk 1L",
              "variant_id": "sku-1",
              "quantity": 6,
              "unit_price": 12000,
              "total_price": 72000,
              "must_have": true,
              "allow_substitutes": true,
              "substitute_sku": "sku-sub",
              "substitute_name": "Milk Gold 1L",
              "original_sku": "sku-1",
              "price_gap_minor": 18000,
              "shortage_action": "SUBSTITUTE"
            }
          ]
        }
        """#.utf8)
        let order = try JSONDecoder().decode(Order.self, from: orderJson)
        #expect(order.autoConfirmOptIn == true)
        let line = try #require(order.items.first)
        #expect(line.mustHave == true)
        #expect(line.allowSubstitutes == true)
        #expect(line.substituteSku == "sku-sub")
        #expect(line.substituteName == "Milk Gold 1L")
        #expect(line.priceGapMinor == 18_000)
    }

    // MARK: - PendingOrderReplayer: 409 Variants, 429 & Multi-Endpoint Procurement

    @Test func replayer_preserves409OwnerApprovalInProgressAnd429_andPurgesIdempotent409() async throws {
        let context = try makeInMemoryModelContext()

        let ownerApprovalOrder = PendingOrder(
            payloadJson: #"{"retailer_id":"r1","items":[{"product_id":"p1","quantity":6}]}"#,
            endpoint: "/v1/orders/edit-preorder",
            method: "POST",
            idempotencyKey: "k-owner"
        )
        let inProgressOrder = PendingOrder(
            payloadJson: #"{"retailer_id":"r1","items":[{"product_id":"p2","quantity":12}]}"#,
            endpoint: "/v1/order/create",
            method: "POST",
            idempotencyKey: "k-progress"
        )
        let rateLimitedOrder = PendingOrder(
            payloadJson: #"{"retailer_id":"r1","payment_gateway":"CASH","latitude":0,"longitude":0,"items":[{"sku_id":"s1","quantity":6,"unit_price":10000}]}"#,
            endpoint: "/v1/checkout/unified",
            method: "POST",
            idempotencyKey: "k-rate"
        )
        let duplicateOrder = PendingOrder(
            payloadJson: #"{"retailer_id":"r1","items":[{"product_id":"p3","quantity":6}]}"#,
            endpoint: "/v1/order/create",
            method: "POST",
            idempotencyKey: "k-dup"
        )
        context.insert(ownerApprovalOrder)
        context.insert(inProgressOrder)
        context.insert(rateLimitedOrder)
        context.insert(duplicateOrder)
        try context.save()

        var dispatchedEndpoints: [String] = []
        var notifiedTitles: [String] = []
        let ownerError = APIError.problemDetail(
            ProblemDetail(status: 409, detail: "Needs owner approval", code: "owner_approval_required")
        )
        let dupError = APIError.problemDetail(
            ProblemDetail(status: 409, detail: "Already processed", code: "already_created")
        )

        let result = await PendingOrderReplayer.flush(
            modelContext: context,
            dispatcher: { endpoint, _, key in
                dispatchedEndpoints.append(endpoint)
                switch key {
                case "k-owner":
                    throw ownerError
                case "k-progress":
                    throw APIError.serverError(
                        statusCode: 409,
                        message: #"{"code":"request_in_progress","detail":"Still running"}"#
                    )
                case "k-rate":
                    throw APIError.serverError(statusCode: 429, message: #"{"code":"rate_limited"}"#)
                case "k-dup":
                    throw dupError
                default:
                    break
                }
            },
            notifier: { title, _, _ in
                notifiedTitles.append(title)
            }
        )

        #expect(dispatchedEndpoints.contains("/v1/orders/edit-preorder"))
        #expect(dispatchedEndpoints.contains("/v1/order/create"))
        #expect(dispatchedEndpoints.contains("/v1/checkout/unified"))
        #expect(result.parkedCount == 3)
        #expect(result.completedCount == 1)
        #expect(notifiedTitles.count == 3)

        let remaining = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(remaining.count == 3)
        let remainingByKey = Dictionary(uniqueKeysWithValues: remaining.map { ($0.idempotencyKey, $0) })
        #expect(remainingByKey["k-owner"]?.status == PendingOrder.statusAwaitingOwnerApproval)
        #expect(remainingByKey["k-progress"]?.status == PendingOrder.statusInProgress)
        #expect(remainingByKey["k-rate"]?.status == PendingOrder.statusRateLimited)
        #expect(remainingByKey["k-dup"] == nil)
    }

    // MARK: - CheckoutViewModel & ProcurementViewModel Contract Behavior

    @Test func checkoutViewModel_handlesPackMultipleRequired_andQueuesOfflineFallback() async throws {
        let context = try makeInMemoryModelContext()
        let cart = CartManager()
        let (product, variant) = makePackProduct(variantId: "sku-pack", name: "Ayran 1L", packCount: 6, priceMinor: 15_000)
        cart.add(product: product, variant: variant, quantity: 8, mustHave: true, allowSubstitutes: false)

        let previewOk = try JSONDecoder().decode(
            CheckoutPreviewResponse.self,
            from: Data(#"{"ok":true,"stock_warnings":[]}"#.utf8)
        )
        var mode = "PACK"
        let vm = CheckoutViewModel(
            previewFetcher: { _ in previewOk },
            checkoutSender: { _, _ in
                if mode == "PACK" {
                    throw APIError.problemDetail(
                        ProblemDetail(
                            status: 422,
                            detail: "Quantity must match pack size",
                            code: "PACK_MULTIPLE_REQUIRED",
                            skuId: "sku-pack",
                            packSize: 6,
                            requested: 8,
                            suggestedFloor: 6,
                            suggestedCeil: 12,
                            suggestedNearest: 6
                        )
                    )
                }
                throw URLError(.notConnectedToInternet)
            }
        )
        vm.autoConfirmOptIn = true

        await vm.submitOrder(cart: cart, modelContext: context)
        #expect(vm.packSuggestions.count == 1)
        #expect(vm.packSuggestions.first?.suggestedNearest == 6)
        #expect(try context.fetch(FetchDescriptor<PendingOrder>()).isEmpty)

        vm.applyPackSuggestion(cart: cart, skuId: "sku-pack", suggestedQuantity: 6)
        #expect(vm.packSuggestions.isEmpty)
        #expect(cart.items.first?.quantity == 6)

        mode = "OFFLINE"
        await vm.submitOrder(cart: cart, modelContext: context)
        let queued = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(queued.count == 1)
        #expect(queued.first?.endpoint == "/v1/checkout/unified")
        #expect(queued.first?.status == PendingOrder.statusPending)
        let queuedPayload = try JSONDecoder().decode(
            UnifiedCheckoutPayload.self,
            from: Data((queued.first?.payloadJson ?? "").utf8)
        )
        #expect(queuedPayload.autoConfirmOptIn == true)
        #expect(queuedPayload.items.first?.mustHave == true)
        #expect(queuedPayload.items.first?.quantity == 6)
    }

    @Test func procurementViewModel_appliesPackFixAndQueues409OwnerApprovalOffline() async throws {
        let context = try makeInMemoryModelContext()
        let (product, _) = makePackProduct(id: "prod-flour", variantId: "sku-flour", name: "Flour", packCount: 6)
        var mode = "PACK"
        let vm = ProcurementViewModel(
            orderSender: { _, _ in
                if mode == "PACK" {
                    throw APIError.serverError(
                        statusCode: 422,
                        message: #"{"code":"PACK_MULTIPLE_REQUIRED","sku_id":"prod-flour","pack_size":6,"requested":10,"suggested_floor":6,"suggested_ceil":12,"suggested_nearest":12}"#
                    )
                }
                throw APIError.problemDetail(
                    ProblemDetail(status: 409, detail: "Owner approval pending", code: "owner_approval_required")
                )
            }
        )
        vm.products = [product]
        let forecast = DemandForecast(
            id: "fc-flour",
            productId: "prod-flour",
            productName: "Flour",
            predictedQuantity: 10,
            confidence: 0.92,
            reasoning: "Weekend spike",
            suggestedOrderDate: "2026-11-14",
            blocked: false,
            blockedReason: nil,
            label: nil
        )
        vm.toggleSelection(for: forecast)
        vm.toggleMustHave(for: forecast.id)
        vm.toggleAllowSubstitutes(for: forecast.id)
        vm.autoConfirmOptIn = true

        await vm.createOrder(modelContext: context, refreshCenter: RetailerRefreshCenter.shared)
        #expect(vm.packSuggestions.count == 1)
        #expect(vm.packSuggestions.first?.suggestedNearest == 12)
        #expect(try context.fetch(FetchDescriptor<PendingOrder>()).isEmpty)

        vm.applyPackSuggestion(skuId: "prod-flour", suggestedQuantity: 12)
        #expect(vm.packSuggestions.isEmpty)
        #expect(vm.quantities[forecast.id] == 12)

        mode = "OWNER_409"
        await vm.createOrder(modelContext: context, refreshCenter: RetailerRefreshCenter.shared)
        let queued = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(queued.count == 1)
        #expect(queued.first?.endpoint == "/v1/order/create")
        #expect(queued.first?.status == PendingOrder.statusAwaitingOwnerApproval)
        let decoded = try JSONDecoder().decode(
            ProcurementOrderRequest.self,
            from: Data((queued.first?.payloadJson ?? "").utf8)
        )
        #expect(decoded.autoConfirmOptIn == true)
        #expect(decoded.items.first?.mustHave == true)
        #expect(decoded.items.first?.allowSubstitutes == true)
        #expect(decoded.items.first?.quantity == 12)
    }

    // MARK: - Workstream W2: Claims formatting/errors, 409 Owner Approval + Fallback, Owner Governance

    @Test func retailerMoney_formatsWholeSomWithoutRawTiyin_andConvertsMandateInputs() {
        #expect(RetailerMoney.formatWholeCurrency(125_000) == "1 250 so'm")
        #expect(RetailerMoney.formatWholeCurrency(125_060) == "1 251 so'm")
        #expect(!RetailerMoney.formatWholeCurrency(125_000).contains("125000"))
        #expect(RetailerMoney.formatWholeCurrency(50_000, currency: "USD").hasSuffix("USD"))

        #expect(RetailerMoney.somInputToMinor("1 500") == 150_000)
        #expect(RetailerMoney.somInputToMinor("0") == 0)
        #expect(RetailerMoney.somInputToMinor("-5") == nil)
        #expect(RetailerMoney.minorToSomInput(150_000) == "1500")
        #expect(RetailerMoney.minorToSomInput(0) == "0")
    }

    @Test func retailerW2ErrorParser_mapsReusedPhotoError_and409OwnerApprovalRequired() {
        let reusedErr = APIError.serverError(
            statusCode: 422,
            message: #"{"code":"reused_photo_evidence_refused","message":"Photo hash already used on claim clm-1"}"#
        )
        let claimMsg = RetailerW2ErrorParser.userFriendlyClaimError(for: reusedErr)
        #expect(!claimMsg.contains("{"))
        #expect(claimMsg == String(localized: "mobile_retailer.claims.error.reused_photo"))

        let approvalJson = #"{"code":"owner_approval_required","error":"owner_approval_required","message":"Credit cap exceeded","request_id":"req-9","client_order_id":"ord-9","status":"PENDING_OWNER","tripped_rule":"DAILY_CREDIT_CAP_EXCEEDED","amount_minor":250000,"threshold_minor":200000,"remaining_cap_minor":50000,"fallback_payment_methods":["CASH","CARD"]}"#
        let parsed = RetailerW2ErrorParser.parseOwnerApprovalRequired(
            from: APIError.serverError(statusCode: 409, message: approvalJson)
        )
        #expect(parsed?.requestId == "req-9")
        #expect(parsed?.trippedRule == "DAILY_CREDIT_CAP_EXCEEDED")
        #expect(parsed?.amountMinor == 250_000)
        #expect(parsed?.formattedAmount == "2 500 so'm")
        #expect(parsed?.formattedThreshold == "2 000 so'm")
        #expect(parsed?.formattedRemainingCap == "500 so'm")
        #expect(parsed?.allowsCashFallback == true)
        #expect(parsed?.allowsCardFallback == true)
    }

    @Test func checkoutViewModel_handles409OwnerApproval_andRetriesCashFallback() async throws {
        let context = try makeInMemoryModelContext()
        let cart = CartManager()
        let (product, variant) = makePackProduct(variantId: "sku-w2", name: "Cheese 1kg", packCount: 1, priceMinor: 25_000)
        cart.add(product: product, variant: variant, quantity: 4, mustHave: false, allowSubstitutes: false)

        let previewOk = try JSONDecoder().decode(
            CheckoutPreviewResponse.self,
            from: Data(#"{"ok":true,"stock_warnings":[]}"#.utf8)
        )
        let checkoutOk = try JSONDecoder().decode(
            CheckoutResponse.self,
            from: Data(#"{"status":"CREATED","invoice_id":"inv-cash-1","total":100000}"#.utf8)
        )

        var capturedPayloads: [UnifiedCheckoutPayload] = []
        let vm = CheckoutViewModel(
            previewFetcher: { _ in previewOk },
            checkoutSender: { payload, _ in
                capturedPayloads.append(payload)
                if capturedPayloads.count == 1 {
                    let body = #"{"code":"owner_approval_required","request_id":"req-co-1","tripped_rule":"DAILY_CREDIT_CAP_EXCEEDED","amount_minor":100000,"threshold_minor":50000,"remaining_cap_minor":0,"fallback_payment_methods":["CASH","CARD"]}"#
                    throw APIError.serverError(statusCode: 409, message: body)
                }
                return checkoutOk
            }
        )

        await vm.submitOrder(cart: cart, modelContext: context)
        #expect(vm.ownerApprovalRequired?.requestId == "req-co-1")
        #expect(vm.ownerApprovalRequired?.allowsCashFallback == true)
        #expect(vm.showError == false)
        let parked = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(parked.count == 1)
        #expect(parked.first?.status == PendingOrder.statusAwaitingOwnerApproval)

        await vm.retryWithFallbackMethod("CASH", cart: cart, modelContext: context)
        #expect(capturedPayloads.count == 2)
        #expect(capturedPayloads.last?.paymentMethod == "CASH")
        #expect(capturedPayloads.last?.paymentGateway == "CASH")
        #expect(vm.ownerApprovalRequired == nil)
        #expect(vm.showSuccess == true)
        #expect(try context.fetch(FetchDescriptor<PendingOrder>()).isEmpty)
    }

    @Test func procurementViewModel_handles409OwnerApproval_andRetriesCardFallback() async throws {
        let context = try makeInMemoryModelContext()
        var capturedBodies: [ProcurementOrderRequest] = []
        let procOk = ProcurementOrderResponse(status: "CREATED", orderId: "ord-card-1", total: 120_000)

        let vm = ProcurementViewModel(
            orderSender: { body, _ in
                capturedBodies.append(body)
                if capturedBodies.count == 1 {
                    let json = #"{"code":"owner_approval_required","request_id":"req-pr-1","tripped_rule":"MAX_SINGLE_ORDER_EXCEEDED","amount_minor":120000,"threshold_minor":100000,"fallback_payment_methods":["CARD"]}"#
                    throw APIError.serverError(statusCode: 409, message: json)
                }
                return procOk
            }
        )
        let forecast = DemandForecast(
            id: "fc-tea",
            productId: "prod-tea",
            productName: "Green Tea",
            predictedQuantity: 6,
            confidence: 0.9,
            reasoning: "Restock",
            suggestedOrderDate: "2026-11-14",
            blocked: false,
            blockedReason: nil,
            label: nil
        )
        vm.toggleSelection(for: forecast)

        await vm.createOrder(modelContext: context, refreshCenter: RetailerRefreshCenter.shared)
        #expect(vm.ownerApprovalRequired?.requestId == "req-pr-1")
        #expect(vm.ownerApprovalRequired?.allowsCardFallback == true)
        #expect(vm.showError == false)
        let parked = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(parked.count == 1)
        #expect(parked.first?.status == PendingOrder.statusAwaitingOwnerApproval)

        await vm.retryWithFallbackMethod("CARD", modelContext: context, refreshCenter: RetailerRefreshCenter.shared)
        #expect(capturedBodies.count == 2)
        #expect(capturedBodies.last?.paymentMethod == "CARD")
        #expect(capturedBodies.last?.paymentGateway == "GLOBAL_PAY")
        #expect(vm.ownerApprovalRequired == nil)
        #expect(vm.showSuccess == true)
        #expect(try context.fetch(FetchDescriptor<PendingOrder>()).isEmpty)
    }

    @Test func ownerGovernanceViewModel_gatesNonOwner_andHandlesApprovalsAndMandateInSom() async throws {
        let nonOwnerMe = try JSONDecoder().decode(
            RetailerMeResponse.self,
            from: Data(#"{"user_id":"u-mgr","retailer_id":"r-1","retailer_org_id":"o-1","retailer_role":"MANAGER","is_owner":false,"name":"Manager","permissions":[],"capabilities":[]}"#.utf8)
        )
        var loadedApprovalsForNonOwner = false
        let nonOwnerVm = OwnerGovernanceViewModel(
            meLoader: { nonOwnerMe },
            approvalsLoader: { _, _, _ in
                loadedApprovalsForNonOwner = true
                return RetailerApprovalsResponse(retailerId: "r-1", items: [], nextCursor: nil)
            },
            approver: { _, _ in ApprovalDecisionResponse(status: "APPROVED", orderId: nil) },
            rejector: { _, _ in ApprovalDecisionResponse(status: "REJECTED", orderId: nil) },
            mandateLoader: { _ in throw URLError(.badURL) },
            mandateUpdater: { _, _ in throw URLError(.badURL) }
        )
        await nonOwnerVm.loadRoleAndApprovals()
        #expect(nonOwnerVm.isOwner == false)
        #expect(loadedApprovalsForNonOwner == false)

        let ownerMe = try JSONDecoder().decode(
            RetailerMeResponse.self,
            from: Data(#"{"user_id":"u-own","retailer_id":"r-1","retailer_org_id":"o-1","retailer_role":"OWNER","is_owner":true,"name":"Owner","permissions":[],"capabilities":[]}"#.utf8)
        )
        let approvalsRes = try JSONDecoder().decode(
            RetailerApprovalsResponse.self,
            from: Data(#"{"retailer_id":"r-1","items":[{"request_id":"req-1","retailer_id":"r-1","requested_by_user":"u-sel","requested_by_name":"Seller 1","amount_minor":500000,"tripped_rule":"DAILY_CREDIT_CAP_EXCEEDED","status":"PENDING_OWNER","payment_method":"CREDIT"}]}"#.utf8)
        )
        let mandateRes = try JSONDecoder().decode(
            SellerMandateResponse.self,
            from: Data(#"{"user_id":"u-sel","retailer_id":"r-1","retailer_role":"SELLER","is_owner":false,"mandate":{"can_receive_goods":true,"can_pay_cash":true,"can_use_credit":true,"daily_credit_cap_minor":500000,"daily_cash_cap_minor":200000,"max_single_order_minor":100000},"daily_usage":{"usage_date":"2026-11-13","spent_credit_minor":50000,"spent_cash_minor":0,"order_count":1}}"#.utf8)
        )
        var approvedId: String?
        var approvedReason: String?
        var patchedUserId: String?
        var patchedBody: SellerMandatePatchBody?

        let ownerVm = OwnerGovernanceViewModel(
            meLoader: { ownerMe },
            approvalsLoader: { _, _, _ in approvalsRes },
            approver: { requestId, reason in
                approvedId = requestId
                approvedReason = reason
                return ApprovalDecisionResponse(status: "APPROVED", orderId: "ord-99")
            },
            rejector: { _, _ in ApprovalDecisionResponse(status: "REJECTED", orderId: nil) },
            mandateLoader: { _ in mandateRes },
            mandateUpdater: { userId, body in
                patchedUserId = userId
                patchedBody = body
                return mandateRes
            }
        )
        await ownerVm.loadRoleAndApprovals()
        #expect(ownerVm.isOwner == true)
        #expect(ownerVm.approvals.count == 1)

        await ownerVm.approveRequest("req-1", reason: "Approved by owner")
        #expect(approvedId == "req-1")
        #expect(approvedReason == "Approved by owner")

        await ownerVm.openMandateEditor(userId: "u-sel", memberName: "Seller 1")
        #expect(ownerVm.mandateForm?.dailyCreditCapSom == "5000")
        #expect(ownerVm.mandateForm?.dailyCashCapSom == "2000")
        #expect(ownerVm.mandateForm?.maxSingleOrderSom == "1000")

        ownerVm.mandateForm?.dailyCreditCapSom = "7500"
        ownerVm.mandateForm?.dailyCashCapSom = "2500"
        ownerVm.mandateForm?.maxSingleOrderSom = "1500"
        await ownerVm.saveMandate()
        #expect(patchedUserId == "u-sel")
        #expect(patchedBody?.dailyCreditCapMinor == 750_000)
        #expect(patchedBody?.dailyCashCapMinor == 250_000)
        #expect(patchedBody?.maxSingleOrderMinor == 150_000)
        #expect(ownerVm.mandateForm == nil)
    }

    @Test func assessPendingOrderError_covers401_409_422_429_andOffline() {
        let offline = RetailerOrderContractHelper.assessPendingOrderError(URLError(.notConnectedToInternet))
        #expect(offline.disposition == .retryTransient)
        #expect(offline.queueStatus == PendingOrder.statusPending)

        let auth401 = RetailerOrderContractHelper.assessPendingOrderError(
            APIError.serverError(statusCode: 401, message: "Unauthorized")
        )
        #expect(auth401.disposition == .parkAuthRequired)
        #expect(auth401.queueStatus == PendingOrder.statusAuthRequired)
        #expect(auth401.shouldNotify == true)

        let conflict409 = RetailerOrderContractHelper.assessPendingOrderError(
            APIError.problemDetail(ProblemDetail(status: 409, detail: "Needs owner", code: "owner_approval_required"))
        )
        #expect(conflict409.disposition == .parkAwaitingOwnerApproval)
        #expect(conflict409.queueStatus == PendingOrder.statusAwaitingOwnerApproval)

        let rate429 = RetailerOrderContractHelper.assessPendingOrderError(
            APIError.serverError(statusCode: 429, message: #"{"code":"rate_limited"}"#)
        )
        #expect(rate429.disposition == .parkRateLimited)
        #expect(rate429.queueStatus == PendingOrder.statusRateLimited)

        let packErr = APIError.problemDetail(
            ProblemDetail(
                status: 422,
                detail: "Pack required",
                code: "PACK_MULTIPLE_REQUIRED",
                skuId: "sku-pack",
                packSize: 6,
                requested: 8,
                suggestedFloor: 6,
                suggestedCeil: 12,
                suggestedNearest: 6
            )
        )
        let parsedPack = RetailerOrderContractHelper.parsePackMultipleSuggestions(
            from: packErr,
            skuNamesBySku: ["sku-pack": "Ayran 1L"]
        )
        #expect(parsedPack.count == 1)
        #expect(parsedPack.first?.suggestedNearest == 6)
    }

    @Test @MainActor func checkoutViewModel_parksAuth401OrderInQueueWithAuthRequiredStatus() async throws {
        let context = try makeInMemoryModelContext()
        let cart = CartManager()
        let (product, variant) = makePackProduct(variantId: "sku-401", name: "Tea", packCount: 1, priceMinor: 10_000)
        cart.add(product: product, variant: variant, quantity: 2, mustHave: false, allowSubstitutes: true)

        let previewOk = try JSONDecoder().decode(
            CheckoutPreviewResponse.self,
            from: Data(#"{"ok":true,"stock_warnings":[]}"#.utf8)
        )
        let vm = CheckoutViewModel(
            previewFetcher: { _ in previewOk },
            checkoutSender: { _, _ in
                throw APIError.serverError(statusCode: 401, message: "Unauthorized")
            }
        )

        await vm.submitOrder(cart: cart, modelContext: context)
        let queued = try context.fetch(FetchDescriptor<PendingOrder>())
        #expect(queued.count == 1)
        #expect(queued.first?.status == PendingOrder.statusAuthRequired)
        #expect(vm.pendingQueueNotice?.status == PendingOrder.statusAuthRequired)
        cart.clear()
    }
}
