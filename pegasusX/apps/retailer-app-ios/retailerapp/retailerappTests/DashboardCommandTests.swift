import Foundation
import Testing
@testable import retailerapp

struct DashboardCommandTests {
    @Test func controlTowerPulseDecodesFiscalFailedSupplierFacet() throws {
        let raw = Data("""
        {
          "source": "spanner",
          "empty": false,
          "open_orders": 1,
          "active_fulfillments": 0,
          "dock_pending": 0,
          "pos_open_sessions": 0,
          "open_shifts": 0,
          "open_assist_tickets": 0,
          "low_stock_sku_bins": 0,
          "shift_variances_7d": 0,
          "sales_minor_7d": 0,
          "loyalty": { "enrolled": false },
          "orders_by_status": { "FISCAL_FAILED": 1, "COMPLETED": 1 },
          "orders_by_supplier": [
            { "supplier_id": "sup-a", "orders_by_status": { "FISCAL_FAILED": 1 } },
            { "supplier_id": "sup-b", "orders_by_status": { "COMPLETED": 1 } }
          ]
        }
        """.utf8)
        let pulse = try JSONDecoder().decode(ControlTowerPulseWire.self, from: raw)
        #expect(pulse.source == "spanner")
        #expect(pulse.empty == false)
        #expect(pulse.ordersByStatus["FISCAL_FAILED"] == 1)
        #expect(pulse.ordersBySupplier.first { $0.supplierId == "sup-a" }?.ordersByStatus["FISCAL_FAILED"] == 1)
        #expect(pulse.ordersBySupplier.first { $0.supplierId == "sup-b" }?.ordersByStatus["COMPLETED"] == 1)
        #expect(pulse.loyalty?.enrolled == false)
        #expect(orderStatusFunnel.contains("FISCAL_FAILED"))
        #expect(orderStatusFunnel.count == 17)
        #expect(statusStackModel(dictionary: orderStatusFunnel, counts: pulse.ordersByStatus).rows.count == 17)
    }

    @Test func commandChipMatchesCanonicalStatusAndSupplier() {
        #expect(retailerOrderMatchesCommand(statusRaw: "FISCAL_FAILED", supplierId: "sup-a", commandStatus: "FISCAL_FAILED", commandSupplierId: nil))
        #expect(retailerOrderMatchesCommand(statusRaw: "DISPATCHED", supplierId: "sup-a", commandStatus: "LOADED", commandSupplierId: "sup-a"))
        #expect(!retailerOrderMatchesCommand(statusRaw: "FISCAL_FAILED", supplierId: "sup-a", commandStatus: "FISCAL_FAILED", commandSupplierId: "sup-b"))
    }

    @Test func pulseHonestyKeepsPreviousOnFailure() {
        let previous = ["old"]
        let failed = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: previous)
        #expect(failed.events == previous)
        #expect(failed.error == PulseHonesty.failed)
        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: previous)
        #expect(empty.events.isEmpty)
        #expect(empty.error == nil)
    }

    @Test func commandPulseHonestyKeepsPreviousOnFailure() {
        let previous = "last"
        let failed = PulseHonesty.applyObject(ok: false, incoming: nil as String?, previous: previous)
        #expect(failed.value == previous)
        #expect(failed.error == PulseHonesty.commandFailed)
        let empty = PulseHonesty.applyObject(ok: true, incoming: "empty", previous: previous)
        #expect(empty.value == "empty")
        #expect(empty.error == nil)
    }

    // MARK: - W5 Fast Home, Cutoff Timer, Language Switcher & Telegram Quick-Bind

    @Test @MainActor func preferencesStore_defaultsToUz_andSwitchesLanguagesImmediatelyWithPersistence() throws {
        let suiteName = "test.w5.prefs.\(UUID().uuidString)"
        let defaults = try #require(UserDefaults(suiteName: suiteName))
        defer { defaults.removePersistentDomain(forName: suiteName) }

        let store = RetailerPreferencesStore(defaults: defaults)
        #expect(store.currentLanguage == .uz)
        #expect(store.currentLanguage.nativeName == "O'zbekcha")
        #expect(store.localizedString("mobile_retailer.workspace_mode.fast") == "Tezkor rejim")

        store.setLanguage(.ru)
        #expect(store.currentLanguage == .ru)
        #expect(store.localizedString("mobile_retailer.workspace_mode.fast") == "Быстрый режим")

        let reloaded = RetailerPreferencesStore(defaults: defaults)
        #expect(reloaded.currentLanguage == .ru)

        reloaded.setLanguage(.en)
        #expect(reloaded.localizedString("mobile_retailer.workspace_mode.fast") == "Fast Mode")
        #expect(reloaded.localizedString("mobile_retailer.fast_home.what_we_owe") == "What We Owe")

        let prevSharedLang = RetailerPreferencesStore.shared.currentLanguage
        let prevSharedMode = RetailerPreferencesStore.shared.workspaceModeOverride
        defer {
            RetailerPreferencesStore.shared.setLanguage(prevSharedLang)
            RetailerPreferencesStore.shared.setWorkspaceMode(prevSharedMode)
        }
        RetailerPreferencesStore.shared.setLanguage(.uz)
        #expect(L10n.string("mobile_retailer.workspace_mode.fast") == "Tezkor rejim")
        RetailerPreferencesStore.shared.setLanguage(.ru)
        #expect(L10n.string("mobile_retailer.workspace_mode.fast") == "Быстрый режим")
        RetailerPreferencesStore.shared.setLanguage(.en)
        #expect(L10n.string("mobile_retailer.workspace_mode.fast") == "Fast Mode")
    }

    @Test @MainActor func fastHomeContractHelper_computesServerSyncedCountdownImmuneToClockSkew() {
        let serverNowISO = "2026-11-13T12:00:00.250Z"
        let cutoffISO = "2026-11-13T13:00:00.250Z"
        // Simulate a phone clock drifted +2 hours ahead of server
        let skewedDeviceNow = FastHomeContractHelper.parseISO8601Date("2026-11-13T14:00:00.250Z")!

        let offset = FastHomeContractHelper.computeServerClockOffset(
            serverNowISO: serverNowISO,
            deviceNow: skewedDeviceNow
        )
        #expect(abs(offset + 7200) < 0.01)

        let countdown = FastHomeContractHelper.computeCutoffCountdown(
            cutoffAtISO: cutoffISO,
            serverClockOffset: offset,
            now: skewedDeviceNow
        )
        #expect(countdown?.isExpired == false)
        #expect(countdown?.totalSecondsRemaining == 3600)
        #expect(countdown?.formattedHMS == "01:00:00")

        let sixtySecondsLater = skewedDeviceNow.addingTimeInterval(60)
        let laterCountdown = FastHomeContractHelper.computeCutoffCountdown(
            cutoffAtISO: cutoffISO,
            serverClockOffset: offset,
            now: sixtySecondsLater
        )
        #expect(laterCountdown?.totalSecondsRemaining == 3540)
        #expect(laterCountdown?.formattedHMS == "00:59:00")

        let pastCutoff = skewedDeviceNow.addingTimeInterval(3605)
        let expired = FastHomeContractHelper.computeCutoffCountdown(
            cutoffAtISO: cutoffISO,
            serverClockOffset: offset,
            now: pastCutoff
        )
        #expect(expired?.isExpired == true)
        #expect(expired?.formattedHMS == "00:00:00")
    }

    @Test @MainActor func dashboardViewModel_roleDefaults_persistence_fastHomeDecode_andProposalActions() async throws {
        let suiteName = "test.w5.dash.\(UUID().uuidString)"
        let defaults = try #require(UserDefaults(suiteName: suiteName))
        defer { defaults.removePersistentDomain(forName: suiteName) }

        #expect(FastHomeContractHelper.resolveDefaultWorkspaceMode(retailerRole: "BUYER", isOwner: false) == .fast)
        #expect(FastHomeContractHelper.resolveDefaultWorkspaceMode(retailerRole: "CASHIER", isOwner: false) == .fast)
        #expect(FastHomeContractHelper.resolveDefaultWorkspaceMode(retailerRole: "MANAGER", isOwner: false) == .fast)
        #expect(FastHomeContractHelper.resolveDefaultWorkspaceMode(retailerRole: "OWNER", isOwner: true) == .full)

        let rawFastHome = Data("""
        {
          "server_now": "2026-11-13T10:00:00Z",
          "cutoff_at": "2026-11-13T12:30:00Z",
          "next_delivery_date": "2026-11-14",
          "tomorrow_delivery": {
            "order_id": "ord-tmrw-1",
            "supplier_id": "sup-1",
            "supplier_name": "Milk Trade",
            "status": "DISPATCHED",
            "eta_text": "2026-11-14T09:00:00Z",
            "items_count": 4,
            "total_minor": 45000000,
            "currency": "UZS"
          },
          "balance_owed_minor": 125000000,
          "overdue_minor": 25000000,
          "days_overdue": 3,
          "active_promise": {
            "promise_id": "prom-1",
            "promised_until": "2026-11-16",
            "amount_minor": 25000000,
            "status": "OPEN"
          },
          "credit_headroom_minor": 375000000,
          "pending_proposal": {
            "order_id": "prop-1",
            "supplier_id": "sup-1",
            "supplier_name": "Milk Trade",
            "order_source": "AI",
            "confirmation_status": "PENDING",
            "requested_delivery_date": "2026-11-14",
            "auto_confirm_at": "2026-11-13T12:30:00Z",
            "items_count": 1,
            "total_minor": 9000000,
            "currency": "UZS",
            "line_items": [
              { "sku": "sku-1", "name": "Ayran 1L", "quantity": 8, "unit_price_minor": 150000, "pack_size": 6 }
            ]
          },
          "pending_approvals_count": 2,
          "locale": "uz"
        }
        """.utf8)
        let decodedFastHome = try JSONDecoder().decode(RetailerFastHomeResponse.self, from: rawFastHome)
        #expect(RetailerMoney.formatWholeCurrency(decodedFastHome.balanceOwedMinor, currency: "UZS") == "1 250 000 so'm")
        #expect(decodedFastHome.pendingApprovalsCount == 2)

        let buyerMe = try JSONDecoder().decode(
            RetailerMeResponse.self,
            from: Data(#"{"user_id":"u-1","retailer_id":"r-1","retailer_org_id":"o-1","retailer_role":"BUYER","is_owner":false,"name":"Buyer","permissions":[],"capabilities":[]}"#.utf8)
        )
        let ownerMe = try JSONDecoder().decode(
            RetailerMeResponse.self,
            from: Data(#"{"user_id":"u-2","retailer_id":"r-1","retailer_org_id":"o-1","retailer_role":"OWNER","is_owner":true,"name":"Owner","permissions":[],"capabilities":[]}"#.utf8)
        )

        var confirmedOrderId: String?
        var heldOrderId: String?
        var editedPayload: (String, String, [APIClient.EditPreorderItem])?
        let fixedNow = FastHomeContractHelper.parseISO8601Date("2026-11-13T10:00:00Z")!

        let prefs = RetailerPreferencesStore(defaults: defaults)
        let buyerVm = DashboardViewModel(
            preferencesStore: prefs,
            retailerIdProvider: { "r-1" },
            dateProvider: { fixedNow },
            fastHomeLoader: { decodedFastHome },
            meLoader: { buyerMe },
            ordersLoader: { _ in [] },
            predictionsLoader: { [] },
            confirmAiAction: { confirmedOrderId = $0 },
            rejectAiAction: { id, _ in heldOrderId = id },
            editPreorderAction: { id, date, items in editedPayload = (id, date, items) },
            shouldLoadSecondaryModules: false
        )

        await buyerVm.loadData()
        #expect(buyerVm.workspaceMode == .fast)
        #expect(buyerVm.fastHomeUsingFallback == false)
        #expect(buyerVm.cutoffCountdown?.formattedHMS == "02:30:00")

        await buyerVm.confirmFastProposal()
        #expect(confirmedOrderId == "prop-1")

        await buyerVm.holdFastProposal()
        #expect(heldOrderId == "prop-1")

        await buyerVm.editFastProposal(updatedItems: [
            FastHomeProposalItem(sku: "sku-1", name: "Ayran 1L", quantity: 8, unitPriceMinor: 150_000, packSize: 6)
        ])
        #expect(editedPayload?.0 == "prop-1")
        #expect(editedPayload?.1 == "2026-11-14")
        #expect(editedPayload?.2.first?.quantity == 6)

        // Owner starts in Full ERP by default until manually toggled, and manual override persists
        let ownerPrefs = RetailerPreferencesStore(defaults: defaults)
        ownerPrefs.setWorkspaceMode(nil)
        let ownerVm = DashboardViewModel(
            preferencesStore: ownerPrefs,
            retailerIdProvider: { "r-1" },
            dateProvider: { fixedNow },
            fastHomeLoader: { decodedFastHome },
            meLoader: { ownerMe },
            ordersLoader: { _ in [] },
            predictionsLoader: { [] },
            shouldLoadSecondaryModules: false
        )
        await ownerVm.loadData()
        #expect(ownerVm.workspaceMode == .full)

        ownerVm.switchWorkspaceMode(.fast)
        #expect(ownerVm.workspaceMode == .fast)

        let reloadedOwnerVm = DashboardViewModel(
            preferencesStore: RetailerPreferencesStore(defaults: defaults),
            retailerIdProvider: { "r-1" },
            dateProvider: { fixedNow },
            fastHomeLoader: { decodedFastHome },
            meLoader: { ownerMe },
            ordersLoader: { _ in [] },
            predictionsLoader: { [] },
            shouldLoadSecondaryModules: false
        )
        await reloadedOwnerVm.loadData()
        #expect(reloadedOwnerVm.workspaceMode == .fast)
    }

    @Test @MainActor func dashboardViewModel_fallsBackGracefullyWhenFastHomeUnavailable() async throws {
        let suiteName = "test.w5.fallback.\(UUID().uuidString)"
        let defaults = try #require(UserDefaults(suiteName: suiteName))
        defer { defaults.removePersistentDomain(forName: suiteName) }

        let fallbackOrder = Order(
            id: "ord-sched-1",
            retailerId: "r-1",
            supplierId: "sup-9",
            supplierName: "Nestle Dist",
            status: .scheduled,
            items: [
                OrderLineItem(
                    id: "li-1",
                    productId: "sku-9",
                    productName: "Milk 1L",
                    variantId: "sku-9",
                    variantSize: "1L",
                    quantity: 12,
                    unitPrice: 120_000,
                    totalPrice: 1_440_000
                )
            ],
            totalAmount: 1_440_000,
            currency: "UZS",
            autoConfirmAt: "2026-11-13T14:00:00Z",
            deliverBefore: "2026-11-14",
            orderSource: "RETAILER",
            confirmationStatus: "PENDING",
            createdAt: "2026-11-13T08:00:00Z",
            updatedAt: "2026-11-13T08:00:00Z",
            estimatedDelivery: "2026-11-14T10:00:00Z",
            qrCode: nil
        )
        let creditProfile = try JSONDecoder().decode(
            CreditProfile.self,
            from: Data(#"{"retailer_id":"r-1","supplier_id":"sup-9","credit_limit_minor":10000000,"current_balance_minor":2500000,"available_credit_minor":7500000,"delinquency_count":1,"status":"ACTIVE"}"#.utf8)
        )
        let fixedNow = FastHomeContractHelper.parseISO8601Date("2026-11-13T11:00:00Z")!

        let vm = DashboardViewModel(
            preferencesStore: RetailerPreferencesStore(defaults: defaults),
            retailerIdProvider: { "r-1" },
            dateProvider: { fixedNow },
            fastHomeLoader: { throw APIError.serverError(statusCode: 404, message: "Not Found") },
            meLoader: { throw APIError.serverError(statusCode: 404, message: "Not Found") },
            ordersLoader: { _ in [fallbackOrder] },
            predictionsLoader: { [] },
            creditProfileLoader: { creditProfile },
            creditRelationshipsLoader: { [] },
            shouldLoadSecondaryModules: false
        )

        await vm.loadData()
        #expect(vm.fastHomeUsingFallback == true)
        #expect(vm.fastHome?.tomorrowDelivery?.orderId == "ord-sched-1")
        #expect(vm.fastHome?.balanceOwedMinor == 2_500_000)
        #expect(vm.fastHome?.creditHeadroomMinor == 7_500_000)
        #expect(vm.fastHome?.daysOverdue == 1)
        #expect(vm.fastHome?.pendingProposal?.orderId == "ord-sched-1")
        #expect(vm.cutoffCountdown?.formattedHMS == "03:00:00")
    }

    @Test @MainActor func profileViewModel_telegramQuickBind_andLanguageSwitch() async throws {
        let suiteName = "test.w5.profile.\(UUID().uuidString)"
        let defaults = try #require(UserDefaults(suiteName: suiteName))
        defer { defaults.removePersistentDomain(forName: suiteName) }

        let prefs = RetailerPreferencesStore(defaults: defaults)
        let status = try JSONDecoder().decode(
            TelegramLinkStatusResponse.self,
            from: Data(#"{"linked":true,"telegram_chat_id":998877,"username":"shakhzod_shop","bot_deep_link":"https://t.me/pegasus_bot?start=ok"}"#.utf8)
        )
        let otpResp = try JSONDecoder().decode(
            TelegramLinkRequestResponse.self,
            from: Data(#"{"ok":true,"otp_code":"654321","expires_at":"2026-11-13T15:00:00Z","bot_deep_link":"https://t.me/pegasus_bot?start=654321"}"#.utf8)
        )

        let profileVm = ProfileViewModel(
            preferencesStore: prefs,
            telegramStatusLoader: { status },
            telegramRequestLoader: { otpResp }
        )

        #expect(profileVm.currentLanguage == .uz)
        profileVm.setLanguage(.en)
        #expect(profileVm.currentLanguage == .en)

        await profileVm.loadTelegramStatus()
        #expect(profileVm.telegramStatus?.linked == true)
        #expect(profileVm.telegramStatus?.telegramChatId == "998877")

        await profileVm.requestTelegramOtp()
        #expect(profileVm.telegramLinkRequest?.otpCode == "654321")
        #expect(profileVm.telegramLinkRequest?.botDeepLink?.contains("654321") == true)
    }
}
