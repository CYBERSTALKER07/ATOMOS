This file is a merged representation of a subset of the codebase, containing specifically included files and files not matching ignore patterns, combined into a single document by Repomix.
The content has been processed where comments have been removed, empty lines have been removed, content has been compressed (code blocks are separated by ⋮---- delimiter).

# File Summary

## Purpose
This file contains a packed representation of a subset of the repository's contents that is considered the most important context.
It is designed to be easily consumable by AI systems for analysis, code review,
or other automated processes.

## File Format
The content is organized as follows:
1. This summary section
2. Repository information
3. Directory structure
4. Repository files (if enabled)
5. Multiple file entries, each consisting of:
  a. A header with the file path (## File: path/to/file)
  b. The full contents of the file in a code block

## Usage Guidelines
- This file should be treated as read-only. Any changes should be made to the
  original repository files, not this packed version.
- When processing this file, use the file path to distinguish
  between different files in the repository.
- Be aware that this file may contain sensitive information. Handle it with
  the same level of security as you would the original repository.

## Notes
- Some files may have been excluded based on .gitignore rules and Repomix's configuration
- Binary files are not included in this packed representation. Please refer to the Repository Structure section for a complete list of file paths, including binary files
- Only files matching these patterns are included: apps/*-ios/**/*.swift, apps/control-tower-macos/**/*.swift
- Files matching these patterns are excluded: **/*.lproj/**, **/*.xcodeproj/**, **/Tests/**, **/*Tests.swift, **/Generated/**, repomix*
- Files matching patterns in .gitignore are excluded
- Files matching default ignore patterns are excluded
- Code comments have been removed from supported file types
- Empty lines have been removed from all files
- Content has been compressed - code blocks are separated by ⋮---- delimiter
- Files are sorted by Git change count (files with more changes are at the bottom)

# Directory Structure
```
apps/
  control-tower-macos/
    Sources/
      Models/
        AnalyticsModels.swift
        FinOpsModels.swift
        InfraModels.swift
        TelemetryModels.swift
      Services/
        ControlTowerAPIClient.swift
        ControlTowerHaptics.swift
        ExportService.swift
        WebSocketStreamClient.swift
      Theme/
        ControlTowerTheme.swift
      ViewModels/
        ControlTowerViewModel.swift
      Views/
        AnalyticsScorecardView.swift
        DesktopKeyboardCheatsheetView.swift
        FinOpsView.swift
        InfraHealthView.swift
        InspectorView.swift
        MainSplitView.swift
        MenuBarStatusView.swift
        OverviewView.swift
        SidebarView.swift
        TelemetryStreamView.swift
      PegasusControlTowerApp.swift
  driver-app-ios/
    driverapp/
      driverapp/
        Services/
          AutoUpdater.swift
          EnterpriseUpdateConfig.swift
      Views/
        Home/
          RescueAcceptanceDialog.swift
    driverappios/
      driverappios/
        Database/
          OfflineDeliveryStore.swift
        Models/
          DriverEarnings.swift
          DriverNotification.swift
          FoundryFleetModels.swift
          LineItem.swift
          Mission.swift
          OfflineDelivery.swift
          Order.swift
          PendingCollection.swift
          PulseModels.swift
          QRPayload.swift
          RouteGeometry.swift
          RouteManifest.swift
          TelemetryPayload.swift
        Offline/
          DriverOfflineActionCatalog.swift
          DriverOfflineQueue.swift
          OfflineHttpSemantics.swift
          QueuedDriverAction.swift
          QueuedMutationRecord.swift
        Services/
          APIClient.swift
          APIClient+FoundryFleet.swift
          AppDelegate.swift
          AuthService.swift
          AutoUpdater.swift
          CellApi.swift
          DeliveryActivityAttributes.swift
          DriverLiveActivityManager.swift
          DriverSessionReconcile.swift
          DriverWsRefresh.swift
          EnterpriseUpdateConfig.swift
          FirebaseAuthHelper.swift
          FleetService.swift
          FleetServiceLive.swift
          ManifestService.swift
          ManifestServiceLive.swift
          MarketPack.swift
          MediaUploadService.swift
          NetworkLifecycleEngine.swift
          ProfileService.swift
          PushNotificationManager.swift
          SyncService.swift
          SyncServiceLive.swift
          TelemetryService.swift
          TelemetryServiceLive.swift
        Theme/
          PegasusTheme.swift
        Utilities/
          DriverIdempotency.swift
          DriverReconnectRecovery.swift
          HandoffPathResolver.swift
          Haptics.swift
          Haversine.swift
          LocationInterpolator.swift
          MapCameraConfig.swift
          NavigationVoiceAnnouncer.swift
          PlannedRoute.swift
          RemainingStops.swift
          RouteDeviation.swift
          RouteNavigation.swift
          SHA256Helper.swift
        ViewModels/
          CashCollectionViewModel.swift
          CorrectionViewModel.swift
          DriverHomeViewModel.swift
          DriverNotificationInboxViewModel.swift
          FleetDVIRViewModel.swift
          FleetMapViewModel.swift
          FleetViewModel.swift
          LoginViewModel.swift
          MidRouteRescueViewModel.swift
          OfflineVerifierViewModel.swift
          OffloadReviewViewModel.swift
          PaymentWaitingViewModel.swift
          ProfileViewModel.swift
          RequestRescueViewModel.swift
          RidesListViewModel.swift
          RootViewModel.swift
          ScannerViewModel.swift
          ShopClosedWaitingViewModel.swift
          SupplyTransfersViewModel.swift
          SyncQueueViewModel.swift
          TelemetryViewModel.swift
          YardGateViewModel.swift
        Views/
          Components/
            Home/
              FactorySupplyCard.swift
              FieldMoneyStrip.swift
              MapButton.swift
              QuickActionsSection.swift
              RecentActivitySection.swift
              RemainingStopsStepper.swift
              TodaySummaryCard.swift
              TransitControlCard.swift
              VehicleInfoCard.swift
            Manifest/
              Correction/
                DeliveryCorrectionHeaderView.swift
                DeliveryCorrectionLineItemCard.swift
                DeliveryCorrectionSummaryBar.swift
              DistanceInfo.swift
              EarlyCompleteSheet.swift
              InfoChip.swift
              ManifestEmptyView.swift
              ManifestHeader.swift
              ManifestLoadingView.swift
              RideCard.swift
            Map/
              ActiveDeliverySheet.swift
              GlassSheet.swift
              MapTopOverlay.swift
              NavigationCueBanner.swift
              OrderPickerSheet.swift
              OrderPreviewSheet.swift
              SheetHandle.swift
            Offload/
              CollectCashView.swift
              FiscalFailedView.swift
              FiscalizingView.swift
              OffloadActionFooter.swift
              OffloadSummaryCard.swift
            Profile/
              DriverCard.swift
              HistorySection.swift
              ProfileHeader.swift
              QuickActions.swift
              StatsSection.swift
            ActiveRideBar.swift
            AnimationModifiers.swift
            ClientPolicyBanner.swift
            CollapsibleSidebar.swift
            DriverEmptyView.swift
            DriverLoadingStateCard.swift
            DriverStateCard.swift
            DriverStateViews.swift
            DriverUiComponents.swift
            GPSErrorBanner.swift
            HandoffInboxCard.swift
            KpiTile.swift
            MapMarkerDetailSheet.swift
            PegasusEmptyView.swift
            PegasusMonochromeTheme.swift
            PegasusMotion.swift
            PegasusStateViews.swift
            PulseStrip.swift
            SignaturePadView.swift
            SourceChip.swift
            StatusPill.swift
            StatusStack.swift
            TelemetryBadge.swift
          CashCollectionView.swift
          DeliveryCorrectionView.swift
          DriverNotificationInboxView.swift
          EndSessionView.swift
          FleetDVIRView.swift
          FleetMapView.swift
          HomeView.swift
          LoginView.swift
          MainTabView.swift
          MidRouteRescueView.swift
          MissionDetailSheet.swift
          MissionListCard.swift
          OfflineVerifierView.swift
          OffloadReviewView.swift
          PaymentWaitingView.swift
          ProfileView.swift
          QRScannerView.swift
          RequestRescueSheet.swift
          RidesListView.swift
          ShopClosedWaitingView.swift
          SupplyTransfersView.swift
          SyncQueueView.swift
          YardGateView.swift
        L10n.swift
        PegasusDriverApp.swift
  factory-app-ios/
    FactoryApp/
      Design/
        CellApi.swift
        CollapsibleSidebar.swift
        DeliveryActivityAttributes.swift
        DeliveryLiveActivityWidget.swift
        MarketPack.swift
        PegasusEmptyView.swift
        PegasusMonochromeTheme.swift
        PegasusMotion.swift
        PegasusStateViews.swift
        RealtimeRefresh.swift
        SourceChip.swift
        StatusStack.swift
      factoryapp/
        Services/
          AutoUpdater.swift
          EnterpriseUpdateConfig.swift
      Layout/
        FactoryAdaptiveShell.swift
        FactorySection.swift
        ResponsiveGridContentWrapper.swift
      Models/
        FactoryModels.swift
      Services/
        APIClient.swift
        AppDelegate.swift
        FactoryRealtimeClient.swift
        FactoryService.swift
        FactorySessionReconcile.swift
        FirebaseAuthHelper.swift
        GeocodeService.swift
        NetworkLifecycleEngine.swift
        PushNotificationManager.swift
        TokenStore.swift
      Theme/
        PegasusTheme.swift
      Utilities/
        FactoryIdempotency.swift
        GeocodeLocationSupport.swift
        JwtPayload.swift
      ViewModels/
        AuthAndLocationViewModels.swift
        DashboardAndInsightsViewModels.swift
        ManifestsAndFleetViewModels.swift
        StaffAndSupplyViewModels.swift
        TransfersAndBayViewModels.swift
      Views/
        Analytics/
          AnalyticsView.swift
        Components/
          Dashboard/
            DashboardHeroCard.swift
            DashboardMetric.swift
            WorkflowLaunchCard.swift
          Insights/
            InsightRow.swift
          LoadingBay/
            BaySection.swift
            BayTransferCard.swift
            FactoryHandoffTimelineSection.swift
            LoadingBayOverviewCard.swift
          ManifestExceptions/
            ExceptionRow.swift
          PayloadOverride/
            MetricBox.swift
            MoveTransferSheet.swift
            OverrideManifestCard.swift
            OverrideTransferRow.swift
          Staff/
            StaffListContent.swift
            StaffRow.swift
          SupplyRequests/
            SupplyRequestRow.swift
          Transfers/
            TransferDetailCards.swift
            TransferListSummary.swift
            TransferRow.swift
            TransferRowMetric.swift
          AddressLocationField.swift
          ClientPolicyBanner.swift
          FactoryInlineStateViews.swift
          FactoryRuntimeBanner.swift
          FactoryStateViews.swift
          FactoryStatusBadge.swift
          FactoryUiComponents.swift
        Dashboard/
          DashboardView.swift
        Exceptions/
          ManifestExceptionsView.swift
        Fleet/
          FleetView.swift
        Insights/
          InsightsView.swift
        LoadingBay/
          LoadingBayView.swift
        Location/
          LocationSettingsView.swift
        Manifests/
          ManifestDetailView.swift
          ManifestsView.swift
          PayloadLoadView.swift
        Notifications/
          NotificationInboxView.swift
        PayloadOverride/
          PayloadOverrideView.swift
        Production/
          ProductionKioskScreen.swift
        Setup/
          LocationSetupView.swift
        Staff/
          StaffDetailView.swift
          StaffView.swift
        SupplyRequests/
          SupplyRequestDetailView.swift
          SupplyRequestsHubView.swift
        Transfers/
          CreateTransferView.swift
          TransferDetailView.swift
          TransferListView.swift
        FactoryMoreHubView.swift
        LoginView.swift
        MainTabView.swift
      L10n.swift
      PegasusFactoryApp.swift
      RootView.swift
  payload-app-ios/
    payload-app-ios/
      Design/
        CellApi.swift
        CollapsibleSidebar.swift
        DeliveryActivityAttributes.swift
        DeliveryLiveActivityWidget.swift
        MarketPack.swift
        PegasusFeedbackViews.swift
        PegasusMonochromeTheme.swift
        PegasusMotion.swift
        PegasusStateViews.swift
        RealtimeRefresh.swift
        StatusStack.swift
        StatusStackChips.swift
      Models/
        Models.swift
      Services/
        APIClient.swift
        AppDelegate.swift
        AutoUpdater.swift
        EnterpriseUpdateConfig.swift
        FirebaseAuthHelper.swift
        NetworkLifecycleEngine.swift
        OfflineQueue.swift
        PushNotificationManager.swift
        TokenStore.swift
        WebSocketClient.swift
      Theme/
        TermTheme.swift
      Utilities/
        HandoffPathResolver.swift
        ManifestBoard.swift
        PayloadIdempotency.swift
      ViewModels/
        ExceptionsViewModel.swift
        HomeViewModel.swift
        InboundReturnsViewModel.swift
        LoginViewModel.swift
        RootViewModel.swift
      Views/
        Components/
          ClientPolicyBanner.swift
          ExplainAndHandoffCards.swift
          HomeBannerOverlay.swift
          KpiTile.swift
          ManifestKpiGrid.swift
          OnlineDot.swift
          OrderChecklistRows.swift
          PayloadFeedbackViews.swift
          PayloadLoadingViews.swift
          PayloadSectionHeader.swift
          PostSealCountdownView.swift
          PulseStrip.swift
          TruckHeader.swift
        Sheets/
          ExceptionReasonSheet.swift
          InjectOrderSheet.swift
          ProductScannerSheet.swift
          ReDispatchSheet.swift
        ExceptionsSheet.swift
        HomeView.swift
        InboundReturnsList.swift
        InboundReturnsView.swift
        LoginView.swift
        ManifestDetailPane.swift
        NotificationsSheet.swift
        OrderChecklistSection.swift
        PayloadStateView.swift
        RecommendationRow.swift
        RootView.swift
        TruckSidebar.swift
      L10n.swift
      PegasusPayloadApp.swift
      ResponsiveGridContentWrapper.swift
    payloadapp/
      payloadapp/
        Services/
          AutoUpdater.swift
          EnterpriseUpdateConfig.swift
  retailer-app-ios/
    retailerapp/
      retailerapp/
        Components/
          AutoOrder/
            AutoOrderExplainerCard.swift
            AutoOrderGlobalToggleCard.swift
            AutoOrderHeaderCard.swift
            AutoOrderOverridesSection.swift
            AutoOrderPredictionsSection.swift
            DemandSourceChips.swift
          Catalog/
            CatalogAllProductsGrid.swift
            CatalogBentoGrid.swift
            CatalogBrowseChips.swift
            CatalogBrowseMode.swift
            CatalogNoResultsState.swift
            CatalogSearchBar.swift
            CatalogSearchResults.swift
          Dashboard/
            AiPredictionSection.swift
            KpiGrid.swift
            QuickReorderSection.swift
            ServiceGrid.swift
            ServiceTileViews.swift
          Insights/
            CategoryBreakdownView.swift
            DailySpendingChartView.swift
            FlowLayout.swift
            KPICard.swift
            OrdersByStateView.swift
            WeekdayPatternView.swift
          Orders/
            ActiveOrderCard.swift
            AiPlannedCard.swift
            DeliveryProposalReviewSheet.swift
            OrderStatusTimeline.swift
          Profile/
            CreditProfileSection.swift
            EmpathyEngineSection.swift
            OrderHistoryLink.swift
            PreferencesSection.swift
            SettingsComponents.swift
            SettingsRowViews.swift
            SettingsToggle.swift
            StatsRowView.swift
            UserCard.swift
          Tracking/
            ActiveCountBadge.swift
            DriverMarker.swift
            OrderInfoCard.swift
            SupplierChip.swift
          AnimatedNumberText.swift
          BottomSheetOverlay.swift
          ClientPolicyBanner.swift
          ErrorStateView.swift
          FloatingActiveOrdersBar.swift
          GradientHeaderCard.swift
          KpiTile.swift
          OrderCardView.swift
          PegasusButton.swift
          PegasusCard.swift
          ProductCardView.swift
          PulseStripView.swift
          QuantityStepper.swift
          RescueWarningBanner.swift
          RetailerEmptyView.swift
          RetailerSectionHeader.swift
          RetailerStateViews.swift
          SidebarMenu.swift
          SkeletonOrderViews.swift
          SkeletonViews.swift
        Design/
          FlexibleStatusChips.swift
          StatusStack.swift
        Layout/
          ResponsiveGridContentWrapper.swift
          RetailerSection.swift
        Models/
          AppModels.swift
          AuthModels.swift
          CardModels.swift
          Category.swift
          ClaimModels.swift
          Order.swift
          PendingOrder.swift
          Product.swift
          Supplier.swift
        Screens/
          AccountProfileView.swift
          ActiveDeliveriesView.swift
          ActiveOrderView.swift
          AdjustStockSheet.swift
          ArrivalView.swift
          AssistView.swift
          AutoOrderView.swift
          CapabilitiesView.swift
          CartView.swift
          CatalogView.swift
          CategoryProductsView.swift
          CategorySuppliersView.swift
          CheckoutPaymentPickerSheet.swift
          CheckoutView.swift
          ConnectSupplierSheet.swift
          CreditPartnersView.swift
          DashboardView.swift
          DeliveriesHubView.swift
          DeliveryMapView.swift
          DeliveryPaymentSheetView.swift
          FamilyMembersView.swift
          FileClaimView.swift
          FutureDemandView.swift
          HistoryView.swift
          HqView.swift
          InsightsView.swift
          LocalSkusView.swift
          LocationPickerView.swift
          LocationsView.swift
          LoginView.swift
          MySuppliersView.swift
          NotificationInboxView.swift
          OrderDetailSheet.swift
          OrderStatusHistorySection.swift
          OrdersView.swift
          PlanogramView.swift
          PosView.swift
          ProcurementView.swift
          ProductDetailView.swift
          ProfileView.swift
          PutawaySheet.swift
          ReceiveDeliverySheet.swift
          ReportsProView.swift
          RetailerMoreHubView.swift
          SavedCardsView.swift
          SearchView.swift
          SectionsView.swift
          SelectOrgView.swift
          SetupView.swift
          ShiftsView.swift
          ShopClosedBanner.swift
          StoreStockOrderPickerSheet.swift
          StoreStockView.swift
          SupplierProductsView.swift
          TeamView.swift
        Services/
          APIClient.swift
          AppDelegate.swift
          AuthManager.swift
          AutoUpdater.swift
          CartManager.swift
          CellApi.swift
          DeliveryActivityAttributes.swift
          EnterpriseUpdateConfig.swift
          FirebaseAuthHelper.swift
          GeocodeService.swift
          LocationManager.swift
          MarketPack.swift
          MediaUploadService.swift
          NetworkLifecycleEngine.swift
          PendingOrderReplayer.swift
          PendingPosStore.swift
          RetailerCheckoutService.swift
          RetailerLiveActivityManager.swift
          RetailerProfileService.swift
          RetailerRefreshCenter.swift
          RetailerSessionReconcile.swift
          RetailerSupplierDiscoveryService.swift
          RetailerWebSocket.swift
        Theme/
          AnimationConstants.swift
          PegasusIcon.swift
          Theme.swift
        Utilities/
          CountdownTimer.swift
          Haptics.swift
          PushNotificationManager.swift
          RetailerAsync.swift
          RetailerErrorSupport.swift
          RetailerIdempotency.swift
        ViewModels/
          AccountProfileViewModel.swift
          ActiveDeliveriesViewModel.swift
          ActiveOrderViewModel.swift
          AppShellViewModel.swift
          ArrivalViewModel.swift
          AssistViewModel.swift
          AutoOrderViewModel.swift
          CapabilitiesViewModel.swift
          CatalogViewModel.swift
          CategoryProductsViewModel.swift
          CategorySuppliersViewModel.swift
          CheckoutViewModel.swift
          ConnectSupplierViewModel.swift
          CreditPartnersViewModel.swift
          DashboardViewModel.swift
          DeliveryMapViewModel.swift
          DeliveryPaymentViewModel.swift
          FamilyMembersViewModel.swift
          FileClaimViewModel.swift
          FutureDemandViewModel.swift
          HistoryViewModel.swift
          HqViewModel.swift
          InsightsViewModel.swift
          LocalSkusViewModel.swift
          LocationPickerViewModel.swift
          LocationsViewModel.swift
          LoginViewModel.swift
          MySuppliersViewModel.swift
          OnboardingViewModel.swift
          OrderDetailViewModel.swift
          OrdersViewModel.swift
          PlanogramViewModel.swift
          PosViewModel.swift
          ProcurementViewModel.swift
          ProductDetailViewModel.swift
          ProfileViewModel.swift
          SearchViewModel.swift
          ShopClosedBannerViewModel.swift
          StoreStockViewModel.swift
          SupplierProductsViewModel.swift
        ContentView.swift
        L10n.swift
        retailerappApp.swift
      Views/
        Dock/
          ShelfAlertModal.swift
  supplier-app-ios/
    SupplierApp/
      Design/
        CellApi.swift
        CollapsibleSidebar.swift
        DeliveryActivityAttributes.swift
        DeliveryLiveActivityWidget.swift
        FlexibleStatusChips.swift
        MarketPack.swift
        PegasusEmptyView.swift
        PegasusMonochromeTheme.swift
        PegasusMotion.swift
        PegasusStateViews.swift
        RealtimeRefresh.swift
        StatusStack.swift
      Layout/
        ResponsiveGridContentWrapper.swift
        SupplierAdaptiveShell.swift
        SupplierSection.swift
        SupplierSectionHeader.swift
      Models/
        FoundrySandboxModels.swift
        SupplierModels.swift
        SupplierOpsModels.swift
      Services/
        APIClient.swift
        AppDelegate.swift
        GeocodeService.swift
        NetworkLifecycleEngine.swift
        PushNotificationManager.swift
        SupplierOperationsService.swift
        SupplierRealtimeClient.swift
        SupplierRealtimeHub.swift
        SupplierService.swift
        SupplierService+Foundry.swift
        SupplierSessionReconcile.swift
        TokenStore.swift
      supplierapp/
        Services/
          AutoUpdater.swift
          EnterpriseUpdateConfig.swift
      Theme/
        SupplierTheme.swift
      Utilities/
        FleetDriverMarkerAnimator.swift
        ForecastConfidenceSupport.swift
        SupplierIdempotencyKeys.swift
        SupplierPortalFeature.swift
        SupplierPortalLinks.swift
      ViewModels/
        CatalogDetailAndCreateViewModels.swift
        CatalogViewModel.swift
        DashboardViewModel.swift
        DispatchPreviewViewModel.swift
        FleetViewModel.swift
        InventoryViewModel.swift
        MoreAndNotificationsViewModels.swift
        NetworkViewModels.swift
        OnboardingViewModel.swift
        OperationsViewModel.swift
        OpsAndManifestsViewModels.swift
        OrderDetailViewModel.swift
        OrdersViewModel.swift
        PlanningAndExceptionsViewModels.swift
        SettingsViewModels.swift
        SupplierShellAndAuthViewModels.swift
        TreasuryDetailViewModels.swift
        TreasuryViewModel.swift
      Views/
        Activity/
          ActivityView.swift
        AI/
          AIRecommendationsView.swift
        Analytics/
          AnalyticsView.swift
          DemandHistoryView.swift
        Auth/
          BusinessSetupView.swift
          RegisterView.swift
        Catalog/
          CatalogBarcodeField.swift
          CatalogCreateSheet.swift
          CatalogDetailView.swift
          CatalogList.swift
          CatalogProductImageButton.swift
          CatalogView.swift
        Components/
          AddressLocationField.swift
          ClientPolicyBanner.swift
          KpiTile.swift
          SupplierStateViews.swift
        Dashboard/
          DashboardView.swift
          NetworkPulseStrip.swift
        Dispatch/
          DispatchPreviewMapView.swift
          DispatchPreviewView.swift
        Earnings/
          EarningsView.swift
        Exceptions/
          ClaimsView.swift
          EarlyCompleteView.swift
          ExceptionsList.swift
          ExceptionsView.swift
          NegotiationsView.swift
          ShopClosedView.swift
        Fleet/
          FleetLiveMapView.swift
          FleetOrdersView.swift
          FleetRowViews.swift
          FleetView.swift
        Foundry/
          FoundrySandboxView.swift
          FoundrySandboxViewModel.swift
        Inventory/
          InventoryImportView.swift
          InventoryList.swift
          InventoryView.swift
        Manifests/
          ManifestDetailView.swift
          ManifestExceptionsView.swift
          ManifestsList.swift
          ManifestsView.swift
        More/
          CreditAdminDisableView.swift
          JSONFeedView.swift
          MoreHubView.swift
          PlaybooksView.swift
          ScoredExceptionsView.swift
        Network/
          CRMView.swift
          DeliveryZonesList.swift
          DeliveryZonesView.swift
          EntityResolutionView.swift
          FactoriesView.swift
          FactoryList.swift
          GeoReportLanesList.swift
          GeoReportView.swift
          LoyaltyProgramView.swift
          SupplyLaneRows.swift
          SupplyLanesList.swift
          SupplyLanesView.swift
          TopologyCenteredEmptyState.swift
          TopologyNodeForms.swift
          TopologyView.swift
          WarehouseList.swift
          WarehousesView.swift
        Notifications/
          NotificationInboxView.swift
        Operations/
          OperationsView.swift
          OperatorBroadcastView.swift
          PaymentBypassView.swift
          ReplenishmentActionView.swift
        Orders/
          OrderDetailPanel.swift
          OrderRow.swift
          OrdersHubView.swift
          OrdersList.swift
          OrdersView.swift
        OrgFleet/
          CreateDriverSheet.swift
          CreateOrgMemberSheet.swift
          CreateVehicleSheet.swift
          DriverListView.swift
          EditOrgMemberSheet.swift
          MetricsOverviewView.swift
          OrgFleetUtils.swift
          OrgFleetView.swift
          OrgMemberListView.swift
          VehicleListView.swift
        Planning/
          CreateOverrideForm.swift
          ForecastConfidenceView.swift
          KnowledgeGraphView.swift
          PlanningBrainView.swift
          PlanningSettingsView.swift
          ReplenishmentPoliciesView.swift
          SeasonalOverridesList.swift
        Portal/
          PortalHandoffView.swift
        Pricing/
          PricingView.swift
          ProductPricingDetailView.swift
          RetailerOverridesView.swift
        Profile/
          ProfileView.swift
          SupplierIdentityCard.swift
        Promotions/
          PromotionsView.swift
        Returns/
          ReturnsView.swift
        Settings/
          NotificationPreferencesView.swift
          ReturnPolicySettingsView.swift
        Treasury/
          CashReconciliationsView.swift
          ChargebacksView.swift
          ClaimChargebacksView.swift
          ComplianceAuditView.swift
          CreditNotesListView.swift
          CreditProfilesView.swift
          LedgerView.swift
          PaymentsView.swift
          PayoutsView.swift
          ReconciliationView.swift
          RoutePerformanceListView.swift
          TreasuryHubView.swift
        BillingGateView.swift
        LoginView.swift
      L10n.swift
      PegasusSupplierApp.swift
      RootView.swift
  warehouse-app-ios/
    WarehouseApp/
      Components/
        Dispatch/
          DispatchDriverList.swift
        Inventory/
          InventoryStockList.swift
        Orders/
          OrderLineItems.swift
          OrderOpsActions.swift
        Tactical/
          TacticalActivityTimeline.swift
          TacticalGaugeCard.swift
          TacticalInspectorView.swift
          TacticalKpiCard.swift
          TacticalLedProgressBar.swift
          TacticalStageStepper.swift
          TacticalStatusBadge.swift
      Design/
        CollapsibleSidebar.swift
        DeliveryActivityAttributes.swift
        DeliveryLiveActivityWidget.swift
        PegasusFeedbackViews.swift
        PegasusMonochromeTheme.swift
        PegasusMotion.swift
        PegasusStateViews.swift
        RealtimeRefresh.swift
      Layout/
        FlexibleStatusChips.swift
        ResponsiveGridContentWrapper.swift
        ResponsiveGridView.swift
        StatusStack.swift
        TacticalWorkbenchView.swift
        WarehouseAdaptiveShell.swift
        WarehouseSection.swift
        WarehouseSectionHeader.swift
      Models/
        FoundryCashierModels.swift
        WarehouseLocationModels.swift
        WarehouseModels.swift
        WarehouseOpsModels.swift
      Services/
        APIClient.swift
        AppDelegate.swift
        CellApi.swift
        FirebaseAuthHelper.swift
        GeocodeService.swift
        MarketPack.swift
        NetworkLifecycleEngine.swift
        PushNotificationManager.swift
        TokenStore.swift
        WarehouseOperationsService.swift
        WarehouseRealtimeClient.swift
        WarehouseRealtimeHub.swift
        WarehouseService.swift
        WarehouseService+Foundry.swift
        WarehouseSessionReconcile.swift
      Theme/
        PegasusTheme.swift
        WarehouseTacticalTheme.swift
      Utilities/
        FleetDriverMarkerAnimator.swift
        GeocodeLocationSupport.swift
        JwtPayload.swift
        WarehouseIdempotency.swift
        WarehousePortalFeature.swift
        WarehousePortalLinks.swift
      ViewModels/
        AddressLocationFieldViewModel.swift
        AdjustInventoryViewModel.swift
        AnalyticsViewModel.swift
        BarcodeScannerViewModel.swift
        CashierViewModel.swift
        ClaimsViewModel.swift
        CoverageViewModel.swift
        CRMViewModel.swift
        DashboardViewModel.swift
        DemandForecastViewModel.swift
        DispatchSettingsViewModel.swift
        DispatchViewModel.swift
        DriversViewModel.swift
        ExceptionsViewModel.swift
        FleetLiveMapViewModel.swift
        InventoryViewModel.swift
        LaborCapacityViewModel.swift
        LocationSettingsViewModel.swift
        LocationSetupViewModel.swift
        LoginViewModel.swift
        ManifestsViewModel.swift
        MoreHubFeedsViewModel.swift
        NotificationInboxViewModel.swift
        OperationsViewModel.swift
        OpsSettingsViewModel.swift
        OrderDetailViewModel.swift
        OrdersViewModel.swift
        PaymentConfigViewModel.swift
        PreordersViewModel.swift
        ProductsViewModel.swift
        ReplenishmentViewModel.swift
        RescuesViewModel.swift
        ReturnPolicySettingsViewModel.swift
        ReturnsViewModel.swift
        RootViewModel.swift
        StaffViewModel.swift
        StockCommitmentsViewModel.swift
        SupplyRequestsViewModels.swift
        TomorrowBoardViewModel.swift
        TransferActionsViewModel.swift
        TreasuryViewModel.swift
        VehicleDetailViewModel.swift
        VehiclesViewModel.swift
        YardGateViewModel.swift
      Views/
        Analytics/
          AnalyticsChartGrid.swift
          AnalyticsView.swift
        Cashier/
          CashierBlindCountView.swift
        Claims/
          ClaimsView.swift
        Components/
          AddressLocationField.swift
          ClientPolicyBanner.swift
          ForecastConfidenceView.swift
          KpiTile.swift
          WarehouseFeedbackViews.swift
          WarehouseStateViews.swift
        Coverage/
          CoverageView.swift
        CRM/
          CRMView.swift
        Dashboard/
          DashboardView.swift
        DemandForecast/
          DemandForecastView.swift
          ForecastChartPanel.swift
          ForecastSkuTable.swift
        Dispatch/
          DispatchOrderList.swift
          DispatchPreviewMapView.swift
          DispatchSettingsView.swift
          DispatchSupportViews.swift
          DispatchView.swift
        Drivers/
          DriversList.swift
          DriversView.swift
        Exceptions/
          ExceptionsView.swift
        Fleet/
          FleetLiveMapSection.swift
          FleetLiveMapView.swift
        Inventory/
          InventoryView.swift
          LocationSettingsView.swift
          OpsSettingsForm.swift
          OpsSettingsView.swift
          ReturnPolicySettingsView.swift
        LaborCapacity/
          LaborCapacityView.swift
        Manifests/
          ManifestsView.swift
        More/
          MoreHubView.swift
          WarehouseJSONFeedView.swift
          WarehouseScoredExceptionsView.swift
        Notifications/
          NotificationInboxView.swift
        Operations/
          OperationsBroadcastForm.swift
          OperationsPricingPreview.swift
          OperationsView.swift
        Orders/
          OrderDetailView.swift
          OrdersList.swift
          OrdersView.swift
        PaymentConfig/
          PaymentConfigView.swift
        Portal/
          PortalHandoffView.swift
        Preorders/
          PreordersList.swift
          PreordersView.swift
        Products/
          ProductsView.swift
        Replenishment/
          ReplenishmentList.swift
          ReplenishmentView.swift
        Rescues/
          RescuesView.swift
        Returns/
          ReturnsList.swift
          ReturnsView.swift
        Scanner/
          BarcodeScannerScreen.swift
        Setup/
          LocationSetupView.swift
        Staff/
          StaffList.swift
          StaffView.swift
        StockCommitments/
          StockCommitmentsView.swift
        SupplyRequests/
          CreateSupplyRequestSheet.swift
          SupplyRequestDetailView.swift
          SupplyRequestsHubView.swift
        TomorrowBoard/
          TomorrowBoardView.swift
        Transfers/
          TransferActionsView.swift
        Treasury/
          TreasuryTransactionList.swift
          TreasuryView.swift
        Vehicles/
          VehicleDetailPanels.swift
          VehicleDetailView.swift
          VehiclesList.swift
          VehiclesView.swift
        Yard/
          YardGateView.swift
        LoginView.swift
      warehouseapp/
        Services/
          AutoUpdater.swift
          EnterpriseUpdateConfig.swift
      L10n.swift
      PegasusWarehouseApp.swift
      RootView.swift
```

# Files

## File: apps/control-tower-macos/Sources/Services/ControlTowerHaptics.swift
```swift
public enum ControlTowerHaptics {
public static func alignment() {
⋮----
public static func levelChange() {
⋮----
public static func generic() {
```

## File: apps/driver-app-ios/driverapp/driverapp/Services/AutoUpdater.swift
```swift
enum AutoUpdater {
private static let defaultManifestURL: URL = URL(
⋮----
static func checkForUpdates(updateURL: String? = nil) {
⋮----
private static func openStoreListing() {
⋮----
private static func promptInstall(manifestURL: URL) {
⋮----
private static func triggerOTAInstall(manifestURL: URL) {
```

## File: apps/driver-app-ios/driverapp/driverapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/driver-app-ios/driverapp/Views/Home/RescueAcceptanceDialog.swift
```swift
struct RescueAcceptanceDialog: View {
let targetDriverId: String
let ordersToRescue: Int
let location: String
let onAccept: () -> Void
let onDecline: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Database/OfflineDeliveryStore.swift
```swift
final class OfflineDeliveryStore {
⋮----
private let modelContext: ModelContext
⋮----
init(modelContext: ModelContext) {
⋮----
func enqueue(orderId: String, signature: String, status: String) {
⋮----
func fetchPending() -> [OfflineDelivery] {
⋮----
func fetchAll() -> [OfflineDelivery] {
⋮----
func delete(_ delivery: OfflineDelivery) {
⋮----
func deleteSynced(orderIds: [String]) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/DriverNotification.swift
```swift
struct DriverNotification: Identifiable, Decodable {
let id: String
let type: String
let title: String
let body: String
let payload: String
let channel: String
let readAt: String?
let createdAt: String
let handoffMetadata: HandoffCardMetadata?
⋮----
var isUnread: Bool { readAt == nil }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DriverNotificationsResponse: Decodable {
let notifications: [DriverNotification]
let unreadCount: Int
let hasMore: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/LineItem.swift
```swift
enum LineItemStatus: String, Codable {
⋮----
struct LineItem: Codable, Identifiable {
let line_item_id: String
let sku_id: String
let quantity: Int
let unit_price: Int
var status: LineItemStatus
⋮----
var id: String { line_item_id }
⋮----
var lineTotal: Int { quantity * unit_price }
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/Mission.swift
```swift
struct Mission: Codable, Identifiable, Hashable {
let order_id: String
let state: String
let target_lat: Double
let target_lng: Double
let amount: Int
let gateway: String
let estimated_arrival_at: String?
let route_id: String?
let sequence_index: Int?
⋮----
var id: String { order_id }
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/OfflineDelivery.swift
```swift
class OfflineDelivery {
var orderId: String
var signature: String
var timestamp: Double
var status: String
var syncStatus: String
⋮----
init(orderId: String, signature: String, timestamp: Double, status: String) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/PendingCollection.swift
```swift
struct PendingCollection: Codable, Hashable, Identifiable {
let orderId: String
let retailerId: String
let amount: Int64
let state: String
let updatedAt: String
⋮----
var id: String { orderId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(orderId: String, retailerId: String, amount: Int64, state: String, updatedAt: String) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/PulseModels.swift
```swift
struct PulseEvent: Decodable, Identifiable {
let id: String
let kind: String
let title: String
let description: String?
let occurredAt: String
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PulseResponse: Decodable {
let events: [PulseEvent]
let fetchedAt: String
let unreadCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct StatusExplain: Decodable, Equatable {
let code: String
let title: String
let summary: String
let nextSteps: [String]?
let deepLink: String?
let recoverable: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct HandoffCardMetadata: Decodable, Equatable {
let kind: String
let title: String
let subtitle: String?
let primaryCta: String?
let primaryLink: String?
let entityType: String?
let entityId: String?
let fields: [String: String]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ApiErrorBody: Decodable {
let error: String?
let detail: String?
let message: String?
let explain: StatusExplain?
⋮----
enum ApiExplainParser {
static func parse(from data: Data) -> (message: String, explain: StatusExplain?)? {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/QRPayload.swift
```swift
struct QRPayload: Codable {
let order_id: String
let token: String
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/RouteGeometry.swift
```swift
struct RouteCoordinate: Codable {
let lat: Double
let lng: Double
⋮----
struct RouteStep: Codable {
let instruction: String
let distanceM: Double
let durationS: Double
let maneuver: String?
let lat: Double
let lng: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RouteGeometryResponse: Codable {
let routeId: String
let encodedPolyline: String?
let coordinates: [RouteCoordinate]
let source: String
let stopCount: Int
let steps: [RouteStep]
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/RouteManifest.swift
```swift
struct RouteManifest: Codable {
let driver_id: String
let date: String
let expires_at: TimeInterval
let hashes: [String: String]
⋮----
var isValid: Bool { Date().timeIntervalSince1970 < expires_at }
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/TelemetryPayload.swift
```swift
struct TelemetryPayload: Codable {
let driver_id: String
let latitude: Double
let longitude: Double
let accuracy: Double?
let timestamp: Double
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/AppDelegate.swift
```swift
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
static let syncTaskId = "com.pegasus.driver.sync"
⋮----
func application(
⋮----
func applicationDidEnterBackground(_ application: UIApplication) {
⋮----
func application(
⋮----
func application(
⋮----
nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
⋮----
private static func scheduleSyncTask() {
⋮----
private static func handleSyncTask(_ task: BGAppRefreshTask) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/AutoUpdater.swift
```swift
enum AutoUpdater {
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
⋮----
private static var lastManifest: Manifest?
⋮----
static func evaluate(
⋮----
static func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
⋮----
static func triggerOTAInstall(manifestURL: URL) {
⋮----
static func openStoreListing(force: Bool = false) {
⋮----
private static func fetchManifest(from urlString: String) async -> Manifest? {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/DriverLiveActivityManager.swift
```swift
final class DriverLiveActivityManager {
static let shared = DriverLiveActivityManager()
⋮----
private var currentActivity: Activity<DeliveryActivityAttributes>?
private var activeOrderId: String?
⋮----
private init() {}
⋮----
func startNavigationActivity(
⋮----
func updateNavigationState(
⋮----
func endActivity(status: String = "COMPLETED") {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/DriverWsRefresh.swift
```swift
enum DriverWsRefresh {
static func shouldRefreshManifest(eventType: String) -> Bool {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/driver-app-ios/driverappios/driverappios/Services/FirebaseAuthHelper.swift
```swift
func sendPhoneVerification(phone: String) async throws {
⋮----
func verifySmsCode(_ code: String) async throws -> String {
⋮----
func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
⋮----
func signOut() {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/ManifestService.swift
```swift
protocol ManifestServiceProtocol {
⋮----
func downloadManifest(bearerToken: String) async throws -> RouteManifest
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/MediaUploadService.swift
```swift
enum MediaUploadService {
struct Ticket: Decodable {
let uploadURL: String
let publicURL: String
let contentType: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
static func uploadJPEG(
⋮----
static func uploadJPEGData(
⋮----
static func savePodJPEG(_ data: Data, prefix: String) throws -> String {
⋮----
static func readLocalJPEG(_ path: String) throws -> Data {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/ProfileService.swift
```swift
final class ProfileService {
static let shared = ProfileService()
⋮----
private var pollingTask: Task<Void, Never>?
private let baseInterval: UInt64 = 60_000_000_000
private var consecutiveFailures = 0
⋮----
private init() {}
⋮----
func startPolling() {
⋮----
func stopPolling() {
⋮----
private func pollOnce() async {
⋮----
private func applyIfChanged(_ profile: DriverProfileResponse) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
func userNotificationCenter(
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/SyncService.swift
```swift
struct SyncResult: Codable {
let status: String
let processed: [String]
let skipped: Int
⋮----
struct SyncDeliveryDTO: Sendable {
let orderId: String
let signature: String
let timestamp: Double
let status: String
⋮----
protocol SyncServiceProtocol {
⋮----
func uploadBatch(driverId: String, deliveries: [SyncDeliveryDTO], bearerToken: String) async throws -> SyncResult
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/TelemetryService.swift
```swift
protocol TelemetryServiceProtocol: AnyObject {
⋮----
func connect(url: URL) async
⋮----
func send(payload: TelemetryPayload)
⋮----
func disconnect()
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/TelemetryServiceLive.swift
```swift
final class TelemetryServiceLive: TelemetryServiceProtocol {
⋮----
static let shared = TelemetryServiceLive()
⋮----
private var webSocketTask: URLSessionWebSocketTask?
private var pingTask: Task<Void, Never>?
private var reconnectTask: Task<Void, Never>?
private let session = URLSession(configuration: .default)
private let encoder = JSONEncoder()
⋮----
private(set) var isConnected: Bool = false
private var reconnectURL: URL?
private var reconnectAttempt: Int = 0
private var intentionalDisconnect = false
private static let maxReconnectDelay: UInt64 = 60_000_000_000
private static let baseReconnectDelay: UInt64 = 5_000_000_000
private static let pingInterval: UInt64 = 30_000_000_000
⋮----
deinit {
⋮----
func connect(url: URL) async {
⋮----
private func establishConnection(url: URL) async {
⋮----
private func startPingLoop() {
⋮----
private func sendPing() async -> Bool {
⋮----
private func handleDisconnect() {
⋮----
private func scheduleReconnect() {
⋮----
func send(payload: TelemetryPayload) {
⋮----
func disconnect() {
⋮----
private func disconnect(intentional: Bool) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/DriverReconnectRecovery.swift
```swift
enum DriverReconnectRecovery {
static let hint = "Connection restored — verify status before retrying."
⋮----
static func recoverInFlight(wasInFlight: Bool) async {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/HandoffPathResolver.swift
```swift
enum HandoffDestination: Equatable {
⋮----
enum HandoffPathResolver {
static func resolve(_ link: String) -> HandoffDestination {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/Haptics.swift
```swift
enum Haptics {
static func heavy()            { UIImpactFeedbackGenerator(style: .heavy).impactOccurred() }
static func medium()           { UIImpactFeedbackGenerator(style: .medium).impactOccurred() }
static func light()            { UIImpactFeedbackGenerator(style: .light).impactOccurred() }
static func soft()             { UIImpactFeedbackGenerator(style: .soft).impactOccurred() }
static func success()          { UINotificationFeedbackGenerator().notificationOccurred(.success) }
static func error()            { UINotificationFeedbackGenerator().notificationOccurred(.error) }
static func warning()          { UINotificationFeedbackGenerator().notificationOccurred(.warning) }
static func selectionChanged() { UISelectionFeedbackGenerator().selectionChanged() }
static func selection()        { selectionChanged() }
static func maneuver() {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/Haversine.swift
```swift
func haversineDistance(from: CLLocationCoordinate2D, to: CLLocationCoordinate2D) -> CLLocationDistance {
⋮----
func offsetCoordinate(
⋮----
func formattedDistance(_ meters: Double) -> String {
⋮----
var degreesToRadians: Double { self * .pi / 180 }
var radiansToDegrees: Double { self * 180 / .pi }
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/LocationInterpolator.swift
```swift
final class LocationInterpolator {
private struct Sample {
let coordinate: CLLocationCoordinate2D
let bearing: CLLocationDirection
let speedMps: CLLocationSpeed
let time: Date
⋮----
private(set) var displayCoordinate: CLLocationCoordinate2D?
private(set) var displayBearing: CLLocationDirection = 0
private(set) var displaySpeedMps: CLLocationSpeed = 0
⋮----
private var from: Sample?
private var to: Sample?
⋮----
func onGps(_ location: CLLocation) {
⋮----
func tick(now: Date = .now) {
⋮----
func clear() {
⋮----
private func publish(_ sample: Sample) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/MapCameraConfig.swift
```swift
enum MapCameraConfig {
static let idleRecenterMs: UInt64 = 8_000_000_000
static let interpolationFrameNs: UInt64 = 33_000_000
static let cameraAnimationSeconds = 0.5
static let minZoomDistance: Double = 400
static let maxZoomDistance: Double = 2_500
static let navigationPitch: Double = 60
static let speedZoomFactor = 20.0
static let lookAheadSeconds = 2.0
static let maxLookAheadMeters = 200.0
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/NavigationVoiceAnnouncer.swift
```swift
final class NavigationVoiceAnnouncer {
private let synthesizer = AVSpeechSynthesizer()
⋮----
func announce(_ cue: NavigationCue) {
⋮----
func stop() {
⋮----
private func activateVoicePromptSession() {
⋮----
func shouldAnnounceManeuverAdvance(previousIndex: Int, nextIndex: Int) -> Bool {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/PlannedRoute.swift
```swift
func buildPlannedRouteCoordinates(orders: [Order], activeRouteId: String?) -> [CLLocationCoordinate2D] {
⋮----
func resolveActiveRouteId(orders: [Order]) -> String? {
⋮----
enum MapCameraMath {
static func lookAheadDistance(speedMps: CLLocationSpeed) -> Double {
⋮----
static func trackingCamera(
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/RemainingStops.swift
```swift
struct RemainingStop: Equatable, Identifiable {
let id: String
let title: String
let state: OrderState
let sequenceIndex: Int
var firstClass: Bool { RemainingStops.isFirstClass(state) }
⋮----
enum RemainingStops {
static func isFirstClass(_ state: OrderState) -> Bool {
⋮----
static func remaining(_ orders: [Order]) -> [RemainingStop] {
⋮----
static func remaining(from inputs: [RemainingStop]) -> [RemainingStop] {
⋮----
struct MoneyHealthCounts: Equatable {
var pendingCash: Int
var openFiscal: Int
var creditLeave: Int
⋮----
static func from(orders: [Order]) -> MoneyHealthCounts {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/RouteDeviation.swift
```swift
enum RouteDeviationAction {
⋮----
enum RouteDeviationConfig {
static let offRouteThresholdMeters: CLLocationDistance = 45
static let sustainedOffRouteSeconds: TimeInterval = 4
static let minRerouteIntervalSeconds: TimeInterval = 30
⋮----
func distanceToPolylineMeters(
⋮----
func distancePointToSegmentMeters(
⋮----
final class RouteDeviationTracker {
private var offRouteSince: Date?
private var lastRerouteAt: Date?
⋮----
func evaluate(
⋮----
func reset() {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/RouteNavigation.swift
```swift
struct NavigationCue: Equatable {
let instruction: String
let distanceM: Double
let maneuver: String?
⋮----
enum RouteNavigation {
static let stepPassedMeters: CLLocationDistance = 35
⋮----
static func advanceStepIndex(
⋮----
static func resolveCue(
⋮----
static func formatDistance(_ meters: Double) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/SHA256Helper.swift
```swift
func sha256Hex(_ input: String) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/CashCollectionViewModel.swift
```swift
enum CashFiscalPhase {
⋮----
final class CashCollectionViewModel {
⋮----
let orderId: String
let amount: Int
⋮----
var isCompleting = false
var errorMessage: String?
var phase: CashFiscalPhase = .collect
var amountReceivedText: String = ""
var shortfallNote: String?
⋮----
private let fleetService: FleetServiceProtocol
⋮----
init(orderId: String, amount: Int, fleetService: FleetServiceProtocol = FleetServiceLive.shared) {
⋮----
var receivedAmountDisplay: Int {
⋮----
func refreshShortfallNote() {
⋮----
func handleSocketEvent(_ event: DriverSocketState.DriverEvent?) {
⋮----
func completeWithCash() async {
⋮----
func retryFiscal() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/CorrectionViewModel.swift
```swift
final class CorrectionViewModel {
⋮----
var lineItems: [LineItem] = []
var rejectionReasons: [String: RejectionReason] = [:]
var isLoading = false
var showConfirmation = false
var isSubmitting = false
var submitError: String?
var evidencePhotoURL = ""
var isUploadingPhoto = false
var previewImage: UIImage?
⋮----
private let fleetService: FleetServiceProtocol
⋮----
// MARK: - Init
⋮----
convenience init() {
⋮----
init(fleetService: FleetServiceProtocol) {
⋮----
// MARK: - Computed
⋮----
var rejectedCount: Int {
⋮----
var originalTotal: Int {
⋮----
var adjustedTotal: Int {
⋮----
var refundDelta: Int {
⋮----
var hasRejections: Bool { rejectedCount > 0 }
⋮----
var needsPhotoProof: Bool {
⋮----
// MARK: - Actions
⋮----
func handlePickedPhoto(_ item: PhotosPickerItem?, orderId: String) async {
⋮----
func uploadEvidence(image: UIImage, orderId: String) async {
⋮----
func loadLineItems(orderId: String) async {
⋮----
func toggleStatus(for itemId: String) {
⋮----
func reason(for itemId: String) -> RejectionReason {
⋮----
func setReason(_ reason: RejectionReason, for itemId: String) {
⋮----
func startTransitForPartialOrder(orderId: String) async -> Bool {
⋮----
func submitAmendment(orderId: String, driverId: String) async -> Bool {
⋮----
// G1.C: skip mid-delivery update (no durable writer). Missing-items / amend is SoT.
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/DriverHomeViewModel.swift
```swift
final class DriverHomeViewModel {
var showNotificationInbox = false
var showSupplyTransfers = false
var showRescueSheet = false
var showSyncQueue = false
var showQRScanner = false
var showOfflineVerifier = false
var selectedRecentMission: Mission?
⋮----
var showHandoffAlert = false
var handoffAlertMessage: String?
⋮----
var pulseEvents: [PulseEvent] = []
var pulseLoading = true
var pulseError: String?
var offlinePending = 0
⋮----
private let fetchPulse: @Sendable () async throws -> PulseResponse
⋮----
init(
⋮----
func loadPulse() async {
⋮----
func refreshOfflinePending(container: ModelContainer) {
⋮----
func handleHandoffLink(_ link: String, onNavigate: ((HandoffDestination) -> Void)?) {
⋮----
func clearHandoffAlert() {
⋮----
func greetingText(for date: Date = Date(), calendar: Calendar = .current) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/DriverNotificationInboxViewModel.swift
```swift
private struct NotificationEmptyResponse: Decodable {}
⋮----
final class DriverNotificationInboxViewModel {
var items: [DriverNotification] = []
var unreadCount: Int = 0
var isLoading = true
var errorMessage: String?
⋮----
private let api: APIClient
⋮----
init(api: APIClient = .shared) {
⋮----
func load() async {
⋮----
private static func message(for error: Error) -> String {
⋮----
func markRead(_ id: String) async {
struct Payload: Encodable { let notification_ids: [String] }
⋮----
func markAllRead() async {
struct Payload: Encodable { let mark_all: Bool }
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/FleetMapViewModel.swift
```swift
final class FleetMapViewModel {
var navPath = NavigationPath()
var cameraPosition: MapCameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
var isCameraLocked: Bool = false
var userPannedAt: Date?
⋮----
var phase: MapPhase = .pickingOrder
var selectedMission: Mission?
var zoomFocus: ZoomFocus = .both
var validatedQR: ValidateQRResponse?
var scannedQRToken: String = ""
var offloadResponse: ConfirmOffloadResponse?
var showRescueSheet = false
⋮----
func resetNavigationAndReturnToPicker() {
⋮----
func handleValidatedQR(_ response: ValidateQRResponse, token: String) {
⋮----
func handleConfirmedOffload(_ result: ConfirmOffloadResponse) {
⋮----
func handleIdleRecenter(for pannedAt: Date?) async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/LoginViewModel.swift
```swift
enum LoginMode {
⋮----
final class LoginViewModel {
var mode: LoginMode = .otp
var phone: String = "+998"
var otpCode: String = ""
var pin: String = ""
var otpSent: Bool = false
var loading: Bool = false
var error: String?
⋮----
func setMode(_ next: LoginMode) {
⋮----
func setPin(_ value: String) {
⋮----
func setOtp(_ value: String) {
⋮----
func sendOtp() async {
⋮----
func verifyOtp() async {
⋮----
func submitPin() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/MidRouteRescueViewModel.swift
```swift
final class MidRouteRescueViewModel {
⋮----
var originalRouteId: String = "route-tashkent-042"
var brokenVehicleId: String = "vh-isuzu-01"
var originalDriverId: String = "driver-uz-09"
var rescueVehicleId: String = "vh-isuzu-04"
var rescueDriverId: String = "driver-uz-12"
var lastCompletedStopSeq: Int64 = 2
var newRescueBoltSeal: String = "SEAL-RESCUE-8812"
⋮----
var totes: [RescueToteItem] = [
⋮----
var isExecuting: Bool = false
var rescueResult: MidRouteRescueResult? = nil
var statusMessage: String? = nil
var errorMessage: String? = nil
⋮----
private let apiClient: APIClient
⋮----
init(apiClient: APIClient = .shared) {
⋮----
var originalDriverLiability: Int64 {
⋮----
var remainingTotes: [RescueToteItem] {
⋮----
var spoiledTotes: [RescueToteItem] {
⋮----
var transferredTotes: [RescueToteItem] {
⋮----
var rescueDriverLiability: Int64 {
⋮----
func executeRescue() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/OfflineVerifierViewModel.swift
```swift
enum VerificationState: Equatable {
⋮----
final class OfflineVerifierViewModel {
⋮----
var state: VerificationState = .idle
private var manifest: RouteManifest?
private var scanLocked = false
⋮----
private let manifestService: ManifestServiceProtocol
private let syncService: SyncServiceProtocol
var store: OfflineDeliveryStore?
⋮----
convenience init() {
⋮----
init(manifestService: ManifestServiceProtocol, syncService: SyncServiceProtocol = SyncServiceLive.shared) {
⋮----
var statusLabel: String {
⋮----
var statusColor: Color {
⋮----
func syncManifest() async {
⋮----
// Also attempt to sync any pending offline deliveries
⋮----
func activateScanner() {
⋮----
func cancelScanner() {
⋮----
func resetTerminal() {
⋮----
func nextDelivery() {
⋮----
// MARK: - Handle Barcode Scan
⋮----
func handleBarcodeScan(_ stringValue: String) {
⋮----
// Unlock after 2 seconds
⋮----
// Parse QR
⋮----
// Lookup order in manifest
⋮----
func syncPendingDeliveries() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/OffloadReviewViewModel.swift
```swift
final class OffloadReviewViewModel {
let response: ValidateQRResponse
let scannedToken: String
⋮----
var rejectedQty: [String: Int] = [:]
var rejectionReasons: [String: RejectionReason] = [:]
var customReasons: [String: String] = [:]
var isSubmitting = false
var errorMessage: String?
var pickedPhoto: PhotosPickerItem?
var previewImage: UIImage?
var evidencePhotoURL = ""
var evidencePhotoLocalPath = ""
var signatureURL = ""
var signatureLocalPath = ""
var isUploadingPhoto = false
⋮----
private let fleetService: FleetServiceProtocol
private let reportMissing: @Sendable (String, [MissingItemRequest], String?) async throws -> Void
private let uploadPhotoJPEG: @Sendable (UIImage, String, String) async throws -> String
private let uploadSigJPEG: @Sendable (Data, String, String) async throws -> String
⋮----
init(
⋮----
var hasRejections: Bool {
⋮----
var needsPhotoProof: Bool {
⋮----
func selectedReason(for item: OrderLineItem) -> RejectionReason {
⋮----
func reasonLabel(for reason: RejectionReason) -> String {
⋮----
func decrementRejection(for item: OrderLineItem) {
⋮----
func incrementRejection(for item: OrderLineItem) {
⋮----
func setReason(_ reason: RejectionReason, for itemId: String) {
⋮----
func setCustomReason(_ text: String, for itemId: String) {
⋮----
func validateCreditLeave() -> (photoURL: String, signatureURL: String, photoLocal: String?, sigLocal: String?)? {
⋮----
func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
⋮----
func processSelectedImage(_ image: UIImage) async {
⋮----
func saveSignature(_ data: Data) async {
⋮----
func confirmOffload() async -> ConfirmOffloadResponse? {
⋮----
func handleReconnect() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/PaymentWaitingViewModel.swift
```swift
final class PaymentWaitingViewModel {
let orderId: String
let amount: Int
let driverId: String
⋮----
var isSettled = false
var isFiscalizing = false
var isFiscalFailed = false
var isCompleting = false
var isDone = false
var requiresCashCollection = false
var errorMessage: String?
⋮----
private let fleetService: FleetServiceProtocol
⋮----
init(
⋮----
var headerIcon: String {
⋮----
var headerTint: Color {
⋮----
var headerTitle: String {
⋮----
var subtitle: String {
⋮----
func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
⋮----
func finalizeDelivery() async {
⋮----
func retryFiscal() async {
⋮----
func handleReconnect() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/ProfileViewModel.swift
```swift
final class ProfileViewModel {
⋮----
var showOfflineVerifier = false
var showSyncQueue = false
var showEndSession = false
var showFleetDVIR = false
var showMidRouteRescue = false
var showYardGate = false
var pendingCount = 0
⋮----
private let queue: DriverOfflineQueue
⋮----
convenience init() {
⋮----
init(queue: DriverOfflineQueue) {
⋮----
func attach(container: ModelContainer) {
⋮----
func refreshPendingCount() {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/RequestRescueViewModel.swift
```swift
final class RequestRescueViewModel {
static let availableReasons = ["Engine Failure", "Flat Tire", "Accident", "Other"]
⋮----
var reason: String = "Engine Failure"
var note: String = ""
var isSubmitting = false
var errorMessage: String?
⋮----
private let fleetService: FleetServiceProtocol
⋮----
init(fleetService: FleetServiceProtocol = FleetServiceLive.shared) {
⋮----
func submitRescue() async -> Bool {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/RidesListViewModel.swift
```swift
final class RidesListViewModel {
var loadingMode = false
var showEarlyComplete = false
var isRequestingEarlyComplete = false
var earlyCompleteMessage: String?
⋮----
private let sendEarlyComplete: @Sendable (String, String) async throws -> Void
⋮----
init(
⋮----
func submitEarlyComplete(reason: String, note: String, reloadMissions: () async -> Void) async {
⋮----
func handleReconnect(reloadMissions: () async -> Void) async {
⋮----
func loadSeqLabel(for index: Int, total: Int) -> String? {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/RootViewModel.swift
```swift
struct DriverClientPolicyResponse: Decodable, Sendable {
let outdated: Bool
let forceUpdate: Bool
let updateDeferred: Bool
let minimumVersion: String
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class RootViewModel {
var clientPolicyMessage: String?
var clientPolicyForce = false
var pendingManifest: AutoUpdater.Manifest?
⋮----
private let fetchPolicy: @Sendable (String) async throws -> DriverClientPolicyResponse
⋮----
init(
⋮----
func dismissBanner() {
⋮----
func loadClientPolicy(currentVersion: String? = nil) async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/ScannerViewModel.swift
```swift
final class ScannerViewModel {
⋮----
var isProcessing = false
var alertTitle = ""
var alertMessage = ""
var showAlert = false
var scanSucceeded = false
var validatedResponse: ValidateQRResponse?
⋮----
private var scanLocked = false
private let fleetService: FleetServiceProtocol
⋮----
// MARK: - Init
⋮----
convenience init() {
⋮----
init(fleetService: FleetServiceProtocol) {
⋮----
// MARK: - Camera Permission
⋮----
func checkCameraPermission() async -> Bool {
⋮----
// MARK: - Handle QR Scan → Validate Only (no state change)
⋮----
func handleScan(_ stringValue: String, onValidated: @escaping (ValidateQRResponse, String) -> Void) {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/ShopClosedWaitingViewModel.swift
```swift
final class ShopClosedWaitingViewModel {
let orderId: String
let driverId: String
⋮----
var reported = false
var isReporting = false
var reportError: String?
var retailerResponse: String?
var bypassToken: String?
var bypassInput = ""
var isSubmittingBypass = false
var bypassError: String?
var isEscalated = false
var countdown: Int = 180
var pickedPhoto: PhotosPickerItem?
var photoURL = ""
var photoLocalPath = ""
var isUploadingPhoto = false
⋮----
private var timer: Timer?
private let sendShopClosed: @Sendable (String, String, String) async throws -> Void
private let sendBypass: @Sendable (String, String) async throws -> Void
private let uploadPhotoJPEG: @Sendable (UIImage, String, String) async throws -> String
⋮----
init(
⋮----
var statusIcon: String {
⋮----
var statusColor: Color {
⋮----
var statusTitle: String {
⋮----
var countdownFormatted: String {
⋮----
var hasPhoto: Bool {
⋮----
func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
⋮----
func reportShopClosed() async {
⋮----
func submitBypass() async -> Bool {
⋮----
func startCountdown() {
⋮----
func stopCountdown() {
⋮----
func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
⋮----
func handleReconnect() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/SupplyTransfersViewModel.swift
```swift
final class SupplyTransfersViewModel {
var transfers: [SupplyTransferRow] = []
var isLoading = false
var isArriving: String?
var errorMessage: String?
var successMessage: String?
⋮----
private let arriveableStates: Set<String> = ["IN_TRANSIT", "IN_TRANSIT_TO_WAREHOUSE", "DISPATCHED"]
⋮----
var activeCount: Int {
⋮----
func refresh() async {
⋮----
func canArrive(_ transfer: SupplyTransferRow) -> Bool {
⋮----
func markArrived(_ transfer: SupplyTransferRow) async {
⋮----
private func currentCoordinate() async -> CLLocationCoordinate2D {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/SyncQueueViewModel.swift
```swift
final class SyncQueueViewModel {
⋮----
var pending: [QueuedDriverAction] = []
var dead: [QueuedDriverAction] = []
var status: String?
var flushing: Bool = false
⋮----
private let queue: DriverOfflineQueue
⋮----
convenience init() {
⋮----
init(queue: DriverOfflineQueue) {
⋮----
func attach(container: ModelContainer) {
⋮----
func reload() {
⋮----
func dismissAllDead() {
⋮----
func flush() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/YardGateViewModel.swift
```swift
final class YardGateViewModel {
⋮----
var depotId: String = "depot-tashkent-south"
var supplierId: String = "supplier-tashkent-1"
var vehicleId: String = "vh-isuzu-01"
var licensePlate: String = "01 777 AAA"
var direction: String = "GATE_OUT"
var vehiclePurpose: String = "OUTBOUND_ROUTE"
var boltSealBarcode: String = "SEAL-UZ-44912"
var sealIntact: Bool = true
var assignedBaySlot: String = "OUTBOUND_LOADING"
⋮----
var isSubmitting: Bool = false
var recordedEvent: YardGateCheckEvent? = nil
var statusMessage: String? = nil
var errorMessage: String? = nil
⋮----
private let apiClient: APIClient
⋮----
init(apiClient: APIClient = .shared) {
⋮----
func updateBaySlot() {
⋮----
func submitGateCheck() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/FactorySupplyCard.swift
```swift
struct FactorySupplyCard: View {
@Binding var showSupplyTransfers: Bool
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/FieldMoneyStrip.swift
```swift
struct FieldMoneyStrip: View {
let counts: MoneyHealthCounts
⋮----
var body: some View {
⋮----
private func strip(_ label: String, _ count: Int) -> some View {
⋮----
struct ManifestBulletMeter: View {
let state: String?
let usedVU: Double?
let maxVU: Double
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/MapButton.swift
```swift
struct MapButton: View {
var pendingCount: Int
let onOpenMap: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/RemainingStopsStepper.swift
```swift
struct RemainingStopsStepper: View {
let stops: [RemainingStop]
var onSelect: ((String) -> Void)?
⋮----
var body: some View {
⋮----
private func dotColor(_ stop: RemainingStop) -> Color {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/TransitControlCard.swift
```swift
struct TransitControlCard: View {
var vm: FleetViewModel
⋮----
var body: some View {
⋮----
private func openWarehouseInMaps() {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/VehicleInfoCard.swift
```swift
struct VehicleInfoCard: View {
var vm: FleetViewModel
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/Correction/DeliveryCorrectionHeaderView.swift
```swift
struct DeliveryCorrectionHeaderView: View {
let orderId: String
let isPartial: Bool
let hasRejections: Bool
let rejectedCount: Int
let onClose: () -> Void
@Binding var showStartTransitAlert: Bool
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/Correction/DeliveryCorrectionLineItemCard.swift
```swift
struct DeliveryCorrectionLineItemCard: View {
let item: LineItem
let currentReason: RejectionReason
let onToggleStatus: () -> Void
let onSetReason: (RejectionReason) -> Void
⋮----
private func reasonLabel(for reason: RejectionReason) -> String {
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/Correction/DeliveryCorrectionSummaryBar.swift
```swift
struct DeliveryCorrectionSummaryBar: View {
let vm: CorrectionViewModel
@Binding var pickedPhoto: PhotosPickerItem?
@Binding var showConfirmAlert: Bool
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/DistanceInfo.swift
```swift
struct DistanceInfo: View {
let mission: Mission
let location: CLLocationCoordinate2D?
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/EarlyCompleteSheet.swift
```swift
struct EarlyCompleteSheet: View {
let onConfirm: (String, String) -> Void
@Environment(\.dismiss) private var dismiss
⋮----
@State private var selectedReason = "FATIGUE"
@State private var note = ""
⋮----
private let reasons: [(id: String, label: String)] = [
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/InfoChip.swift
```swift
struct InfoChip: View {
let icon: String
let text: String
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/ManifestEmptyView.swift
```swift
struct ManifestEmptyView: View {
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/ManifestHeader.swift
```swift
struct ManifestHeader: View {
@Binding var loadingMode: Bool
let pendingMissionsCount: Int
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/ManifestLoadingView.swift
```swift
struct ManifestLoadingView: View {
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Manifest/RideCard.swift
```swift
struct RideCard: View {
let mission: Mission
let index: Int
let loadSeqLabel: String?
let location: CLLocationCoordinate2D?
let onSelect: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/GlassSheet.swift
```swift
struct GlassSheet: View {
@Environment(\.colorScheme) private var cs
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/NavigationCueBanner.swift
```swift
struct NavigationCueBanner: View {
let cue: NavigationCue
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/OrderPickerSheet.swift
```swift
struct OrderPickerSheet: View {
var vm: FleetViewModel
@Binding var selectedMission: Mission?
@Binding var phase: MapPhase
@Binding var zoomFocus: ZoomFocus
var bottomInset: Double
var zoomTo: (ZoomFocus, Mission) -> Void
⋮----
var body: some View {
⋮----
private var loadingRow: some View {
⋮----
private var emptyRow: some View {
⋮----
private func pickerRow(_ mission: Mission, index: Int) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/SheetHandle.swift
```swift
struct SheetHandle: View {
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Offload/CollectCashView.swift
```swift
struct CollectCashView: View {
let orderId: String
let amount: Int
@Binding var amountReceivedText: String
let shortfallNote: String?
⋮----
var body: some View {
⋮----
private var receivedAmountDisplay: Int {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Offload/FiscalFailedView.swift
```swift
struct FiscalFailedView: View {
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Offload/FiscalizingView.swift
```swift
struct FiscalizingView: View {
let amount: Int
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Offload/OffloadActionFooter.swift
```swift
struct OffloadActionFooter: View {
let orderId: String
let hasRejections: Bool
let isSubmitting: Bool
let onShopClosed: ((String) -> Void)?
let onCreditDelivery: ((String) -> Void)?
let onReportMissing: ((String) -> Void)?
let onConfirm: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Offload/OffloadSummaryCard.swift
```swift
struct OffloadSummaryCard: View {
let retailerName: String
let totalAmount: Int
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Profile/DriverCard.swift
```swift
struct DriverCard: View {
var vm: FleetViewModel
⋮----
var body: some View {
⋮----
struct InfoTile: View {
let label: String
let value: String
let icon: String
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Profile/HistorySection.swift
```swift
struct HistorySection: View {
var vm: FleetViewModel
⋮----
var body: some View {
⋮----
struct HistoryRow: View {
let row: DriverHistoryRow
let index: Int
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Profile/ProfileHeader.swift
```swift
struct ProfileHeader: View {
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Profile/StatsSection.swift
```swift
struct StatsSection: View {
var vm: FleetViewModel
⋮----
var body: some View {
⋮----
private var totalValue: String {
⋮----
private var deliveries: String {
⋮----
struct StatCard: View {
let title: String
let value: String
let icon: String
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/ActiveRideBar.swift
```swift
struct ActiveRideBar: View {
@Environment(\.colorScheme) private var cs
let mission: Mission
let driverLocation: CLLocationCoordinate2D?
let onTap: () -> Void
⋮----
@State private var appeared = false
@State private var pulse = false
⋮----
private var distance: Double? {
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/AnimationModifiers.swift
```swift
struct PressableButtonStyle: ButtonStyle {
var scale: Double = 0.96
⋮----
func makeBody(configuration: Configuration) -> some View {
⋮----
static var pressable: PressableButtonStyle { PressableButtonStyle() }
static func pressable(scale: Double) -> PressableButtonStyle {
⋮----
static var slideUp: AnyTransition {
⋮----
static var fadeScale: AnyTransition {
⋮----
struct ShimmerModifier: ViewModifier {
@State private var phase: Double = 0
⋮----
func body(content: Content) -> some View {
⋮----
func shimmer() -> some View {
⋮----
struct StaggeredAppear: ViewModifier {
let index: Int
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var appeared = false
⋮----
func body(content: Content) -> some View {
⋮----
func staggeredAppear(index: Int) -> some View {
⋮----
struct GlowPulse: ViewModifier {
let color: Color
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var pulse = false
⋮----
func body(content: Content) -> some View {
⋮----
func glowPulse(color: Color) -> some View {
⋮----
struct LabCardModifier: ViewModifier {
func body(content: Content) -> some View {
⋮----
func labCard() -> some View {
⋮----
func labReadableWidth() -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/DriverEmptyView.swift
```swift
struct DriverEmptyView: View {
let icon: String
let title: String
let message: String
var actionTitle: String? = nil
var action: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/DriverLoadingStateCard.swift
```swift
struct DriverLoadingStateCard: View {
let title: String
let message: String
⋮----
var body: some View {
⋮----
private struct DriverStateSkeletonBar: View {
let width: CGFloat?
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/DriverStateCard.swift
```swift
struct DriverStateCard: View {
let icon: String
let title: String
let message: String
var actionTitle: String? = nil
var action: (() -> Void)? = nil
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var pulse = false
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/DriverStateViews.swift
```swift
struct DriverLoadingView: View {
let title: String
var message: String = "Syncing manifest, route, and delivery assignments."
⋮----
@State private var animating = false
⋮----
var body: some View {
⋮----
struct DriverErrorView: View {
let title: String
let message: String
let retry: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/DriverUiComponents.swift
```swift
struct DriverSectionHeader: View {
let title: String
var subtitle: String? = nil
var trailing: String? = nil
var trailingTint: Color = LabTheme.fg
⋮----
var body: some View {
⋮----
struct DriverStatusBadge: View {
let text: String
var tint: Color? = nil
var large = false
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/GPSErrorBanner.swift
```swift
struct GPSErrorBanner: View {
let message: String
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/HandoffInboxCard.swift
```swift
struct HandoffInboxCard: View {
let metadata: HandoffCardMetadata
var onAction: ((String) -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/KpiTile.swift
```swift
struct KpiTile: View {
let label: String
let value: String
var icon: String? = nil
var tint: Color = LabTheme.fg
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift
```swift
struct MapMarkerDetailSheet: View {
@Environment(\.colorScheme) private var cs
⋮----
let mission: Mission
let driverLocation: CLLocationCoordinate2D?
let isInRange: Bool
let onScan: () -> Void
let onCorrection: () -> Void
let onDismiss: () -> Void
⋮----
@State private var dragOffset: Double = 0
@State private var appeared = false
@State private var autoHideWorkItem: DispatchWorkItem?
⋮----
private var distance: Double? {
⋮----
var body: some View {
⋮----
private var headerSection: some View {
⋮----
private var endpointCard: some View {
⋮----
private var actionButtons: some View {
⋮----
private func startAutoHide() {
⋮----
private func cancelAutoHide() {
⋮----
private func dismiss() {
⋮----
private var bottomInset: Double {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/PegasusEmptyView.swift
```swift
public struct PegasusEmptyView: View {
public let title: String
public let message: String
⋮----
public init(title: String, message: String) {
⋮----
public var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/SignaturePadView.swift
```swift
struct SignaturePadView: View {
var onSave: (Data) -> Void
⋮----
@State private var strokes: [[CGPoint]] = []
@State private var current: [CGPoint] = []
⋮----
var body: some View {
⋮----
private func renderJPEG() -> Data? {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/SourceChip.swift
```swift
struct SourceChip: View {
let source: String
⋮----
var body: some View {
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/StatusPill.swift
```swift
struct StatusPill: View {
let label: String
let color: Color
@State private var appeared = false
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/TelemetryBadge.swift
```swift
struct TelemetryBadge: View {
let isLive: Bool
@State private var pulse = false
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/DeliveryCorrectionView.swift
```swift
struct DeliveryCorrectionView: View {
let orderId: String
let driverId: String
var isPartial: Bool = false
let onClose: () -> Void
let onAmended: () -> Void
⋮----
@State private var vm = CorrectionViewModel()
@State private var showConfirmAlert = false
@State private var showStartTransitAlert = false
@State private var pickedPhoto: PhotosPickerItem?
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/EndSessionView.swift
```swift
enum OfflineReason: String, CaseIterable {
⋮----
var label: String {
⋮----
var icon: String {
⋮----
struct EndSessionView: View {
@Bindable var vm: FleetViewModel
@Environment(\.dismiss) private var dismiss
⋮----
@State private var selectedReason: OfflineReason?
@State private var note = ""
⋮----
private var canConfirm: Bool {
⋮----
var body: some View {
⋮----
// Drag handle
⋮----
// Title
⋮----
private func reasonOption(_ reason: OfflineReason) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/MissionDetailSheet.swift
```swift
struct MissionDetailSheet: View {
@Environment(\.dismiss) private var dismiss
⋮----
let mission: Mission
let driverLocation: CLLocationCoordinate2D?
let isInRange: Bool
let distance: Double?
let onScan: () -> Void
let onCorrection: () -> Void
⋮----
@State private var showOutOfRangeAlert = false
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/MissionListCard.swift
```swift
struct MissionListCard: View {
⋮----
let missions: [Mission]
let isLoading: Bool
let driverLocation: CLLocationCoordinate2D?
let onSelect: (Mission) -> Void
let onRefresh: () -> Void
⋮----
var body: some View {
⋮----
private func missionRow(_ mission: Mission, index: Int) -> some View {
⋮----
private func distanceLabel(for mission: Mission) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/OffloadReviewView.swift
```swift
struct OffloadReviewView: View {
let response: ValidateQRResponse
let scannedToken: String
let driverId: String
let onConfirm: (ConfirmOffloadResponse) -> Void
let onCancel: () -> Void
var onShopClosed: ((String) -> Void)?
⋮----
var onCreditDelivery: ((String, String, String, String?, String?) -> Void)?
var onReportMissing: ((String) -> Void)?
⋮----
@State private var vm: OffloadReviewViewModel
@State private var driverSocketState = DriverSocketState.shared
⋮----
init(
⋮----
var body: some View {
⋮----
private struct OffloadLineItemRow: View {
let item: OrderLineItem
@Bindable var vm: OffloadReviewViewModel
@State private var customReasonText: String = ""
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/PaymentWaitingView.swift
```swift
struct PaymentWaitingView: View {
let orderId: String
let amount: Int
let driverId: String
let onCompleted: () -> Void
var onCashCollectionRequired: () -> Void = {}
⋮----
@State private var vm: PaymentWaitingViewModel
@State private var driverSocketState = DriverSocketState.shared
⋮----
init(
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/QRScannerView.swift
```swift
struct QRScannerView: View {
let onValidated: (ValidateQRResponse, String) -> Void
let onCancel: () -> Void
⋮----
@State private var vm = ScannerViewModel()
@State private var cameraPermissionDenied = false
⋮----
var body: some View {
⋮----
private var reticleOverlay: some View {
⋮----
struct QRCameraPreview: UIViewRepresentable {
let onScan: (String) -> Void
⋮----
func makeUIView(context: Context) -> UIView {
⋮----
func updateUIView(_ uiView: UIView, context: Context) {
⋮----
func makeCoordinator() -> Coordinator {
⋮----
class Coordinator: NSObject, AVCaptureMetadataOutputObjectsDelegate {
let onScan: (String) -> Void
var session: AVCaptureSession?
var previewLayer: AVCaptureVideoPreviewLayer?
⋮----
init(onScan: @escaping (String) -> Void) {
⋮----
func metadataOutput(
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/RequestRescueSheet.swift
```swift
struct RequestRescueSheet: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = RequestRescueViewModel()
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/ShopClosedWaitingView.swift
```swift
struct ShopClosedWaitingView: View {
let orderId: String
let driverId: String
let onResolved: () -> Void
let onCancel: () -> Void
⋮----
@State private var vm: ShopClosedWaitingViewModel
@State private var driverSocketState = DriverSocketState.shared
⋮----
init(
⋮----
var body: some View {
⋮----
private func retailerResponseView(_ response: String) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/SupplyTransfersView.swift
```swift
struct SupplyTransfersView: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = SupplyTransfersViewModel()
⋮----
var body: some View {
⋮----
private struct SupplyTransferRowView: View {
let transfer: SupplyTransferRow
let canArrive: Bool
let isArriving: Bool
let onArrive: () -> Void
⋮----
var body: some View {
⋮----
private var statusTint: Color {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/SyncQueueView.swift
```swift
struct SyncQueueView: View {
@Environment(\.modelContext) private var modelContext
@State private var vm = SyncQueueViewModel()
⋮----
var body: some View {
⋮----
private func actionRow(_ row: QueuedDriverAction, dead: Bool = false) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/L10n.swift
```swift
enum L10n {
static func format(_ key: String, _ args: CVarArg...) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/PegasusDriverApp.swift
```swift
struct LabDriverApp: App {
@State private var tokenStore = TokenStore.shared
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
⋮----
var body: some Scene {
⋮----
struct RootView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var driverSocketState = DriverSocketState.shared
@State private var rootVM = RootViewModel()
⋮----
var body: some View {
⋮----
private struct DriverOutdatedGate: View {
let message: String
let onSignOut: () -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/PegasusEmptyView.swift
```swift
public struct PegasusEmptyView: View {
public let title: String
public let message: String
⋮----
public init(title: String, message: String) {
⋮----
public var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/SourceChip.swift
```swift
struct SourceChip: View {
let source: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/factoryapp/Services/AutoUpdater.swift
```swift
final class AutoUpdater {
static let shared = AutoUpdater()
⋮----
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
⋮----
private(set) var lastManifest: Manifest?
⋮----
private init() {}
⋮----
func evaluate(
⋮----
func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
⋮----
func triggerOTAInstall(manifestURL: URL) {
⋮----
func openStoreListing(force: Bool = false) {
⋮----
private func fetchManifest(from urlString: String) async -> Manifest? {
```

## File: apps/factory-app-ios/FactoryApp/factoryapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/factory-app-ios/FactoryApp/Layout/FactorySection.swift
```swift
enum FactorySection: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var icon: String {
⋮----
static var compactTabs: [FactorySection] { [.dashboard, .loadingBay, .payloadLoad, .transfers] }
⋮----
static var primarySections: [FactorySection] {
⋮----
static var operationsSections: [FactorySection] {
⋮----
static var intelligenceSections: [FactorySection] {
⋮----
enum FactoryCompactTab: Hashable {
```

## File: apps/factory-app-ios/FactoryApp/Layout/ResponsiveGridContentWrapper.swift
```swift
struct ResponsiveGridContentWrapper<Content: View>: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@ViewBuilder let content: () -> Content
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Services/AppDelegate.swift
```swift
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
func application(
⋮----
func application(
⋮----
func application(
⋮----
nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
```

## File: apps/factory-app-ios/FactoryApp/Services/FactoryService.swift
```swift
enum FactoryService {
private static let api = APIClient.shared
⋮----
static func login(phone: String, password: String) async throws -> AuthResponse {
⋮----
static func login(idToken: String) async throws -> AuthResponse {
⋮----
static func register(body: [String: String]) async throws -> AuthResponse {
⋮----
static func refresh() async throws -> AuthResponse {
struct EmptyBody: Encodable {}
⋮----
static func setup(
⋮----
static func dashboard() async throws -> DashboardStats {
⋮----
static func profile() async throws -> FactoryProfile {
⋮----
static func analyticsOverview(from: String? = nil, to: String? = nil) async throws -> FactoryAnalyticsOverview {
⋮----
static func transfers(state: String? = nil, limit: Int = 50) async throws -> TransferListResponse {
⋮----
static func transfer(id: String) async throws -> Transfer {
⋮----
static func transitionTransfer(id: String, target: String) async throws -> Transfer {
⋮----
static func createTransfer(_ req: FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse {
⋮----
// MARK: - Loading Bay
static func loadingBayTransfers() async throws -> TransferListResponse {
⋮----
static func pulse() async throws -> FactoryPulseResponse {
⋮----
static func dispatch(transferIds: [String]) async throws -> DispatchResponse {
⋮----
static func supplyRequests() async throws -> [SupplyRequest] {
⋮----
static func acceptSupplyRequest(id: String, body: [String: String] = [:]) async throws {
⋮----
static func transitionSupplyRequest(id: String, action: String) async throws -> SupplyRequestTransitionResponse {
⋮----
static func supplyFulfillOptions(id: String) async throws -> SupplyFulfillOptions {
⋮----
static func supplyRequestQC(id: String) async throws -> SupplyRequestQCResponse {
⋮----
static func postSupplyRequestQC(id: String, result: String) async throws -> SupplyRequestQCResponse {
⋮----
static func loadingManifests() async throws -> ManifestListResponse {
⋮----
static func manifests() async throws -> ManifestListResponse {
⋮----
static func manifestDetail(id: String) async throws -> ManifestDetailSnapshot {
⋮----
static func transitionManifest(id: String, action: String) async throws -> FactoryManifestTransitionResponse {
struct EmptyBody: Encodable {}
⋮----
static func rebalanceManifestTransfer(sourceManifestId: String, targetManifestId: String, transferId: String) async throws -> ManifestRebalanceResponse {
⋮----
static func cancelManifestTransfer(manifestId: String, transferId: String) async throws -> ManifestCancelTransferResponse {
⋮----
static func cancelManifest(manifestId: String) async throws -> ManifestCancelResponse {
⋮----
static func fleet() async throws -> VehicleListResponse {
⋮----
static func fleetLiveMap() async throws -> FactoryFleetLiveMapResponse {
⋮----
static func fleetDrivers() async throws -> [FactoryFleetDriverRow] {
⋮----
static func fleetVehicles() async throws -> [FactoryFleetVehicleRow] {
⋮----
static func staff() async throws -> StaffListResponse {
⋮----
static func createStaff(name: String, role: String) async throws -> StaffMember {
⋮----
static func staffDetail(id: String) async throws -> StaffMember {
⋮----
static func setStaffPassword(id: String, pin: String) async throws {
struct Body: Encodable { let pin: String }
struct Resp: Decodable { let password_set: Bool? }
⋮----
static func insights() async throws -> InsightListResponse {
⋮----
static func manifestExceptions(escalatedOnly: Bool = false) async throws -> ManifestExceptionListResponse {
⋮----
static func resolveManifestException(
⋮----
static func factoryLocation() async throws -> FactoryLocationResponse {
⋮----
static func patchFactoryLocation(address: String, placeId: String?, lat: Double, lng: Double) async throws -> FactoryLocationResponse {
```

## File: apps/factory-app-ios/FactoryApp/Services/FirebaseAuthHelper.swift
```swift
func sendPhoneVerification(phone: String) async throws {
⋮----
func verifySmsCode(_ code: String) async throws -> String {
⋮----
func resetFlow() {
⋮----
func signOut() {
```

## File: apps/factory-app-ios/FactoryApp/Services/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
func userNotificationCenter(
```

## File: apps/factory-app-ios/FactoryApp/Services/TokenStore.swift
```swift
final class TokenStore {
static let shared = TokenStore()
⋮----
private(set) var token: String?
private(set) var refreshToken: String?
private(set) var factoryId: String?
⋮----
var isAuthenticated: Bool { token != nil }
⋮----
var isConfigured: Bool { JwtPayload.isConfigured(token: token) }
⋮----
var hasAssignedFactory: Bool { JwtPayload.homeNodeId(token: token) != nil }
⋮----
private let service = "com.pegasusx.factory-app"
⋮----
private init() {
⋮----
func store(auth: AuthResponse) {
⋮----
func updateTokens(token: String, refresh: String, factoryId: String? = nil) {
⋮----
func clear() {
⋮----
private func writeKeychain(account: String, value: String) {
⋮----
private func readKeychain(account: String) -> String? {
⋮----
private func deleteKeychain(account: String) {
```

## File: apps/factory-app-ios/FactoryApp/Utilities/FactoryIdempotency.swift
```swift
enum FactoryIdempotency {
private static func factoryId() -> String {
⋮----
static func startLoading(manifestId: String) -> String {
⋮----
static func seal(manifestId: String) -> String {
⋮----
static func dispatch(manifestId: String) -> String {
⋮----
static func complete(manifestId: String) -> String {
⋮----
static func batchDispatch(transferIds: [String]) -> String {
⋮----
static func rebalance(
⋮----
static func cancelTransfer(manifestId: String, transferId: String) -> String {
⋮----
static func cancelManifest(manifestId: String, reason: String = "") -> String {
⋮----
static func transferCreate(
⋮----
static func transferTransition(transferId: String, targetState: String) -> String {
⋮----
static func supplyRequestTransition(requestId: String, action: String) -> String {
⋮----
static func supplyRequestAccept(requestId: String) -> String {
⋮----
static func supplyRequestQC(requestId: String, result: String) -> String {
⋮----
static func opsLocation(lat: Double, lng: Double, placeId: String? = nil) -> String {
⋮----
static func forLifecycleAction(_ action: String, manifestId: String) -> String {
⋮----
private static func stableHash(_ input: String) -> String {
```

## File: apps/factory-app-ios/FactoryApp/Utilities/JwtPayload.swift
```swift
enum JwtPayload {
static func isConfigured(token: String?) -> Bool {
⋮----
static func homeNodeId(token: String?) -> String? {
⋮----
private static func decode(_ token: String?) -> [String: Any]? {
```

## File: apps/factory-app-ios/FactoryApp/ViewModels/AuthAndLocationViewModels.swift
```swift
enum LoginMode: Equatable {
⋮----
final class LoginViewModel {
var mode: LoginMode = .otp
var phone: String = "+998"
var otpCode: String = ""
var password: String = ""
var otpSent: Bool = false
var loading: Bool = false
var error: String?
⋮----
private let configureAuth: @MainActor () -> Void
private let resetAuthFlow: @MainActor () -> Void
private let sendVerification: @MainActor (String) async throws -> Void
private let verifyCode: @MainActor (String) async throws -> String
private let loginWithIdToken: @MainActor (String) async throws -> AuthResponse
private let loginWithPassword: @MainActor (String, String) async throws -> AuthResponse
⋮----
init(
⋮----
var canSendOtp: Bool {
⋮----
var canVerifyOtp: Bool {
⋮----
var canPasswordLogin: Bool {
⋮----
func configure() {
⋮----
func sanitizeOtp(_ raw: String) {
⋮----
func switchMode() {
⋮----
func resetForResend() {
⋮----
func sendOtp() async {
⋮----
func verifyOtp(onAuthenticated: (AuthResponse) -> Void) async {
⋮----
func passwordLogin(onAuthenticated: (AuthResponse) -> Void) async {
⋮----
final class LocationSetupViewModel {
var factoryName: String = ""
var location = AddressLocationValue()
var loading: Bool = false
var submitting: Bool = false
var error: String?
⋮----
private let fetchLocation: @MainActor () async throws -> FactoryLocationResponse
private let patchLocation: @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse
private let refreshAuth: @MainActor () async throws -> AuthResponse
private let setupFactory: @MainActor (String, String, String?, Double, Double) async throws -> FactorySetupResponse
private let resolveLocation: @MainActor (AddressLocationValue) async -> AddressLocationValue?
⋮----
init(
⋮----
func loadExistingIfNeeded(hasAssignedFactory: Bool) async {
⋮----
func submit(
⋮----
final class LocationSettingsViewModel {
var factoryName: String = ""
var location = AddressLocationValue()
var loading: Bool = true
var saving: Bool = false
var error: String?
var saveMessage: String?
⋮----
private let fetchLocation: @MainActor () async throws -> FactoryLocationResponse
private let patchLocation: @MainActor (String, String?, Double, Double) async throws -> FactoryLocationResponse
private let resolveLocation: @MainActor (AddressLocationValue) async -> AddressLocationValue?
⋮----
init(
⋮----
func load() async {
⋮----
func save() async {
⋮----
final class AddressLocationViewModel {
var query: String = ""
var suggestions: [GeocodePrediction] = []
var error: String?
var locating: Bool = false
var resolving: Bool = false
⋮----
private var searchTask: Task<Void, Never>?
⋮----
func syncAddress(_ address: String) {
⋮----
func cancelSearch() {
⋮----
func scheduleSearch(_ text: String) {
⋮----
private func search(_ text: String) async {
⋮----
func pick(placeId: String, fallback: String, current: AddressLocationValue) async -> AddressLocationValue {
⋮----
func resolveOnBlur(current: AddressLocationValue) async -> AddressLocationValue? {
⋮----
func useMyLocation() async -> AddressLocationValue? {
⋮----
private func resolveText(_ text: String, current: AddressLocationValue) async -> AddressLocationValue? {
⋮----
private func apply(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
```

## File: apps/factory-app-ios/FactoryApp/ViewModels/DashboardAndInsightsViewModels.swift
```swift
struct FactoryNotificationsPage: Decodable {
let notifications: [FactoryNotificationItem]
let unreadCount: Int
let hasMore: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryNotificationItem: Decodable, Identifiable {
let id: String
let type: String
let title: String
let body: String
let payload: String
let channel: String
let readAt: String?
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
private struct MarkNotificationsReadRequest: Encodable {
let markAll: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class DashboardViewModel {
var stats: DashboardStats = .empty
var loading: Bool = true
var error: String?
var clientPolicyMessage: String?
var clientPolicyForce: Bool = false
var pendingManifest: AutoUpdater.Manifest?
var showManifests: Bool = false
var showNotifications: Bool = false
var showCreateTransfer: Bool = false
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let refreshNanos: UInt64 = 60_000_000_000
private let fetchDashboard: @MainActor () async throws -> DashboardStats
⋮----
init(
⋮----
var dashboardMetrics: [DashboardMetric] {
⋮----
func load(silent: Bool = false) async {
⋮----
func runLifecycleLoop() async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
func dismissPolicyBanner() {
⋮----
func promptInstallUpdate() {
⋮----
func loadClientPolicy() async {
⋮----
struct ClientPolicy: Decodable {
let outdated: Bool
let forceUpdate: Bool
let updateDeferred: Bool?
let minimumVersion: String
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class AnalyticsViewModel {
var overview = FactoryAnalyticsOverview(
⋮----
var loading: Bool = true
var error: String?
⋮----
private let fetchAnalytics: @MainActor () async throws -> FactoryAnalyticsOverview
⋮----
init(
⋮----
func load() async {
⋮----
final class InsightsViewModel {
var insights: [Insight] = []
var loading: Bool = true
var error: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchInsights: @MainActor () async throws -> [Insight]
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class NotificationsViewModel {
var items: [FactoryNotificationItem] = []
var unreadCount: Int = 0
var loading: Bool = true
var error: String?
⋮----
private let fetchPage: @MainActor () async throws -> FactoryNotificationsPage
private let markAllReadAction: @MainActor () async throws -> Void
⋮----
init(
⋮----
func load() async {
⋮----
func markAllRead() async {
```

## File: apps/factory-app-ios/FactoryApp/ViewModels/ManifestsAndFleetViewModels.swift
```swift
final class PayloadOverrideViewModel {
var manifests: [Manifest] = []
var loading: Bool = true
var error: String?
⋮----
var moveCandidate: MoveCandidate?
var selectedTargetManifestId: String?
⋮----
var cancelTransferCandidate: CancelTransferCandidate?
var showReleaseAlert: Bool = false
⋮----
var cancelManifestCandidate: Manifest?
var showCancelManifestAlert: Bool = false
⋮----
var isProcessing: Bool = false
var processingError: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchLoadingManifests: @MainActor () async throws -> [Manifest]
private let rebalanceAction: @MainActor (String, String, String) async throws -> ManifestRebalanceResponse
private let cancelTransferAction: @MainActor (String, String) async throws -> ManifestCancelTransferResponse
private let cancelManifestAction: @MainActor (String) async throws -> ManifestCancelResponse
⋮----
init(
⋮----
func promptMove(manifestId: String, transfer: ManifestTransfer) {
⋮----
func dismissMove() {
⋮----
func promptRelease(manifestId: String, transfer: ManifestTransfer) {
⋮----
func clearReleaseAlert() {
⋮----
func promptCancelManifest(_ manifest: Manifest) {
⋮----
func clearCancelManifestAlert() {
⋮----
func load(silent: Bool = false) async {
⋮----
func moveTransfer(_ candidate: MoveCandidate, targetManifestId: String) async {
⋮----
func confirmReleaseTransfer() async {
⋮----
func confirmCancelManifest() async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class ManifestsViewModel {
var manifests: [Manifest] = []
var loading: Bool = true
var error: String?
var selectedManifestID: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchManifests: @MainActor () async throws -> [Manifest]
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class PayloadLoadViewModel {
var manifests: [Manifest] = []
var loading: Bool = true
var error: String?
var actingId: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchManifests: @MainActor () async throws -> [Manifest]
private let transitionAction: @MainActor (String, String) async throws -> FactoryManifestTransitionResponse
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func run(_ manifest: Manifest, _ action: ManifestLifecycleAction) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class ManifestDetailViewModel {
let manifestId: String
var detail: ManifestDetailSnapshot?
var loading: Bool = true
var acting: Bool = false
var error: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchDetail: @MainActor (String) async throws -> ManifestDetailSnapshot
private let transitionAction: @MainActor (String, String) async throws -> FactoryManifestTransitionResponse
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func runLifecycle(_ action: ManifestLifecycleAction) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class ManifestExceptionsViewModel {
var exceptions: [ManifestException] = []
var loading: Bool = true
var error: String?
var escalatedOnly: Bool = false
var resolvingId: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchExceptions: @MainActor (Bool) async throws -> [ManifestException]
private let resolveAction: @MainActor (String) async throws -> ResolveManifestExceptionResponse
⋮----
init(
⋮----
func toggleEscalated() {
⋮----
func load(silent: Bool = false) async {
⋮----
func resolve(_ exceptionId: String) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class FleetViewModel {
var vehicles: [Vehicle] = []
var liveRoutes: [FactoryFleetLiveRoute] = []
var loading: Bool = true
var error: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchVehicles: @MainActor () async throws -> [Vehicle]
private let fetchLiveRoutes: @MainActor () async throws -> [FactoryFleetLiveRoute]
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
```

## File: apps/factory-app-ios/FactoryApp/ViewModels/StaffAndSupplyViewModels.swift
```swift
final class StaffViewModel {
var staff: [StaffMember] = []
var loading: Bool = true
var error: String?
var showCreate: Bool = false
⋮----
var newStaffName: String = ""
var newStaffRole: String = "FACTORY_OPERATOR"
var submittingCreate: Bool = false
var createError: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchStaff: @MainActor () async throws -> [StaffMember]
private let createStaffAction: @MainActor (String, String) async throws -> StaffMember
⋮----
init(
⋮----
var canSubmitCreate: Bool {
⋮----
func presentCreate() {
⋮----
func load() async {
⋮----
func createStaff() async -> Bool {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class StaffDetailViewModel {
let staffId: String
var staff: StaffMember?
var loading: Bool = true
var error: String?
var pin: String = ""
var setMsg: String?
var setting: Bool = false
⋮----
private let fetchDetail: @MainActor (String) async throws -> StaffMember
private let setPasswordAction: @MainActor (String, String) async throws -> Void
⋮----
init(
⋮----
var canSetPassword: Bool {
⋮----
func load() async {
⋮----
func setPassword() async {
⋮----
struct SupplyRequestActionSpec: Equatable {
let title: String
let action: String
let emphasized: Bool
⋮----
func supplyRequestActions(for state: String) -> [SupplyRequestActionSpec] {
⋮----
struct SupplyFulfillModalItem: Identifiable {
let request: SupplyRequest
let options: SupplyFulfillOptions
var id: String { request.id }
⋮----
func handleAction(request: SupplyRequest, action: String) async {
⋮----
func confirmFulfill(_ item: SupplyFulfillModalItem) async {
⋮----
func handleQC(request: SupplyRequest, result: String) async {
⋮----
func availableActions(for state: String) -> [String] {
⋮----
func load() async {
⋮----
func performAction(_ action: String) async {
⋮----
func load() async {
```

## File: apps/factory-app-ios/FactoryApp/ViewModels/TransfersAndBayViewModels.swift
```swift
final class LoadingBayViewModel {
var transfers: [Transfer] = []
var loading: Bool = true
var error: String?
var dispatching: Bool = false
var handoffEvents: [FactoryPulseEvent] = []
var handoffLoading: Bool = true
var handoffError: String?
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchBayTransfers: @MainActor () async throws -> [Transfer]
private let fetchPulse: @MainActor () async throws -> [FactoryPulseEvent]
private let dispatchTransfers: @MainActor ([String]) async throws -> DispatchResponse
⋮----
init(
⋮----
var approved: [Transfer] { transfers.filter { $0.state == "APPROVED" } }
var loadingState: [Transfer] { transfers.filter { $0.state == "LOADING" } }
var dispatched: [Transfer] { transfers.filter { $0.state == "DISPATCHED" } }
⋮----
func load() async {
⋮----
func batchDispatch() async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class TransferListViewModel {
static let filters = [
⋮----
var transfers: [Transfer] = []
var loading: Bool = true
var error: String?
var selectedFilter: String = "ALL"
var selectedTransferID: String?
var showCreateTransfer: Bool = false
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchTransfers: @MainActor (String?) async throws -> [Transfer]
⋮----
init(
⋮----
var selectedTransfer: Transfer? {
⋮----
func applyInitialFilter(_ initialFilter: String?) {
⋮----
func load(silent: Bool = false) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class TransferDetailViewModel {
let transferId: String
var transfer: Transfer?
var loading: Bool = true
var error: String?
var transitioning: Bool = false
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchTransfer: @MainActor (String) async throws -> Transfer
private let transitionAction: @MainActor (String, String) async throws -> Transfer
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func transition(to target: String) async {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
⋮----
final class CreateTransferViewModel {
var loadingFleet: Bool = true
var submitting: Bool = false
var error: String?
var drivers: [FactoryFleetDriverRow] = []
var vehicles: [FactoryFleetVehicleRow] = []
var orderId: String = ""
var totalVu: String = "25"
var driverId: String = ""
var vehicleId: String = ""
⋮----
private let realtimeClient = FactoryRealtimeClient()
private let fetchDrivers: @MainActor () async throws -> [FactoryFleetDriverRow]
private let fetchVehicles: @MainActor () async throws -> [FactoryFleetVehicleRow]
private let createAction: @MainActor (FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse
⋮----
init(
⋮----
func loadFleet() async {
⋮----
func submit() async -> String? {
⋮----
func startRealtime() {
⋮----
func stopRealtime() {
```

## File: apps/factory-app-ios/FactoryApp/Views/Analytics/AnalyticsView.swift
```swift
struct AnalyticsView: View {
@State private var vm = AnalyticsViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Dashboard/DashboardHeroCard.swift
```swift
struct DashboardHeroCard: View {
let stats: DashboardStats
⋮----
var body: some View {
⋮----
struct OverviewMetric: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Dashboard/DashboardMetric.swift
```swift
struct DashboardMetric {
let title: String
let value: String
let supporting: String
let icon: String
let tint: Color
var chip: (text: String, tint: Color)? = nil
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Dashboard/WorkflowLaunchCard.swift
```swift
struct WorkflowLaunchCard: View {
let onOpenSupplyRequests: () -> Void
let onOpenPayloadOverride: () -> Void
let onOpenManifestExceptions: () -> Void
let onOpenManifests: () -> Void
let onOpenAnalytics: () -> Void
let onOpenCreateTransfer: () -> Void
let onOpenInsights: () -> Void
⋮----
var body: some View {
⋮----
struct WorkflowLaunchRow: View {
let title: String
let supporting: String
let actionLabel: String
let onTap: () -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Insights/InsightRow.swift
```swift
struct InsightRow: View {
let insight: Insight
⋮----
private var urgencyColor: Color {
⋮----
var body: some View {
⋮----
private func demandWhyText(_ insight: Insight) -> String? {
⋮----
private func number(from codable: AnyCodable?) -> Double? {
⋮----
private func string(from codable: AnyCodable?) -> String? {
⋮----
struct InsightMetricPill: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/LoadingBay/BaySection.swift
```swift
struct BaySection: View {
let title: String
let count: Int
let transfers: [Transfer]
let emptyMessage: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/LoadingBay/BayTransferCard.swift
```swift
struct BayTransferCard: View {
let transfer: Transfer
⋮----
var body: some View {
⋮----
struct BayTransferMetric: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/LoadingBay/FactoryHandoffTimelineSection.swift
```swift
struct FactoryHandoffTimelineSection: View {
let events: [FactoryPulseEvent]
let loading: Bool
var error: String? = nil
⋮----
private var subtitle: String {
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/LoadingBay/LoadingBayOverviewCard.swift
```swift
struct LoadingBayOverviewCard: View {
let readyCount: Int
let loadingCount: Int
let dispatchedCount: Int
⋮----
var body: some View {
⋮----
struct BayOverviewMetric: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/PayloadOverride/MetricBox.swift
```swift
struct MoveCandidate: Identifiable {
let sourceManifestId: String
let transfer: ManifestTransfer
⋮----
var id: String { "\(sourceManifestId):\(transfer.id)" }
⋮----
struct CancelTransferCandidate: Identifiable {
let manifestId: String
let transfer: ManifestTransfer
⋮----
var id: String { "\(manifestId):\(transfer.id)" }
⋮----
struct MetricBox: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/PayloadOverride/MoveTransferSheet.swift
```swift
struct MoveTransferSheet: View {
let candidate: MoveCandidate
let manifests: [Manifest]
@Binding var selectedTargetManifestId: String?
let isProcessing: Bool
let onCancel: () -> Void
let onMove: (String) -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/PayloadOverride/OverrideManifestCard.swift
```swift
struct OverrideManifestCard: View {
let manifest: Manifest
let isProcessing: Bool
let manifestsCount: Int
let onCancelManifest: () -> Void
let onMoveTransfer: (ManifestTransfer) -> Void
let onReleaseTransfer: (ManifestTransfer) -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/PayloadOverride/OverrideTransferRow.swift
```swift
struct OverrideTransferRow: View {
let transfer: ManifestTransfer
let isProcessing: Bool
let canMove: Bool
let onMove: () -> Void
let onRelease: () -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Staff/StaffListContent.swift
```swift
struct StaffListContent: View {
let staff: [StaffMember]
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Staff/StaffRow.swift
```swift
struct StaffRow: View {
let member: StaffMember
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/SupplyRequests/SupplyRequestRow.swift
```swift
struct SupplyRequestRow: View {
let request: SupplyRequest
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Transfers/TransferDetailCards.swift
```swift
struct TransferOverviewCard: View {
let transfer: Transfer
⋮----
var body: some View {
⋮----
struct TransferSummaryCard: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Transfers/TransferListSummary.swift
```swift
struct TransferListSummary: View {
let count: Int
let selectedFilter: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Transfers/TransferRow.swift
```swift
struct TransferRow: View {
let transfer: Transfer
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/Transfers/TransferRowMetric.swift
```swift
struct TransferRowMetric: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/FactoryInlineStateViews.swift
```swift
struct FactoryLoadingView: View {
let title: String
var message: String = "Fetching the latest factory data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
var body: some View {
⋮----
struct FactoryErrorView: View {
let message: String
let retry: () -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/FactoryRuntimeBanner.swift
```swift
struct FactoryRuntimeBanner: View {
let tone: FactoryRuntimeTone
let message: String
⋮----
var body: some View {
⋮----
private func paletteFor(_ tone: FactoryRuntimeTone) -> (icon: String, fill: Color, tint: Color) {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/FactoryStateViews.swift
```swift
enum FactoryStateKind {
⋮----
enum FactoryRuntimeTone {
⋮----
struct FactoryLoadingState: View {
let title: String
let message: String
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
var body: some View {
⋮----
struct FactoryStateView: View {
let kind: FactoryStateKind
let headline: String
let message: String
var actionTitle: String? = nil
var action: (() -> Void)? = nil
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
var body: some View {
⋮----
private func paletteFor(_ kind: FactoryStateKind) -> (icon: String, fill: Color, tint: Color) {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/FactoryStatusBadge.swift
```swift
struct FactoryStatusBadge: View {
let text: String
var tint: Color? = nil
var emphasized: Bool = true
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Exceptions/ManifestExceptionsView.swift
```swift
struct ManifestExceptionsView: View {
@State private var vm = ManifestExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Fleet/FleetView.swift
```swift
struct FleetView: View {
@State private var vm = FleetViewModel()
⋮----
var body: some View {
⋮----
private func resolvedDriverLabel(for route: FactoryFleetLiveRoute) -> String {
⋮----
private func resolvedCoordText(lat: Double?, lng: Double?) -> String {
```

## File: apps/factory-app-ios/FactoryApp/Views/Insights/InsightsView.swift
```swift
struct InsightsView: View {
@State private var vm = InsightsViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/LoadingBay/LoadingBayView.swift
```swift
struct LoadingBayView: View {
@State private var vm = LoadingBayViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Location/LocationSettingsView.swift
```swift
struct LocationSettingsView: View {
@State private var vm = LocationSettingsViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Manifests/ManifestDetailView.swift
```swift
struct ManifestDetailView: View {
let manifestId: String
@State private var vm: ManifestDetailViewModel
⋮----
init(manifestId: String) {
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Manifests/ManifestsView.swift
```swift
struct ManifestsView: View {
var showsCloseButton: Bool = false
@Environment(\.dismiss) private var dismiss
@State private var vm = ManifestsViewModel()
⋮----
var body: some View {
⋮----
struct ManifestRow: View {
let manifest: Manifest
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Manifests/PayloadLoadView.swift
```swift
struct PayloadLoadView: View {
@State private var vm = PayloadLoadViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Notifications/NotificationInboxView.swift
```swift
struct NotificationInboxView: View {
var showsCloseButton: Bool = false
@Environment(\.dismiss) private var dismiss
@State private var vm = NotificationsViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/PayloadOverride/PayloadOverrideView.swift
```swift
struct PayloadOverrideView: View {
@State private var vm = PayloadOverrideViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Production/ProductionKioskScreen.swift
```swift
struct ProductionKioskScreen: View {
let machineId: String = "MAC-123"
@State private var isJammed: Bool = false
@State private var currentStatus: String = "IDLE"
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Setup/LocationSetupView.swift
```swift
struct LocationSetupView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var vm = LocationSetupViewModel()
⋮----
private var hasAssignedFactory: Bool { tokenStore.hasAssignedFactory }
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Staff/StaffDetailView.swift
```swift
struct StaffDetailView: View {
let staffId: String
@State private var vm: StaffDetailViewModel
⋮----
init(staffId: String) {
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Staff/StaffView.swift
```swift
struct StaffView: View {
@State private var realtimeClient = FactoryRealtimeClient()
@State private var staff: [StaffMember] = []
@State private var loading = true
@State private var error: String?
@State private var showCreate = false
⋮----
var body: some View {
⋮----
private func load() {
⋮----
private struct CreateStaffSheet: View {
let onCreated: () -> Void
@Environment(\.dismiss) private var dismiss
@State private var name = ""
@State private var role = "FACTORY_OPERATOR"
@State private var submitting = false
@State private var error: String?
⋮----
var body: some View {
⋮----
private func create() {
⋮----
private struct RequestActionSpec {
let title: String
let action: String
let emphasized: Bool
⋮----
private func requestActions(for state: String) -> [RequestActionSpec] {
⋮----
private struct FulfillModalItem: Identifiable {
let request: SupplyRequest
let options: SupplyFulfillOptions
var id: String { request.id }
⋮----
private func handleAction(request: SupplyRequest, action: String) async {
⋮----
private func handleQC(request: SupplyRequest, result: String) async {
⋮----
private func modeButton(_ mode: String, label: String) -> some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/SupplyRequests/SupplyRequestDetailView.swift
```swift
struct SupplyRequestDetailView: View {
let requestId: String
⋮----
@State private var request: SupplyRequest?
@State private var loading = true
@State private var error: String?
@State private var actionError: String?
@State private var isProcessing = false
⋮----
var body: some View {
⋮----
private func headerSection(_ request: SupplyRequest) -> some View {
⋮----
private func actionSection(_ request: SupplyRequest) -> some View {
⋮----
private func detailsSection(_ request: SupplyRequest) -> some View {
⋮----
private func detailRow(_ label: String, value: String) -> some View {
⋮----
private func load() async {
⋮----
private func performAction(_ action: String) async {
⋮----
private func availableActions(for state: String) -> [String] {
```

## File: apps/factory-app-ios/FactoryApp/Views/SupplyRequests/SupplyRequestsHubView.swift
```swift
struct SupplyRequestsHubView: View {
@Environment(\.dismiss) private var dismiss
@State private var requests: [SupplyRequest] = []
@State private var loading = true
@State private var error: String?
@State private var selectedRequestID: String?
⋮----
var body: some View {
⋮----
private func load() async {
```

## File: apps/factory-app-ios/FactoryApp/Views/Transfers/CreateTransferView.swift
```swift
struct CreateTransferView: View {
@Environment(\.dismiss) private var dismiss
let onCreated: (String) -> Void
⋮----
@State private var vm = CreateTransferViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Transfers/TransferDetailView.swift
```swift
struct TransferDetailView: View {
let transferId: String
@State private var vm: TransferDetailViewModel
⋮----
init(transferId: String) {
⋮----
var body: some View {
⋮----
struct TransferDetailTag: View {
let text: String
var emphasized = true
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Transfers/TransferListView.swift
```swift
struct TransferListView: View {
var initialFilter: String? = nil
@State private var vm = TransferListViewModel()
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/FactoryMoreHubView.swift
```swift
struct FactoryMoreHubView: View {
var body: some View {
⋮----
private func sectionLink(_ section: FactorySection) -> some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/MainTabView.swift
```swift
enum AppTab: String, CaseIterable {
⋮----
var icon: String {
⋮----
struct MainTabView: View {
@State private var selectedTab: AppTab = .dashboard
@State private var pendingTransferFilter: String?
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/L10n.swift
```swift
enum L10n {
static func format(_ key: String, _ args: CVarArg...) -> String {
```

## File: apps/factory-app-ios/FactoryApp/PegasusFactoryApp.swift
```swift
struct LabFactoryApp: App {
@State private var tokenStore = TokenStore.shared
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
⋮----
var body: some Scene {
```

## File: apps/factory-app-ios/FactoryApp/RootView.swift
```swift
struct RootView: View {
@Environment(TokenStore.self) private var tokenStore
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/PegasusFeedbackViews.swift
```swift
public struct PegasusErrorView: View {
public let message: String
public let retry: () -> Void
⋮----
public init(message: String, retry: @escaping () -> Void) {
⋮----
public var body: some View {
⋮----
public struct PegasusEmptyView: View {
public let title: String
public let message: String
⋮----
public init(title: String, message: String) {
⋮----
public var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/StatusStackChips.swift
```swift
struct SourceChip: View {
let source: String
⋮----
var body: some View {
⋮----
struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
⋮----
struct FlowLayout: Layout {
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
```

## File: apps/payload-app-ios/payload-app-ios/Services/AppDelegate.swift
```swift
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
func application(
⋮----
func application(
⋮----
func application(
⋮----
nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
```

## File: apps/payload-app-ios/payload-app-ios/Services/AutoUpdater.swift
```swift
enum AutoUpdater {
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
⋮----
private static var lastManifest: Manifest?
⋮----
static func evaluate(
⋮----
static func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
⋮----
static func triggerOTAInstall(manifestURL: URL) {
⋮----
static func openStoreListing(force: Bool = false) {
⋮----
private static func fetchManifest(from urlString: String) async -> Manifest? {
```

## File: apps/payload-app-ios/payload-app-ios/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/payload-app-ios/payload-app-ios/Services/FirebaseAuthHelper.swift
```swift
func sendPhoneVerification(phone: String) async throws {
⋮----
func verifySmsCode(_ code: String) async throws -> String {
⋮----
func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
⋮----
func signOut() {
```

## File: apps/payload-app-ios/payload-app-ios/Services/OfflineQueue.swift
```swift
private enum PayloadOfflineHTTP {
static let statusPending = "PENDING"
⋮----
static func normalizeEndpoint(_ endpoint: String) -> String {
⋮----
static func isRetryable(_ code: Int) -> Bool {
⋮----
static func isSuccess(_ code: Int) -> Bool {
⋮----
final class QueuedActionModel {
@Attribute(.unique) var id: String
var endpoint: String
var method: String
var body: String
var createdAt: Double
var capturedLat: Double?
var capturedLng: Double?
var capturedAtMs: Double?
var attemptCount: Int
var lastError: String
var status: String
⋮----
init(
⋮----
func bodyForFlush() -> String {
⋮----
final class OfflineQueue {
static let shared = OfflineQueue()
⋮----
let container: ModelContainer
let context: ModelContext
⋮----
init(inMemory: Bool = false) {
⋮----
func read() -> [QueuedActionModel] {
⋮----
func enqueue(_ action: QueuedActionModel) {
⋮----
func flush(api: APIClient) async -> (Int, Int) {
```

## File: apps/payload-app-ios/payload-app-ios/Services/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
var onOpenPanel: (() -> Void)?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
func userNotificationCenter(
⋮----
func userNotificationCenter(
```

## File: apps/payload-app-ios/payload-app-ios/Services/TokenStore.swift
```swift
final class TokenStore {
static let shared = TokenStore()
⋮----
private let service = "com.pegasus.payload"
⋮----
private(set) var token: String?
private(set) var name: String?
private(set) var supplierId: String?
private(set) var warehouseId: String?
private(set) var warehouseName: String?
private(set) var firebaseToken: String?
private(set) var refreshToken: String?
⋮----
var isAuthenticated: Bool { token != nil }
⋮----
private init() {
⋮----
func saveSession(from resp: LoginResponse) {
⋮----
func updateTokens(token: String, refresh: String?) {
⋮----
func logout() {
⋮----
// MARK: - Keychain plumbing
⋮----
private enum Key: String, CaseIterable {
⋮----
private func read(_ key: Key) -> String? {
⋮----
private func readFromService(_ serviceName: String, account: String) -> String? {
⋮----
private func write(_ key: Key, value: String) {
⋮----
private func writeToService(_ serviceName: String, account: String, value: String) {
⋮----
private func delete(_ key: Key) {
⋮----
private func deleteFromService(_ serviceName: String, account: String) {
```

## File: apps/payload-app-ios/payload-app-ios/Utilities/HandoffPathResolver.swift
```swift
enum HandoffDestination: Equatable {
⋮----
enum HandoffPathResolver {
static func resolve(_ link: String) -> HandoffDestination {
```

## File: apps/payload-app-ios/payload-app-ios/Utilities/ManifestBoard.swift
```swift
enum ManifestBoard {
static let states = ["DRAFT", "LOADING", "SEALED", "DISPATCHED"]
⋮----
static func canonicalState(_ state: String?) -> String {
⋮----
static func isBoardState(_ state: String?) -> Bool {
⋮----
struct Column: Equatable {
let state: String
let trucks: [Truck]
⋮----
static func attach(trucks: [Truck], manifests: [Manifest]) -> [Truck] {
⋮----
static func group(_ trucks: [Truck]) -> [Column] {
⋮----
static func unassigned(_ trucks: [Truck]) -> [Truck] {
```

## File: apps/payload-app-ios/payload-app-ios/Utilities/PayloadIdempotency.swift
```swift
enum PayloadIdempotency {
static func key(action: String, entityId: String) -> String {
⋮----
static func recommendReassign(orderId: String) -> String {
⋮----
static func fleetReassign(orderIds: [String]) -> String {
⋮----
static func applyReassign(orderId: String, toDriverId: String) -> String {
⋮----
static func supplierStartLoading(manifestId: String) -> String {
⋮----
static func supplierInjectOrder(manifestId: String, orderId: String) -> String {
⋮----
static func sealCompleted(manifestIds: [String]) -> String {
⋮----
static func manifestException(manifestId: String, orderId: String) -> String {
⋮----
static func orderSeal(orderId: String) -> String {
⋮----
static func missingItems(orderId: String) -> String {
⋮----
static func inboundScan(barcode: String, sessionId: String) -> String {
⋮----
static func inboundConfirm(returnIds: [String], disposition: String) -> String {
```

## File: apps/payload-app-ios/payload-app-ios/ViewModels/ExceptionsViewModel.swift
```swift
final class ExceptionsViewModel {
var exceptions: [ManifestExceptionRow] = []
var loading = false
var error: String?
var markingDamaged = false
var damageReportedMessage: String?
var overflowAttempts = 0
var isEscalatedToDlq: Bool { overflowAttempts >= 3 }
⋮----
private let api: APIClient
⋮----
init(api: APIClient = .shared) {
⋮----
func loadExceptions() async {
⋮----
func reportDockDamage(orderId: String, manifestId: String) async {
```

## File: apps/payload-app-ios/payload-app-ios/ViewModels/InboundReturnsViewModel.swift
```swift
final class InboundReturnsViewModel {
enum Tab: String, CaseIterable, Sendable {
⋮----
var tab: Tab = .queue
var rows: [InboundReturnRow] = []
var history: [InboundReturnRow] = []
var loading = true
var error: String?
var barcode = ""
var sessionId: String?
var scanning = false
var selected = Set<String>()
var statusMessage: String?
var queuedScans = 0
var scannerEnabled = true
⋮----
private let api: APIClient
private let offlineQueue: OfflineQueue
⋮----
init(api: APIClient = .shared, offlineQueue: OfflineQueue = .shared) {
⋮----
func load() async {
⋮----
func ensureSession() async throws -> String {
⋮----
func handleScan(online: Bool) async {
⋮----
func enqueueOfflineScan(barcode: String) {
struct ScanQueueBody: Encodable {
let barcode: String
let qty: Int
let sessionId: String
enum CodingKeys: String, CodingKey {
⋮----
func confirm(disposition: String) async {
⋮----
func toggleSelect(_ id: String) {
```

## File: apps/payload-app-ios/payload-app-ios/ViewModels/LoginViewModel.swift
```swift
enum LoginMode: Equatable {
⋮----
final class LoginViewModel {
var mode: LoginMode = .otp
var phone: String = ""
var otpCode: String = ""
var pin: String = ""
var otpSent: Bool = false
var loading: Bool = false
var error: String?
⋮----
private let sendPhoneVerificationHandler: @MainActor (String) async throws -> Void
private let verifySmsCodeHandler: @MainActor (String) async throws -> String
private let loginWithIdTokenHandler: @MainActor (String) async throws -> LoginResponse
private let loginWithPinHandler: @MainActor (String, String) async throws -> LoginResponse
private let onAuthenticatedHandler: @MainActor (LoginResponse) -> Void
⋮----
init(
⋮----
var canSubmitPin: Bool {
⋮----
func setMode(_ next: LoginMode) {
⋮----
func setPin(_ value: String) {
⋮----
func setOtp(_ value: String) {
⋮----
func sendOtp() async {
⋮----
func verifyOtp() async {
⋮----
func submitPin() async {
```

## File: apps/payload-app-ios/payload-app-ios/ViewModels/RootViewModel.swift
```swift
final class RootViewModel {
⋮----
private(set) var clientPolicyMessage: String?
private(set) var clientPolicyForce = false
private(set) var pendingManifest: AutoUpdater.Manifest?
⋮----
private let fetchPolicy: PolicyFetcher
private let evaluatePolicy: PolicyEvaluator
private let promptInstallHandler: InstallPrompter
⋮----
init(
⋮----
func loadClientPolicy(versionOverride: String? = nil) async {
⋮----
func dismissBanner() {
⋮----
func triggerInstall() {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/ExplainAndHandoffCards.swift
```swift
struct ExplainStatusBanner: View {
let explain: StatusExplain?
var fallbackTitle: String?
var fallbackDetail: String?
⋮----
var body: some View {
⋮----
struct HandoffInboxCard: View {
let metadata: HandoffCardMetadata
var onAction: ((String) -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/HomeBannerOverlay.swift
```swift
struct HomeBannerOverlay: View {
@Bindable var viewModel: HomeViewModel
⋮----
var body: some View {
⋮----
struct InfoBanner: View {
let text: String
let tint: Color
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/KpiTile.swift
```swift
struct KpiTile: View {
let label: String
let value: String
var footer: AnyView? = nil
⋮----
init(label: String, value: String) {
⋮----
init<Footer: View>(label: String, value: String, @ViewBuilder footer: () -> Footer) {
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/OnlineDot.swift
```swift
struct OnlineDot: View {
let online: Bool
let queued: Int
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var pulseScale: CGFloat = 1
⋮----
var body: some View {
⋮----
private var statusLabel: String {
⋮----
private var accessibilityLabel: String {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/OrderChecklistRows.swift
```swift
struct OrderChip: View {
let order: LiveOrder
let selected: Bool
let sealed: Bool
let dispatchCode: String?
let onTap: () -> Void
⋮----
var body: some View {
⋮----
private var background: Color {
⋮----
struct ItemRow: View {
let checked: Bool
let enabled: Bool
let label: String
let quantity: Int
let onToggle: () -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/PayloadFeedbackViews.swift
```swift
struct PayloadErrorView: View {
let message: String
let retry: () -> Void
⋮----
var body: some View {
⋮----
struct PayloadEmptyView: View {
let title: String
let message: String
var variant: PayloadStateVariant = .manifest
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/PayloadLoadingViews.swift
```swift
struct PayloadLoadingView: View {
let title: String
var message: String = "Fetching the latest manifest data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/PostSealCountdownView.swift
```swift
struct PostSealCountdownView: View {
let orderId: String
let dispatchCode: String
let secondsLeft: Int
let reportingMissingItems: Bool
let onDismiss: () -> Void
let onReportMissingItems: () -> Void
⋮----
var body: some View {
⋮----
struct AllSealedSuccessView: View {
let dispatchCodes: [String: String]
let onStartNew: () -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/TruckHeader.swift
```swift
struct TruckHeader: View {
let truck: Truck
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Sheets/ExceptionReasonSheet.swift
```swift
struct ExceptionReasonSheet: View {
let orderId: String
let inFlight: Bool
let onCancel: () -> Void
let onSelect: (String) -> Void
⋮----
private let reasons: [(code: String, label: String, icon: String)] = [
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Sheets/InjectOrderSheet.swift
```swift
final class InjectOrderViewModel {
var orderId: String = ""
var showScanner: Bool = false
⋮----
var trimmedOrderId: String {
⋮----
var canSubmit: Bool {
⋮----
func toggleScanner() {
⋮----
func applyScannedBarcode(_ raw: String) {
⋮----
struct InjectOrderSheet: View {
let injecting: Bool
let onCancel: () -> Void
let onSubmit: (String) -> Void
@State private var form = InjectOrderViewModel()
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Sheets/ProductScannerSheet.swift
```swift
struct ProductScannerSheet: View {
@Bindable var viewModel: HomeViewModel
@Binding var isPresented: Bool
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Sheets/ReDispatchSheet.swift
```swift
struct ReDispatchSheet: View {
let orderId: String
let loading: Bool
let response: RecommendReassignResponse?
let reassigning: Bool
let onClose: () -> Void
let onPick: (String, Bool) -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/ExceptionsSheet.swift
```swift
struct ManifestExceptionsSheet: View {
@Bindable var viewModel: HomeViewModel
@State private var exceptionsViewModel = ExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/InboundReturnsList.swift
```swift
struct InboundReturnsList: View {
var rows: [InboundReturnRow]
var selectable: Bool
@Binding var selected: Set<String>
⋮----
var body: some View {
⋮----
private func inboundCard(_ row: InboundReturnRow, selectable: Bool) -> some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/InboundReturnsView.swift
```swift
struct InboundReturnsView: View {
var online: Bool = true
@Environment(\.dismiss) private var dismiss
@State private var viewModel = InboundReturnsViewModel()
⋮----
var body: some View {
⋮----
private func scanHeader(vm: InboundReturnsViewModel) -> some View {
⋮----
private func queueList(vm: InboundReturnsViewModel) -> some View {
⋮----
private func historyList(vm: InboundReturnsViewModel) -> some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/NotificationsSheet.swift
```swift
struct NotificationsSheet: View {
@Bindable var viewModel: HomeViewModel
⋮----
var body: some View {
⋮----
struct NotificationRow: View {
let item: NotificationItem
let onTap: () -> Void
var onHandoffAction: ((String) -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/OrderChecklistSection.swift
```swift
struct OrderChecklistSection: View {
@Bindable var viewModel: HomeViewModel
let onShowException: (String) -> Void
let onShowReDispatch: (String) -> Void
let onScanProduct: () -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/PayloadStateView.swift
```swift
enum PayloadStateVariant {
⋮----
enum PayloadStateTone {
⋮----
struct PayloadStateView: View {
let variant: PayloadStateVariant
let title: String
let message: String?
var detail: String? = nil
var compact = false
var tone: PayloadStateTone = .default
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var pulse = false
@State private var drift = false
@State private var spin = false
⋮----
private var accent: Color {
⋮----
private var shouldRotate: Bool {
⋮----
private var illustrationSize: CGFloat {
⋮----
var body: some View {
⋮----
private var illustration: some View {
⋮----
private struct TriangleShape: Shape {
func path(in rect: CGRect) -> Path {
```

## File: apps/payload-app-ios/payload-app-ios/Views/RecommendationRow.swift
```swift
struct RecommendationRow: View {
let rec: TruckRecommendation
let onPickComplete: () -> Void
let onPickPartial: () -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/RootView.swift
```swift
struct RootView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var viewModel = RootViewModel()
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/L10n.swift
```swift
enum L10n {
static func format(_ key: String, _ args: CVarArg...) -> String {
```

## File: apps/payload-app-ios/payload-app-ios/PegasusPayloadApp.swift
```swift
struct PegasusPayloadApp: App {
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
@State private var tokenStore = TokenStore.shared
⋮----
var body: some Scene {
```

## File: apps/payload-app-ios/payload-app-ios/ResponsiveGridContentWrapper.swift
```swift
struct ResponsiveGridContentWrapper<Content: View>: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@ViewBuilder let content: () -> Content
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payloadapp/payloadapp/Services/AutoUpdater.swift
```swift
enum AutoUpdater {
private static let defaultManifestURL = URL(
⋮----
static func checkForUpdates(updateURL: String? = nil) {
⋮----
private static func openStoreListing() {
⋮----
private static func promptInstall(manifestURL: URL) {
⋮----
private static func triggerOTAInstall(manifestURL: URL) {
```

## File: apps/payload-app-ios/payloadapp/payloadapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/AutoOrderExplainerCard.swift
```swift
struct AutoOrderExplainerCard: View {
var body: some View {
⋮----
private func explainerRow(num: String, text: String) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/AutoOrderGlobalToggleCard.swift
```swift
struct AutoOrderGlobalToggleCard: View {
@Binding var globalAutoOrder: Bool
let analyticsStartDate: String?
let onToggle: (Bool) -> Void
⋮----
@State private var toggleValue = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/AutoOrderHeaderCard.swift
```swift
struct AutoOrderHeaderCard: View {
let supplierCount: Int
let categoryCount: Int
let productCount: Int
let predictionCount: Int
⋮----
var body: some View {
⋮----
private func miniStat(value: String, label: String) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/AutoOrderOverridesSection.swift
```swift
struct OverrideItem: Identifiable {
let id: String
let label: String
let enabled: Bool
let hasHistory: Bool
let level: OverrideLevel
⋮----
enum OverrideLevel {
⋮----
var subtitle: String {
⋮----
struct AutoOrderOverridesSection: View {
let title: String
let icon: String
let items: [OverrideItem]
@Binding var localToggleStates: [String: Bool]
let onToggle: (OverrideItem, Bool) -> Void
⋮----
var body: some View {
⋮----
private struct OverrideToggleRow: View {
let item: OverrideItem
let externalValue: Bool
let onUserToggle: (Bool) -> Void
⋮----
@State private var toggleValue = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/AutoOrderPredictionsSection.swift
```swift
struct AutoOrderPredictionsSection: View {
let forecasts: [DemandForecast]
⋮----
var body: some View {
⋮----
private func confidenceColor(_ confidence: Double) -> Color {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AutoOrder/DemandSourceChips.swift
```swift
struct DemandSourceChips: View {
let sources: [String]?
⋮----
private var list: [String] {
⋮----
var body: some View {
⋮----
private func isPOS(_ code: String) -> Bool {
⋮----
private func label(for code: String) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogAllProductsGrid.swift
```swift
struct CatalogAllProductsGrid: View {
let products: [Product]
@Binding var selectedProduct: Product?
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogBrowseChips.swift
```swift
struct CatalogBrowseChips: View {
@Binding var browseMode: CatalogBrowseMode
var onNavigateToSuppliers: () -> Void
⋮----
var body: some View {
⋮----
func browseChip(_ title: String, selected: Bool, action: @escaping () -> Void) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogBrowseMode.swift
```swift
enum CatalogBrowseMode {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogNoResultsState.swift
```swift
struct CatalogNoResultsState: View {
let searchText: String
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogSearchBar.swift
```swift
struct CatalogSearchBar: View {
@Binding var searchText: String
@Binding var showFullSearch: Bool
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogSearchResults.swift
```swift
struct CatalogSearchResults: View {
let filteredProducts: [Product]
@Binding var selectedProduct: Product?
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Dashboard/ServiceTileViews.swift
```swift
struct ServiceTileView: View {
let title: String
let icon: String
let subtitle: String?
let height: Double
⋮----
var body: some View {
⋮----
struct ServiceTileSmall: View {
let title: String
let icon: String
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/CategoryBreakdownView.swift
```swift
struct CategoryBreakdownView: View {
let data: [CategorySpend]
⋮----
private var maxTotal: Int { data.map(\.total).max() ?? 1 }
⋮----
var body: some View {
⋮----
private func abbreviate(_ value: Int) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/DailySpendingChartView.swift
```swift
struct DailySpendingChartView: View {
let data: [RetailerDayExpense]
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/FlowLayout.swift
```swift
struct FlowLayout: Layout {
var spacing: CGFloat = 8
⋮----
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
⋮----
private func layout(proposal: ProposedViewSize, subviews: Subviews) -> (offsets: [CGPoint], size: CGSize) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/KPICard.swift
```swift
struct KPICard: View {
let title: String
let value: String
let subtitle: String
var isPositive: Bool = true
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/OrdersByStateView.swift
```swift
let stateColors: [String: Color] = [
⋮----
struct OrdersByStateView: View {
let data: [OrderStateCount]
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Insights/WeekdayPatternView.swift
```swift
struct WeekdayPatternView: View {
let data: [DayOfWeekPattern]
⋮----
var body: some View {
⋮----
private func abbreviate(_ value: Int) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Orders/ActiveOrderCard.swift
```swift
struct ActiveOrderCard: View {
let order: Order
let onDetails: () -> Void
let onQR: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Orders/AiPlannedCard.swift
```swift
struct AiPlannedCard: View {
let item: RetailerAIPrediction
let onConfirm: () -> Void
let onReject: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Orders/DeliveryProposalReviewSheet.swift
```swift
struct DeliveryProposalReviewSheet: View {
let order: Order
let isPending: Bool
let onAccept: () -> Void
let onReject: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Orders/OrderStatusTimeline.swift
```swift
struct OrderStatusTimeline: View {
let currentStep: Int
⋮----
private let steps = OrderStatus.timelineSteps
⋮----
var body: some View {
⋮----
private func dotColor(isCompleted: Bool, isActive: Bool) -> Color {
⋮----
private func labelColor(isCompleted: Bool, isActive: Bool) -> Color {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/EmpathyEngineSection.swift
```swift
struct EmpathyEngineSection: View {
@Binding var globalAutoOrder: Bool
@Binding var showHistoryAlert: Bool
var toggleAction: (Bool, Bool) async -> Void
⋮----
@State private var toggleValue = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/OrderHistoryLink.swift
```swift
struct OrderHistoryLink: View {
var orderCount: Int
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/PreferencesSection.swift
```swift
struct PreferencesSection: View {
@Binding var aiAutoOrder: Bool
@Binding var notificationsEnabled: Bool
var onAutoOrderToggle: (Bool) -> Void = { _ in }
⋮----
@State private var autoOrderToggleValue = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/SettingsComponents.swift
```swift
struct SettingsItem: Identifiable {
let id = UUID()
let icon: String
let title: String
let subtitle: String?
var view: String? = nil
⋮----
struct SettingsSectionView: View {
var title: String
var icon: String
var items: [SettingsItem]
⋮----
var body: some View {
⋮----
struct SectionLabel: View {
var title: String
var icon: String
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/SettingsRowViews.swift
```swift
struct SettingsRow: View {
var item: SettingsItem
⋮----
var body: some View {
⋮----
struct SettingsRowContent: View {
var item: SettingsItem
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/SettingsToggle.swift
```swift
struct SettingsToggle: View {
var icon: String
var title: String
var subtitle: String
var color: Color
@Binding var isOn: Bool
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/UserCard.swift
```swift
struct UserCard: View {
var displayName: String
var displayCompany: String
var userEmail: String?
var profilePhone: String
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Tracking/ActiveCountBadge.swift
```swift
struct ActiveCountBadge: View {
let count: Int
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Tracking/DriverMarker.swift
```swift
struct DriverMarker: View {
let isGreen: Bool
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Tracking/OrderInfoCard.swift
```swift
struct OrderInfoCard: View {
let order: TrackingOrder
let onDismiss: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Tracking/SupplierChip.swift
```swift
struct SupplierChip: View {
let name: String
let isSelected: Bool
let onTap: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/AnimatedNumberText.swift
```swift
struct AnimatedNumberText: View {
let value: Int
var font: Font = .headline
var color: Color = AppTheme.textPrimary
⋮----
var body: some View {
⋮----
struct AnimatedCurrencyText: View {
let value: Int64
var font: Font = .headline
var color: Color = AppTheme.accent
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/ErrorStateView.swift
```swift
struct ErrorStateView: View {
var title: String = "Failed to Load"
var message: String = "Could not load data. Check your connection and try again."
var icon: String = "wifi.exclamationmark"
var retryAction: (() async -> Void)?
⋮----
@State private var isRetrying = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/GradientHeaderCard.swift
```swift
struct GradientHeaderCard<Content: View>: View {
let title: String
let subtitle: String?
let icon: String?
let gradient: LinearGradient
let content: Content
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/KpiTile.swift
```swift
struct KpiTile: View {
let title: String
let value: String
let systemImage: String
let tint: Color
var chip: (text: String, tint: Color)? = nil
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/PegasusButton.swift
```swift
struct LabButtonStyle: ButtonStyle {
enum Variant {
⋮----
let variant: Variant
let fullWidth: Bool
⋮----
init(variant: Variant, fullWidth: Bool = false) {
⋮----
func makeBody(configuration: Configuration) -> some View {
⋮----
private var foregroundColor: Color {
⋮----
private var backgroundColor: Color {
⋮----
var asLinearGradient: LinearGradient {
⋮----
struct LabButton: View {
let title: String
let variant: LabButtonStyle.Variant
let icon: String?
let fullWidth: Bool
let action: () -> Void
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/PegasusCard.swift
```swift
struct LabCard<Content: View>: View {
let content: Content
⋮----
init(@ViewBuilder content: () -> Content) {
⋮----
var body: some View {
⋮----
struct LabCardWithHeader<Content: View>: View {
let title: String
let subtitle: String?
let icon: String?
let content: Content
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/PulseStripView.swift
```swift
enum PulseHonesty {
static let failed = "pulse_failed"
static let commandFailed = "control_tower_pulse_failed"
⋮----
struct Result<T> {
let events: [T]
let error: String?
⋮----
struct ObjectResult<T> {
let value: T?
let error: String?
⋮----
static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
⋮----
static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
⋮----
struct PulseStripView: View {
let events: [RetailerPulseEvent]
let loading: Bool
var error: String? = nil
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/RescueWarningBanner.swift
```swift
struct RescueWarningBanner: View {
let message: String?
let onDismiss: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/RetailerEmptyView.swift
```swift
struct RetailerEmptyView: View {
let title: String
let message: String
var systemImage: String = "tray"
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/RetailerSectionHeader.swift
```swift
struct RetailerSectionHeader: View {
let title: String
var subtitle: String? = nil
var icon: String? = nil
var count: Int? = nil
⋮----
var body: some View {
⋮----
struct RetailerStatusBadge: View {
let text: String
var tint: Color? = nil
var showsLiveDot: Bool = false
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/RetailerStateViews.swift
```swift
struct RetailerLoadingView: View {
let title: String
var message: String = "Fetching the latest retailer data."
⋮----
@State private var animating = false
⋮----
var body: some View {
⋮----
struct RetailerErrorView: View {
let message: String
let retry: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/SkeletonOrderViews.swift
```swift
struct SkeletonOrderCard: View {
var body: some View {
⋮----
struct SkeletonOrderList: View {
var count: Int = 4
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/SkeletonViews.swift
```swift
struct SkeletonProductCard: View {
var body: some View {
⋮----
struct SkeletonProductGrid: View {
var columns: Int = 2
var count: Int = 6
⋮----
private var gridColumns: [GridItem] {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Design/FlexibleStatusChips.swift
```swift
struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Layout/ResponsiveGridContentWrapper.swift
```swift
struct ResponsiveGridContentWrapper<Content: View>: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@ViewBuilder let content: () -> Content
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Layout/RetailerSection.swift
```swift
enum RetailerSection: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var desktopRoute: String {
⋮----
var icon: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/AuthModels.swift
```swift
struct LoginRequest: Encodable {
let phoneNumber: String
let password: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RegisterRequest: Encodable {
let phoneNumber: String
let password: String
let storeName: String
let ownerName: String
let addressText: String
let deliveryAddress: String?
let placeId: String?
let latitude: Double
let longitude: Double
let taxId: String?
let receivingWindowOpen: String?
let receivingWindowClose: String?
let accessType: String?
let storageCeilingHeightCM: Double?
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/CardModels.swift
```swift
struct RetailerCardToken: Identifiable, Codable {
var id: String { tokenId }
⋮----
let tokenId: String
let retailerId: String
let gateway: String
let cardLast4: String
let cardType: String
let isDefault: Bool
let isActive: Bool
let expiresAt: Date?
let createdAt: Date
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerCardsResponse: Codable {
let cards: [RetailerCardToken]
⋮----
struct PSPListing: Codable, Hashable {
let code: String
let status: String
let selectable: Bool
let nationalCards: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerPaymentCatalogResponse: Codable {
let currencyCode: String
let marketCode: String?
let catalog: [PSPListing]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CardInitiateRequest: Codable {
let gateway: String
⋮----
struct CardInitiateResponse: Codable {
let cardToken: String
let requiresOtp: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CardConfirmRequest: Codable {
let cardToken: String
let otpCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CardConfirmResponse: Codable {
let tokenId: String
let cardLast4: String
let cardType: String
let confirmed: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CardIdRequest: Codable {
let tokenId: String
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/ClaimModels.swift
```swift
struct RetailerClaimLine: Codable, Identifiable, Hashable {
var id: String { sku }
let sku: String
let quantity: Int64
let reason: String?
let unitPriceMinor: Int64?
let amountMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerClaimEvidence: Codable, Hashable {
let evidenceType: String
let uri: String
let mimeType: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerClaim: Codable, Identifiable, Hashable {
var id: String { claimId }
let claimId: String
let orderId: String
let claimType: String
let status: String
let description: String?
let amountMinor: Int64?
let currency: String?
let lineItems: [RetailerClaimLine]?
let createdAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerClaimsListResponse: Codable {
let claims: [RetailerClaim]
⋮----
struct ClaimEligibility: Codable, Hashable {
let eligible: Bool
let endsAt: String?
let windowHours: Int
let hoursRemaining: Double
let policySource: String
let photoRequiredTypes: [String]?
let orderStatus: String?
let reason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FileClaimEvidenceBody: Encodable {
let evidenceType: String
let uri: String
let mimeType: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FileClaimLineBody: Encodable {
let sku: String
let quantity: Int64
let reason: String
⋮----
struct FileClaimRequestBody: Encodable {
let claimType: String
let description: String
let lineItems: [FileClaimLineBody]
let evidences: [FileClaimEvidenceBody]
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/PendingOrder.swift
```swift
final class PendingOrder {
var endpoint: String = "/v1/checkout/unified"
var method: String = "POST"
var payloadJson: String
var idempotencyKey: String = ""
var createdAt: Date
var retryCount: Int
var lastError: String?
⋮----
init(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/AccountProfileView.swift
```swift
struct AccountProfileView: View {
@State private var vm = AccountProfileViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/AdjustStockSheet.swift
```swift
struct AdjustStockSheet: View {
let locationId: String
let initialSku: String
let allSkus: [String]
var onComplete: () -> Void
⋮----
@Environment(\.dismiss) private var dismiss
@State private var vm = StoreStockActionViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/AssistView.swift
```swift
struct AssistView: View {
@State private var vm = AssistViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CapabilitiesView.swift
```swift
struct CapabilitiesView: View {
@State private var vm = CapabilitiesViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CheckoutPaymentPickerSheet.swift
```swift
struct CheckoutPaymentPickerSheet: View {
let paymentOptions: [CheckoutPaymentOption]
let selectedPaymentId: String
let onSelect: (CheckoutPaymentOption) -> Void
⋮----
var body: some View {
⋮----
private func paymentIcon(_ option: CheckoutPaymentOption) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ConnectSupplierSheet.swift
```swift
struct ConnectSupplierSheet: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = ConnectSupplierViewModel()
⋮----
let existingSuppliers: [Supplier]
let onConnected: () async -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CreditPartnersView.swift
```swift
struct CreditPartnersView: View {
@State private var vm = CreditPartnersViewModel()
⋮----
var body: some View {
⋮----
struct CreditRelationshipWire: Decodable {
let supplierId: String
let creditLimitMinor: Int64
let currentBalanceMinor: Int64
let termsDays: Int
let onHold: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CreditRelationshipsResponse: Decodable {
let relationships: [CreditRelationshipWire]?
⋮----
struct ArInvoiceWire: Decodable {
let invoiceId: String
let supplierId: String
let balanceMinor: Int64
let status: String
let dueAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ArInvoicesResponse: Decodable {
let invoices: [ArInvoiceWire]?
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/DeliveryPaymentSheetView.swift
```swift
struct DeliveryPaymentSheetView: View {
let event: PaymentRequiredEvent
let onDismiss: () -> Void
⋮----
@State private var vm = DeliveryPaymentViewModel()
⋮----
var body: some View {
⋮----
private var chooseContent: some View {
⋮----
private var cashConfirmContent: some View {
⋮----
private var processingContent: some View {
⋮----
private var cashPendingContent: some View {
⋮----
private var fiscalizingContent: some View {
⋮----
private var successContent: some View {
⋮----
private var failedContent: some View {
⋮----
private var formattedAmount: String {
⋮----
private var formattedOriginalAmount: String {
⋮----
private struct PaymentOptionButton: View {
let icon: String
let label: String
let description: String
let action: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/FamilyMembersView.swift
```swift
struct FamilyMembersView: View {
@State private var vm = FamilyMembersViewModel()
⋮----
var body: some View {
⋮----
struct AddFamilyMemberView: View {
@Environment(\.dismiss) private var dismiss
@State private var name = ""
@State private var phone = ""
var onAdd: (FamilyMemberRequest) -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/InsightsView.swift
```swift
struct MonthlyExpense: Codable, Identifiable {
let month: String
let total: Int
⋮----
var id: String { month }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var shortMonth: String {
⋮----
struct TopSupplierExpense: Codable, Identifiable {
let supplierID: String
let supplierName: String
let total: Int
let orderCount: Int
⋮----
var id: String { supplierID }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TopProductExpense: Codable, Identifiable {
let productID: String
let productName: String
let total: Int
let quantity: Int
⋮----
var id: String { productID }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerAnalytics: Codable {
let monthlyExpenses: [MonthlyExpense]
let topSuppliers: [TopSupplierExpense]
let topProducts: [TopProductExpense]
let totalThisMonth: Int
let totalLastMonth: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerDayExpense: Codable, Identifiable {
let date: String
let total: Int
let count: Int
var id: String { date }
⋮----
var shortDate: String {
⋮----
struct OrderStateCount: Codable, Identifiable {
let state: String
let count: Int
var id: String { state }
⋮----
struct CategorySpend: Codable, Identifiable {
let category: String
let total: Int
let count: Int
var id: String { category }
⋮----
struct DayOfWeekPattern: Codable, Identifiable {
let weekday: String
let avg: Int
let count: Int
var id: String { weekday }
⋮----
struct RetailerDetailedAnalytics: Codable {
let dailySpending: [RetailerDayExpense]
let ordersByState: [OrderStateCount]
let categoryBreakdown: [CategorySpend]
let weekdayPattern: [DayOfWeekPattern]
let totalSpent: Int
let totalOrders: Int
let avgOrderValue: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum DateRange: String, CaseIterable {
⋮----
var days: Int {
⋮----
struct InsightsView: View {
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = InsightsViewModel()
⋮----
var body: some View {
⋮----
private func formatAmount(_ value: Int) -> String {
⋮----
private func abbreviateAmount(_ value: Int) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/LocationsView.swift
```swift
struct LocationsView: View {
@State private var vm = LocationsViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/OrderDetailSheet.swift
```swift
struct OrderDetailSheet: View {
let order: Order
var onCancelled: (() -> Void)? = nil
@Environment(\.dismiss) private var dismiss
@State private var vm = OrderDetailViewModel()
⋮----
private var claimCandidate: Bool {
⋮----
var body: some View {
⋮----
private static let claimIsoFormatter = ISO8601DateFormatter()
⋮----
private func formatClaimEndsAt(_ raw: String?) -> String {
⋮----
private func summaryRow(_ title: String, value: String) -> some View {
⋮----
struct QROverlay: View {
let order: Order
let onDismiss: () -> Void
⋮----
@State private var appeared = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/OrderStatusHistorySection.swift
```swift
struct OrderStatusHistorySection: View {
let orderId: String
⋮----
@State private var vm = OrderDetailViewModel()
⋮----
var body: some View {
⋮----
private func statusLabel(_ entry: OrderTimelineEntry) -> String {
⋮----
private func subtitle(for entry: OrderTimelineEntry) -> String {
⋮----
private static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
private static let isoStandardFormatter = ISO8601DateFormatter()
⋮----
private func formatWhen(_ iso: String) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/OrdersView.swift
```swift
enum OrderTab: String, CaseIterable {
⋮----
var title: String {
⋮----
var icon: String {
⋮----
struct OrdersView: View {
@Environment(\.modelContext) private var modelContext
var initialCommandStatus: String? = nil
var initialSupplierId: String? = nil
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = OrdersViewModel()
⋮----
private var activeOrders: [Order] { vm.activeOrders }
private var pendingOrders: [Order] { vm.pendingOrders }
private var predictions: [RetailerAIPrediction] { vm.predictions }
⋮----
var body: some View {
⋮----
private var commandFilterBanner: some View {
⋮----
private var commandFilteredContent: some View {
⋮----
private var topTabs: some View {
⋮----
private func badgeCount(for tab: OrderTab) -> Int {
⋮----
private var activeContent: some View {
⋮----
private var pendingContent: some View {
⋮----
private var aiPlannedContent: some View {
⋮----
private func tabEmptyState(icon: String, title: String, message: String) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/PlanogramView.swift
```swift
struct PlanogramView: View {
@State private var vm = PlanogramViewModel()
⋮----
var body: some View {
⋮----
private var slottingGridTab: some View {
⋮----
private var walkChecklistTab: some View {
⋮----
private var cameraVisionTab: some View {
⋮----
private struct ChecklistRow: View {
let title: String
@State var isChecked: Bool
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/PutawaySheet.swift
```swift
struct PutawaySheet: View {
let locationId: String
let initialSku: String
let availableSkus: [String]
var onComplete: () -> Void
⋮----
@Environment(\.dismiss) private var dismiss
@State private var vm = StoreStockActionViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ReceiveDeliverySheet.swift
```swift
struct ReceiveDeliverySheet: View {
let locationId: String
var onComplete: () -> Void
⋮----
@Environment(\.dismiss) private var dismiss
@State private var vm = StoreStockActionViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/RetailerMoreHubView.swift
```swift
struct RetailerMoreHubView: View {
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SavedCardsView.swift
```swift
struct SavedCardItem: Identifiable, Codable {
var id: String
var panMasked: String
var scheme: String
var holder: String
var expiry: String
var bankName: String
var isDefault: Bool
⋮----
struct SavedCardsView: View {
@State private var cards: [SavedCardItem] = [
⋮----
@State private var showingAddCard = false
@State private var panInput = ""
@State private var expInput = ""
@State private var otpInput = ""
@State private var bindingStep = 0 // 0: input, 1: otp
@State private var statusMessage: String?
⋮----
var body: some View {
⋮----
private func schemeIcon(_ scheme: String) -> some View {
⋮----
private func schemeColor(_ scheme: String) -> Color {
⋮----
private func deleteCard(_ id: String) {
⋮----
private func setDefaultCard(_ id: String) {
⋮----
private func completeCardBinding() {
⋮----
private func resetAddCardState() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SectionsView.swift
```swift
struct SectionsView: View {
@State private var items: [SectionRowIOS] = []
@State private var name = "Dairy"
@State private var aisle = ""
@State private var selectedId: String?
@State private var skuText = ""
@State private var banner: String?
@State private var saveError: String?
@State private var busy = false
private let api = APIClient.shared
⋮----
var body: some View {
⋮----
private func refresh() async {
⋮----
private func create() async {
⋮----
private func saveSkus() async {
⋮----
struct SectionRowIOS: Identifiable {
let id: String
let name: String
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SelectOrgView.swift
```swift
struct SelectOrgView: View {
@Bindable var auth: AuthManager
@State private var busyId: String?
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SetupView.swift
```swift
struct SetupView: View {
@Environment(AuthManager.self) private var auth
@State private var vm = OnboardingViewModel()
⋮----
var body: some View {
⋮----
private var header: some View {
⋮----
private var progressBar: some View {
⋮----
private var taxStep: some View {
⋮----
private var addressStep: some View {
⋮----
private func setupField(title: String, text: Binding<String>, icon: String) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ShiftsView.swift
```swift
struct ShiftsView: View {
@State private var clockedIn = false
@State private var shifts: [ShiftRow] = []
@State private var floatMinor = "0"
@State private var closingCash = "0"
@State private var registerId: String?
@State private var banner: String?
@State private var busy = false
⋮----
private let api = APIClient.shared
⋮----
var body: some View {
⋮----
private func refresh() async {
⋮----
private func clockIn() async {
⋮----
private func clockOut() async {
⋮----
private func openShift() async {
⋮----
private func closeShift(id: String) async {
⋮----
struct ShiftRow: Identifiable {
let id: String
let status: String
let openingFloatMinor: Int64
let varianceMinor: Int64?
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/StoreStockOrderPickerSheet.swift
```swift
struct StoreStockOrderPickerSheet: View {
@Bindable var vm: StoreStockViewModel
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/TeamView.swift
```swift
struct TeamView: View {
@State private var members: [RetailerOrgMember] = []
@State private var loading = true
@State private var errorText: String?
@State private var banner: String?
@State private var name = ""
@State private var phone = ""
@State private var password = ""
@State private var role = "CASHIER"
@State private var busy = false
⋮----
private let api = APIClient.shared
⋮----
var body: some View {
⋮----
private func load() async {
⋮----
private func invite() async {
⋮----
private func deactivate(_ userId: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/AppDelegate.swift
```swift
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
func application(
⋮----
func application(
⋮----
func application(
⋮----
nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/AutoUpdater.swift
```swift
final class AutoUpdater {
static let shared = AutoUpdater()
⋮----
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
⋮----
private(set) var lastManifest: Manifest?
⋮----
private init() {}
⋮----
func evaluate(
⋮----
func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
⋮----
func triggerOTAInstall(manifestURL: URL) {
⋮----
func openStoreListing(force: Bool = false) {
⋮----
private func fetchManifest(from urlString: String) async -> Manifest? {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/CartManager.swift
```swift
final class CartManager {
var items: [CartItem] = []
var supplierIsActive: Bool = true
private var lastSyncedSignature: String = ""
⋮----
var totalItems: Int {
⋮----
var totalPriceMinor: Int64 {
⋮----
var quotedSubtotalMinor: Int64?
var quotedDiscountMinor: Int64 = 0
⋮----
var checkoutSubtotalMinor: Int64 {
⋮----
var checkoutDiscountMinor: Int64 {
⋮----
var checkoutTotalMinor: Int64 {
⋮----
var displayTotal: String {
⋮----
var isEmpty: Bool { items.isEmpty }
⋮----
// MARK: - Add to Cart
⋮----
func add(product: Product, variant: Variant, quantity: Int = 1) {
⋮----
// MARK: - Remove from Cart
⋮----
func remove(itemId: String) {
⋮----
// MARK: - Update Quantity
⋮----
func updateQuantity(itemId: String, quantity: Int) {
⋮----
// MARK: - Increment / Decrement
⋮----
func increment(itemId: String) {
⋮----
func decrement(itemId: String) {
⋮----
// MARK: - Clear
⋮----
func clear() {
⋮----
func refreshCheckoutQuote() async {
⋮----
// MARK: - Build Order Payload
⋮----
func buildCheckoutPayload(
⋮----
func applyPreviewCaps(_ preview: CheckoutPreviewResponse) {
⋮----
func maxQuantity(for item: CartItem, preview: CheckoutPreviewResponse?) -> Int {
⋮----
struct CheckoutQuoteLine: Encodable {
let productID: String
let quantity: Int64
let unitPriceMinor: Int64
let currency: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(productID: String, quantity: Int64, unitPriceMinor: Int64, currency: String = packCurrency(MarketPackStore.pack)) {
⋮----
struct CheckoutQuoteRequest: Encodable {
let supplierID: String
let lines: [CheckoutQuoteLine]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CheckoutQuoteResponse: Decodable {
let supplierID: String
let subtotalMinor: Int64
let discountMinor: Int64
let totalMinor: Int64
let currency: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct UnifiedCheckoutPayload: Codable {
let retailerId: String
let paymentGateway: String
let latitude: Double
let longitude: Double
let items: [Item]
let deliveryMode: String?
let requestedDeliveryDate: String?
let deliveryPriority: String?
let checkoutPolicyToken: String?
let currency: String?
⋮----
struct Item: Codable {
let skuId: String
let quantity: Int
let unitPriceUzs: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct OrderCurrencyOptions: Codable {
let enabled: Bool
let operatingCurrency: String
let allowlist: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StockWarning: Codable, Hashable {
let sku: String
let requested: Int64
let available: Int64
let backorderQty: Int64
let acceptsBackorder: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CheckoutResponse: Codable {
let status: String
let invoiceId: String
let total: Int64
let supplierOrders: [SupplierOrderResult]?
let backorderedItemCount: Int?
let backorderOrderId: String?
let stockWarnings: [StockWarning]?
⋮----
struct SupplierOrderResult: Codable {
let orderId: String
let supplierId: String
let supplierName: String
let total: Int64
let itemCount: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
private func signature(for cartItems: [CartItem]) -> String {
⋮----
private func cartItem(from serverItem: CartSyncItem, existing: CartItem?) -> CartItem {
⋮----
// MARK: - Server Hydrate
func hydrateFromServer() async {
⋮----
func sync() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/FirebaseAuthHelper.swift
```swift
func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
⋮----
func signOut() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Stored/event currency, else session pack. Empty pack does not invent UZS.
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
/// Card rails the retailer may tap. Empty catalog never invents Adyen/Stripe.
func filterRetailerCardGateways(incoming: [String], catalog: [String]) -> [String] {
⋮----
func selectableRetailerCatalogCodes(_ catalog: [PSPListing]) -> [String] {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/MediaUploadService.swift
```swift
enum MediaUploadService {
struct Ticket: Decodable {
let uploadURL: String
let publicURL: String
let contentType: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
static func uploadJPEG(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/PendingOrderReplayer.swift
```swift
enum PendingOrderReplayer {
⋮----
static func flush(modelContext: ModelContext, api: APIClient = .shared) async {
⋮----
private static func replay(_ order: PendingOrder, api: APIClient) async throws -> Bool {
⋮----
private static func pendingOrderIdempotencyKey(_ order: PendingOrder) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerCheckoutService.swift
```swift
enum RetailerCheckoutService {
static func fetchPreview(
⋮----
static func completeCheckout(
⋮----
enum CheckoutPreviewError: LocalizedError {
⋮----
var errorDescription: String? {
⋮----
struct CheckoutPreviewResponse: Decodable {
let ok: Bool
let blocked: Bool?
let code: String?
let message: String?
let rejectedSkus: [String]?
let oosItems: [String]?
let shortfall: [String: Int64]?
let stockWarnings: [StockWarning]
let maxQuantities: [String: Int64]?
let orderableQuantities: [String: Int64]?
let backorderedItemCount: Int?
let showStockCounts: Bool?
let deliveryFeeMinor: Int64?
let deliveryDistanceKm: Double?
let preorderMinLeadDays: Int64?
let preorderMaxLeadDays: Int64?
let orderLineMinQuantity: Int64?
let orderLineMaxQuantity: Int64?
let defaultOutOfStockPolicy: String?
let checkoutPolicyToken: String?
let checkoutPolicyExpiresAt: String?
let orderAcceptanceOpen: Bool?
let orderAcceptanceWindowLabel: String?
let nextOrderAcceptanceAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerLiveActivityManager.swift
```swift
final class RetailerLiveActivityManager {
static let shared = RetailerLiveActivityManager()
⋮----
private var currentActivity: Activity<DeliveryActivityAttributes>?
private var activeOrderId: String?
⋮----
private init() {}
⋮----
func startInboundDeliveryActivity(
⋮----
func updateInboundDeliveryState(
⋮----
func endActivity(status: String = "COMPLETED") {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerProfileService.swift
```swift
enum RetailerProfileService {
static func saveProfile(
⋮----
private static func profileFingerprint(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerSessionReconcile.swift
```swift
enum RetailerSessionReconcile {
static func run(api: APIClient = .shared) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerSupplierDiscoveryService.swift
```swift
enum RetailerSupplierDiscoveryService {
static func searchSuppliers(api: APIClient, query: String) async throws -> [Supplier] {
⋮----
static func addSupplier(api: APIClient, supplierId: String) async throws {
⋮----
static func removeSupplier(api: APIClient, supplierId: String) async throws {
⋮----
static func filterNewCandidates(_ results: [Supplier], existing: [Supplier]) -> [Supplier] {
⋮----
private struct EmptyRequest: Encodable {}
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Theme/AnimationConstants.swift
```swift
enum AnimationConstants {
⋮----
static let fluid = Animation.spring(response: 0.3, dampingFraction: 0.8)
⋮----
static let express = Animation.spring(response: 0.15, dampingFraction: 0.9)
⋮----
static let bouncy = Animation.spring(response: 0.35, dampingFraction: 0.6)
⋮----
static let tabSwitch = Animation.easeOut(duration: 0.08)
⋮----
static let snappy = Animation.snappy
⋮----
static let sheet = Animation.spring(response: 0.35, dampingFraction: 0.86)
⋮----
static let staggerDelay: Double = 0.05
⋮----
static let hero = Animation.spring(response: 0.6, dampingFraction: 0.75)
⋮----
static let micro = Animation.spring(response: 0.1, dampingFraction: 0.95)
⋮----
static let smooth = Animation.easeInOut(duration: 0.25)
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Theme/PegasusIcon.swift
```swift
enum LabIcon: String, CaseIterable {
⋮----
var systemName: String {
⋮----
var filledSystemName: String {
⋮----
func image(filled: Bool = false) -> Image {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/Haptics.swift
```swift
enum Haptics {
static func light() {
⋮----
static func medium() {
⋮----
static func heavy() {
⋮----
static func success() {
⋮----
static func error() {
⋮----
static func warning() {
⋮----
static func selection() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func userNotificationCenter(
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
private func sendTokenToServer(token: String) async {
⋮----
enum QRCodeGenerator {
static func generate(from string: String, size: CGSize = CGSize(width: 200, height: 200)) -> UIImage? {
⋮----
struct QRCodeView: View {
let data: String
var size: Double = 160
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/RetailerAsync.swift
```swift
enum RetailerAsync {
static func run(_ operation: @escaping @Sendable () async -> Void) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/RetailerErrorSupport.swift
```swift
enum RetailerRequestIssue {
⋮----
enum RetailerErrorSupport {
static func classify(_ error: Error) -> RetailerRequestIssue {
⋮----
static func message(
⋮----
static func retryQueuedMessage(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/RetailerIdempotency.swift
```swift
enum RetailerIdempotency {
static func orderCreate(items: [ProcurementOrderRequest.Item]) -> String {
⋮----
static func confirmCash(orderId: String) -> String {
⋮----
static func cancel(orderId: String) -> String {
⋮----
static func confirmPreorder(orderId: String) -> String {
⋮----
static func confirmAI(orderId: String) -> String {
⋮----
static func requestCancel(orderId: String) -> String {
⋮----
static func shopClosedResponse(orderId: String, response: String) -> String {
⋮----
static func acceptDeliveryProposal(orderId: String) -> String {
⋮----
static func rejectDeliveryProposal(orderId: String, reason: String = "") -> String {
⋮----
static func rejectPreorder(orderId: String, reason: String = "") -> String {
⋮----
static func rejectAI(orderId: String, reason: String = "") -> String {
⋮----
static func editPreorder(orderId: String) -> String {
⋮----
static func setup(retailerId: String) -> String {
⋮----
static func supplierAdd(supplierId: String) -> String {
⋮----
static func supplierRemove(supplierId: String) -> String {
⋮----
static func profileUpdate(retailerId: String, payloadFingerprint: String) -> String {
⋮----
/// Stable file-claim key — same body retries return the same claim_id (G11/G25).
static func claimFile(orderId: String, bodyFingerprint: String) -> String {
⋮----
private static func stableHash(_ input: String) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/AccountProfileViewModel.swift
```swift
final class AccountProfileViewModel {
var retailerId = ""
var name = ""
var company = ""
var phone = ""
var receivingWindowOpen = ""
var receivingWindowClose = ""
var isLoading = false
var isSaving = false
var errorMessage: String?
var saveMessage: String?
var openWindowError: String?
var closeWindowError: String?
⋮----
private let api = APIClient.shared
⋮----
func clearOpenWindowError() {
⋮----
func clearCloseWindowError() {
⋮----
func loadProfile() async {
⋮----
func saveProfile() async {
⋮----
private func validateReceivingWindow(_ raw: String) -> String? {
⋮----
private func normalizeReceivingWindow(_ raw: String) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ActiveDeliveriesViewModel.swift
```swift
final class ActiveDeliveriesViewModel {
var orders: [Order] = []
var isLoading = false
var loadError = false
var selectedOrder: Order?
var qrOverlayOrder: Order?
var showMap = false
var approachingOrderIds: Set<String> = []
⋮----
private let api = APIClient.shared
private let ws = RetailerWebSocket.shared
⋮----
func loadOrders() async {
⋮----
func listenForApproaching() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ActiveOrderViewModel.swift
```swift
final class ActiveOrderViewModel {
var activeOrders: [Order] = []
var isLoading = false
var loadError = false
⋮----
private let api = APIClient.shared
⋮----
func loadActiveOrders() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/AppShellViewModel.swift
```swift
final class AppShellViewModel {
var selectedTab: AppTab = .home
var sideMenuSelection: SideMenuTab = .home
var isSidebarExpanded = true
⋮----
var showSidebar = false
var showFutureDemand = false
var showAutoOrder = false
var showActiveOrderDetail = false
var showProfile = false
var showCart = false
var showInsights = false
var showProcurement = false
var showCredit = false
var showHq = false
var showNotificationInbox = false
⋮----
var notificationCount = 0
var cartBounce = false
var activeOrders: [Order] = []
var paymentEvent: PaymentRequiredEvent?
var shopClosedAlert: ShopClosedAlertEvent?
var clientPolicyMessage: String?
var clientPolicyForce = false
var pendingManifest: AutoUpdater.Manifest?
var rescueWarningMessage: String?
var deliveriesHubInitialTab: DeliveriesHubTab = .map
⋮----
private let api = APIClient.shared
private let ws = RetailerWebSocket.shared
⋮----
func showFloatingBar(horizontalSizeClass: UserInterfaceSizeClass?) -> Bool {
⋮----
func navigateToSuppliersTab(horizontalSizeClass: UserInterfaceSizeClass?) {
⋮----
func handleTabChange(from oldTab: AppTab, to newTab: AppTab) {
⋮----
func handleSidebarNavigation(
⋮----
func bootstrap() async {
⋮----
func refreshOnActive(refreshCenter: RetailerRefreshCenter) async {
⋮----
func loadActiveOrders() async {
⋮----
func loadNotificationCount() async {
⋮----
func loadPendingPayments(reconcile: Bool = false) async {
⋮----
func loadClientPolicy() async {
⋮----
struct ClientPolicy: Decodable {
let outdated: Bool?
let forceUpdate: Bool?
let updateDeferred: Bool?
let minimumVersion: String?
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
enum CodingKeys: String, CodingKey {
⋮----
func connectWebSocket(modelContext: ModelContext, refreshCenter: RetailerRefreshCenter) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ArrivalViewModel.swift
```swift
struct SupplierDockGroup: Identifiable {
var id: String { supplierId }
let supplierId: String
let supplierName: String
let orders: [TrackingOrder]
let totalAmount: Int
let hasApproaching: Bool
let hasArrived: Bool
⋮----
final class ArrivalViewModel {
var orders: [TrackingOrder] = []
var expandedSupplierIds: Set<String> = []
var revealedTokenOrderIds: Set<String> = []
var activeQrOrder: TrackingOrder?
var autoOpenedOrderIds: Set<String> = []
var isLoading = false
var isRefreshing = false
var loadError: String?
⋮----
private let api = APIClient.shared
private let ws = RetailerWebSocket.shared
private let dockStates: Set<String> = [
⋮----
var activeOrders: [TrackingOrder] {
⋮----
var supplierGroups: [SupplierDockGroup] {
⋮----
func toggleSupplierExpanded(_ supplierId: String) {
⋮----
func toggleRevealToken(_ orderId: String) {
⋮----
func loadOrders(refreshing: Bool = false) async {
⋮----
func maybeAutoOpenQr() {
⋮----
func observeWebSocket() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/AssistViewModel.swift
```swift
struct AssistSectionOption: Identifiable, Hashable {
let id: String
let name: String
⋮----
struct AssistRowIOS: Identifiable {
let id: String
let note: String
let status: String
⋮----
final class AssistViewModel {
var sections: [AssistSectionOption] = []
var selectedSectionId = ""
var note = ""
var tickets: [AssistRowIOS] = []
var banner: String?
var busy = false
⋮----
private let api = APIClient.shared
⋮----
var canCreate: Bool {
⋮----
func refresh() async {
⋮----
func create() async {
⋮----
func act(id: String, action: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/AutoOrderViewModel.swift
```swift
struct AutoOrderShadowStats: Codable {
let proposalCount: Int64
let matchedOrders: Int64
let wape: Double
let unmodifiedAcceptRate: Double
let windowDays: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderSoakDecision: Codable {
let allowed: Bool
let reasons: [String]?
let stats: AutoOrderShadowStats?
let bypassSource: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderSoakThresholds: Codable {
let minProposals: Int64?
let maxWape: Double?
let minUnmodified: Double?
let gateDisabled: Bool?
let bypassSource: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderSoakGate: Codable {
let decision: AutoOrderSoakDecision?
let thresholds: AutoOrderSoakThresholds?
let placeFlagEnabled: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderShadowProposal: Codable, Identifiable {
var id: String { proposalId }
let proposalId: String
let retailerId: String
let sku: String
let supplierId: String?
let proposedQty: Int64
let ip: Double
let reorderPoint: Double
let orderUpTo: Double
let bucketDate: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderShadowProposalsResponse: Codable {
let items: [AutoOrderShadowProposal]
⋮----
struct AutoOrderSettings: Codable {
let globalEnabled: Bool
let executionMode: String?
let hasAnyHistory: Bool
let analyticsStartDate: String?
let supplierOverrides: [SupplierOverride]
let categoryOverrides: [CategoryOverride]
let productOverrides: [ProductOverride]
let variantOverrides: [VariantOverride]
let shadowStats: AutoOrderShadowStats?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierOverride: Codable, Identifiable, Hashable {
var id: String { supplierID }
let supplierID: String
let enabled: Bool
let hasHistory: Bool
let supplierName: String?
let analyticsStartDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CategoryOverride: Codable, Identifiable, Hashable {
var id: String { categoryID }
let categoryID: String
let enabled: Bool
let hasHistory: Bool
let analyticsStartDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ProductOverride: Codable, Identifiable, Hashable {
var id: String { productID }
let productID: String
let supplierID: String
let enabled: Bool
let hasHistory: Bool
let productName: String?
let analyticsStartDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct VariantOverride: Codable, Identifiable, Hashable {
var id: String { skuID }
let skuID: String
let productID: String?
let enabled: Bool
let hasHistory: Bool
let skuLabel: String?
let analyticsStartDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
// MARK: - AnyCodable Helper
⋮----
struct AnyCodable: Encodable {
private let value: [String: Any]
⋮----
init(_ value: [String: Any]) {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
private struct DynamicCodingKey: CodingKey {
var stringValue: String
var intValue: Int?
⋮----
init(stringValue: String) {
⋮----
init?(intValue: Int) {
⋮----
// MARK: - ViewModel
⋮----
final class AutoOrderViewModel {
enum EnableTarget {
⋮----
var settings: AutoOrderSettings?
var forecasts: [DemandForecast] = []
var isLoading = true
var syncMessage: String?
var globalAutoOrder = false
var executionMode = "draft"
var pendingTarget: EnableTarget?
var showHistoryAlert = false
var localToggleStates: [String: Bool] = [:]
var runs: [AutoOrderRun] = []
var lastRun: AutoOrderRun?
var running = false
var runningMode: String?
var placeConfirmOpen = false
var reorderSuggestions: [RetailerReorderSuggestion] = []
var shadowProposals: [AutoOrderShadowProposal] = []
var soakGate: AutoOrderSoakGate?
⋮----
private let api = APIClient.shared
⋮----
var alertEntityLabel: String {
⋮----
func handleGlobalToggle(_ newVal: Bool) {
⋮----
func handleOverrideToggle(item: OverrideItem, newVal: Bool) {
⋮----
func cancelPendingEnable() {
⋮----
func loadAll() async {
⋮----
private func normalizeMode(_ raw: String?, global: Bool) -> String {
⋮----
private func loadShadowProposals() async -> [AutoOrderShadowProposal] {
⋮----
private func loadSoakGate() async -> AutoOrderSoakGate? {
⋮----
func setExecutionMode(_ mode: String) async {
⋮----
private func loadRuns() async -> [AutoOrderRun] {
⋮----
private func loadSuggestions() async -> [RetailerReorderSuggestion] {
⋮----
func runAutoOrder(mode: String) async {
⋮----
private func loadSettings() async -> (AutoOrderSettings?, String?) {
⋮----
private func loadForecasts() async -> ([DemandForecast], String?) {
⋮----
func enableGlobal(useHistory: Bool) async {
⋮----
func disableGlobal() async {
⋮----
func toggleOverride(item: OverrideItem, enabled: Bool, useHistory: Bool) async {
⋮----
func confirmEnable(useHistory: Bool) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CapabilitiesViewModel.swift
```swift
final class CapabilitiesViewModel {
var packs: [RetailerCapabilityPack] = []
var loading = true
var errorText: String?
var banner: String?
var busyId: String?
⋮----
private let api = APIClient.shared
⋮----
func load() async {
⋮----
func enable(_ id: String) async {
⋮----
func disable(_ id: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CatalogViewModel.swift
```swift
final class CatalogViewModel {
var searchText = ""
var browseMode: CatalogBrowseMode = .categories
var categories: [ProductCategory] = []
var products: [Product] = []
var selectedProduct: Product?
var isLoading = false
var loadError: String?
var showFullSearch = false
⋮----
private let api = APIClient.shared
⋮----
var isSearching: Bool {
⋮----
var filteredProducts: [Product] {
⋮----
func loadCategories() async {
⋮----
func loadProducts() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CategoryProductsViewModel.swift
```swift
final class CategoryProductsViewModel {
var products: [Product] = []
var isLoading = false
var selectedProduct: Product?
var autoOrderSettings: [String: Bool] = [:]
var togglingIds: Set<String> = []
⋮----
private let api = APIClient.shared
⋮----
var groupedProducts: [(supplier: String, products: [Product])] {
⋮----
func loadProducts(categoryId: String) async {
⋮----
func toggleProductAutoOrder(productId: String, enabled: Bool) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CategorySuppliersViewModel.swift
```swift
final class CategorySuppliersViewModel {
var suppliers: [Supplier] = []
var isLoading = false
var errorMessage: String?
var mySupplierIds: Set<String> = []
var togglingIds: Set<String> = []
⋮----
private let api = APIClient.shared
⋮----
func loadSuppliers(categoryId: String) async {
⋮----
func toggleMySupplier(_ supplier: Supplier) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CheckoutViewModel.swift
```swift
struct CheckoutPaymentOption: Identifiable, Hashable {
let id: String
let label: String
let isToken: Bool
⋮----
final class CheckoutViewModel {
var selectedPaymentId = "GlobalPay"
var paymentOptions: [CheckoutPaymentOption] = [
⋮----
var showPaymentPicker = false
var checkoutSessionId = UUID().uuidString
var isSubmitting = false
var showSuccess = false
var showError = false
var errorMessage = ""
var oosItems: [String] = []
var stockWarnings: [StockWarning] = []
var showSupplierClosedWarning = false
var preview: CheckoutPreviewResponse?
var previewLoading = false
var deliveryMode = "STANDARD"
var deliveryDate = ""
var expressPriority = false
var catalogCodes: [String] = []
var packDisplayCurrency = packCurrency(MarketPackStore.pack)
var showBackorderConfirm = false
var pendingBackorderPreview: CheckoutPreviewResponse?
var skipBackorderConfirm = false
var checkoutPolicyToken: String?
⋮----
private let api = APIClient.shared
private static let isoFormatter = ISO8601DateFormatter()
⋮----
var selectedPaymentLabel: String {
⋮----
var scheduledMinDate: String {
⋮----
var scheduledMaxDate: String? {
⋮----
/// Map UI labels to backend gateway codes expected by /v1/checkout/unified
func gatewayCode(for methodId: String) -> String {
⋮----
func skuFor(_ item: CartItem) -> String {
⋮----
func isOos(_ item: CartItem) -> Bool {
⋮----
func selectPaymentOption(_ option: CheckoutPaymentOption) {
⋮----
func selectDeliveryMode(_ mode: String) {
⋮----
func buildPayload(cart: CartManager, retailerId: String, gateway: String) -> UnifiedCheckoutPayload {
⋮----
func loadPaymentMethods() async {
⋮----
func fetchPreview(cart: CartManager) async {
⋮----
// Keep last preview state when refresh fails transiently.
⋮----
func fetchCards() async {
⋮----
func fetchPaymentCatalog() async {
⋮----
private func catalogPaymentOptions() -> [CheckoutPaymentOption] {
⋮----
func submitOrder(cart: CartManager, modelContext: ModelContext) async {
⋮----
private func queueOfflineOrder(
⋮----
private func checkoutIdempotencyKey(payload: UnifiedCheckoutPayload, gateway: String, sessionId: String? = nil) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ConnectSupplierViewModel.swift
```swift
final class ConnectSupplierViewModel {
var query = ""
var results: [Supplier] = []
var isSearching = false
var connectingSupplierId: String?
var statusMessage: String?
⋮----
private let api = APIClient.shared
⋮----
var hasMinQueryLength: Bool {
⋮----
func search(_ raw: String, existingSuppliers: [Supplier]) async {
⋮----
func connect(_ supplierId: String, onConnected: () async -> Void) async -> Bool {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/CreditPartnersViewModel.swift
```swift
final class CreditPartnersViewModel {
var relationships: [CreditRelationshipWire] = []
var invoices: [ArInvoiceWire] = []
var banner: String?
var isLoading = false
⋮----
private let api = APIClient.shared
⋮----
func load() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/DashboardViewModel.swift
```swift
final class DashboardViewModel {
var activeOrders: [Order] = []
var predictions: [RetailerAIPrediction] = []
var reorderProducts: [Product] = []
var isLoading = false
var actingId: String?
var orderActionPending = false
var loadError: String?
var pulseEvents: [RetailerPulseEvent] = []
var pulseLoading = false
var pulseError: String?
var commandPulse: ControlTowerPulseWire?
var commandPulseError: String?
var commandJump: CommandStatusJump?
⋮----
private let api = APIClient.shared
⋮----
func loadData(silent: Bool = false) async {
⋮----
func pollCommandPulseLoop() async {
⋮----
func loadCommandPulse() async {
⋮----
func cancelOrder(_ orderId: String) async {
⋮----
func confirmAiOrder(_ orderId: String) async {
⋮----
func rejectAiOrder(_ orderId: String) async {
⋮----
func confirmPreorder(_ orderId: String) async {
⋮----
func editPreorder(_ orderId: String) async {
⋮----
func findOrder(_ orderId: String) async -> Order? {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/DeliveryMapViewModel.swift
```swift
enum TrackingLoadIssue {
⋮----
struct SupplierFilter: Identifiable {
let id: String
let name: String
⋮----
final class DeliveryMapViewModel {
var orders: [TrackingOrder] = []
var recentReceipts: [TrackingOrder] = []
var fiscalQRReceipt: TrackingOrder?
var suppliers: [SupplierFilter] = []
var selectedSupplierIds: Set<String> = []
var isLoading = false
var activeFulfillmentCount = 0
var loadIssue: TrackingLoadIssue?
var selectedOrder: TrackingOrder?
var hasFittedInitialCamera = false
var cameraPosition: MapCameraPosition = .region(
⋮----
private let api = APIClient.shared
private let ws = RetailerWebSocket.shared
private let network = NetworkLifecycleEngine.shared
private let pollingInterval: TimeInterval = 15
⋮----
var selectedOrderId: String? {
⋮----
var visibleOrders: [TrackingOrder] {
⋮----
var activeDeliveryCount: Int {
⋮----
var emptyStateMessage: String {
⋮----
func receiptHTMLURL(for receipt: TrackingOrder) -> URL? {
⋮----
func receiptPDFURL(for receipt: TrackingOrder) -> URL? {
⋮----
func startPolling() async {
⋮----
func refreshTrackingState() async {
⋮----
func fetchTracking() async {
⋮----
func fetchActiveFulfillments() async {
⋮----
func observeWebSocket() async {
⋮----
func toggleSupplier(_ id: String) {
⋮----
func fitCamera() {
⋮----
private func resolveLoadIssue(_ error: Error) -> TrackingLoadIssue {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/DeliveryPaymentViewModel.swift
```swift
enum PaymentPhase: Equatable {
⋮----
struct CardGatewayOption: Identifiable {
let gateway: String
let label: String
let description: String
⋮----
var id: String { gateway }
⋮----
final class DeliveryPaymentViewModel {
var phase: PaymentPhase = .choose
var errorMessage: String?
var catalogCodes: [String] = []
var packDisplayCurrency = packCurrency(MarketPackStore.pack)
⋮----
private let api = APIClient.shared
private let ws = RetailerWebSocket.shared
⋮----
func cardGatewayOptions(availableCardGateways: [String]) -> [CardGatewayOption] {
⋮----
func fetchPaymentCatalog() async {
⋮----
func initiateCashCheckout(orderId: String) async {
⋮----
func confirmCash(orderId: String) async {
⋮----
func initiateCardCheckout(orderId: String, gateway: String) async {
⋮----
func listenForCompletion(orderId: String) async {
⋮----
func resetToChoose() {
⋮----
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
⋮----
enum CodingKeys: String, CodingKey {
⋮----
private struct DiscardablePaymentCodable: Decodable {
init(from decoder: Decoder) throws {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/FamilyMembersViewModel.swift
```swift
final class FamilyMembersViewModel {
var members: [FamilyMemberResponse] = []
var isLoading = false
var showAddSheet = false
var errorMessage: String?
var familyWrites = "open"
var migrating = false
var showMigrateConfirm = false
var migrateResult: FamilyMigrateResult?
var banner: String?
⋮----
private let api = APIClient.shared
⋮----
var familyGone: Bool {
⋮----
func copyTempPassword(_ password: String) {
⋮----
func loadMembers() async {
⋮----
func migrateToTeam() async {
⋮----
func addMember(request: FamilyMemberRequest) async {
⋮----
func removeMembers(at offsets: IndexSet) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/FileClaimViewModel.swift
```swift
struct ClaimLineDraft: Identifiable, Hashable {
let id: String
let sku: String
let productName: String
let orderedQuantity: Int
var claimQuantity: Int
⋮----
final class FileClaimViewModel {
let order: Order
let preferredSku: String?
⋮----
var claimType = "CONCEALED_DAMAGE"
var descriptionText = ""
var lineDrafts: [ClaimLineDraft] = []
var photoURL = ""
var pickedItem: PhotosPickerItem?
var previewImage: UIImage?
var isUploadingPhoto = false
var isSubmitting = false
var errorMessage: String?
var skuWarning: String?
var successClaimId: String?
var existing: [RetailerClaim] = []
var eligibility: ClaimEligibility?
var eligLoading = true
⋮----
let claimTypes = ["CONCEALED_DAMAGE", "DAMAGED", "MISSING", "TAMPER", "TEMPERATURE", "OTHER"]
⋮----
private let api = APIClient.shared
private static let isoFormatter = ISO8601DateFormatter()
⋮----
init(order: Order, preferredSku: String? = nil) {
⋮----
var selectedQtyTotal: Int {
⋮----
var canSubmit: Bool {
⋮----
func loadInitial() async {
⋮----
func loadEligibility() async {
⋮----
func loadExisting() async {
⋮----
func applyPreferredSku() {
⋮----
func loadAndUpload(item: PhotosPickerItem?) async {
⋮----
func submit() async {
⋮----
func formatEndsAt(_ raw: String?) -> String {
⋮----
func windowClosedCopy(_ e: ClaimEligibility) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/FutureDemandViewModel.swift
```swift
final class FutureDemandViewModel {
var items: [RetailerAIPrediction] = []
var isLoading = false
var actingId: String?
⋮----
private let api = APIClient.shared
⋮----
var totalUnits: Int64 {
⋮----
func loadPredictions() async {
⋮----
func confirm(_ item: RetailerAIPrediction) async {
⋮----
func reject(_ item: RetailerAIPrediction) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/HistoryViewModel.swift
```swift
final class HistoryViewModel {
var orders: [Order] = []
var isLoading = false
var loadError = false
var filterStatus: OrderStatus?
⋮----
private let api = APIClient.shared
⋮----
var filteredOrders: [Order] {
⋮----
func loadOrders() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/HqViewModel.swift
```swift
struct HqSummaryWire: Decodable {
let locationCount: Int
let skuCount: Int
let netMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct HqLocationWire: Decodable {
let locationId: String
let qtySold: Int
let netMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(locationId: String, qtySold: Int, netMinor: Int64) {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct HqLocationsResponse: Decodable {
let items: [HqLocationWire]?
⋮----
final class HqViewModel {
var summaryLine = "—"
var locations: [HqLocationWire] = []
var loadError: String?
var summaryReady = false
var isLoading = false
⋮----
private let api = APIClient.shared
private static let dayFormatter: ISO8601DateFormatter = {
⋮----
var day: String {
⋮----
func formatMinor(_ minor: Int64) -> String {
⋮----
func load() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/LocalSkusViewModel.swift
```swift
struct LocalSkuRowIOS: Identifiable {
let id: String
let name: String
let priceMinor: Int64
let currency: String
let active: Bool
⋮----
final class LocalSkusViewModel {
var rows: [LocalSkuRowIOS] = []
var name = ""
var barcode = ""
var priceInput = "5000"
var banner: String?
var loadError: String?
var summaryReady = false
var busy = false
⋮----
private let api = APIClient.shared
⋮----
var canCreate: Bool {
⋮----
func load() async {
⋮----
func create() async {
⋮----
func toggle(_ row: LocalSkuRowIOS) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/LocationPickerViewModel.swift
```swift
final class LocationPickerViewModel {
let locationManager = LocationManager()
var cameraPosition: MapCameraPosition
var centerCoordinate: CLLocationCoordinate2D
var addressLabel: String?
var resolving = false
var awaitingUserLocation = false
⋮----
private var resolveTask: Task<Void, Never>?
private var locateTask: Task<Void, Never>?
⋮----
init(initialLatitude: Double = 0.0, initialLongitude: Double = 0.0) {
⋮----
var statusDisplayText: String {
⋮----
func onCameraEnded(_ newCenter: CLLocationCoordinate2D) {
⋮----
func scheduleResolveAddress() {
⋮----
func resolveAddress() async {
⋮----
func handleLocationChange() {
⋮----
func requestAndMoveToUserLocation() {
⋮----
func confirmSelection() async -> (lat: Double, lng: Double, label: String) {
⋮----
func cancelTasks() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/LocationsViewModel.swift
```swift
final class LocationsViewModel {
var items: [RetailerLocationDTO] = []
var activeId = ""
var loading = true
var banner: String?
var name = ""
var address = ""
var busy = false
⋮----
private let api = APIClient.shared
⋮----
var canCreate: Bool {
⋮----
func load() async {
⋮----
func create() async {
⋮----
func switchTo(_ id: String) async {
⋮----
func setPrimary(_ id: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/LoginViewModel.swift
```swift
final class LoginViewModel {
var phone = "+998"
var password = ""
var isLoginMode = true
⋮----
// Registration fields
var storeName = ""
var ownerName = ""
var addressText = ""
var taxId = ""
var receivingWindowOpen = ""
var receivingWindowClose = ""
var accessType = ""           // STREET_PARKING | ALLEYWAY | LOADING_DOCK
var storageCeilingHeightText = ""
⋮----
// Location & presentation state
let locationManager = LocationManager()
var latitude: Double = 0.0
var longitude: Double = 0.0
var locationLabel = ""
var showLocationPicker = false
var locating = false
var logoScale: Double = 0.5
var formOpacity: Double = 0
⋮----
var phoneInput: String {
⋮----
var isFormValid: Bool {
⋮----
func setLocation(lat: Double, lng: Double, label: String) {
⋮----
func toggleAccessType(_ value: String) {
⋮----
func shareCurrentLocation() async {
⋮----
func submit(auth: AuthManager) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/MySuppliersViewModel.swift
```swift
final class MySuppliersViewModel {
var suppliers: [Supplier] = []
var isLoading = false
var errorMessage: String?
var autoOrderSettings = SimpleAutoOrderSettings.default
var showConnectSheet = false
var supplierPendingRemoval: Supplier?
var showRemoveConfirmation = false
⋮----
private let api = APIClient.shared
⋮----
func loadSuppliers() async {
⋮----
func promptRemoval(for supplier: Supplier) {
⋮----
func cancelRemoval() {
⋮----
func removeSupplier(_ supplierId: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/OnboardingViewModel.swift
```swift
final class OnboardingViewModel {
enum SetupStep {
⋮----
var step: SetupStep = .tax
var taxId = ""
var billingAddress = ""
var shippingAddress = ""
var city = ""
var postalCode = ""
var isSubmitting = false
var errorMessage: String?
⋮----
private let api = APIClient.shared
⋮----
func validateTax() -> String? {
⋮----
func validateAddress() -> String? {
⋮----
func advanceFromTax() -> Bool {
⋮----
func submit() async -> Bool {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/OrderDetailViewModel.swift
```swift
final class OrderDetailViewModel {
var showCancelConfirm = false
var isCancelling = false
var cancelError = false
var showFileClaim = false
var claimElig: ClaimEligibility?
⋮----
var timelineItems: [OrderTimelineEntry] = []
var isTimelineLoading = true
⋮----
private let api = APIClient.shared
⋮----
func isClaimCandidate(_ order: Order) -> Bool {
⋮----
func loadClaimEligibility(for order: Order) async {
⋮----
func cancelOrder(_ order: Order) async -> Bool {
⋮----
func loadTimeline(orderId: String) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/PlanogramViewModel.swift
```swift
struct PlanogramSlotIOS: Identifiable {
let id: String
let shelfIndex: Int
let colIndex: Int
let expectedSkuName: String
let facings: Int
let minFacings: Int
⋮----
struct PlanogramFindingIOS: Identifiable {
let id: String
let type: String
let shelfIndex: Int
let colIndex: Int
let expectedSku: String
let detectedSku: String?
let confidence: Double
var status: String
⋮----
final class PlanogramViewModel {
var selectedTab = 0
var banner: String?
var isBusy = false
var isRunningVision = false
var findings: [PlanogramFindingIOS] = [
⋮----
func shelfDescription(for row: Int) -> String {
⋮----
func slotSkuLabel(row: Int, col: Int) -> String {
⋮----
func runVisionAudit() async {
⋮----
func acceptFinding(id: String) {
⋮----
func dismissFinding(id: String) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/PosViewModel.swift
```swift
final class PosViewModel {
var registerId: String?
var locationId: String?
var sessionId: String?
var cart: [PosCartLine] = []
var sku = ""
var priceMajor = "0"
var qty = "1"
var banner: String?
var busy = false
var lastSaleId: String?
var pending: [PendingPosStore.Entry] = []
var holdsEnabled = false
var serverHolds: [PosHoldWire] = []
var holdNote = ""
⋮----
private let api = APIClient.shared
private let store = PendingPosStore.shared
⋮----
var totalMinor: Int64 {
⋮----
func formatMinor(_ minor: Int64) -> String {
⋮----
func parseMajorToMinor(_ string: String) -> Int64 {
⋮----
func refreshPending() {
⋮----
func ensureRegister() async {
⋮----
func loadServerHolds() async {
⋮----
func linesFromHold(_ hold: PosHoldWire) -> [PosCartLine] {
⋮----
func parkHold() async {
⋮----
func resumeHold(_ hold: PosHoldWire) async {
⋮----
func voidHold(_ hold: PosHoldWire) async {
⋮----
func openSession() async {
⋮----
func closeSession() async {
⋮----
func addLine() {
⋮----
func completeSale() async {
⋮----
func syncPending() async {
⋮----
func voidLast() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ProcurementViewModel.swift
```swift
struct ProcurementOrderRequest: Codable {
let retailerId: String
let items: [Item]
⋮----
struct Item: Codable {
let productId: String
let quantity: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ProcurementOrderResponse: Codable {
let status: String
let orderId: String
let total: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class ProcurementViewModel {
var forecasts: [DemandForecast] = []
var selectedItems: Set<String> = []
var quantities: [String: Int] = [:]
var isSubmitting = false
var showSuccess = false
var showError = false
var errorMessage = ""
var isLoading = false
var products: [Product] = []
⋮----
private let api = APIClient.shared
⋮----
var selectedTotalUnits: Int {
⋮----
func toggleSelectAll() {
⋮----
func toggleSelection(for forecast: DemandForecast) {
⋮----
func updateQuantity(for forecastId: String, quantity: Int) {
⋮----
func acknowledgeSuccess() {
⋮----
func addToCart(cart: CartManager) {
⋮----
func createOrder(modelContext: ModelContext, refreshCenter: RetailerRefreshCenter) async {
⋮----
private func queuePendingProcurementOrder(
⋮----
func loadPredictions() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ProductDetailViewModel.swift
```swift
final class ProductDetailViewModel {
var selectedVariant: Variant?
var quantity: Int = 1
var addedToCart = false
var imageScale: Double = 0.95
var variantAutoOrder: [String: Bool] = [:]
var variantHasHistory: [String: Bool] = [:]
var productAutoOrderEnabled: Bool = false
var productHasHistory: Bool = false
var pendingVariantTarget: String?
var pendingProductEnable: Bool = false
var showHistoryAlert: Bool = false
⋮----
private let api = APIClient.shared
⋮----
func configure(for product: Product) {
⋮----
func selectVariant(_ variant: Variant) {
⋮----
func addToCart(product: Product, cart: CartManager) {
⋮----
func cancelPendingEnable() {
⋮----
func toggleVariantAutoOrder(skuId: String, enabled: Bool, useHistory: Bool) async {
⋮----
func patchProductAutoOrder(productId: String, enabled: Bool, useHistory: Bool) async {
⋮----
func loadAutoOrderState(for product: Product) async {
⋮----
func confirmProductOrVariantEnable(productId: String, useHistory: Bool) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ProfileViewModel.swift
```swift
final class ProfileViewModel {
var showHistoryAlert = false
var profileName = ""
var profileCompany = ""
var profilePhone = ""
var profileLocation = ""
var pricingRulesSummary = ""
var orderCount = 0
var totalSpent: Int64 = 0
var totalSpentCurrency = "UZS"
var creditProfile: CreditProfile?
var creditLoading = true
var creditMissing = false
var creditError: String?
⋮----
private let api = APIClient.shared
⋮----
private struct RetailerPricingRulesResponse: Decodable {
let summary: String?
let status: String?
⋮----
func loadAll(retailerId: String, currentUser: User?) async -> Bool? {
⋮----
func loadProfile(currentUser: User?) async {
⋮----
func loadStats(retailerId: String) async -> Bool? {
⋮----
func loadCreditProfile() async {
⋮----
func toggleGlobalAutoOrder(enabled: Bool, useHistory: Bool) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/SearchViewModel.swift
```swift
final class SearchViewModel {
var searchText = ""
var products: [Product] = []
var selectedProduct: Product?
var isLoading = false
⋮----
private let api = APIClient.shared
⋮----
var filteredProducts: [Product] {
⋮----
func loadProducts() async {
⋮----
func clearSearch() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/ShopClosedBannerViewModel.swift
```swift
final class ShopClosedBannerViewModel {
var isSubmitting = false
var errorText: String?
var bypassPending = false
var bypassPhotoURL: String?
var uploading = false
var photoItem: PhotosPickerItem?
⋮----
private let api = APIClient.shared
⋮----
func labelForOption(_ option: String) -> String {
⋮----
func handleReconnect() {
⋮----
func cancelBypass() {
⋮----
func selectOption(_ option: String, orderId: String, onRespond: @escaping (String) -> Void) {
⋮----
func uploadBypassPhoto(_ item: PhotosPickerItem, orderId: String) async {
⋮----
func respond(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/StoreStockViewModel.swift
```swift
// Modal Sheet Controls
⋮----
// Returns / Claim Picker
⋮----
func load() async {
⋮----
func quickPutaway(sku: String, qtyToMove: Int64) async {
⋮----
func openOrderPicker(skuHint: String? = nil) {
⋮----
func configureInitialSku(initialSku: String, candidates: [String]) {
⋮----
func submitReceive(locationId: String) async -> Bool {
⋮----
func submitPutaway(locationId: String) async -> Bool {
⋮----
func submitAdjust(locationId: String) async -> Bool {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/SupplierProductsViewModel.swift
```swift
final class SupplierProductsViewModel {
var products: [Product] = []
var isLoading = false
var errorMessage: String?
var selectedProduct: Product?
var autoOrderSettings: [String: Bool] = [:]
var supplierAutoOrder = false
var isMySupplier = false
var isTogglingMySupplier = false
var isTogglingSupplierAutoOrder = false
var togglingProductIds: Set<String> = []
⋮----
private let api = APIClient.shared
⋮----
func groupedProducts(for supplier: Supplier) -> [(key: String, value: [Product])] {
⋮----
func loadProducts(supplierId: String) async {
⋮----
func loadAutoOrderState(supplierId: String) async {
⋮----
func toggleMySupplier(supplierId: String) async {
⋮----
func toggleSupplierAutoOrder(supplierId: String) async {
⋮----
func toggleProductAutoOrder(productId: String, enabled: Bool) async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/L10n.swift
```swift
enum L10n {
static func format(_ key: String, _ args: CVarArg...) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/retailerappApp.swift
```swift
struct retailerappApp: App {
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
@State private var cartManager = CartManager()
@State private var authManager = AuthManager.shared
⋮----
var body: some Scene {
```

## File: apps/supplier-app-ios/SupplierApp/Design/FlexibleStatusChips.swift
```swift
struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
⋮----
struct StatusChipsFlowLayout: Layout {
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
```

## File: apps/supplier-app-ios/SupplierApp/Design/PegasusEmptyView.swift
```swift
public struct PegasusEmptyView: View {
public let title: String
public let message: String
⋮----
public init(title: String, message: String) {
⋮----
public var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Layout/ResponsiveGridContentWrapper.swift
```swift
struct ResponsiveGridContentWrapper<Content: View>: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@ViewBuilder let content: () -> Content
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Layout/SupplierSection.swift
```swift
enum SupplierSection: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var icon: String {
⋮----
static var compactTabs: [SupplierSection] {
⋮----
static var sidebarSections: [SupplierSection] {
⋮----
static var opsSections: [SupplierSection] {
⋮----
static var intelligenceSections: [SupplierSection] {
⋮----
static var networkSections: [SupplierSection] {
⋮----
static var accountSections: [SupplierSection] {
```

## File: apps/supplier-app-ios/SupplierApp/Services/AppDelegate.swift
```swift

```

## File: apps/supplier-app-ios/SupplierApp/Utilities/FleetDriverMarkerAnimator.swift
```swift
struct AnimatedDriverCoordinate: Identifiable {
let id: String
let lat: Double
let lng: Double
let stale: Bool
⋮----
final class FleetDriverMarkerAnimator {
private struct AnimState {
let driverId: String
var fromLat: Double
var fromLng: Double
var toLat: Double
var toLng: Double
var startMs: TimeInterval
let stale: Bool
⋮----
private var states: [String: AnimState] = [:]
private let durationMs: TimeInterval = 1_200
⋮----
func updateTargets(_ routes: [SupplierFleetLiveRoute]) {
⋮----
func snapshot(now: Date = .now) -> [AnimatedDriverCoordinate] {
⋮----
private func progressAt(nowMs: TimeInterval, state: AnimState) -> Double {
⋮----
private func lerp(_ a: Double, _ b: Double, _ t: Double) -> Double {
⋮----
private func easeOut(_ t: Double) -> Double {
```

## File: apps/supplier-app-ios/SupplierApp/Utilities/SupplierPortalFeature.swift
```swift
enum SupplierPortalFeature: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var title: String {
⋮----
var portalPath: String {
⋮----
var handoffMessage: String {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/CatalogDetailAndCreateViewModels.swift
```swift
final class CatalogCreateViewModel {
var name = ""
var priceMinor = ""
var unitVu = "1"
var barcode = ""
var categories: [CatalogCategory] = []
var categoryId = ""
var currency = packCurrency(MarketPackStore.pack)
var photoItem: PhotosPickerItem?
var imageData: Data?
var creating = false
var error: String?
⋮----
func handlePhotoSelection(_ item: PhotosPickerItem?) async {
⋮----
func loadForm() async {
⋮----
func create() async -> Bool {
⋮----
final class CatalogDetailViewModel {
var product: CatalogProduct?
var loading = true
var error: String?
private var loadGeneration = 0
⋮----
func load(productId: String?, silent: Bool = false) async {
⋮----
final class AddressLocationViewModel {
var query = ""
var suggestions: [GeocodePrediction] = []
var error: String?
var locating = false
private var searchTask: Task<Void, Never>?
⋮----
func scheduleSearch(_ text: String) {
⋮----
func cancelSearch() {
⋮----
func search(_ text: String) async {
⋮----
func pick(placeId: String, fallback: String) async -> AddressLocationValue? {
⋮----
func useMyLocation() async -> AddressLocationValue? {
⋮----
private func makeValue(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/CatalogViewModel.swift
```swift
final class CatalogViewModel {
var products: [CatalogProduct] = []
var draftVU: [String: String] = [:]
var draftBarcode: [String: String] = [:]
var loading = true
var error: String?
var savingId: String?
var imageSavingId: String?
var query = ""
var showCreate = false
⋮----
private var loadGeneration = 0
⋮----
var filtered: [CatalogProduct] {
⋮----
func load(silent: Bool = false) async {
⋮----
func saveUnitVolume(_ product: CatalogProduct) async {
⋮----
func saveProductImage(_ product: CatalogProduct, imageData: Data) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/DashboardViewModel.swift
```swift
private struct DashboardDemandPanel: Sendable {
let generatedAt: String?
let confidence: ForecastConfidence
⋮----
final class DashboardViewModel {
var dashboard: SupplierDashboard?
var meiSummary: SupplierMEIONetworkSummary?
var meiError: String?
var pulseEvents: [SupplierPulseEvent] = []
var pulseLoading = true
var pulseError: String?
var demandConfidence: ForecastConfidence?
var demandGeneratedAt: String?
var demandError: String?
var loading = true
var error: String?
var pack: MarketPack?
var commandJump: CommandStatusJump?
⋮----
private var loadGeneration = 0
⋮----
func loadPack(token: String?) async {
⋮----
func load(tokenStore: TokenStore? = nil, silent: Bool = false) async {
⋮----
private func loadMEI(previous: SupplierMEIONetworkSummary?) async -> PulseHonesty.ObjectResult<SupplierMEIONetworkSummary> {
⋮----
private func loadDemand(previous: DashboardDemandPanel?) async -> PulseHonesty.ObjectResult<DashboardDemandPanel> {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/DispatchPreviewViewModel.swift
```swift
enum DispatchMode: String, CaseIterable {
⋮----
final class DispatchPreviewViewModel {
static let tetrisBuffer = 0.95
⋮----
var preview: SupplierDispatchPreview?
var warehouses: [SupplierTopologyWarehouse] = []
var selectedWarehouseId: String?
var loading = true
var executing = false
var error: String?
var executeMessage: String?
var showAutoConfirm = false
var showCapacityOverride = false
var dispatchMode: DispatchMode = .manual
var selectedDriverId = ""
var selectedOrderIds = Set<String>()
private var loadGeneration = 0
⋮----
func toggleWarehouse(_ warehouseId: String) async {
⋮----
func toggleOrderSelection(_ orderId: String) {
⋮----
func selectAllUndispatched() {
⋮----
func handleReconnect() {
⋮----
func pruneSelectedOrders(validIds: [String]?) {
⋮----
func syncDriverSelection() {
⋮----
func executeAuto() async {
⋮----
func executeManual(forceCapacity: Bool) async {
⋮----
func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/FleetViewModel.swift
```swift
final class FleetViewModel {
var segment = 0
var drivers: [FleetDriver] = []
var vehicles: [FleetVehicle] = []
var loading = true
var error: String?
⋮----
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/MoreAndNotificationsViewModels.swift
```swift
struct CreditAdminDisableBody: Encodable, Sendable {
let ticket_id: String
let reason: String
⋮----
struct CreditAdminDisableResponse: Decodable, Sendable {
let status: String?
let error: String?
⋮----
final class CreditAdminDisableViewModel {
var mode = "relationship"
var supplierId = ""
var retailerId = ""
var ticketId = ""
var reason = ""
var message: String?
var busy = false
⋮----
private let submitter: @Sendable (String, CreditAdminDisableBody, String) async throws -> CreditAdminDisableResponse
⋮----
init(
⋮----
var isProgramMode: Bool {
⋮----
var canSubmit: Bool {
⋮----
var targetPath: String {
⋮----
func submit() async {
⋮----
final class JSONFeedViewModel {
var bodyText = ""
var errorText: String?
var loading = true
private var loadGeneration = 0
private let fetcher: @Sendable (String, [String: String]) async throws -> String
⋮----
init(
⋮----
func load(path: String, query: [String: String], silent: Bool = false) async {
⋮----
final class PlaybooksViewModel {
var rows: [ControlTowerPlaybook] = []
var errorText: String?
var loading = true
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
final class ScoredExceptionsViewModel {
var rows: [ScoredException] = []
var errorText: String?
var loading = true
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
struct SupplierNotificationsPage: Decodable, Sendable {
let notifications: [SupplierNotificationItem]
let unreadCount: Int
let hasMore: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct MarkNotificationsReadRequest: Encodable, Sendable {
let markAll: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class NotificationInboxViewModel {
var items: [SupplierNotificationItem] = []
var unreadCount = 0
var loading = true
var error: String?
private var loadGeneration = 0
⋮----
private let pageFetcher: @Sendable () async throws -> SupplierNotificationsPage
private let markAllHandler: @Sendable (String) async throws -> Void
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func markAllRead() async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/NetworkViewModels.swift
```swift
final class CRMViewModel {
var retailers: [SupplierCRMRetailer] = []
var loading = true
var error: String?
var selected: SupplierCRMRetailerDetail?
private var loadGeneration = 0
private var detailGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
func loadDetail(_ id: String) async {
⋮----
final class TopologyNodesListViewModel {
var warehouses: [SupplierTopologyWarehouse] = []
var factories: [SupplierTopologyFactory] = []
var loading = true
var error: String?
var showAdd = false
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
final class SupplyLanesViewModel {
var lanes: [SupplierSupplyLaneRow] = []
var loading = true
var error: String?
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
final class EntityResolutionViewModel {
static let types = ["ANY", "ORDER", "RETAILER", "WAREHOUSE", "FACTORY", "DRIVER", "VEHICLE", "SUPPLIER"]
⋮----
var entityType = "ANY"
var query = ""
var entityId = ""
var busy = false
var error: String?
var resolved: EntityResolutionResolveResponse?
var explain: EntityResolutionExplainResponse?
⋮----
func runResolve() async {
⋮----
func runExplain(type: String, id: String) async {
⋮----
final class LoyaltyProgramViewModel {
var program: LoyaltyProgram?
var earnBps = "100"
var reason = ""
var status: String?
var error: String?
var loading = true
var busy = false
private var loadGeneration = 0
⋮----
func load(silent: Bool = false) async {
⋮----
func save() async {
⋮----
struct TopologyWarehouseDraft: Identifiable {
let id: String
var warehouseId: String?
var name: String
var location: AddressLocationValue
var coverageRadiusKm: String
⋮----
struct TopologyFactoryDraft: Identifiable {
let id: String
var factoryId: String?
var name: String
var location: AddressLocationValue
var isActive: Bool = true
⋮----
final class TopologyViewModel {
var loading = true
var saving = false
var editing = false
var error: String?
var warehouses: [SupplierTopologyWarehouse] = []
var factories: [SupplierTopologyFactory] = []
var warehouseDrafts: [TopologyWarehouseDraft] = []
var factoryDrafts: [TopologyFactoryDraft] = []
private var loadGeneration = 0
⋮----
func beginEditing() {
⋮----
func cancelEditing() {
⋮----
func addWarehouseDraft() {
⋮----
func addFactoryDraft() {
⋮----
func applyDraftsFromLoaded() {
⋮----
func load(silent: Bool = false) async {
⋮----
func save() async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/OperationsViewModel.swift
```swift
final class OperationsViewModel {
var loading: Bool = true
var loadGeneration: Int = 0
var error: String?
var empathy: SupplierEmpathyAdoption?
var title: String = ""
var bodyText: String = ""
var broadcastRole: String = "ALL"
var orderId: String = ""
var bypassReason: String = ""
var bypassToken: String?
var showBypassConfirm: Bool = false
var broadcasting: Bool = false
var replenishing: Bool = false
var bypassing: Bool = false
var statusMessage: String?
var templateDate: String = ""
⋮----
func loadEmpathy(silent: Bool = false) async {
⋮----
func sendBroadcast(supplierId: String?) async {
⋮----
func runReplenishment() {
⋮----
func issueBypass() async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/OpsAndManifestsViewModels.swift
```swift
final class PromotionsViewModel {
var promotions: [SupplierPromotion] = []
var loading = true
private var loadGeneration = 0
var error: String?
var showCreate = false
var showEdit = false
var editingPromotion: SupplierPromotion?
var name = ""
var discountBps = "500"
var simResults: [String: PromoSimulateResult] = [:]
var simulatingId: String?
⋮----
func beginEdit(_ promo: SupplierPromotion) {
⋮----
func cancelEdit() {
⋮----
func load(silent: Bool = false) async {
⋮----
func create() async {
⋮----
func saveEdit() async {
⋮----
func simulate(_ promo: SupplierPromotion) async {
⋮----
func deactivate(_ promotionId: String) async {
⋮----
final class ProfileViewModel {
var profile: SupplierProfile?
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false, tokenStore: TokenStore) async {
⋮----
final class EarningsViewModel {
var earnings: SupplierEarnings?
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
final class InventoryImportViewModel {
var csvText = ""
var fileName = ""
var step = 0
var sessionId: String?
var sessionStatus = ""
var mappings: [SupplierImportMappingCandidate] = []
var applyResult: SupplierImportApplyResponse?
var directResult: SupplierInventoryImportResult?
var busy = false
var error: String?
⋮----
func directImport() async {
⋮----
func startWizard() async {
⋮----
func approveAndApply(_ sessionId: String) async {
⋮----
final class ReturnsViewModel {
var items: [SupplierReturnRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var resolvingId: String?
var resolution = "RETURN_TO_STOCK"
var notes = ""
var actionLoading: String?
⋮----
func startDispute(returnId: String) {
⋮----
func load(silent: Bool = false) async {
⋮----
func resolve(returnId: String) async {
⋮----
final class ManifestDetailViewModel {
var detail: SupplierManifestDetail?
var loading = true
private var loadGeneration = 0
var error: String?
var actionError: String?
var busy = false
var injectOrderId = ""
⋮----
func handleReconnect() {
⋮----
private func idem(_ manifestId: String, prefix: String, extra: String = "") -> String {
⋮----
func load(manifestId: String, silent: Bool = false) async {
⋮----
func startLoading(manifestId: String) async {
⋮----
func injectOrder(manifestId: String) async {
⋮----
func seal(manifestId: String) async {
⋮----
private func runAction(manifestId: String, _ action: () async throws -> Void) async {
⋮----
final class ManifestExceptionsViewModel {
var rows: [SupplierManifestExceptionRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var escalatedOnly = false
⋮----
func load(silent: Bool = false) async {
⋮----
final class ManifestsViewModel {
var rows: [SupplierManifestRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
final class FleetLiveMapViewModel {
var routes: [SupplierFleetLiveRoute] = []
var exceptionCells: [ExceptionMapCell] = []
var zoneOverrides: [ControlTowerZoneOverride] = []
var loading = true
private var loadGeneration = 0
var error: String?
var overlayError: String?
var cameraPosition: MapCameraPosition = .region(
⋮----
var animator = FleetDriverMarkerAnimator()
var hasFittedInitialCamera = false
var publishAction = "REROUTE"
var publishStatus: String?
var publishing = false
var showPublishSheet = false
var mapRegion = MKCoordinateRegion(
⋮----
func load(silent: Bool = false) async {
⋮----
private func fitCamera() {
⋮----
func startPolling() async {
⋮----
func publishZoneOverride() async {
⋮----
private func polygonFromCurrentCamera() -> GeoJSONPolygonPayload {
⋮----
final class FleetOrdersViewModel {
var rows: [SupplierFleetOrderRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/OrderDetailViewModel.swift
```swift
final class OrderDetailViewModel {
var warehouseDetail: WarehouseOrderDetail?
var detailError: String?
var opsReason = ""
var proposeDate = Date()
var showProposeSheet = false
var showRejectDialog = false
⋮----
private static let tashkentIsoFormatter: ISO8601DateFormatter = {
⋮----
func isoDeliveryDate() -> String {
⋮----
func loadWarehouseDetail(for order: SupplierOrder) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/PlanningAndExceptionsViewModels.swift
```swift
final class KnowledgeGraphViewModel {
var graph: SupplierKnowledgeGraph?
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
private struct DemandPanel {
let generatedAt: String?
let confidence: ForecastConfidence
⋮----
final class PlanningBrainViewModel {
var tab: PlanBrainTab = .planning
var sandop: PlanningSAndOPSnapshot?
var scenario: PlanningScenarioResult?
var downtimeHours = 8.0
var demandDeltaPct = 10.0
var loading = true
private var loadGeneration = 0
var running = false
var error: String?
var demandConfidence: ForecastConfidence?
var demandGeneratedAt: String?
var demandError: String?
var retailerId = ""
var sparsity: SparsityGateResult?
var sparsityError: String?
var pushStatus: String?
⋮----
func load(silent: Bool = false) async {
⋮----
private func loadDemand(previous: DemandPanel?) async -> PulseHonesty.ObjectResult<DemandPanel> {
⋮----
func runScenario() async {
⋮----
func checkSparsity() async {
⋮----
func runPush() async {
⋮----
private struct NetworkModeRead {
let mode: String
let planningEnabled: Bool
⋮----
final class PlanningSettingsViewModel {
var data: SeasonalTemplatesResponse?
var loading = true
private var loadGeneration = 0
var saving = false
var error: String?
var formError: String?
var templateId = ""
var name = ""
var startDate = ""
var endDate = ""
var multiplier = ""
var networkMode = ""
var planningEnabled = false
var modeKnown = false
var modeError: String?
var opsStatus: String?
var killReason = ""
var opsBusy = false
⋮----
func load(silent: Bool = false) async {
⋮----
private func loadNetworkMode(previous: NetworkModeRead?) async -> PulseHonesty.ObjectResult<NetworkModeRead> {
⋮----
func createOverride() async {
⋮----
func putMode(_ mode: String) async {
⋮----
func runPullMatrix() async {
⋮----
func runPredictivePush() async {
⋮----
func runKillSwitch() async {
⋮----
final class ReplenishmentPoliciesViewModel {
var policy: SupplierReplenishmentPolicy?
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
final class ClaimsViewModel {
var claims: [SupplierClaim] = []
var statusFilter = "OPEN"
var settlementMode = "LEDGER_ONLY"
var loading = true
private var loadGeneration = 0
var error: String?
var busyId: String?
var lastSettlement: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func approve(_ claimId: String) async {
⋮----
func reject(_ claimId: String) async {
⋮----
final class EarlyCompleteViewModel {
var driverId = ""
var busy = false
var error: String?
var success: String?
⋮----
func handleReconnect() {
⋮----
func approve() async {
⋮----
final class ExceptionsViewModel {
var rows: [SupplierExceptionRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var busyKey: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func resolve(_ row: SupplierExceptionRow) async {
⋮----
final class ShopClosedViewModel {
var rows: [ShopClosedAttemptRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var busyId: String?
var statusMessage: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func handleReconnect() async {
⋮----
func resolve(_ attemptId: String, action: String) async {
⋮----
final class ActivityViewModel {
var rows: [SupplierActivityEvent] = []
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
final class AIRecommendationsViewModel {
var filter = "PENDING"
var loading = true
private var loadGeneration = 0
var error: String?
var items: [SupplierAIRecommendation] = []
var pendingDecisionId: String?
var feedback: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func record(_ rec: SupplierAIRecommendation, decision: String) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/SettingsViewModels.swift
```swift
struct EditableNotificationPreference: Identifiable, Equatable {
let id: String
let eventType: String
let channel: String
var enabled: Bool
⋮----
final class NotificationPreferencesViewModel {
var items: [EditableNotificationPreference] = []
var loading = true
var saved = false
var error: String?
private var loadGeneration = 0
⋮----
func load() async {
⋮----
func save() async {
⋮----
final class ReturnPolicySettingsViewModel {
static let hourPresets: [Int64] = [8, 24, 48, 72]
⋮----
var loading = true
var hasLoaded = false
var saving = false
var error: String?
var saved = false
var hours: Int64 = 48
var hoursText = "48"
var concealed = ""
var requirePhoto = true
var allowExpired = false
var sourceHint = ""
private var loadGeneration = 0
⋮----
func selectPreset(_ preset: Int64) {
⋮----
func sanitizeHoursText(_ newValue: String) {
⋮----
func sanitizeConcealed(_ newValue: String) {
⋮----
func load(silent: Bool = false) async {
⋮----
func save() async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/SupplierShellAndAuthViewModels.swift
```swift
final class SupplierShellViewModel {
var sidebarSelection: SupplierSection? = .dashboard
var isSidebarExpanded = true
var compactTab: CompactTab = .dashboard
var clientPolicyMessage: String?
var clientPolicyForce = false
var pendingManifest: AutoUpdater.Manifest?
⋮----
func dismissBanner() {
⋮----
func triggerUpdateInstall() {
⋮----
func loadClientPolicy() async {
⋮----
struct ClientPolicy: Decodable {
let outdated: Bool
let forceUpdate: Bool
let updateDeferred: Bool
let minimumVersion: String
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct BillingGatewayOption: Identifiable, Equatable {
let id: String
var isSelected: Bool
⋮----
final class BillingGateViewModel {
var bankName = ""
var accountHolder = ""
var accountNumber = ""
var swiftBic = ""
var iban = ""
var gatewayOptions: [BillingGatewayOption] = [
⋮----
var loading = false
var error: String?
⋮----
var selectedGateways: [String] {
⋮----
var canSubmit: Bool {
⋮----
func submit(tokenStore: TokenStore) async {
⋮----
final class SupplierLoginViewModel {
var phone = ""
var password = ""
var loading = false
var error: String?
⋮----
var canLogin: Bool {
⋮----
func login(tokenStore: TokenStore) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/TreasuryDetailViewModels.swift
```swift
final class CashReconciliationsViewModel {
var rows: [CashReconciliationRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var busyId: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func accept(_ id: String) async {
⋮----
final class ChargebacksViewModel {
var orderId = ""
var retailerId = ""
var gateway = "ADYEN"
var amount = ""
var currency = packCurrency(MarketPackStore.pack)
var sessionId = ""
var busy = false
var chargebackMessage: String?
var reversalMessage: String?
var error: String?
⋮----
func handleReconnect() {
⋮----
func submitChargeback() async {
⋮----
func submitReversal() async {
⋮----
final class ClaimChargebacksViewModel {
var items: [PaymentLedgerEntry] = []
var orderFilter = ""
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
var total: Int64? {
⋮----
func load(silent: Bool = false) async {
⋮----
final class ComplianceAuditViewModel {
var loading = true
private var loadGeneration = 0
var error: String?
var summary: ComplianceSummary?
var forceCompletes: [ComplianceForceCompleteRow] = []
var openFiscal: [ComplianceFiscalOpenRow] = []
⋮----
private let loadDashboardFn: @Sendable () async throws -> ComplianceDashboardResponse
private let fetchReceiptFn: @Sendable (String) async throws -> OrderReceiptMeta
private let openURLFn: @MainActor (URL) -> Void
⋮----
init(
⋮----
func openReceipt(orderId: String) async {
⋮----
func load(silent: Bool = false) async {
⋮----
final class CreditNotesListViewModel {
var rows: [CreditNoteRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var busyId: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func issue(_ id: String) async {
⋮----
final class CreditProfilesViewModel {
var rows: [CreditProfileRow] = []
var loading = true
private var loadGeneration = 0
var error: String?
var busyId: String?
⋮----
func load(silent: Bool = false) async {
⋮----
func setStatus(_ row: CreditProfileRow, status: String) async {
⋮----
final class LedgerViewModel {
var rows: [PaymentLedgerEntry] = []
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
⋮----
final class PaymentsViewModel {
var loading = true
private var loadGeneration = 0
var error: String?
var authority: SettlementAuthorityResponse?
var mismatches: [ReconciliationMismatchRow] = []
⋮----
func load(silent: Bool = false) async {
⋮----
func load(silent: Bool = false) async {
⋮----
func savePolicy() async {
⋮----
func generate() async {
⋮----
func load(silent: Bool = false) async {
⋮----
func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogProductImageButton.swift
```swift
final class CatalogImagePickerViewModel {
var photoItem: PhotosPickerItem?
var error: String?
⋮----
func handleSelection(item: PhotosPickerItem?, onUpload: (Data) async -> Void) async {
⋮----
struct CatalogProductImageButton: View {
let product: CatalogProduct
let isSaving: Bool
let onUpload: (Data) async -> Void
⋮----
@State private var vm = CatalogImagePickerViewModel()
⋮----
var body: some View {
⋮----
private var label: String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/ExceptionsList.swift
```swift
private let resolvableKinds: Set<String> = ["CASH_DISCREPANCY", "CREDIT_NOTE_DRAFT", "CREDIT_FREEZE"]
⋮----
struct ExceptionsList: View {
let rows: [SupplierExceptionRow]
var busyKey: String? = nil
var onResolve: (SupplierExceptionRow) -> Void = { _ in }
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Fleet/FleetRowViews.swift
```swift
struct DriverRow: View {
let driver: FleetDriver
⋮----
var body: some View {
⋮----
struct VehicleRow: View {
let vehicle: FleetVehicle
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Manifests/ManifestsList.swift
```swift
struct ManifestsList: View {
let items: [SupplierManifestRow]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/More/CreditAdminDisableView.swift
```swift
struct CreditAdminDisableView: View {
@State private var vm = CreditAdminDisableViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/More/JSONFeedView.swift
```swift
struct JSONFeedView: View {
let title: String
let path: String
var query: [String: String] = [:]
@State private var vm = JSONFeedViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/More/PlaybooksView.swift
```swift
struct PlaybooksView: View {
@State private var vm = PlaybooksViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/More/ScoredExceptionsView.swift
```swift
struct ScoredExceptionsView: View {
@State private var vm = ScoredExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/CRMView.swift
```swift
enum CRMDetailLoad {
struct Outcome<T> {
let value: T?
let failure: Error?
⋮----
static func selection<T, Failure: Error>(
⋮----
struct CRMView: View {
@State private var vm = CRMViewModel()
⋮----
var body: some View {
⋮----
static func formatCRMOrderAmount(_ amount: Int64?, currency: String = packCurrency(MarketPackStore.pack)) -> String {
⋮----
static func formatCRMOrderLineAmount(_ amountMinor: Int64?, currency: String = packCurrency(MarketPackStore.pack)) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/EntityResolutionView.swift
```swift
struct EntityResolutionView: View {
@State private var vm = EntityResolutionViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/FactoryList.swift
```swift
struct FactoryList: View {
let factories: [SupplierTopologyFactory]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/LoyaltyProgramView.swift
```swift
struct LoyaltyProgramView: View {
@State private var vm = LoyaltyProgramViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/SupplyLaneRows.swift
```swift
struct LaneMetricRow: View {
let label: String
let value: String
⋮----
var body: some View {
⋮----
struct LaneUtilizationRow: View {
let pct: Double
⋮----
private var tint: Color {
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/TopologyCenteredEmptyState.swift
```swift
struct TopologyCenteredEmptyState: View {
let title: String
let message: String
let actionLabel: String
let action: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Orders/OrderRow.swift
```swift
struct OrderRow: View {
let order: SupplierOrder
var showWarehouseMenu: Bool = false
var onDelay: (() -> Void)?
var onReject: (() -> Void)?
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/OrgFleetUtils.swift
```swift
enum OrgFleetUtils {
static func nodeLabel(type: String, id: String, topology: SupplierTopologyResponse?) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Portal/PortalHandoffView.swift
```swift
struct PortalHandoffView: View {
let feature: SupplierPortalFeature
@Environment(\.openURL) private var openURL
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/CashReconciliationsView.swift
```swift
struct CashReconciliationsView: View {
@State private var viewModel = CashReconciliationsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/CreditNotesListView.swift
```swift
struct CreditNotesListView: View {
@State private var viewModel = CreditNotesListViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/CreditProfilesView.swift
```swift
struct CreditProfilesView: View {
@State private var viewModel = CreditProfilesViewModel()
⋮----
var body: some View {
⋮----
static func formatCreditProfileSummary(limitMinor: Int64?, balanceMinor: Int64?, availMinor: Int64?) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/PayoutsView.swift
```swift
struct PayoutsView: View {
@State private var viewModel = PayoutsViewModel()
⋮----
var body: some View {
⋮----
static func formatBatchPayout(status: String, netPayoutMinor: Int64?, currency: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/RoutePerformanceListView.swift
```swift
struct RoutePerformanceListView: View {
@State private var viewModel = RoutePerformanceListViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/PegasusSupplierApp.swift
```swift
struct PegasusSupplierApp: App {
@State private var tokenStore = TokenStore.shared
@State private var realtimeHub = SupplierRealtimeHub()
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
⋮----
var body: some Scene {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Dispatch/DispatchDriverList.swift
```swift
struct DispatchDriverList: View {
let availableDrivers: [AvailableDriver]
let unavailableDrivers: [AvailableDriver]
⋮----
var body: some View {
⋮----
private func driverRow(_ driver: AvailableDriver, unavailableDetail: Bool) -> some View {
⋮----
private func driverSubtitle(_ driver: AvailableDriver, unavailableDetail: Bool) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Orders/OrderLineItems.swift
```swift
struct OrderLineItems: View {
let lineItems: [LineItem]
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Orders/OrderOpsActions.swift
```swift
struct OrderOpsActions: View {
let state: String
@Binding var proposeDate: Date
@Binding var reasonInput: String
@Binding var showProposeSheet: Bool
@Binding var pendingAction: OrderMutationAction?
let mutating: Bool
let onLoadRecommendations: () -> Void
⋮----
var body: some View {
⋮----
private func showOps(for state: String) -> Bool {
⋮----
private func canProposeDate(_ state: String) -> Bool {
⋮----
private func canReject(_ state: String) -> Bool {
⋮----
private func canOverflow(_ state: String) -> Bool {
⋮----
private func canReassign(_ state: String) -> Bool {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalActivityTimeline.swift
```swift
struct TacticalTimelineItem: Identifiable {
let id: String
let timestamp: String
let title: String
let description: String
let actor: String
var icon: String = "circle.fill"
var tint: Color = TacticalTheme.accentCobalt
⋮----
struct TacticalActivityTimeline: View {
let items: [TacticalTimelineItem]
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalGaugeCard.swift
```swift
struct TacticalGaugeCard: View {
let title: String
let percentage: Double
var subtitle: String? = nil
var tint: Color = TacticalTheme.accentCobalt
⋮----
private var percentageClamped: Double {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalKpiCard.swift
```swift
struct TacticalKpiCard: View {
let title: String
let value: String
let systemImage: String
var tint: Color = TacticalTheme.accentCobalt
var delta: String? = nil
var deltaIsPositive: Bool? = nil
var subtitle: String? = nil
var alertChip: String? = nil
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalLedProgressBar.swift
```swift
struct TacticalLedProgressBar: View {
let title: String
let current: Int
let total: Int
var tint: Color = TacticalTheme.accentCobalt
var blockCount: Int = 12
⋮----
private var activeBlocks: Int {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalStageStepper.swift
```swift
struct TacticalStageStepper: View {
let stages: [String]
let currentStageIndex: Int
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalStatusBadge.swift
```swift
struct TacticalStatusBadge: View {
let status: String
var overrideTint: Color? = nil
var showPulse: Bool = true
⋮----
private var tint: Color {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/PegasusFeedbackViews.swift
```swift
public struct PegasusErrorView: View {
public let message: String
public let retry: () -> Void
⋮----
public init(message: String, retry: @escaping () -> Void) {
⋮----
public var body: some View {
⋮----
public struct PegasusEmptyView: View {
public let title: String
public let message: String
⋮----
public init(title: String, message: String) {
⋮----
public var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/FlexibleStatusChips.swift
```swift
struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
⋮----
private struct FlowLayout: Layout {
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/ResponsiveGridContentWrapper.swift
```swift
struct ResponsiveGridContentWrapper<Content: View>: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@ViewBuilder let content: () -> Content
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/ResponsiveGridView.swift
```swift
struct ResponsiveGridView<Data: RandomAccessCollection, Content: View>: View where Data.Element: Identifiable {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
let data: Data
@ViewBuilder let content: (Data.Element) -> Content
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/TacticalWorkbenchView.swift
```swift
struct TacticalWorkbenchView: View {
let section: WarehouseSection
var onInspect: (() -> Void)? = nil
⋮----
private let stages = ["Inbound ASN", "Dock Receiving", "Pick Waves", "Staging Lanes", "Gate Dispatch"]
⋮----
var body: some View {
⋮----
private func stagingLaneRow(
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/WarehouseSectionHeader.swift
```swift
struct WarehouseSectionHeader: View {
let title: String
var subtitle: String? = nil
⋮----
var body: some View {
⋮----
struct WarehouseStatusBadge: View {
let text: String
var tint: Color? = nil
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Models/WarehouseLocationModels.swift
```swift
struct WarehouseLocationResponse: Decodable {
let warehouseId: String
let name: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
let countryCode: String
let packCountryCode: String
let currencyCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseLocationPatchRequest: Encodable {
let address: String
let placeId: String?
let lat: Double
let lng: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseSetupRequest: Encodable {
let name: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseSetupResponse: Decodable {
let warehouseId: String
let token: String?
let refreshToken: String?
let isConfigured: Bool
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/warehouse-app-ios/WarehouseApp/Models/WarehouseOpsModels.swift
```swift
struct EmergencyTransferRequest: Encodable {
let totalVolumeVu: Double
let notes: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ForceReceiveRequest: Encodable {
let factoryId: String?
let totalVolumeVu: Double
let notes: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TransferMutationResponse: Decodable {
let transferId: String
let state: String
let notes: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReplenishmentInsight: Decodable, Identifiable {
let id: String
let warehouseId: String
let warehouseName: String
let productId: String
let productName: String
let urgency: String
let currentStock: Int64
let avgDailyVelocity: Double
let daysUntilStockout: Int
let reorderQuantity: Int64
let status: String
let createdAt: String
let reasonCode: String?
let demandBreakdown: [String: AnyCodable]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReplenishmentInsightsResponse: Decodable {
let insights: [ReplenishmentInsight]
let data: [ReplenishmentInsight]?
⋮----
var rows: [ReplenishmentInsight] {
⋮----
struct ReplenishmentInsightActionResponse: Decodable {
let insightId: String
let status: String
let transferId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchSettingsResponse: Decodable {
let warehouseId: String
let autoDispatchEnabled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchSettingsPatchRequest: Encodable {
let autoDispatchEnabled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct OpsFinancialsResponse: Decodable {
let warehouseId: String
let period: String
let currency: String
let totalRevenue: Int64
let completedOrders: Int64
let avgOrderValue: Int64
let platformFee: Int64
let netPayout: Int64
let cashPending: Int64
let cashCollected: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOrderMutationRequest: Encodable {
let reason: String?
⋮----
struct WarehouseProposeDeliveryRequest: Encodable {
let proposedDeliveryDate: String
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOrderMutationResponse: Decodable {
let orderId: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RefreshTokenRequest: Encodable {
let refreshToken: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RouteCoordinateWire: Decodable {
let lat: Double
let lng: Double
⋮----
struct RouteGeometryWire: Decodable {
let routeId: String?
let encodedPolyline: String?
let coordinates: [RouteCoordinateWire]
let source: String
let stopCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseDriverLocationWire: Decodable {
let driverId: String
let supplierId: String?
let lat: Double
let lng: Double
let latitude: Double
let longitude: Double
let reportedAt: String
let receivedAt: String
let staleAfterSeconds: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var resolvedLatitude: Double { latitude != 0 ? latitude : lat }
var resolvedLongitude: Double { longitude != 0 ? longitude : lng }
⋮----
struct WarehouseFleetLiveRoute: Decodable, Identifiable {
var id: String { manifestId }
let manifestId: String
let routeId: String
let driverId: String
let driverName: String?
let manifestState: String
let routeGeometry: RouteGeometryWire?
let driverLocation: WarehouseDriverLocationWire?
let liveLocationAvailable: Bool
let locationStale: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseFleetLiveMapResponse: Decodable {
let routes: [WarehouseFleetLiveRoute]
let warehouseId: String
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct BroadcastTemplate: Decodable, Identifiable {
let id: String
let category: String
let title: String
let body: String
let defaultRole: String
let scope: String
let source: String?
let warehouseId: String?
let placeholderKeys: [String]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct BroadcastTemplatesResponse: Decodable {
let templates: [BroadcastTemplate]
⋮----
struct WarehouseBroadcastRequest: Encodable {
let title: String
let body: String
let role: String?
⋮----
struct WarehouseBroadcastResponse: Decodable {
let status: String
let warehouseId: String
let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseBroadcastTemplateCreateRequest: Encodable {
let title: String
let body: String
let defaultRole: String?
let category: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct BroadcastTemplateDeleteResponse: Decodable {
let status: String
let templateId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOverridePreview: Decodable {
let retailersOnSkuCount: Int
let activeOverrideCount: Int
let catalogListPrice: Int64
let marginDeltaPerUnit: Int64
let marginEstimateLabel: String
let affectedRetailerIds: [String]?
let readOnly: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOverridePreviewRequest: Encodable {
let retailerId: String?
let productId: String?
let skuId: String?
let proposedPrice: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehousePulseEvent: Decodable, Identifiable {
let id: String
let kind: String
let title: String
let description: String?
let occurredAt: String
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehousePulseResponse: Decodable {
let events: [WarehousePulseEvent]
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum HandoffPulseSupport {
private static let handoffKinds: Set<String> = [
⋮----
static func filter(_ events: [WarehousePulseEvent]) -> [WarehousePulseEvent] {
⋮----
private static func isHandoff(_ event: WarehousePulseEvent) -> Bool {
⋮----
struct RecommendReassignRequest: Encodable {
let orderId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TruckRecommendation: Decodable {
let driverId: String
let driverName: String
let vehicleId: String
let vehicleClass: String
let licensePlate: String
let score: Double
let reason: String
let toRoute: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RecommendReassignResponse: Decodable {
let orderId: String
let retailerName: String
let orderVolumeVu: Double
let currentDriver: String
let recommendations: [TruckRecommendation]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReassignOrderRequest: Encodable {
let orderId: String
let toDriverId: String
let reason: String
let toManifestId: String?
let isPartial: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ScoredException: Decodable, Identifiable {
var id: String { exceptionId }
let exceptionId: String
let type: String
let severity: String
let orderId: String
let score: Int64
let ageMinutes: Int64
let topPlaybookName: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ScoredExceptionsResponse: Decodable {
let exceptions: [ScoredException]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case exceptions }
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/AppDelegate.swift
```swift
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
func application(
⋮----
func application(
⋮----
func application(
⋮----
nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
func userNotificationCenter(
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/TokenStore.swift
```swift
final class TokenStore {
static let shared = TokenStore()
⋮----
private(set) var token: String?
private(set) var refreshToken: String?
private(set) var warehouseId: String?
⋮----
var isAuthenticated: Bool { token != nil }
⋮----
var isConfigured: Bool { JwtPayload.isConfigured(token: token) }
⋮----
var hasAssignedWarehouse: Bool { JwtPayload.homeNodeId(token: token) != nil }
⋮----
private let service = "com.pegasus.warehouse"
⋮----
private init() {
⋮----
func store(auth: AuthResponse) {
⋮----
func updateTokens(token: String, refresh: String, warehouseId: String? = nil) {
⋮----
func clear() {
⋮----
private func writeKeychain(account: String, value: String) {
⋮----
private func readKeychain(account: String) -> String? {
⋮----
private func deleteKeychain(account: String) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/FleetDriverMarkerAnimator.swift
```swift
struct AnimatedDriverCoordinate: Identifiable {
let id: String
let lat: Double
let lng: Double
let stale: Bool
⋮----
final class FleetDriverMarkerAnimator {
private struct AnimState {
let driverId: String
var fromLat: Double
var fromLng: Double
var toLat: Double
var toLng: Double
var startMs: TimeInterval
let stale: Bool
⋮----
private var states: [String: AnimState] = [:]
private let durationMs: TimeInterval = 1_200
⋮----
func updateTargets(_ routes: [WarehouseFleetLiveRoute]) {
⋮----
func snapshot(now: Date = .now) -> [AnimatedDriverCoordinate] {
⋮----
private func progressAt(nowMs: TimeInterval, state: AnimState) -> Double {
⋮----
private func lerp(_ a: Double, _ b: Double, _ t: Double) -> Double {
⋮----
private func easeOut(_ t: Double) -> Double {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/JwtPayload.swift
```swift
enum JwtPayload {
static func isConfigured(token: String?) -> Bool {
⋮----
static func homeNodeId(token: String?) -> String? {
⋮----
private static func decode(_ token: String?) -> [String: Any]? {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/WarehousePortalFeature.swift
```swift
enum WarehousePortalFeature: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var title: String {
⋮----
var subtitle: String {
⋮----
var portalPath: String {
⋮----
var systemImage: String {
⋮----
var handoffMessage: String {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/AddressLocationFieldViewModel.swift
```swift
final class AddressLocationFieldViewModel {
var query = ""
var suggestions: [GeocodePrediction] = []
var error: String?
var locating = false
var resolving = false
⋮----
private var searchTask: Task<Void, Never>?
⋮----
private let autocompleteHandler: @Sendable (String) async throws -> [GeocodePrediction]
private let resolvePlaceHandler: @Sendable (String) async throws -> ResolvedLocation
private let reverseHandler: @Sendable (Double, Double) async throws -> ResolvedLocation
private let resolveLocationValueHandler: @MainActor (AddressLocationValue) async -> AddressLocationValue?
private let coordinateProvider: @MainActor () -> CLLocationCoordinate2D?
⋮----
init(
⋮----
func syncExternalAddress(_ address: String) {
⋮----
func scheduleSearch(_ text: String, debounceNanoseconds: UInt64 = 280_000_000) {
⋮----
func cancelSearch() {
⋮----
func search(_ text: String) async {
⋮----
func pick(placeId: String, fallback: String, currentValue: AddressLocationValue) async -> AddressLocationValue {
⋮----
func resolveOnBlur(currentValue: AddressLocationValue) async -> AddressLocationValue {
⋮----
func resolveText(_ text: String, currentValue: AddressLocationValue) async -> AddressLocationValue {
⋮----
func useMyLocation(currentValue: AddressLocationValue) async -> AddressLocationValue {
⋮----
private func apply(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/AdjustInventoryViewModel.swift
```swift
final class AdjustInventoryViewModel {
⋮----
let item: InventoryItem
var qty: String
var reason = ""
var showConfirm = false
var submitting = false
var error: String?
⋮----
private let adjustStock: AdjustStock
⋮----
init(
⋮----
var skuLabel: String {
⋮----
var parsedQuantity: Int? {
⋮----
var canReview: Bool {
⋮----
func handleReconnect() {
⋮----
func save() async -> Bool {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/AnalyticsViewModel.swift
```swift
final class AnalyticsViewModel {
⋮----
var data = AnalyticsData.empty
var loading = true
var error: String?
var period = "30d"
⋮----
private let fetchAnalytics: FetchAnalytics
⋮----
init(fetchAnalytics: FetchAnalytics? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/BarcodeScannerViewModel.swift
```swift
struct ScannedBarcodeRecord: Identifiable, Sendable, Hashable {
let id = UUID()
let code: String
let timestamp: Date
⋮----
final class BarcodeScannerViewModel {
enum ScanState: Equatable {
⋮----
var scanState: ScanState = .idle
var scannedRecords: [ScannedBarcodeRecord] = []
var manualBarcode: String = ""
var scannerActive: Bool = true
⋮----
private var resetTask: Task<Void, Never>?
private let resetDelay: Duration
private let playSounds: Bool
⋮----
init(resetDelay: Duration = .seconds(2), playSounds: Bool = true) {
⋮----
var isManualSubmitDisabled: Bool {
⋮----
func clearHistory() {
⋮----
func submitManualBarcode() -> Bool {
⋮----
func handleBarcodeReceived(_ rawCode: String) {
⋮----
func deleteRecords(at offsets: IndexSet) {
⋮----
private func triggerErrorFeedback(message: String) {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/CashierViewModel.swift
```swift
struct DenominationRowItem: Identifiable, Sendable {
var id: Int64 { noteUzs }
let noteUzs: Int64
var count: Int64
⋮----
var totalUzs: Int64 {
⋮----
var totalTiyin: Int64 {
⋮----
final class CashierViewModel {
private static let isoFormatter = ISO8601DateFormatter()
⋮----
var depotId: String = "depot-tashkent-south"
var supplierId: String = "supplier-tashkent-1"
var driverId: String = "driver-uz-09"
var routeId: String = "route-tashkent-042"
var bagSealBarcode: String = "BAG-UZ-99120"
⋮----
var session: CashierSessionView? = nil
var attemptNumber: Int64 = 1
⋮----
var denominations: [Int64: Int64] = [
⋮----
var countedCashMinor: Int64 {
⋮----
var isBlindCountMasked: Bool = true
var isRecountRequired: Bool = false
var isBalanced: Bool = false
var isQuarantined: Bool = false
var clearanceTokenHash: String? = nil
var microSkimmingFlagged: Bool = false
⋮----
var isLoading: Bool = false
var statusMessage: String? = nil
var errorMessage: String? = nil
⋮----
var amlExposureMinor: Int64 = 0
var amlCeilingMinor: Int64 = 0
var amlExceeded: Bool = false
var lastVaultEntry: DepotVaultLedgerEntry? = nil
⋮----
var showRRNSheet: Bool = false
var showCITSheet: Bool = false
var showAMLSheet: Bool = false
⋮----
var rrnOrderId: String = "order-uz-8812"
var rrnGateway: String = "123456789012"
var rrnDisputedText: String = "500000"
⋮----
var citBadgeId: String = "INKAS-9921"
var citSealBarcode: String = "CIT-SEAL-7712"
var citHandoverText: String = "1425000"
⋮----
var amlTin: String = "123456789"
var amlExposureText: String = "15000000"
⋮----
var totalBanknotesCount: Int64 {
⋮----
private let startSessionHandler: @Sendable (StartCashierSessionRequest) async throws -> CashierSessionView
private let submitBlindCountHandler: @Sendable (String, BlindCountRequest) async throws -> CashierSessionView
private let reclassifyTenderHandler: @Sendable (RRNTenderReclassificationRequest) async throws -> CashierSessionView
private let executeCITHandoverHandler: @Sendable (CITHandoverRequest) async throws -> DepotVaultLedgerEntry
private let checkAMLHandler: @Sendable (AMLCheckRequest) async throws -> AMLCheckResponse
⋮----
init(
⋮----
func setDenominationCount(noteUzs: Int64, count: Int64) {
⋮----
func adjustCount(noteUzs: Int64, delta: Int64) {
⋮----
func addStrap(noteUzs: Int64, strapCount: Int64) {
⋮----
func clearCounts() {
⋮----
func quickFillSampleBalanced() {
⋮----
func quickFillSampleDiscrepant() {
⋮----
static func formatUzs(amount: Int64) -> String {
⋮----
func startSession(expectedCashMinor: Int64 = 1425000000) async {
⋮----
func submitBlindCount(toleranceThresholdMinor: Int64 = 500000) async {
⋮----
func reclassifyTender(
⋮----
func executeCITHandover(
⋮----
func checkAMLExposure(tin: String, cashMinor: Int64) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ClaimsViewModel.swift
```swift
final class ClaimsViewModel {
⋮----
let statusFilters = ["OPEN", "UNDER_REVIEW", "RESOLVED", "REJECTED", ""]
⋮----
var status = "OPEN"
var claims: [WarehouseClaim] = []
var loading = true
var error: String?
⋮----
private let fetchClaims: FetchClaims
⋮----
init(fetchClaims: FetchClaims? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/CoverageViewModel.swift
```swift
final class CoverageViewModel {
⋮----
var coverage: WarehouseCoverageResponse?
var factory: WarehouseSupplyFactoryResponse?
var factoryError: String?
var loading = true
var error: String?
⋮----
private let fetchCoverage: FetchCoverage
private let fetchFactory: FetchFactory
⋮----
init(
⋮----
func coverageModeLabel(_ mode: String?) -> String {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/CRMViewModel.swift
```swift
final class CRMViewModel {
⋮----
var retailers: [Retailer] = []
var loading = true
var error: String?
⋮----
private let fetchRetailers: FetchRetailers
⋮----
init(fetchRetailers: FetchRetailers? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/DashboardViewModel.swift
```swift
final class DashboardViewModel {
⋮----
var stats = DashboardData.empty
var loading = true
var hasData = false
var error: String?
var commandJump: CommandStatusJump?
⋮----
private let fetchDashboard: FetchDashboard
⋮----
init(fetchDashboard: FetchDashboard? = nil) {
⋮----
func load(silent: Bool = false) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/DemandForecastViewModel.swift
```swift
final class DemandForecastViewModel {
⋮----
var horizon = 7
var forecast = DemandForecastResponse()
var loading = true
var error: String?
var selectedSegment = 0
⋮----
var criticalCount: Int {
⋮----
var urgentCount: Int {
⋮----
var normalCount: Int {
⋮----
private let fetchForecast: FetchForecast
private let fetchInsights: FetchInsights
⋮----
init(
⋮----
func load() async {
⋮----
static func demandForecastFromInsights(_ insights: [ReplenishmentInsight], horizon: Int) -> DemandForecastResponse {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/DispatchSettingsViewModel.swift
```swift
final class DispatchSettingsViewModel {
⋮----
var autoDispatchEnabled = false
var hasLoaded = false
var loading = true
var saving = false
var error: String?
var saveMessage: String?
⋮----
private var lastSyncedValue: Bool?
private let fetchSettings: FetchSettings
private let patchSettings: PatchSettings
⋮----
init(
⋮----
func load() async {
⋮----
func handleToggleChange(_ newValue: Bool) async {
⋮----
func save(enabled: Bool) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/DispatchViewModel.swift
```swift
enum DispatchAssignmentMode: String, CaseIterable, Identifiable, Sendable {
⋮----
var id: String { rawValue }
var label: String {
⋮----
struct DispatchOrderDetailRoute: Identifiable, Hashable, Sendable {
let id: String
init(id: String) {
⋮----
struct DispatchActionAlert: Identifiable, Sendable {
let id = UUID()
let title: String
let message: String
init(title: String, message: String) {
⋮----
final class DispatchViewModel {
var preview: DispatchPreview?
var fleetVehicles: [Vehicle] = []
var vehicleReasons: [String: String] = [:]
var vehicleNotes: [String: String] = [:]
var mutatingFleetVehicleId: String?
var fleetAlert: String?
var supplyRequests: [WarehouseSupplyRequest] = []
var dispatchLocks: [WarehouseDispatchLock] = []
var loading = true
var error: String?
var selectedSegment = 0
var realtimeClient = WarehouseRealtimeClient()
var realtimeStatus: WarehouseRealtimeStatus = .idle
var showCreateSupplyRequest = false
var showAcquireDispatchLock = false
var requestPendingCancellation: WarehouseSupplyRequest?
var lockPendingRelease: WarehouseDispatchLock?
var actionAlert: DispatchActionAlert?
var executing = false
var selectedDriverId = ""
var showRescueDialog = false
var lookupDriverId = ""
var lookupLoading = false
var lookupError: String? = nil
var earlyCompleteReq: [String: String]? = nil
var rescueAction: String? = nil
var rescueWindowStart = ""
var rescueWindowEnd = ""
var rescueMutating = false
⋮----
var selectedOrderIds: Set<String> = []
var showCapacityDialog = false
var capacityDialogAutoMode = false
var showSmartConfirm = false
var dispatchMode: DispatchAssignmentMode = .smart
var capacityWarnings: [DispatchCapacityWarning] = []
var proposeTarget: String?
var rejectRoute: DispatchOrderDetailRoute?
var opsReasonInput = ""
var proposeDate = Date()
var detailRoute: DispatchOrderDetailRoute?
var handoffEvents: [WarehousePulseEvent] = []
var handoffLoading = true
var handoffError: String?
⋮----
var showCancelSupplyRequestAlert: Bool {
⋮----
var showReleaseDispatchLockAlert: Bool {
⋮----
var proposeRoute: DispatchOrderDetailRoute? {
⋮----
static let dispatchIsoFractionalFormatter: ISO8601DateFormatter = {
⋮----
static func dispatchIsoDeliveryDate(from date: Date) -> String {
⋮----
var capacitySuggestedUnselect: [String] {
⋮----
var capacityExceededMessage: String {
⋮----
var cancellableSupplyStates: Set<String> {
⋮----
var hasActiveManualDispatchLock: Bool {
⋮----
var realtimeBanner: (systemImage: String, title: String, tint: Color)? {
⋮----
init() {}
⋮----
func connectRealtime(onReconnect: (@Sendable () -> Void)? = nil) {
⋮----
func disconnectRealtime() {
⋮----
func load(silent: Bool = false) {
⋮----
func driverPickerLabel(_ driver: AvailableDriver) -> String {
⋮----
func driverSubtitle(_ driver: AvailableDriver, unavailableDetail: Bool) -> String {
⋮----
func dispatchLockSubtitle(_ lock: WarehouseDispatchLock) -> String {
⋮----
func dispatchLockScope(_ lock: WarehouseDispatchLock) -> String {
⋮----
func createSupplyRequest(form: SupplyRequestFormData) async {
⋮----
func cancelSupplyRequest(_ request: WarehouseSupplyRequest) async {
⋮----
func runManualDispatch(forceCapacity: Bool) async {
⋮----
func runSmartDispatch(forceCapacity: Bool = false, acceptPartial: Bool = false) async {
⋮----
func acquireDispatchLock() async {
⋮----
func releaseDispatchLock(_ lock: WarehouseDispatchLock) async {
⋮----
func reloadDispatchPreview() async {
⋮----
func reloadSupplyRequests() async {
⋮----
func reloadDispatchLocks() async {
⋮----
func reloadFleetVehicles() async {
⋮----
func updateFleetVehicle(_ vehicle: Vehicle, isActive: Bool, reason: String? = nil, note: String? = nil) async {
⋮----
func describe(_ error: Error, fallback: String = "Request failed") -> String {
⋮----
func submitDispatchPropose(orderId: String) async {
⋮----
func submitDispatchReject(orderId: String) async {
⋮----
func findRescueDriver() async {
⋮----
func confirmRescue(action: String) async {
⋮----
func resetRescue() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/DriversViewModel.swift
```swift
final class DriversViewModel {
⋮----
var drivers: [Driver] = []
var vehicles: [Vehicle] = []
var loading = true
var error: String?
var showCreate = false
var createdPin: String?
var showPinAlert = false
var updatingDriverId: String?
⋮----
var draftName = ""
var draftPhone = ""
var submitting = false
var createError: String?
⋮----
var canCreate: Bool {
⋮----
private let fetchDrivers: FetchDrivers
private let fetchVehicles: FetchVehicles
private let assignDriverFn: AssignDriver
private let createDriverFn: CreateDriver
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func assign(driverId: String, vehicleId: String?) async {
⋮----
func prepareCreateSheet() {
⋮----
func createDriver() async -> Bool {
⋮----
func acknowledgePin() {
⋮----
func handleReconnectForAssignment() {
⋮----
func handleReconnectDuringCreate() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ExceptionsViewModel.swift
```swift
final class ExceptionsViewModel {
⋮----
var rows: [WarehouseOpsException] = []
var loading = true
var error: String?
⋮----
private let fetchExceptions: FetchExceptions
⋮----
init(fetchExceptions: FetchExceptions? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/FleetLiveMapViewModel.swift
```swift
final class FleetLiveMapViewModel {
⋮----
var routes: [WarehouseFleetLiveRoute] = []
var loading = true
var error: String?
var cameraPosition: MapCameraPosition = .region(
⋮----
var animator = FleetDriverMarkerAnimator()
var hasFittedInitialCamera = false
⋮----
private let fetchLiveMap: FetchLiveMap
private let warehouseIdProvider: WarehouseIdProvider
private let pollIntervalNanoseconds: UInt64
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func startPolling() async {
⋮----
func fitCamera() {
⋮----
func routeColor(_ index: Int) -> Color {
⋮----
func routeColor(for route: WarehouseFleetLiveRoute) -> Color {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/InventoryViewModel.swift
```swift
final class InventoryViewModel {
⋮----
var items: [InventoryItem] = []
var loading = true
var error: String?
var lowOnly = false
var adjustItem: InventoryItem?
var policySavingId: String?
⋮----
let policies = ["INHERIT", "REJECT", "ACCEPT_BACKORDER"]
⋮----
private let fetchInventory: FetchInventory
private let patchPolicy: PatchPolicy
private let adjustStock: AdjustStock
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func updatePolicy(item: InventoryItem, policy: String) async {
⋮----
func adjustInventory(productId: String, quantity: Int) async throws {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/LaborCapacityViewModel.swift
```swift
final class LaborCapacityViewModel {
var date: String
var zones: [LaborZoneCapacity] = []
var loading = true
var error: String?
var driverId = ""
var score: LaborDriverScore?
var availHours = "8"
var availStatus = "AVAILABLE"
var zoneH3 = ""
var saving = false
var statusMessage: String?
⋮----
private static let utcDateFormatter: DateFormatter = {
⋮----
static func todayUTC() -> String {
⋮----
init() {
⋮----
func loadZones() async {
⋮----
func loadScore() async {
⋮----
func saveAvailability() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/LocationSettingsViewModel.swift
```swift
final class LocationSettingsViewModel {
⋮----
var warehouseName = ""
var location = AddressLocationValue()
var loading = true
var saving = false
var error: String?
var saveMessage: String?
var packCountry = ""
⋮----
private let fetchLocation: FetchLocation
private let resolveLocation: ResolveLocation
private let patchLocation: PatchLocation
⋮----
init(
⋮----
func load() async {
⋮----
func save() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/LocationSetupViewModel.swift
```swift
final class LocationSetupViewModel {
var warehouseName = ""
var location = AddressLocationValue()
var loading = false
var submitting = false
var error: String?
var packCountry = ""
⋮----
private let fetchLocationHandler: @Sendable () async throws -> WarehouseLocationResponse
private let resolveLocationHandler: @MainActor (AddressLocationValue) async -> AddressLocationValue?
private let patchLocationHandler: @Sendable (String, String?, Double, Double) async throws -> WarehouseLocationResponse
private let refreshTokenHandler: @Sendable (String) async throws -> AuthResponse
private let setupHandler: @Sendable (String, String, String?, Double, Double) async throws -> WarehouseSetupResponse
⋮----
init(
⋮----
func loadExistingIfNeeded(hasAssignedWarehouse: Bool) async {
⋮----
func submit(tokenStore: TokenStore) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/LoginViewModel.swift
```swift
final class LoginViewModel {
var phone = ""
var pin = ""
var otpCode = ""
var useOtp = true
var otpSent = false
var loading = false
var error: String?
⋮----
private let configureAuthHandler: @MainActor () -> Void
private let resetFlowHandler: @MainActor () -> Void
private let sendOtpHandler: @MainActor (String) async throws -> Void
private let verifySmsHandler: @MainActor (String) async throws -> String
private let loginIdTokenHandler: @Sendable (String) async throws -> AuthResponse
private let loginPinHandler: @Sendable (String, String) async throws -> AuthResponse
private let uploadPushTokenHandler: @MainActor () async -> Void
⋮----
init(
⋮----
var isSubmitDisabled: Bool {
⋮----
func configure() {
⋮----
func toggleAuthMode() {
⋮----
func clampOtpCode() {
⋮----
func clampPin() {
⋮----
func sendOtp() async {
⋮----
func verifyOtp(tokenStore: TokenStore) async {
⋮----
func loginWithPin(tokenStore: TokenStore) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ManifestsViewModel.swift
```swift
final class ManifestsViewModel {
⋮----
var manifests: [Manifest] = []
var loading = true
var error: String?
⋮----
private let fetchManifests: FetchManifests
⋮----
init(fetchManifests: FetchManifests? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/MoreHubFeedsViewModel.swift
```swift
final class WarehouseJSONFeedViewModel {
⋮----
var bodyText = ""
var errorText: String?
var loading = true
⋮----
private let fetchJSON: FetchJSON
⋮----
init(fetchJSON: FetchJSON? = nil) {
⋮----
func load(path: String, query: [String: String] = [:]) async {
⋮----
final class WarehouseScoredExceptionsViewModel {
⋮----
var rows: [ScoredException] = []
var errorText: String?
var loading = true
⋮----
private let fetchScoredExceptions: FetchScoredExceptions
⋮----
init(fetchScoredExceptions: FetchScoredExceptions? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/NotificationInboxViewModel.swift
```swift
struct WarehouseNotificationsPage: Decodable {
let notifications: [WarehouseNotificationItem]
let unreadCount: Int
let hasMore: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(notifications: [WarehouseNotificationItem], unreadCount: Int, hasMore: Bool = false) {
⋮----
struct WarehouseNotificationItem: Decodable, Identifiable {
let id: String
let type: String
let title: String
let body: String
let payload: String
let channel: String
let readAt: String?
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
struct MarkNotificationsReadRequest: Encodable {
let markAll: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class NotificationInboxViewModel {
⋮----
var items: [WarehouseNotificationItem] = []
var unreadCount = 0
var loading = true
var error: String?
⋮----
private let fetchPage: FetchPage
private let markAllReadFn: MarkAllRead
⋮----
init(
⋮----
func load() async {
⋮----
func markAllRead() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/OperationsViewModel.swift
```swift
final class OperationsViewModel {
let broadcastRoles = ["DRIVER", "RETAILER", "ALL"]
⋮----
var loading = true
var error: String?
var statusMessage: String?
var templates: [BroadcastTemplate] = []
⋮----
var title = ""
var bodyText = ""
var broadcastRole = "DRIVER"
var templateDate = ""
var customReason = ""
var saveAsTemplate = false
var broadcasting = false
var savingTemplate = false
⋮----
var productId = ""
var retailerId = ""
var proposedPrice = ""
var preview: RetailerOverridePreview?
var previewLoading = false
private var previewTask: Task<Void, Never>?
⋮----
init() {}
⋮----
func loadTemplates(silent: Bool = false) async {
⋮----
func applyTemplate(_ template: BroadcastTemplate) {
⋮----
func sendBroadcast() async {
⋮----
func deleteTemplate(_ template: BroadcastTemplate) async {
⋮----
func schedulePreview() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/OpsSettingsViewModel.swift
```swift
final class OpsSettingsViewModel {
⋮----
var policy = "REJECT"
var showStockCounts = false
var preorderMinLeadDays = "3"
var preorderMaxLeadDays = "90"
var orderLineMin = ""
var orderLineMax = ""
var clearOrderLineMin = true
var clearOrderLineMax = true
var expressEnabled = false
var expressStockFloor = "0"
var feeBaseMinor = "0"
var feeCurrency = ""
var feeTiers: [FeeTierDraft] = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
var clearFeeRules = true
var scheduleJSON = "{\n  \"is_24h\": true\n}"
var enforceOrderAcceptance = false
var scheduleIs24h = true
var scheduleTimezone = "UTC"
var weekdayOpen = "09:00"
var weekdayClose = "17:00"
var loading = true
var saving = false
var error: String?
var scheduleError: String?
var saveMessage: String?
⋮----
private let fetchOpsSettings: FetchOpsSettings
private let fetchPaymentConfig: FetchPaymentConfig
private let patchOpsSettings: PatchOpsSettings
⋮----
init(
⋮----
func applyScheduleFields(from schedule: [String: AnyCodable]) {
⋮----
func buildScheduleForSave() -> [String: Any]? {
⋮----
func load() async {
⋮----
func save() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/OrderDetailViewModel.swift
```swift
enum OrderMutationAction: String, Identifiable {
⋮----
var id: String { rawValue }
⋮----
final class OrderDetailViewModel {
⋮----
private(set) var orderId: String
var order: Order?
var loading = true
var error: String?
var mutating = false
var pendingAction: OrderMutationAction?
var reasonInput = ""
var proposeDate = Date()
var showProposeSheet = false
var statusMessage: String?
var recommendations: [TruckRecommendation] = []
var showReassignSheet = false
var selectedDriverId: String?
var selectedManifestId: String?
var isPartialReassign = false
⋮----
private let fetchOrder: FetchOrder
private let proposeDeliveryOp: ProposeDelivery
private let fetchReceiptMeta: FetchReceiptMeta
private let issuePaymentBypassOp: IssuePaymentBypass
private let rejectOrderOp: RejectOrder
private let overflowOrderOp: OverflowOrder
private let recommendReassignOp: RecommendReassign
private let reassignOrderOp: ReassignOrder
⋮----
init(
⋮----
var trimmedReason: String {
⋮----
func updateOrderIdIfNeeded(_ newOrderId: String) {
⋮----
func selectRecommendation(_ rec: TruckRecommendation) {
⋮----
func load() async {
⋮----
func submitPropose() async {
⋮----
func resolveReceiptURL() async -> URL? {
⋮----
func runMutation(_ action: OrderMutationAction) async {
⋮----
func loadRecommendations() async {
⋮----
func submitReassign() async {
⋮----
func alertTitle(for action: OrderMutationAction) -> String {
⋮----
func reasonPrompt(for action: OrderMutationAction) -> String {
⋮----
func confirmLabel(for action: OrderMutationAction) -> String {
⋮----
static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
static func isoDeliveryDate(from date: Date) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/OrdersViewModel.swift
```swift
struct ProposeSheetOrder: Identifiable, Equatable, Sendable {
let id: String
⋮----
final class OrdersViewModel {
⋮----
let states = ["ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED"]
⋮----
var hubTab: OrdersHubTab = .active
var orders: [Order] = []
var preorders: [WarehousePreorderRow] = []
var loading = true
var error: String?
var selectedState = "ALL"
var proposeTarget: String?
var proposeDate = Date()
var rejectTarget: String?
var reasonInput = ""
var statusMessage: String?
⋮----
private let fetchOrders: FetchOrders
private let fetchPreorders: FetchPreorders
private let proposeOrderDeliveryOp: ProposeOrderDelivery
private let proposePreorderDeliveryOp: ProposePreorderDelivery
private let rejectOrderOp: RejectOrder
private let rejectPreorderOp: RejectPreorder
⋮----
init(
⋮----
var filterStates: [String] {
⋮----
var trimmedReason: String {
⋮----
var proposeSheetOrder: ProposeSheetOrder? {
⋮----
var showRejectAlert: Bool {
⋮----
func applyInitialState(_ state: String?) {
⋮----
func beginPropose(orderId: String) {
⋮----
func beginReject(orderId: String) {
⋮----
func cancelPropose() {
⋮----
func cancelReject() {
⋮----
func load() async {
⋮----
func submitPropose(orderId: String) async {
⋮----
func submitReject() async {
⋮----
func autoClearStatusMessage() async {
⋮----
static let tashkentIsoFractionalFormatter: ISO8601DateFormatter = {
⋮----
static func isoDeliveryDate(from date: Date) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/PaymentConfigViewModel.swift
```swift
final class PaymentConfigViewModel {
⋮----
var listings: [PSPListing] = []
var currency = ""
var loading = true
var error: String?
⋮----
private let fetchPaymentConfig: FetchPaymentConfig
⋮----
init(fetchPaymentConfig: FetchPaymentConfig? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/PreordersViewModel.swift
```swift
enum PreorderSheet: Identifiable {
⋮----
var id: String {
⋮----
final class PreordersViewModel {
⋮----
var rows: [WarehousePreorderRow] = []
var loading = true
var acting = false
var activeSheet: PreorderSheet?
var proposeDate = Date()
var reasonInput = ""
var statusMessage: String?
⋮----
private let fetchPreorders: FetchPreorders
private let proposeDelivery: ProposeDelivery
private let rejectPreorder: RejectPreorder
⋮----
init(
⋮----
var trimmedReason: String {
⋮----
var hasBlockingError: Bool {
⋮----
func beginPropose(_ row: WarehousePreorderRow) {
⋮----
func beginReject(_ row: WarehousePreorderRow) {
⋮----
func load() async {
⋮----
func submitPropose(_ row: WarehousePreorderRow) async {
⋮----
func submitReject(_ row: WarehousePreorderRow) async {
⋮----
static let tashkentDayFormatter: DateFormatter = {
⋮----
static let tashkentIsoFormatter: ISO8601DateFormatter = {
⋮----
static func initialProposeDate(for row: WarehousePreorderRow) -> Date {
⋮----
static func isoDeliveryDate(from date: Date) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ProductsViewModel.swift
```swift
final class ProductsViewModel {
⋮----
var products: [Product] = []
var loading = true
var error: String?
⋮----
private let fetchProducts: FetchProducts
⋮----
init(fetchProducts: FetchProducts? = nil) {
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ReplenishmentViewModel.swift
```swift
final class ReplenishmentViewModel {
⋮----
var insights: [ReplenishmentInsight] = []
var loading = true
var error: String?
var actingId: String?
var statusMessage: String?
⋮----
private let fetchInsights: FetchInsights
private let performInsightAction: PerformInsightAction
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func runAction(insightId: String, action: String) async {
⋮----
func handleReconnect() {
⋮----
func autoClearStatusMessage() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/RescuesViewModel.swift
```swift
final class RescuesViewModel {
⋮----
var brokenDrivers: [AvailableDriver] = []
var selectedBroken: AvailableDriver?
var rescueOptions: [RescueOption] = []
var loading = true
var previewLoading = false
var proposeLoading = false
var error: String?
var statusMessage: String?
⋮----
private let fetchDispatchPreview: FetchDispatchPreview
private let previewRescueFn: PreviewRescue
private let proposeRescueFn: ProposeRescue
⋮----
init(
⋮----
func loadDrivers() async {
⋮----
func findRescue(_ driver: AvailableDriver) async {
⋮----
func propose(_ option: RescueOption) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ReturnPolicySettingsViewModel.swift
```swift
final class ReturnPolicySettingsViewModel {
⋮----
var loading = true
var saving = false
var error: String?
var saved = false
var reverseSla = "24"
var canOverride = false
var retailerWindow = ""
var supplierId = ""
⋮----
private let warehouseIdProvider: WarehouseIdProvider
private let fetchPolicy: FetchPolicy
private let savePolicy: SavePolicy
⋮----
init(
⋮----
func sanitizeReverseSla(_ newValue: String) {
⋮----
func sanitizeRetailerWindow(_ newValue: String) {
⋮----
func load() async {
⋮----
func save() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/ReturnsViewModel.swift
```swift
enum ReturnsTab: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
final class ReturnsViewModel {
⋮----
var tab: ReturnsTab = .queue
var returns: [InboundReturnRow] = []
var history: [InboundReturnRow] = []
var reverseTasks: [ReverseLogisticsTask] = []
var loading = true
var error: String?
var barcode = ""
var sessionId: String?
var scanning = false
var selected: Set<String> = []
var statusMessage: String?
var scannerEnabled = true
var receivingTaskId: String?
⋮----
private let warehouseIdProvider: WarehouseIdProvider
private let fetchInboundReturns: FetchInboundReturns
private let fetchReturnsHistory: FetchReturnsHistory
private let fetchReverseLogistics: FetchReverseLogistics
private let createInboundSessionOp: CreateInboundSession
private let scanInboundBarcodeOp: ScanInboundBarcode
private let confirmInboundReturnsOp: ConfirmInboundReturns
private let receiveReverseLogisticsOp: ReceiveReverseLogistics
⋮----
init(
⋮----
var trimmedBarcode: String {
⋮----
func load() async {
⋮----
private func ensureSession() async throws -> String {
⋮----
func handleScannedCode(_ code: String) async {
⋮----
func handleScan() async {
⋮----
func confirm(disposition: String) async {
⋮----
func receive(_ task: ReverseLogisticsTask) async {
⋮----
static func parseReceivedQty(_ raw: String?) -> [String: Int] {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/RootViewModel.swift
```swift
struct WarehouseClientPolicyResponse: Decodable, Sendable {
let outdated: Bool
let forceUpdate: Bool
let updateDeferred: Bool?
let minimumVersion: String
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
final class RootViewModel {
var clientPolicyMessage: String?
var clientPolicyForce = false
var pendingManifest: AutoUpdater.Manifest?
⋮----
private let realtime = WarehouseRealtimeClient()
private let fetchPolicyHandler: @Sendable (String) async throws -> WarehouseClientPolicyResponse
private let evaluatePolicyHandler: @Sendable (WarehouseClientPolicyResponse) async -> AutoUpdater.UpdateState
⋮----
init(
⋮----
func loadClientPolicy(versionOverride: String? = nil) async {
⋮----
func dismissBanner() {
⋮----
func promptInstall() {
⋮----
func connectRealtime(realtimeHub: WarehouseRealtimeHub) {
⋮----
func disconnectRealtime() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/StaffViewModel.swift
```swift
final class StaffViewModel {
⋮----
var staff: [StaffMember] = []
var loading = true
var error: String?
var showCreate = false
var createdPin: String?
var showPinAlert = false
⋮----
var draftName = ""
var draftPhone = ""
var draftRole = "WAREHOUSE_ADMIN"
var submitting = false
var createError: String?
⋮----
var canCreate: Bool {
⋮----
private let fetchStaff: FetchStaff
private let createStaffFn: CreateStaff
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func prepareCreateSheet() {
⋮----
func createStaff() async -> Bool {
⋮----
func acknowledgePin() {
⋮----
func handleReconnectDuringCreate() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/StockCommitmentsViewModel.swift
```swift
final class StockCommitmentsViewModel {
⋮----
var commitments: [StockCommitmentRow] = []
var loading = true
var error: String?
⋮----
private let fetchCommitments: FetchCommitments
⋮----
init(fetchCommitments: FetchCommitments? = nil) {
⋮----
func load(silent: Bool = false) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/SupplyRequestsViewModels.swift
```swift
struct ManualSupplyLine: Identifiable, Sendable {
let id: UUID
var productId: String
var quantity: String
⋮----
init(id: UUID = UUID(), productId: String = "", quantity: String = "") {
⋮----
final class CreateSupplyRequestViewModel {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
var factoryId = ""
var factoryLocked = false
var priority = "NORMAL"
var notes = ""
var useForecast = true
var deliveryDate = Date()
var includeDeliveryDate = false
var forecast: [DemandForecastProduct] = []
var forecastLoading = false
var manualItems: [ManualSupplyLine] = [ManualSupplyLine()]
⋮----
var canSubmit: Bool {
⋮----
private let fetchFactory: FetchFactory
private let fetchForecast: FetchForecast
⋮----
init(
⋮----
func loadEngineFactory() async {
⋮----
func loadForecast() async {
⋮----
func addManualLine() {
⋮----
func buildFormData() -> SupplyRequestFormData {
⋮----
final class SupplyRequestDetailViewModel {
⋮----
var request: WarehouseSupplyRequest?
var qcResult = ""
var loading = true
var error: String?
var busy = false
var statusMessage: String?
⋮----
private let fetchRequest: FetchRequest
private let fetchQC: FetchQC
private let cancelRequestFn: CancelRequest
⋮----
init(
⋮----
func load(requestId: String, silent: Bool = false) async {
⋮----
func cancelRequest(requestId: String) async {
⋮----
func handleReconnect() {
⋮----
final class SupplyRequestsHubViewModel {
⋮----
let filters = ["ALL", "OPEN", "CANCELLED"]
⋮----
var requests: [WarehouseSupplyRequest] = []
var loading = true
var error: String?
var stateFilter = "ALL"
var showCreate = false
⋮----
var filtered: [WarehouseSupplyRequest] {
⋮----
private let fetchRequests: FetchRequests
private let createRequestFn: CreateRequest
⋮----
init(
⋮----
func load() async {
⋮----
func submitCreate(form: SupplyRequestFormData) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/TomorrowBoardViewModel.swift
```swift
final class TomorrowBoardViewModel {
⋮----
private static let utcDateFormatter: DateFormatter = {
⋮----
var date = Calendar.current.date(byAdding: .day, value: 1, to: Date()) ?? Date()
var preorders: [WarehouseOpsBoardOrder] = []
var deliverBefore: [WarehouseOpsBoardOrder] = []
var loading = true
var error: String?
⋮----
var dateString: String {
⋮----
private let fetchBoard: FetchBoard
⋮----
init(fetchBoard: FetchBoard? = nil) {
⋮----
func load(silent: Bool = false) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/TransferActionsViewModel.swift
```swift
final class TransferActionsViewModel {
⋮----
var volumeInput = "20"
var transferIdInput = ""
var notesInput = ""
var putawayProduct = ""
var putawayLocation = ""
var putawayQty = "1"
var putawayExpiry = ""
var pickManifestId = ""
var pickScan = ""
var pickScannedQty = 0
var activeWave: PickWave?
var busy = false
var statusMessage: String?
⋮----
private let emergencyTransferOp: EmergencyTransfer
private let forceReceiveOp: ForceReceive
private let receiveTransferOp: ReceiveTransfer
private let createBinOp: CreateBin
private let putawayLotOp: PutawayLot
private let createPickWaveOp: CreatePickWave
private let listPickWavesOp: ListPickWaves
private let getPickWaveOp: GetPickWave
private let confirmPickTaskOp: ConfirmPickTask
private let createCycleCountOp: CreateCycleCount
private let listCycleCountsOp: ListCycleCounts
private let submitCycleCountOp: SubmitCycleCount
⋮----
init(
⋮----
var nextPendingTask: PickTask? {
⋮----
func handleReconnect() {
⋮----
func createEmergencyTransfer() async {
⋮----
func forceReceivePayload() async {
⋮----
func receiveInboundTransfer() async {
⋮----
func ensureStageBin() async {
⋮----
func performPutawayLot() async {
⋮----
func createPickWave() async {
⋮----
func loadPickWaves() async {
⋮----
func confirmCurrentPick(wave: PickWave, task: PickTask) async {
⋮----
func createCycleCount() async {
⋮----
func submitFirstOpenExpected() async {
⋮----
func onPickBarcode(_ code: String) async {
⋮----
private func runMutation(
⋮----
private func runAction(
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/TreasuryViewModel.swift
```swift
final class TreasuryViewModel {
⋮----
var overview = TreasuryOverview.empty
var invoices: [Invoice] = []
var loading = true
var error: String?
var selectedSegment = 0
⋮----
private let fetchOverview: FetchOverview
private let fetchFallbackFinancials: FetchFallbackFinancials
private let fetchInvoices: FetchInvoices
⋮----
init(
⋮----
func load() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/VehicleDetailViewModel.swift
```swift
final class VehicleDetailViewModel {
⋮----
var vehicle: Vehicle?
var loading = true
var error: String?
var mutating = false
var selectedReason = "MANUAL_HOLD"
var customNote = ""
⋮----
private let fetchVehicle: FetchVehicle
private let updateAvailabilityFn: UpdateAvailability
⋮----
init(
⋮----
func load(vehicleId: String, silent: Bool = false) async {
⋮----
func updateAvailability(vehicleId: String, isActive: Bool, reason: String? = nil, note: String? = nil) async {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/VehiclesViewModel.swift
```swift
final class VehiclesViewModel {
⋮----
let vehicleClasses = [
⋮----
var vehicles: [Vehicle] = []
var loading = true
var error: String?
var showCreate = false
⋮----
var draftLabel = ""
var draftPlate = ""
var selectedClass = "CLASS_A"
var submitting = false
var createError: String?
⋮----
var canCreate: Bool {
⋮----
private let fetchVehicles: FetchVehicles
private let createVehicleFn: CreateVehicle
⋮----
init(
⋮----
func load(silent: Bool = false) async {
⋮----
func prepareCreateSheet() {
⋮----
func createVehicle() async -> Bool {
⋮----
func handleReconnectDuringCreate() {
```

## File: apps/warehouse-app-ios/WarehouseApp/ViewModels/YardGateViewModel.swift
```swift
struct YardVehiclePreset: Identifiable, Sendable, Equatable {
let id: String
let plate: String
let vehicleId: String
let label: String
⋮----
final class YardGateViewModel {
var depotId: String = "depot-tashkent-south"
var supplierId: String = "supplier-tashkent-1"
var vehicleId: String = "vh-isuzu-01"
var licensePlate: String = "01 777 AAA"
var direction: String = "GATE_IN"
var vehiclePurpose: String = "INBOUND_SUPPLIER_DROP"
var boltSealBarcode: String = "SEAL-UZ-11029"
var sealIntact: Bool = true
var assignedBaySlot: String = "BAY_3_INSPECTION"
⋮----
var isSubmitting: Bool = false
var recordedEvent: YardGateCheckEvent? = nil
var statusMessage: String? = nil
var errorMessage: String? = nil
⋮----
let vehiclePresets: [YardVehiclePreset] = [
⋮----
let sealPresets = ["SEAL-UZ-11029", "SEAL-UZ-88123", "SEAL-UZ-99412"]
⋮----
var isOutboundGateBlocked: Bool {
⋮----
private let recordGateEventHandler: @Sendable (YardGateCheckEvent) async throws -> YardGateCheckEvent
⋮----
init(
⋮----
func applyPreset(_ preset: YardVehiclePreset) {
⋮----
func updateSealIntegrity(_ intact: Bool) {
⋮----
func submitGateEvent() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Analytics/AnalyticsChartGrid.swift
```swift
struct AnalyticsChartGrid: View {
let daily: [DailyMetric]
⋮----
private var maxRevenue: Int {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Analytics/AnalyticsView.swift
```swift
struct AnalyticsView: View {
@State private var vm = AnalyticsViewModel()
⋮----
private let columns = [
⋮----
var body: some View {
⋮----
private struct AnalyticsKpiCard: View {
let title: String
let value: String
let icon: String
let index: Int
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Claims/ClaimsView.swift
```swift
struct ClaimsView: View {
@State private var vm = ClaimsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/KpiTile.swift
```swift
struct KpiTile: View {
let title: String
let value: String
let systemImage: String
let tint: Color
var chip: (text: String, tint: Color)? = nil
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/WarehouseFeedbackViews.swift
```swift
struct WarehouseErrorView: View {
let message: String
let retry: () -> Void
⋮----
var body: some View {
⋮----
struct WarehouseEmptyView: View {
let title: String
let message: String
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/WarehouseStateViews.swift
```swift
struct WarehouseLoadingView: View {
let title: String
var message: String = "Fetching the latest warehouse data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Coverage/CoverageView.swift
```swift
struct CoverageView: View {
@State private var vm = CoverageViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/CRM/CRMView.swift
```swift
struct CRMView: View {
@State private var vm = CRMViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/DemandForecast/DemandForecastView.swift
```swift
struct DemandForecastView: View {
@State private var vm = DemandForecastViewModel()
⋮----
var body: some View {
⋮----
private var forecastToolbar: some ToolbarContent {
⋮----
private var forecastBody: some View {
⋮----
private var productsSegment: some View {
⋮----
private var seriesSegment: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/DemandForecast/ForecastChartPanel.swift
```swift
struct ForecastChartPanel: View {
let criticalCount: Int
let urgentCount: Int
let normalCount: Int
⋮----
private let summaryColumns = [
⋮----
var body: some View {
⋮----
private struct ForecastSummaryCard: View {
let title: String
let count: Int
let subtitle: String
let tint: Color
let index: Int
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchOrderList.swift
```swift
struct DispatchOrderList: View {
let preview: DispatchPreview
let fleetVehicles: [Vehicle]
@Binding var vehicleReasons: [String: String]
@Binding var vehicleNotes: [String: String]
let mutatingFleetVehicleId: String?
@Binding var dispatchMode: DispatchAssignmentMode
@Binding var selectedDriverId: String
@Binding var selectedOrderIds: Set<String>
let executing: Bool
⋮----
var onManualDispatch: () -> Void
var onSmartDispatch: () -> Void
var onProposeDate: (_ orderId: String) -> Void
var onReject: (_ orderId: String) -> Void
var onOrderDoubleTap: (_ orderId: String) -> Void
var onMarkVehicleUnavailable: (_ vehicle: Vehicle) -> Void
var onRestoreVehicle: (_ vehicle: Vehicle) -> Void
⋮----
private let dispatchTetrisBuffer: Double = 0.95
⋮----
var body: some View {
⋮----
private func driverPickerLabel(_ driver: AvailableDriver) -> String {
⋮----
private struct DispatchFleetVehicleRow: View {
let vehicle: Vehicle
@Binding var vehicleReasons: [String: String]
@Binding var vehicleNotes: [String: String]
let mutating: Bool
let onMarkUnavailable: () -> Void
let onRestore: () -> Void
⋮----
@State private var selectedReason: String
@State private var customNote: String
⋮----
init(
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchSettingsView.swift
```swift
struct DispatchSettingsView: View {
@State private var vm = DispatchSettingsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchSupportViews.swift
```swift
struct DispatchStatusBanner: View {
let systemImage: String
let title: String
let tint: Color
⋮----
var body: some View {
⋮----
struct HandoffTimelineSection: View {
let events: [WarehousePulseEvent]
let loading: Bool
var error: String? = nil
⋮----
private var subtitle: String {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Drivers/DriversList.swift
```swift
struct DriversList: View {
let drivers: [Driver]
let vehicles: [Vehicle]
let updatingDriverId: String?
let onAssign: (String, String?) -> Void
⋮----
var body: some View {
⋮----
private func assignedVehicleLabel(for driver: Driver) -> String {
⋮----
private func assignedVehicleReason(for driver: Driver) -> String? {
⋮----
private func assignableVehicles(for driver: Driver) -> [Vehicle] {
⋮----
private func vehicleLabel(for vehicle: Vehicle) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Drivers/DriversView.swift
```swift
struct DriversView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = DriversViewModel()
⋮----
var body: some View {
⋮----
private struct CreateDriverSheet: View {
@Bindable var vm: DriversViewModel
@Environment(\.dismiss) private var dismiss
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Exceptions/ExceptionsView.swift
```swift
struct ExceptionsView: View {
@State private var vm = ExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Fleet/FleetLiveMapView.swift
```swift
struct FleetLiveMapView: View {
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Inventory/LocationSettingsView.swift
```swift
struct LocationSettingsView: View {
@State private var vm = LocationSettingsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Inventory/OpsSettingsForm.swift
```swift
struct FeeTierDraft: Identifiable, Equatable {
let id = UUID()
var maxKm: String
var feeMinor: String
⋮----
init(maxKm: String = "", feeMinor: String = "0") {
⋮----
struct OpsSettingsForm: View {
@Binding var preorderMinLeadDays: String
@Binding var preorderMaxLeadDays: String
@Binding var policy: String
@Binding var showStockCounts: Bool
@Binding var clearOrderLineMin: Bool
@Binding var orderLineMin: String
@Binding var clearOrderLineMax: Bool
@Binding var orderLineMax: String
@Binding var expressEnabled: Bool
@Binding var expressStockFloor: String
@Binding var clearFeeRules: Bool
@Binding var feeBaseMinor: String
@Binding var feeCurrency: String
@Binding var feeTiers: [FeeTierDraft]
@Binding var enforceOrderAcceptance: Bool
@Binding var scheduleIs24h: Bool
@Binding var scheduleTimezone: String
@Binding var weekdayOpen: String
@Binding var weekdayClose: String
@Binding var scheduleJSON: String
let scheduleError: String?
⋮----
@State private var acceptBackorder: Bool
⋮----
init(
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Inventory/OpsSettingsView.swift
```swift
struct OpsSettingsView: View {
@State private var vm = OpsSettingsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Inventory/ReturnPolicySettingsView.swift
```swift
struct ReturnPolicySettingsView: View {
@State private var vm = ReturnPolicySettingsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Manifests/ManifestsView.swift
```swift
struct ManifestsView: View {
@State private var vm = ManifestsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/More/WarehouseJSONFeedView.swift
```swift
struct WarehouseJSONFeedView: View {
let title: String
let path: String
var query: [String: String] = [:]
@State private var vm = WarehouseJSONFeedViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/More/WarehouseScoredExceptionsView.swift
```swift
struct WarehouseScoredExceptionsView: View {
@State private var vm = WarehouseScoredExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Notifications/NotificationInboxView.swift
```swift
struct NotificationInboxView: View {
@State private var vm = NotificationInboxViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Operations/OperationsBroadcastForm.swift
```swift
struct OperationsBroadcastForm: View {
let broadcastRoles: [String]
let templates: [BroadcastTemplate]
⋮----
@Binding var templateDate: String
@Binding var customReason: String
@Binding var title: String
@Binding var broadcastRole: String
@Binding var bodyText: String
@Binding var saveAsTemplate: Bool
⋮----
let broadcasting: Bool
let savingTemplate: Bool
⋮----
let onApplyTemplate: (BroadcastTemplate) -> Void
let onDeleteTemplate: (BroadcastTemplate) async -> Void
let onSendBroadcast: () async -> Void
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Operations/OperationsPricingPreview.swift
```swift
struct OperationsPricingPreview: View {
@Binding var productId: String
@Binding var retailerId: String
@Binding var proposedPrice: String
⋮----
let previewLoading: Bool
let preview: RetailerOverridePreview?
⋮----
let onSchedulePreview: () -> Void
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Operations/OperationsView.swift
```swift
struct OperationsView: View {
@State private var vm = OperationsViewModel()
⋮----
var body: some View {
⋮----
private var operationsForm: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Orders/OrdersList.swift
```swift
enum OrdersHubTab: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
struct OrdersList: View {
let hubTab: OrdersHubTab
let loading: Bool
let error: String?
let orders: [Order]
let preorders: [WarehousePreorderRow]
let onRetry: () -> Void
let onProposeActive: (String) -> Void
let onRejectActive: (String) -> Void
let onProposePreorder: (String) -> Void
let onRejectPreorder: (String) -> Void
⋮----
var body: some View {
⋮----
private struct OrderActionFlags {
let canDelay: Bool
let canReject: Bool
⋮----
private func orderActionFlags(_ state: String) -> OrderActionFlags {
⋮----
private struct OrderOpsCardView: View {
let title: String
let orderId: String
let state: String
let amountLabel: String
var badge: String?
var canDelay: Bool
var canReject: Bool
var delayLabel: String = "Propose date"
var onDelay: (() -> Void)?
var onReject: (() -> Void)?
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/PaymentConfig/PaymentConfigView.swift
```swift
struct PaymentConfigView: View {
@State private var vm = PaymentConfigViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Portal/PortalHandoffView.swift
```swift
struct PortalHandoffView: View {
let feature: WarehousePortalFeature
@Environment(\.openURL) private var openURL
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Preorders/PreordersList.swift
```swift
struct PreordersList: View {
let rows: [WarehousePreorderRow]
let acting: Bool
let onPropose: (WarehousePreorderRow) -> Void
let onReject: (WarehousePreorderRow) -> Void
⋮----
var body: some View {
⋮----
private func showsReviewBadge(_ row: WarehousePreorderRow) -> Bool {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Products/ProductsView.swift
```swift
struct ProductsView: View {
@State private var vm = ProductsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Replenishment/ReplenishmentList.swift
```swift
struct ReplenishmentList: View {
let insights: [ReplenishmentInsight]
let actingId: String?
let onApprove: (String) -> Void
let onDismiss: (String) -> Void
⋮----
var body: some View {
⋮----
private func urgencyTint(_ urgency: String) -> Color {
⋮----
private func demandWhyText(_ insight: ReplenishmentInsight) -> String? {
⋮----
private func number(from codable: AnyCodable?) -> Double? {
⋮----
private func string(from codable: AnyCodable?) -> String? {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Replenishment/ReplenishmentView.swift
```swift
struct ReplenishmentView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = ReplenishmentViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Rescues/RescuesView.swift
```swift
struct RescuesView: View {
@State private var vm = RescuesViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Setup/LocationSetupView.swift
```swift
struct LocationSetupView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var vm = LocationSetupViewModel()
⋮----
private var hasAssignedWarehouse: Bool { tokenStore.hasAssignedWarehouse }
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffView.swift
```swift
struct StaffView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = StaffViewModel()
⋮----
var body: some View {
⋮----
private struct CreateStaffSheet: View {
@Bindable var vm: StaffViewModel
@Environment(\.dismiss) private var dismiss
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/StockCommitments/StockCommitmentsView.swift
```swift
struct StockCommitmentsView: View {
@State private var vm = StockCommitmentsViewModel()
⋮----
var body: some View {
⋮----
private struct StockCommitmentCard: View {
let row: StockCommitmentRow
⋮----
var body: some View {
⋮----
private func metricRow(_ label: String, _ value: String, isHeader: Bool = false, valueColor: Color = .primary) -> some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/SupplyRequests/SupplyRequestDetailView.swift
```swift
struct SupplyRequestDetailView: View {
let requestId: String
⋮----
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = SupplyRequestDetailViewModel()
⋮----
var body: some View {
⋮----
private struct LabKeyValueRow: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/SupplyRequests/SupplyRequestsHubView.swift
```swift
struct SupplyRequestsHubView: View {
@State private var vm = SupplyRequestsHubViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Transfers/TransferActionsView.swift
```swift
struct TransferActionsView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = TransferActionsViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Treasury/TreasuryTransactionList.swift
```swift
struct TreasuryTransactionList: View {
let invoices: [Invoice]
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Treasury/TreasuryView.swift
```swift
struct TreasuryView: View {
@State private var vm = TreasuryViewModel()
⋮----
private var gridMin: CGFloat { 160 }
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Vehicles/VehicleDetailPanels.swift
```swift
struct VehicleAvailabilityPanel: View {
let vehicle: Vehicle
let mutating: Bool
@Binding var selectedReason: String
@Binding var customNote: String
let onMarkUnavailable: (String, String?) -> Void
let onRestore: () -> Void
⋮----
var body: some View {
⋮----
struct FleetTruckDispatchCard: View {
let vehicle: Vehicle
@Binding var selectedReason: String
@Binding var customNote: String
let mutating: Bool
let onMarkUnavailable: () -> Void
let onRestore: () -> Void
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Vehicles/VehiclesList.swift
```swift
struct VehiclesList: View {
let vehicles: [Vehicle]
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Vehicles/VehiclesView.swift
```swift
struct VehiclesView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = VehiclesViewModel()
⋮----
var body: some View {
⋮----
private struct CreateVehicleSheet: View {
@Bindable var vm: VehiclesViewModel
@Environment(\.dismiss) private var dismiss
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/warehouseapp/Services/AutoUpdater.swift
```swift
final class AutoUpdater {
static let shared = AutoUpdater()
⋮----
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
⋮----
private(set) var lastManifest: Manifest?
⋮----
private init() {}
⋮----
func evaluate(
⋮----
func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
⋮----
func triggerOTAInstall(manifestURL: URL) {
⋮----
func openStoreListing(force: Bool = false) {
⋮----
private func fetchManifest(from urlString: String) async -> Manifest? {
```

## File: apps/warehouse-app-ios/WarehouseApp/L10n.swift
```swift
enum L10n {
static func format(_ key: String, _ args: CVarArg...) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/PegasusWarehouseApp.swift
```swift
struct LabWarehouseApp: App {
@State private var tokenStore = TokenStore.shared
@State private var realtimeHub = WarehouseRealtimeHub()
@UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
⋮----
var body: some Scene {
```

## File: apps/warehouse-app-ios/WarehouseApp/RootView.swift
```swift
struct RootView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = RootViewModel()
⋮----
var body: some View {
```

## File: apps/control-tower-macos/Sources/Models/AnalyticsModels.swift
```swift
public struct RolePerformanceScorecard: Identifiable, Codable, Sendable {
public var id: String { role }
public let role: String
public let primaryMetricName: String
public let primaryMetricValue: String
public let primaryMetricSubtext: String
public let secondaryMetricName: String
public let secondaryMetricValue: String
public let complianceRatePercent: Double
public let activeAlertCount: Int
⋮----
public init(
```

## File: apps/control-tower-macos/Sources/Models/FinOpsModels.swift
```swift
public enum RolePerspective: String, CaseIterable, Identifiable, Codable, Sendable {
⋮----
public var id: String { rawValue }
⋮----
public var systemIcon: String {
⋮----
public struct TenderWaterfallSlice: Identifiable, Codable, Sendable {
public var id: String { label }
public let label: String
public let amountTiyin: Int64
public let percentage: Double
public let isDeduction: Bool
public let detail: String
⋮----
public var amountUzsFormatted: String {
⋮----
public init(label: String, amountTiyin: Int64, percentage: Double, isDeduction: Bool, detail: String) {
⋮----
public struct AttributedCostItem: Identifiable, Codable, Sendable {
public var id: String { "\(role)-\(entityId)-\(resourceName)" }
public let role: String
public let entityId: String
public let resourceName: String
public let monthlyCostCents: Int64
public let usageMetric: String
public let trendPercent: Double
⋮----
public var monthlyCostUsdFormatted: String {
⋮----
public init(
⋮----
public struct BillingInvoice: Identifiable, Codable, Sendable {
public var id: String { invoiceNumber }
public let invoiceNumber: String
public let entityId: String
public let role: String
public let period: String
public let totalAmountTiyin: Int64
public let status: InvoiceStatus
public let generatedAt: String
⋮----
public enum InvoiceStatus: String, Codable, Sendable {
⋮----
public var amountUzsFormatted: String {
⋮----
public init(
```

## File: apps/control-tower-macos/Sources/Models/InfraModels.swift
```swift
public struct CellHealth: Identifiable, Codable, Sendable {
public var id: String { cellId }
public let cellId: String
public let region: String
public let role: String
public let activeNodes: Int
public let qps: Int
public let cpuPercent: Double
public let memoryPercent: Double
public let status: HealthStatus
public let p99LatencyMs: Double
⋮----
public enum HealthStatus: String, Codable, Sendable {
⋮----
public init(
⋮----
public struct SpannerClusterMetrics: Codable, Sendable {
public let p99ReadMs: Double
public let p99WriteMs: Double
public let p99CommitMs: Double
public let activeSessions: Int
public let maxSessions: Int
public let staleReadOffloadPercent: Double
public let hourlyHistory: [LatencyDataPoint]
⋮----
public struct LatencyDataPoint: Identifiable, Codable, Sendable {
public var id: String { hour }
public let hour: String
public let readMs: Double
public let writeMs: Double
public let commitMs: Double
⋮----
public init(hour: String, readMs: Double, writeMs: Double, commitMs: Double) {
⋮----
public init(
⋮----
public struct KafkaPartitionLag: Identifiable, Codable, Sendable {
public var id: String { topic }
public let topic: String
public let partitions: Int
public let maxLagMs: Int
public let dlqCount: Int
public let consumerGroup: String
public let status: String
⋮----
public init(
⋮----
public struct RedisPermitPool: Identifiable, Codable, Sendable {
public var id: String { poolName }
public let poolName: String
public let activePermits: Int
public let maxPermits: Int
public let queueWaitMs: Double
⋮----
public var utilizationPercent: Double {
⋮----
public init(poolName: String, activePermits: Int, maxPermits: Int, queueWaitMs: Double) {
```

## File: apps/control-tower-macos/Sources/Models/TelemetryModels.swift
```swift
public enum LogSeverity: String, CaseIterable, Identifiable, Codable, Sendable {
⋮----
public var id: String { rawValue }
⋮----
public var systemColor: String {
⋮----
public struct TelemetryLogEvent: Identifiable, Codable, Sendable {
public var id: String { eventId }
public let eventId: String
public let traceId: String
public let timestampIso: String
public let severity: LogSeverity
public let role: String
public let entityId: String
public let service: String
public let message: String
public let metadata: [String: String]
⋮----
public init(
⋮----
public struct DiagnosticBundle: Codable, Sendable {
public let bundleId: String
public let generatedAt: String
public let appVersion: String
public let activeCell: String
public let logs: [TelemetryLogEvent]
public let spannerStatus: SpannerClusterMetrics
public let kafkaPartitions: [KafkaPartitionLag]
⋮----
public init(
```

## File: apps/control-tower-macos/Sources/Services/ExportService.swift
```swift
public enum ExportService {
public static func exportDiagnosticBundle(bundle: DiagnosticBundle) {
⋮----
public static func exportWaterfallCSV(slices: [TenderWaterfallSlice], role: RolePerspective) {
```

## File: apps/control-tower-macos/Sources/Services/WebSocketStreamClient.swift
```swift
public final class WebSocketStreamClient {
public static var defaultStreamURL: URL {
⋮----
public private(set) var isConnected: Bool = false
public private(set) var events: [TelemetryLogEvent] = []
public var isPaused: Bool = false
⋮----
public let url: URL
private var webSocketTask: URLSessionWebSocketTask?
private let maxBufferSize: Int = 50
private var retryAttempt: Int = 0
⋮----
public init(url: URL? = nil) {
⋮----
public func connect() {
⋮----
public func disconnect() {
⋮----
public func togglePause() {
⋮----
public func pushEvent(_ event: TelemetryLogEvent) {
⋮----
public func ingestRawPayload(_ data: Data) {
⋮----
private func receiveNextMessage() {
⋮----
private func handleIncomingString(_ text: String) {
⋮----
private func scheduleReconnect() {
⋮----
private func seedInitialEvents() {
```

## File: apps/control-tower-macos/Sources/Theme/ControlTowerTheme.swift
```swift
init(hex: UInt32, alpha: Double = 1.0) {
⋮----
init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum ControlTowerTheme {
⋮----
public static var canvas: Color {
⋮----
public static var surface: Color {
⋮----
public static var surfaceSubtle: Color {
⋮----
public static var surfaceElevated: Color {
⋮----
public static var inputWell: Color {
⋮----
public static var border: Color {
⋮----
public static var borderStrong: Color {
⋮----
public static var textPrimary: Color {
⋮----
public static var textSecondary: Color {
⋮----
public static var textTertiary: Color {
⋮----
public static var accentTeal: Color {
⋮----
public static var accentTealHover: Color {
⋮----
public static var accentTealSoft: Color {
⋮----
public static var slateTeal: Color {
⋮----
public static var warmCopper: Color {
⋮----
public static var warmCopperSoft: Color {
⋮----
public static var statusNominal: Color {
⋮----
public static var statusWarning: Color {
⋮----
public static var statusCritical: Color {
⋮----
public static var statusInfo: Color {
⋮----
public struct ControlTowerCardModifier: ViewModifier {
var borderColor: Color
var backgroundColor: Color
⋮----
public init(borderColor: Color = ControlTowerTheme.border, backgroundColor: Color = ControlTowerTheme.surface) {
⋮----
public func body(content: Content) -> some View {
⋮----
func controlTowerCard(borderColor: Color = ControlTowerTheme.border, backgroundColor: Color = ControlTowerTheme.surface) -> some View {
```

## File: apps/control-tower-macos/Sources/Views/DesktopKeyboardCheatsheetView.swift
```swift
public struct DesktopKeyboardCheatsheetView: View {
@Binding var isPresented: Bool
@State private var searchText: String = ""
⋮----
public init(isPresented: Binding<Bool>) {
⋮----
public struct ShortcutEntry: Identifiable {
public let id = UUID()
public let keyCombination: [String]
public let description: String
public let category: String
⋮----
private let shortcuts: [ShortcutEntry] = [
// Navigation
⋮----
private var filteredCategories: [String] {
⋮----
public var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/DriverEarnings.swift
```swift
struct DailyEarning: Codable, Hashable, Identifiable {
let date: String
let deliveryCount: Int64
let volume: Int64
⋮----
var id: String { date }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DriverEarningsResponse: Codable {
let totalDeliveries: Int64
let totalVolume: Int64
let totalRoutes: Int64
let last30Days: [DailyEarning]
let todayMinor: Int64
let weekMinor: Int64
let monthMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct DriverHistoryRow: Codable, Hashable, Identifiable {
let orderId: String
let status: String
let totalMinor: Int64
let currency: String
let completedAt: String
⋮----
var id: String { orderId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(orderId: String, status: String, totalMinor: Int64, currency: String, completedAt: String) {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DriverHistoryResponse: Codable {
let rows: [DriverHistoryRow]
⋮----
init(rows: [DriverHistoryRow] = []) {
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case rows }
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/Order.swift
```swift
enum OrderState: String, Codable, CaseIterable {
⋮----
var label: String {
⋮----
var customerStage: CustomerStage {
⋮----
var isActive: Bool {
⋮----
enum CustomerStage: String, Codable, CaseIterable {
⋮----
var displayName: String {
⋮----
var progressPercent: Int {
⋮----
struct OrderLineItem: Codable, Identifiable, Hashable {
let productId: String
let productName: String
let quantity: Int
let unitPrice: Int
⋮----
var id: String { productId }
var lineTotal: Int { quantity * unitPrice }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct Order: Codable, Identifiable, Hashable {
let id: String
let retailerId: String
let retailerName: String
let state: OrderState
let totalAmount: Int
let deliveryFeeMinor: Int?
let deliveryDistanceKm: Double?
let deliveryAddress: String
let latitude: Double
let longitude: Double
let qrToken: String
let paymentGateway: String?
let createdAt: String
let updatedAt: String
let items: [OrderLineItem]
let estimatedArrivalAt: String?
let etaDurationSec: Int?
let etaDistanceM: Int?
let routeId: String?
let sequenceIndex: Int?
let isPartial: Bool?
let splitGroupId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var displayTotal: String { totalAmount.formattedAmount }
⋮----
var deliveryFeeLabel: String? {
⋮----
struct DeliverySubmitRequest: Codable {
let orderId: String
let qrToken: String
let latitude: Double
let longitude: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReorderStopsRequest: Codable {
let routeId: String
let orderSequence: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CollectCashRequest: Codable {
let orderId: String
let latitude: Double
let longitude: Double
⋮----
let amountReceivedMinor: Int64?
let note: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(orderId: String, latitude: Double, longitude: Double, amountReceivedMinor: Int64? = nil, note: String? = nil) {
⋮----
struct DeliverySubmitResponse: Codable {
let success: Bool
let message: String
let newState: OrderState?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum RejectionReason: String, Codable, CaseIterable {
⋮----
struct AmendItemPayload: Codable {
let productId: String
let acceptedQty: Int
let rejectedQty: Int
let reason: String
let customReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AmendOrderRequest: Codable {
let orderId: String
let items: [AmendItemPayload]
let driverNotes: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AmendOrderResponse: Codable {
let success: Bool
let message: String
let adjustedTotal: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ValidateQRResponse: Codable {
let orderId: String
let retailerName: String
let totalAmount: Int
let state: String
let items: [OrderLineItem]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ConfirmOffloadResponse: Codable {
let orderId: String
let state: String
let paymentMethod: String
let amount: Int
let invoiceId: String?
let retailerId: String
let message: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DeliveryScanQRResponse: Codable {
let valid: Bool
let orderId: String
let state: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CollectCashResponse: Codable {
let orderId: String
let state: String
let amount: Int
let amountReceivedMinor: Int?
let shortfallMinor: Int?
let overageMinor: Int?
let distanceM: Double
let message: String
let attemptId: String?
let fiscalStatus: String?
let currency: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct VerifyHandshakeRequest: Codable {
let orderId: String
let token: String
let latitude: Double
let longitude: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct VerifyHandshakeResponse: Codable {
let success: Bool
let message: String
⋮----
struct UpdateOrderDuringDeliveryRequest: Codable {
let orderId: String
let latitude: Double
let longitude: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct UpdateOrderDuringDeliveryResponse: Codable {
let success: Bool
let message: String
⋮----
struct MissingItemRequest: Codable {
let skuId: String
let missingQty: Int
let reason: String?
let photoURL: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(skuId: String, missingQty: Int, reason: String? = nil, photoURL: String? = nil) {
⋮----
struct MissingItemsResponse: Codable {
let status: String
let orderId: String
let adjustedTotal: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SplitPaymentResponse: Codable {
let status: String
```

## File: apps/driver-app-ios/driverappios/driverappios/Offline/DriverOfflineQueue.swift
```swift
final class DriverOfflineQueue {
static let shared = DriverOfflineQueue()
⋮----
private var container: ModelContainer?
private var migrated = false
⋮----
private init() {}
⋮----
func attach(container: ModelContainer) {
⋮----
var context: ModelContext? { container?.mainContext }
⋮----
func enqueue(
⋮----
func enqueueJSONObject(
⋮----
func pending() -> [QueuedDriverAction] {
⋮----
func dead() -> [QueuedDriverAction] {
⋮----
func pendingCount() -> Int { pending().count }
⋮----
func delete(_ action: QueuedDriverAction) {
⋮----
func markDead(_ action: QueuedDriverAction, error: String) {
⋮----
func recordAttempt(_ action: QueuedDriverAction, error: String) {
⋮----
func clearDead() {
⋮----
func flush(api: APIClient) async -> (sent: Int, remaining: Int) {
⋮----
private func resolvePodLocalFiles(endpoint: String, bodyJSON: String, orderId: String) async throws -> String? {
⋮----
private func flushDeliverBatch(_ actions: [QueuedDriverAction], api: APIClient) async -> Int {
⋮----
private func migrateLegacyDeliveriesIfNeeded() {
⋮----
private func extractOrderId(_ json: String) -> String? {
⋮----
private static func coords(from json: String) -> (lat: Double?, lng: Double?) {
⋮----
private static func ensureCoords(in json: String, lat: Double?, lng: Double?) -> String {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private func isoToEpoch(_ iso: String) -> TimeInterval? {
⋮----
private func isoToEpochMs(_ iso: String) -> Double? {
```

## File: apps/driver-app-ios/driverappios/driverappios/Offline/OfflineHttpSemantics.swift
```swift
public enum OfflineHttpSemantics {
public static let statusPending = "PENDING"
public static let statusDead = "DEAD"
public static let maxAttemptsDefault = 8
⋮----
public static func normalizeEndpoint(_ endpoint: String) -> String {
⋮----
public static func isRetryableHTTP(_ code: Int) -> Bool {
⋮----
public static func isSuccessHTTP(_ code: Int) -> Bool {
⋮----
public static func isDeadHTTP(_ code: Int) -> Bool {
⋮----
public enum FlushOutcome: Sendable {
⋮----
public static func outcome(forHTTP code: Int) -> FlushOutcome {
```

## File: apps/driver-app-ios/driverappios/driverappios/Offline/QueuedDriverAction.swift
```swift
final class QueuedDriverAction {
@Attribute(.unique) var id: String
var endpoint: String
var method: String
var bodyJSON: String
var priority: Int
var clientTimestampIso: String
var attemptCount: Int
var lastError: String
var status: String
var orderId: String
var createdAt: Double
⋮----
var capturedLat: Double?
var capturedLng: Double?
var capturedAtMs: Double?
⋮----
init(
⋮----
func bodyJSONForFlush() -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Offline/QueuedMutationRecord.swift
```swift
public struct QueuedMutationRecord: Codable, Sendable, Equatable {
public var id: String
public var endpoint: String
public var method: String
public var payloadJSON: String
public var idempotencyKey: String
public var capturedLat: Double?
public var capturedLng: Double?
public var capturedAtMs: Double?
public var clientTimestampIso: String
public var attemptCount: Int
public var lastError: String
public var status: String
public var orderId: String
public var priority: Int
public var createdAtMs: Double
⋮----
public init(
⋮----
/// Merge capture-time coords into the JSON body for flush.
public func payloadJSONWithCapturedCoords() -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/DriverSessionReconcile.swift
```swift
enum DriverSessionReconcile {
private static let dateFormatter: DateFormatter = {
⋮----
static func run() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/FleetService.swift
```swift
protocol FleetServiceProtocol {
⋮----
func fetchActiveMissions(routeId: String) async throws -> [Mission]
⋮----
func deliverOrder(orderId: String, scannedToken: String) async throws
⋮----
func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse
⋮----
func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse
⋮----
func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse
⋮----
func completeOrder(orderId: String) async throws
⋮----
func collectCash(orderId: String, amountReceivedMinor: Int64?) async throws -> CollectCashResponse
⋮----
func retryFiscal(orderId: String) async throws -> CollectCashResponse
⋮----
func fetchOrderLineItems(orderId: String) async throws -> [LineItem]
⋮----
func amendOrder(orderId: String, driverId: String, items: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus, reason: String, customReason: String?)]) async throws
⋮----
func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse
⋮----
func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse
⋮----
func markCreditDelivery(orderId: String, photoProofUrl: String?, signatureUrl: String?) async throws -> [String: String]
⋮----
func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String?) async throws -> SplitPaymentResponse
⋮----
func requestRescue(reason: String, note: String) async throws
⋮----
func respondRescue(rescueId: String, brokenDriverId: String, accept: Bool) async throws
⋮----
func reassignHandshake(orderId: String) async throws
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/ManifestServiceLive.swift
```swift
final class ManifestServiceLive: ManifestServiceProtocol {
⋮----
static let shared = ManifestServiceLive()
private static let isoFormatter = ISO8601DateFormatter()
⋮----
func downloadManifest(bearerToken: String) async throws -> RouteManifest {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.driver.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/SyncServiceLive.swift
```swift
final class SyncServiceLive: SyncServiceProtocol {
⋮----
static let shared = SyncServiceLive()
⋮----
func uploadBatch(
⋮----
private struct BatchUploadRequest: Encodable {
let driverId: String
let deliveries: [BatchDelivery]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
private struct BatchDelivery: Encodable {
let orderId: String
let signature: String
let timestamp: Double
let status: String
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/driver-app-ios/driverappios/driverappios/Theme/PegasusTheme.swift
```swift
enum LabTheme {
⋮----
static let fg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 244/255.0, green: 244/255.0, blue: 245/255.0, alpha: 1.0) : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
⋮----
static let fgSecondary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) : UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) })
⋮----
static let fgTertiary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) : UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) })
⋮----
static let card = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 24/255.0, green: 24/255.0, blue: 27/255.0, alpha: 1.0) : UIColor.white })
⋮----
static let bg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 14/255.0, green: 14/255.0, blue: 16/255.0, alpha: 1.0) : UIColor(red: 237/255.0, green: 237/255.0, blue: 240/255.0, alpha: 1.0) })
⋮----
static let iconTile = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 226/255.0, green: 226/255.0, blue: 230/255.0, alpha: 1.0) })
⋮----
static let buttonFg = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) : UIColor.white })
⋮----
static let separator = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 228/255.0, green: 228/255.0, blue: 231/255.0, alpha: 1.0) })
⋮----
static let cardBorder = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) })
⋮----
static let destructive = Color(uiColor: .systemRed)
static let success     = Color(uiColor: .systemGreen)
static let warning     = Color(uiColor: .systemOrange)
static let live        = Color(uiColor: .systemGreen)
static let offline     = Color(uiColor: .systemRed)
static let transit     = Color(uiColor: .systemBlue)
⋮----
static func statusTint(for status: String) -> Color {
⋮----
static let readableMaxWidth: CGFloat = 560
⋮----
static let cardRadius: Double   = 24
static let buttonRadius: Double = 16
static let pillRadius: Double   = 100
static let modalRadius: Double  = 30
static let sheetRadius: Double  = 34
⋮----
static let s4: Double  = 4
static let s8: Double  = 8
static let s12: Double = 12
static let s16: Double = 16
static let s20: Double = 20
static let s24: Double = 24
static let s32: Double = 32
⋮----
static let mono: Font = .system(.caption, design: .monospaced, weight: .semibold)
⋮----
enum Anim {
static let snappy      = Animation.spring(response: 0.35, dampingFraction: 0.7)
static let bouncy      = Animation.spring(response: 0.5, dampingFraction: 0.72)
static let sheetReveal = Animation.spring(response: 0.55, dampingFraction: 0.78)
static let settle      = Animation.spring(response: 0.6, dampingFraction: 0.85)
static let micro       = Animation.spring(response: 0.25, dampingFraction: 0.8)
static let breathe     = Animation.easeInOut(duration: 1.6).repeatForever(autoreverses: true)
⋮----
static func stagger(_ index: Int, base: Animation = .spring(response: 0.4, dampingFraction: 0.75)) -> Animation {
⋮----
var formattedAmount: String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Utilities/DriverIdempotency.swift
```swift
enum DriverIdempotency {
private static func driverId() -> String {
⋮----
static func deliver(orderId: String) -> String {
⋮----
static func offload(orderId: String) -> String {
⋮----
static func complete(orderId: String) -> String {
⋮----
static func fiscalRetry(orderId: String) -> String {
⋮----
static func collectCash(orderId: String) -> String {
⋮----
static func confirmPaymentBypass(orderId: String) -> String {
⋮----
static func bypassOffload(orderId: String) -> String {
⋮----
static func reportShopClosed(orderId: String) -> String {
⋮----
static func proximityUnlock(orderId: String) -> String {
⋮----
static func partialOffload(orderId: String, fingerprint: String) -> String {
⋮----
static func depart(truckId: String) -> String {
⋮----
static func returnComplete(truckId: String) -> String {
⋮----
static func cashReconciliation(declaredMinor: Int64) -> String {
⋮----
static func syncBatch(orderSignatures: [String]) -> String {
⋮----
static func markArrived(orderId: String) -> String {
⋮----
static func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64) -> String {
⋮----
static func creditDelivery(orderId: String) -> String {
⋮----
static func missingItems(orderId: String) -> String {
⋮----
static func reportDamage(orderId: String) -> String {
⋮----
static func reportCondition(orderId: String, conditionType: String) -> String {
⋮----
static func requestEarlyComplete(reason: String) -> String {
⋮----
static func requestRescue(reason: String, note: String) -> String {
⋮----
static func respondRescue(rescueId: String, accept: Bool) -> String {
⋮----
static func routeReorder(routeId: String, orderSequence: [String]) -> String {
⋮----
static func supplyTransferArrive(transferId: String) -> String {
⋮----
static func amendOrder(orderId: String, items: [AmendItemPayload]) -> String {
⋮----
static func transitionState(orderId: String, newState: String) -> String {
⋮----
static func availability(onShift: Bool, reason: String = "", note: String? = nil) -> String {
⋮----
private static func stableHash(_ input: String) -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/QuickActionsSection.swift
```swift
struct QuickActionsSection: View {
@Binding var showRescueSheet: Bool
var onScanQR: (() -> Void)? = nil
var onOfflineVerify: (() -> Void)? = nil
⋮----
var body: some View {
⋮----
private func actionTileButton(icon: String, label: String, tint: Color = LabTheme.fg, action: @escaping () -> Void) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/RecentActivitySection.swift
```swift
struct RecentActivitySection: View {
var vm: FleetViewModel
var onSelectMission: ((Mission) -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Home/TodaySummaryCard.swift
```swift
struct TodaySummaryCard: View {
var vm: FleetViewModel
var onSelectPending: (() -> Void)? = nil
var onSelectDone: (() -> Void)? = nil
var onSelectRevenue: (() -> Void)? = nil
⋮----
var body: some View {
⋮----
private var totalRevenue: String {
⋮----
private var todayDate: String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/ActiveDeliverySheet.swift
```swift
struct ActiveDeliverySheet: View {
let mission: Mission
var vm: FleetViewModel
var bottomInset: Double
@Binding var navPath: NavigationPath
@Binding var showRescueSheet: Bool
⋮----
var body: some View {
⋮----
private static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
private static let isoStandardFormatter = ISO8601DateFormatter()
private static let etaTimeFormatter: DateFormatter = {
⋮----
private func formatETATime(_ iso: String) -> String {
⋮----
private func formatDuration(_ totalSec: Int) -> String {
⋮----
private func formatETADistance(_ meters: Int) -> String {
⋮----
private func openDestinationInMaps(lat: Double, lng: Double, name: String) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/MapTopOverlay.swift
```swift
struct MapTopOverlay: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Binding var phase: MapPhase
@Binding var selectedMission: Mission?
@Binding var isCameraLocked: Bool
@Binding var userPannedAt: Date?
@Binding var cameraPosition: MapCameraPosition
@Binding var zoomFocus: ZoomFocus
var currentTarget: Mission?
⋮----
var vm: FleetViewModel
var isLive: Bool
var goBack: () -> Void
var cycleZoom: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Map/OrderPreviewSheet.swift
```swift
struct OrderPreviewSheet: View {
let mission: Mission
var vm: FleetViewModel
var bottomInset: Double
@Binding var phase: MapPhase
@Binding var selectedMission: Mission?
@Binding var isCameraLocked: Bool
@Binding var userPannedAt: Date?
var startTelemetry: () -> Void
⋮----
var body: some View {
⋮----
private static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
private static let isoStandardFormatter = ISO8601DateFormatter()
private static let etaTimeFormatter: DateFormatter = {
⋮----
private func formatETATime(_ iso: String) -> String {
⋮----
private func formatDuration(_ totalSec: Int) -> String {
⋮----
private func formatETADistance(_ meters: Int) -> String {
⋮----
private func openDestinationInMaps(lat: Double, lng: Double, name: String) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/Profile/QuickActions.swift
```swift
struct QuickActions: View {
let onOfflineVerifier: () -> Void
let onSyncQueue: () -> Void
let onEndSession: () -> Void
var onFleetDVIR: () -> Void = {}
var onMidRouteRescue: () -> Void = {}
var onYardGate: () -> Void = {}
var pendingCount: Int = 0
⋮----
var body: some View {
⋮----
struct ActionRow: View {
let icon: String
let title: String
let subtitle: String
var destructive: Bool = false
let action: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/CollapsibleSidebar.swift
```swift
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
let title: String
@Binding var isExpanded: Bool
@Binding var selection: Item?
let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
@ViewBuilder let detail: () -> Detail
let footer: (() -> AnyView)?
⋮----
init(
⋮----
var body: some View {
⋮----
private func sidebar(effectiveExpanded: Bool, sidebarWidth: CGFloat, canToggle: Bool) -> some View {
⋮----
private func sidebarRow(_ item: CollapsibleSidebarItem<Item>, effectiveExpanded: Bool) -> some View {
⋮----
struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
let id: String
let tag: Item
let label: String
let icon: String
⋮----
init(id: String? = nil, tag: Item, label: String, icon: String) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/PegasusMonochromeTheme.swift
```swift
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum PegasusMonochromeTheme {
public static let readableMaxWidth: CGFloat = 960
⋮----
public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
⋮----
public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
⋮----
public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
⋮----
public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
⋮----
public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
⋮----
public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
⋮----
public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
⋮----
public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
⋮----
public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
⋮----
public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
⋮----
public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
⋮----
public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
⋮----
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)
⋮----
public static let spacingXS: CGFloat = 4
public static let spacingSM: CGFloat = 8
public static let spacingMD: CGFloat = 12
public static let spacingLG: CGFloat = 16
public static let spacingXL: CGFloat = 24
public static let spacingXXL: CGFloat = 32
⋮----
public static let radiusXS: CGFloat = 4
public static let radiusSM: CGFloat = 8
public static let radiusMD: CGFloat = 12
public static let radiusLG: CGFloat = 16
public static let radiusCard: CGFloat = 24
public static let radiusXL: CGFloat = 28
⋮----
public static func statusTint(for status: String) -> Color {
⋮----
public struct PegasusMonochromeCard: ViewModifier {
public init() {}
public func body(content: Content) -> some View {
⋮----
public struct MinimumHitTargetModifier: ViewModifier {
public var minSize: CGFloat
⋮----
public init(minSize: CGFloat = 44) {
⋮----
public func body(content: Content) -> some View {
⋮----
func pegasusMonochromeCard() -> some View {
⋮----
func pegasusReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
⋮----
public enum PegasusHaptics {
public static func light() {
⋮----
public static func medium() {
⋮----
public static func heavy() {
⋮----
public static func selection() {
⋮----
public static func success() {
⋮----
public static func warning() {
⋮----
public static func error() {
⋮----
public struct PegasusPressableButtonStyle: ButtonStyle {
public init() {}
public func makeBody(configuration: Configuration) -> some View {
⋮----
static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/PegasusMotion.swift
```swift
public enum PegasusAnim {
public static let snappy = Animation.snappy(duration: 0.3)
public static let smooth = Animation.smooth(duration: 0.35)
public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
public static let quick = Animation.easeOut(duration: 0.15)
⋮----
public static func staggerDelay(index: Int) -> Double {
⋮----
public static func resolved(_ animation: Animation = snappy, reduceMotion: Bool) -> Animation? {
⋮----
public enum PegasusMotionDuration {
public static let short4: Double = 0.2
public static let medium2: Double = 0.3
public static let medium4: Double = 0.4
⋮----
func pegasusNumericMetric<V: Equatable>(value: V) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/PegasusStateViews.swift
```swift
public struct PegasusLoadingView: View {
public let title: String
public var message: String = "Fetching the latest data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
public init(title: String, message: String = "Fetching the latest data.") {
⋮----
public var body: some View {
⋮----
public struct PegasusErrorView: View {
public let message: String
public let retry: () -> Void
⋮----
public init(message: String, retry: @escaping () -> Void) {
⋮----
public var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/PulseStrip.swift
```swift
enum PulseHonesty {
static let failed = "pulse_failed"
⋮----
struct Result<T> {
let events: [T]
let error: String?
⋮----
static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
⋮----
struct PulseStrip: View {
let events: [PulseEvent]
let loading: Bool
var error: String? = nil
⋮----
var body: some View {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private func formatTimestamp(_ iso: String) -> String {
⋮----
struct ExplainStatusBanner: View {
let explain: StatusExplain?
var fallbackTitle: String?
var fallbackDetail: String?
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/Components/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
⋮----
private struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
⋮----
private struct FlowLayout: Layout {
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/CashCollectionView.swift
```swift
struct CashCollectionView: View {
⋮----
let orderId: String
let amount: Int
let onCompleted: () -> Void
let onCancel: () -> Void
var onSplitPayment: ((String, Int) -> Void)?
⋮----
@State private var vm: CashCollectionViewModel
@State private var driverSocketState = DriverSocketState.shared
⋮----
init(
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/DriverNotificationInboxView.swift
```swift
struct DriverNotificationInboxView: View {
var onHandoffAction: ((String) -> Void)? = nil
@State private var vm = DriverNotificationInboxViewModel()
@Environment(\.dismiss) private var dismiss
⋮----
var body: some View {
⋮----
private struct DriverNotifRow: View {
let notification: DriverNotification
var onHandoffAction: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private var typeIcon: String {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private var timeAgo: String {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/FleetMapView.swift
```swift
enum MapPhase: Equatable {
⋮----
enum ZoomFocus: Int {
⋮----
var next: ZoomFocus { ZoomFocus(rawValue: (rawValue + 1) % 3) ?? .me }
⋮----
var icon: String {
⋮----
var label: String {
⋮----
struct FleetMapView: View {
@Environment(\.colorScheme) private var cs
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(\.modelContext) private var modelContext
@Bindable var vm: FleetViewModel
var goBack: () -> Void = {}
⋮----
@State private var mapVM = FleetMapViewModel()
@State private var telemetryVM = TelemetryViewModel()
@State private var driverSocketState = DriverSocketState.shared
⋮----
var body: some View {
⋮----
// MARK: - Map Body
⋮----
private var mapBody: some View {
⋮----
private var currentTarget: Mission? {
⋮----
private var bottomSheet: some View {
⋮----
private func cycleZoom() {
⋮----
private func zoomTo(_ focus: ZoomFocus, mission: Mission) {
⋮----
private var bottomInset: Double {
⋮----
private func missionPin(for mission: Mission) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/LoginView.swift
```swift
struct LoginView: View {
let onAuthenticated: () -> Void
⋮----
@State private var viewModel = LoginViewModel()
@FocusState private var focusedField: Field?
⋮----
private enum Field: Hashable { case phone, otp, pin }
⋮----
var body: some View {
⋮----
private func authButtonLabel(_ title: String) -> some View {
⋮----
private func completeAuth(_ action: () async -> Void) async {
⋮----
private struct PinDots: View {
let filled: Int
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/MainTabView.swift
```swift
enum AppTab: CaseIterable {
⋮----
var icon: String {
⋮----
var label: String {
⋮----
struct MainTabView: View {
@Environment(\.scenePhase) private var scenePhase
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var selectedTab: AppTab = .home
@State private var vm = FleetViewModel()
private var network = NetworkLifecycleEngine.shared
private var wasOffline: Bool { !network.isOnline }
⋮----
var body: some View {
⋮----
private var phoneLayout: some View {
⋮----
private var ipadCockpitLayout: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/MidRouteRescueView.swift
```swift
struct MidRouteRescueView: View {
⋮----
@State private var vm = MidRouteRescueViewModel()
let onBack: () -> Void
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/OfflineVerifierView.swift
```swift
struct OfflineVerifierView: View {
@State private var vm = OfflineVerifierViewModel()
let modelContext: ModelContext
⋮----
var body: some View {
⋮----
private var headerView: some View {
⋮----
private var statusBar: some View {
⋮----
private var stateContent: some View {
⋮----
private var idleCard: some View {
⋮----
private var syncingCard: some View {
⋮----
private func readyCard(_ manifest: RouteManifest) -> some View {
⋮----
private var scanningCard: some View {
⋮----
private func verifiedCard(_ orderId: String) -> some View {
⋮----
private func fraudCard(_ reason: String) -> some View {
⋮----
private func errorCard(_ reason: String) -> some View {
⋮----
private func infoRow(_ label: String, _ value: String) -> some View {
⋮----
private func monoButton(_ title: String, action: @escaping () -> Void) -> some View {
⋮----
private func destructiveButton(_ title: String, action: @escaping () -> Void) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/YardGateView.swift
```swift
struct YardGateView: View {
⋮----
@State private var vm = YardGateViewModel()
let onBack: () -> Void
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/factory-app-ios/FactoryApp/Design/CollapsibleSidebar.swift
```swift
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
let title: String
@Binding var isExpanded: Bool
@Binding var selection: Item?
let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
@ViewBuilder let detail: () -> Detail
let footer: (() -> AnyView)?
⋮----
init(
⋮----
var body: some View {
⋮----
private func sidebar(effectiveExpanded: Bool, sidebarWidth: CGFloat, canToggle: Bool) -> some View {
⋮----
private func sidebarRow(_ item: CollapsibleSidebarItem<Item>, effectiveExpanded: Bool) -> some View {
⋮----
struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
let id: String
let tag: Item
let label: String
let icon: String
⋮----
init(id: String? = nil, tag: Item, label: String, icon: String) {
```

## File: apps/factory-app-ios/FactoryApp/Design/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/factory-app-ios/FactoryApp/Design/DeliveryLiveActivityWidget.swift
```swift
public struct DeliveryLiveActivityWidget: Widget {
public init() {}
⋮----
public var body: some WidgetConfiguration {
⋮----
private func formatDistance(_ meters: Double) -> String {
⋮----
private struct LockScreenLiveActivityView: View {
let context: ActivityViewContext<DeliveryActivityAttributes>
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Empty pack prints the number only. Never invents UZS.
func formatPackMoney(_ minor: Int64, pack: MarketPack?, decimalPlaces: Int = 2) -> String {
⋮----
/// Stored/event currency, else session pack. Empty pack does not invent UZS.
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/PegasusMonochromeTheme.swift
```swift
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum PegasusMonochromeTheme {
public static let readableMaxWidth: CGFloat = 960
⋮----
public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
⋮----
public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
⋮----
public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
⋮----
public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
⋮----
public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
⋮----
public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
⋮----
public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
⋮----
public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
⋮----
public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
⋮----
public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
⋮----
public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
⋮----
public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
⋮----
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)
⋮----
public static let spacingXS: CGFloat = 4
public static let spacingSM: CGFloat = 8
public static let spacingMD: CGFloat = 12
public static let spacingLG: CGFloat = 16
public static let spacingXL: CGFloat = 24
public static let spacingXXL: CGFloat = 32
⋮----
public static let radiusXS: CGFloat = 4
public static let radiusSM: CGFloat = 8
public static let radiusMD: CGFloat = 12
public static let radiusLG: CGFloat = 16
public static let radiusCard: CGFloat = 24
public static let radiusXL: CGFloat = 28
⋮----
public static func statusTint(for status: String) -> Color {
⋮----
public struct PegasusMonochromeCard: ViewModifier {
public init() {}
public func body(content: Content) -> some View {
⋮----
public struct MinimumHitTargetModifier: ViewModifier {
public var minSize: CGFloat
⋮----
public init(minSize: CGFloat = 44) {
⋮----
public func body(content: Content) -> some View {
⋮----
func pegasusMonochromeCard() -> some View {
⋮----
func pegasusReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
⋮----
public enum PegasusHaptics {
public static func light() {
⋮----
public static func medium() {
⋮----
public static func heavy() {
⋮----
public static func selection() {
⋮----
public static func success() {
⋮----
public static func warning() {
⋮----
public static func error() {
⋮----
public struct PegasusPressableButtonStyle: ButtonStyle {
public init() {}
public func makeBody(configuration: Configuration) -> some View {
⋮----
static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
```

## File: apps/factory-app-ios/FactoryApp/Design/PegasusMotion.swift
```swift
public enum PegasusAnim {
public static let snappy = Animation.snappy(duration: 0.3)
public static let smooth = Animation.smooth(duration: 0.35)
public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
public static let quick = Animation.easeOut(duration: 0.15)
⋮----
public static func staggerDelay(index: Int) -> Double {
⋮----
public static func resolved(_ animation: Animation = snappy, reduceMotion: Bool) -> Animation? {
⋮----
public enum PegasusMotionDuration {
public static let short4: Double = 0.2
public static let medium2: Double = 0.3
public static let medium4: Double = 0.4
⋮----
func pegasusNumericMetric<V: Equatable>(value: V) -> some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/PegasusStateViews.swift
```swift
public struct PegasusLoadingView: View {
public let title: String
public var message: String = "Fetching the latest data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
public init(title: String, message: String = "Fetching the latest data.") {
⋮----
public var body: some View {
⋮----
public struct PegasusErrorView: View {
public let message: String
public let retry: () -> Void
⋮----
public init(message: String, retry: @escaping () -> Void) {
⋮----
public var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Design/RealtimeRefresh.swift
```swift
public struct SilentRefreshModifier: ViewModifier {
private let refreshEpoch: Int
private let reconnectEpoch: Int
private let reload: (_ silent: Bool) -> Void
⋮----
@State private var coalesceTask: Task<Void, Never>?
⋮----
public init(
⋮----
public func body(content: Content) -> some View {
⋮----
private func scheduleCoalescedReload() {
⋮----
func silentRealtimeRefresh(
⋮----
public enum RealtimeLoadState {
public static func showFullScreenLoading(loading: Bool, hasData: Bool) -> Bool {
⋮----
public enum PulseHonesty {
public static let failed = "pulse_failed"
public static let commandFailed = "control_tower_pulse_failed"
⋮----
public struct Result<T> {
public let events: [T]
public let error: String?
⋮----
public struct ObjectResult<T> {
public let value: T?
public let error: String?
⋮----
public static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
⋮----
public static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
```

## File: apps/factory-app-ios/FactoryApp/Design/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
⋮----
private struct FlexibleStatusChips: View {
let rows: [StatusStackRow]
var onSelect: ((String) -> Void)? = nil
⋮----
var body: some View {
⋮----
private func chip(_ row: StatusStackRow) -> some View {
⋮----
private struct FlowLayout: Layout {
func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
⋮----
func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
```

## File: apps/factory-app-ios/FactoryApp/Models/FactoryModels.swift
```swift
struct LoginRequest: Encodable {
var phone: String = ""
var password: String = ""
var idToken: String = ""
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct AuthResponse: Decodable {
let token: String
let refreshToken: String
let factoryId: String
let factoryName: String
let isConfigured: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct DeviceTokenRequest: Encodable, Equatable {
let token: String
let platform: String
⋮----
// MARK: - Dashboard
struct DashboardStats: Decodable {
let pendingTransfers: Int
let loadingTransfers: Int
let activeManifests: Int
let dispatchedToday: Int
let vehiclesTotal: Int
let vehiclesAvailable: Int
let staffOnShift: Int
let criticalInsights: Int
let source: String
let plane: String
let transfersByState: [String: Int]
let manifestsByState: [String: Int]
let vehiclesByState: [String: Int]
let driverDuty: [String: Int]
let slaByStatus: [String: Int]
let qcByResult: [String: Int]
let qcAvailable: Bool
let bayLoadingTransfers: Int
let bayLoadingManifests: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
static let empty = DashboardStats()
⋮----
struct Transfer: Decodable, Identifiable {
let id: String
let factoryId: String
let warehouseId: String
let warehouseName: String
let state: String
let priority: String
let totalItems: Int
let totalVolumeL: Double
let notes: String
let createdAt: String
let updatedAt: String
let items: [TransferItem]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct TransferItem: Decodable, Identifiable {
let id: String
let productId: String
let productName: String
let quantity: Int
let quantityAvailable: Int
let unitVolumeL: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct TransferListResponse: Decodable {
let transfers: [Transfer]
let total: Int
⋮----
struct TransitionRequest: Encodable {
let targetState: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyRequest: Decodable, Identifiable {
let id: String
let warehouseId: String
let factoryId: String
let supplierId: String
let state: String
let priority: String
let requestedDeliveryDate: String?
let totalVolumeVU: Double
let notes: String
let transferOrderId: String
let createdBy: String
let createdAt: String
let updatedAt: String?
let slaStatus: String?
let slaDueAt: String?
let slaHoursRemaining: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyRequestListResponse: Decodable {
let requests: [SupplyRequest]
⋮----
struct SupplyRequestQCResponse: Decodable {
let requestId: String
let result: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplyRequestQCRequest: Encodable {
let result: String
⋮----
func slaBadgeVisible(_ status: String?) -> Bool {
⋮----
func slaHoursLabel(_ hours: Double?) -> String? {
⋮----
func supplyDeliveryWithSLA(_ request: SupplyRequest) -> String {
⋮----
struct SupplyRequestTransitionRequest: Encodable {
let action: String
let transferOrderId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyRequestTransitionResponse: Decodable {
let requestId: String
let state: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyFulfillOptions: Decodable {
let transferMode: String
let warehouseId: String
let warehouseName: String
let coLocated: Bool
let outcomeInternal: String
let outcomeTruck: String
let linkedDriverETA: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct Manifest: Decodable, Identifiable {
let id: String
let factoryId: String
let driverId: String
let driverName: String
let vehicleId: String
let vehicleLabel: String
let truckId: String
let truckPlate: String
let state: String
let status: String
let totalVolumeVU: Double
let maxVolumeVU: Double
let maxCapacityVU: Double
let stopCount: Int
let regionCode: String
let createdAt: String
let transfers: [ManifestTransfer]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ManifestTransfer: Decodable, Identifiable {
let id: String
let productName: String
let quantity: Int
let volumeVU: Double
let state: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ManifestListResponse: Decodable {
let manifests: [Manifest]
let total: Int
⋮----
struct ManifestDetailCore: Decodable, Identifiable {
let manifestId: String
let state: String
let transferCount: Int
let totalVolumeVU: Int
let maxVolumeVU: Int
let driverId: String
let vehicleId: String
let updatedAt: String
⋮----
var id: String { manifestId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestTransitionRow: Decodable, Identifiable {
let action: String
let fromState: String
let toState: String
let at: String
let reason: String
⋮----
var id: String { "\(action)-\(at)" }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestDetailSnapshot: Decodable {
let manifest: ManifestDetailCore
let transfers: [ManifestTransfer]
let transitions: [ManifestTransitionRow]
let exceptions: [ManifestException]
let routeId: String
let stopCount: Int
let orderCount: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestRebalanceRequest: Encodable {
let sourceManifestId: String
let targetManifestId: String
let transferIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestRebalanceResponse: Decodable {
let sourceManifestId: String
let targetManifestId: String
let transfersMoved: Int
let volumeMovedVU: Double
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestCancelTransferRequest: Encodable {
let manifestId: String
let transferId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestCancelTransferResponse: Decodable {
let manifestId: String
let transferId: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestCancelRequest: Encodable {
let manifestId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestCancelResponse: Decodable {
let manifestId: String
let status: String
let transfersReleased: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct Vehicle: Decodable, Identifiable {
let id: String
let plateNumber: String
let driverName: String
let status: String
let capacityKg: Double
let capacityL: Double
let currentRoute: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct FactoryFleetDriverLocation: Decodable {
let driverId: String?
let lat: Double?
let lng: Double?
let latitude: Double?
let longitude: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetLiveRoute: Decodable, Identifiable {
var id: String { manifestId }
let manifestId: String
let driverId: String?
let driverName: String?
let manifestState: String?
let liveLocationAvailable: Bool?
let locationStale: Bool?
let driverLocation: FactoryFleetDriverLocation?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetLiveMapResponse: Decodable {
let routes: [FactoryFleetLiveRoute]
let factoryId: String?
let fetchedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct VehicleListResponse: Decodable {
let vehicles: [Vehicle]
⋮----
struct StaffMember: Decodable, Identifiable {
let id: String
let name: String
let phone: String
let role: String
let status: String
let joinedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct StaffListResponse: Decodable {
let staff: [StaffMember]
⋮----
struct CreateStaffRequest: Encodable {
let name: String
let role: String
⋮----
// MARK: - Insight
struct Insight: Decodable, Identifiable {
let id: String
let warehouseId: String
let warehouseName: String
let productId: String
let productName: String
let urgency: String
let currentStock: Int
let avgDailyVelocity: Double
let daysUntilStockout: Int
let reorderQuantity: Int
let status: String
let reasonCode: String?
let demandBreakdown: [String: AnyCodable]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct InsightListResponse: Decodable {
let insights: [Insight]
⋮----
// MARK: - Manifest exceptions
struct ManifestException: Decodable, Identifiable {
let exceptionId: String
let manifestId: String
let transferId: String
let reason: String
let metadata: String
let attemptCount: Int
let escalated: Bool
let createdAt: String
let correlationId: String
⋮----
var id: String { exceptionId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ManifestExceptionListResponse: Decodable {
let exceptions: [ManifestException]
⋮----
struct ResolveManifestExceptionRequest: Encodable {
let resolution: String
let note: String
⋮----
init(resolution: String = "RESOLVED", note: String = "") {
⋮----
struct ResolveManifestExceptionResponse: Decodable {
let exceptionId: String
let manifestId: String
let resolution: String
let note: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - Dispatch
struct DispatchRequest: Encodable {
var mode: String = "AUTO"
let transferIds: [String]
var forceCapacity: Bool = false
var acceptPartial: Bool = false
var reason: String = "factory-app-ios"
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchResponse: Decodable {
let status: String
let manifestId: String
let truckPlate: String
let stopCount: Int
let createdManifestCount: Int
let optimizerClass: String
let dispatchAlgo: String
let unassigned: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - Extended Factory Contracts
struct FactoryProfile: Decodable {
let factoryId: String
let supplierId: String
let name: String
let address: String
let lat: Double
let lng: Double
let h3Index: String
let regionCode: String
let leadTimeDays: Int
let productionCapacityVU: Double
let productTypes: [String]
let isActive: Bool
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryAnalyticsDayBucket: Decodable {
let date: String
let transfers: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryAnalyticsOverview: Decodable {
let dailyActivity: [FactoryAnalyticsDayBucket]
let transfersTotal: Int64
let manifestsActive: Int64
let exceptionQueue: Int64
let avgLeadTimeMins: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryCreateTransferRequest: Encodable {
let orderId: String?
let totalVu: Int
let driverId: String?
let vehicleId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryCreateTransferResponse: Decodable {
let transferId: String
let state: String
let totalVu: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetDriverRow: Decodable, Identifiable {
let driverId: String
let name: String
let onShift: Bool
⋮----
var id: String { driverId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetDriversEnvelope: Decodable {
let drivers: [FactoryFleetDriverRow]
⋮----
struct FactoryFleetVehicleRow: Decodable, Identifiable {
let vehicleId: String
let plateNo: String
let state: String
⋮----
var id: String { vehicleId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetVehiclesEnvelope: Decodable {
let vehicles: [FactoryFleetVehicleRow]
⋮----
struct FactoryFleetDriver: Decodable, Identifiable {
let id: String
let name: String
let phone: String
let driverType: String
let vehicleType: String
let licensePlate: String
let isActive: Bool
let truckStatus: String
let createdAt: String
let vehicleId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetDriverListResponse: Decodable {
let data: [FactoryFleetDriver]
⋮----
struct FactoryFleetVehicle: Decodable, Identifiable {
let id: String
let supplierId: String
let vehicleClass: String
let maxVolumeVU: Double
let licensePlate: String
let isActive: Bool
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryFleetVehicleListResponse: Decodable {
let data: [FactoryFleetVehicle]
⋮----
struct FactoryStaffDetail: Decodable {
let staffId: String
let factoryId: String
let name: String
let phone: String
let staffRole: String
let isActive: Bool
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryManifestTransitionResponse: Decodable {
let status: String
let manifestId: String?
let state: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum ManifestLifecycleAction: String {
⋮----
var label: String {
⋮----
static func next(for state: String) -> ManifestLifecycleAction? {
⋮----
struct FactoryLocationResponse: Decodable {
let factoryId: String
let name: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryLocationPatchRequest: Encodable {
let address: String
let placeId: String?
let lat: Double
let lng: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactorySetupRequest: Encodable {
let factoryName: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
let facilityType: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactorySetupResponse: Decodable {
let factoryId: String
let token: String?
let refreshToken: String?
let isConfigured: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryPulseEvent: Decodable, Identifiable {
let id: String
let kind: String
let title: String
let description: String?
let occurredAt: String
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FactoryPulseResponse: Decodable {
let events: [FactoryPulseEvent]
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum FactoryHandoffPulseSupport {
private static let handoffKinds: Set<String> = [
⋮----
static func filter(_ events: [FactoryPulseEvent]) -> [FactoryPulseEvent] {
⋮----
private static func isHandoff(_ event: FactoryPulseEvent) -> Bool {
⋮----
struct AnyCodable: Codable {
let value: Any
⋮----
init(_ value: Any) {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
```

## File: apps/factory-app-ios/FactoryApp/Services/APIClient.swift
```swift
private actor FactoryHTTPCoalescer {
private struct CacheEntry {
let data: Data
let response: URLResponse
let timestamp: Date
⋮----
private var inFlightGets: [String: Task<(Data, URLResponse), Error>] = [:]
private var microCache: [String: CacheEntry] = [:]
private let microCacheTTL: TimeInterval = 2.0
private var refreshTask: Task<Bool, Error>?
⋮----
func fetchGet(
⋮----
func invalidateCache() {
⋮----
func refreshSingleFlight(
⋮----
else if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
⋮----
func get<T: Decodable>(_ path: String, query: [String: String] = [:]) async throws -> T {
⋮----
func post<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
⋮----
func patch<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
⋮----
func postVoid<B: Encodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws {
⋮----
private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
⋮----
private func attachToken(_ request: inout URLRequest) async {
⋮----
private func attemptRefresh() async throws -> Bool {
⋮----
private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
⋮----
func factoryWebSocketURL(token: String, lastEventId: String? = nil) -> URL {
⋮----
enum APIError: LocalizedError {
⋮----
var errorDescription: String? {
```

## File: apps/factory-app-ios/FactoryApp/Services/FactoryRealtimeClient.swift
```swift
enum FactoryRealtimeStatus: Equatable {
⋮----
enum FactoryRealtimeEventType: String {
⋮----
struct FactoryLiveEvent: Decodable {
let type: String
let timestamp: String?
let eventId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
var eventType: FactoryRealtimeEventType? {
⋮----
final class FactoryRealtimeClient {
private let session: URLSession
private let decoder = JSONDecoder()
private let pathMonitor = NWPathMonitor()
private let workQueue = DispatchQueue(label: "factory.realtime.client")
⋮----
private var task: URLSessionWebSocketTask?
private var heartbeatTask: Task<Void, Never>?
private var reconnectWorkItem: DispatchWorkItem?
private var reconnectAttempt = 0
private var hasConnectedOnce = false
private var closed = true
private var networkAvailable = true
private var lastEventId: String?
private var stateHandler: (@MainActor (FactoryRealtimeStatus) -> Void)?
private var eventHandler: (@MainActor (FactoryLiveEvent) -> Void)?
private var reconnectHandler: (@MainActor () -> Void)?
⋮----
init(session: URLSession = .shared) {
⋮----
deinit {
⋮----
func connect(
⋮----
func disconnect() {
⋮----
private func openSocket(isReconnect: Bool) {
⋮----
private func startHeartbeat() {
⋮----
private func receiveLoop() {
⋮----
private func handleSocketDrop() {
⋮----
private func scheduleReconnect() {
⋮----
private func publish(_ status: FactoryRealtimeStatus) {
```

## File: apps/factory-app-ios/FactoryApp/Services/FactorySessionReconcile.swift
```swift
enum FactorySessionReconcile {
static func run() async {
```

## File: apps/factory-app-ios/FactoryApp/Services/GeocodeService.swift
```swift
struct GeocodePrediction: Decodable, Equatable {
let placeId: String
let description: String
⋮----
init(placeId: String, description: String) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ResolvedLocation: Decodable, Equatable {
let address: String
let lat: Double
let lng: Double
let placeId: String?
⋮----
init(address: String, lat: Double, lng: Double, placeId: String? = nil) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum GeocodeService {
static func autocomplete(_ input: String) async throws -> [GeocodePrediction] {
⋮----
static func resolvePlace(_ placeId: String) async throws -> ResolvedLocation {
⋮----
static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
⋮----
static func forward(address: String) async throws -> ResolvedLocation {
struct ForwardRequest: Encodable { let address: String }
⋮----
private struct AutocompleteResponse: Decodable {
let predictions: [GeocodePrediction]
```

## File: apps/factory-app-ios/FactoryApp/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.factory.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/factory-app-ios/FactoryApp/Theme/PegasusTheme.swift
```swift
enum Anim {
static let snappy = Animation.snappy(duration: 0.3)
static let smooth = Animation.smooth(duration: 0.35)
static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
static let quick = Animation.easeOut(duration: 0.15)
⋮----
struct LabCardModifier: ViewModifier {
func body(content: Content) -> some View {
⋮----
func labCard() -> some View {
⋮----
func labReadableWidth() -> some View {
⋮----
struct PressableButtonStyle: ButtonStyle {
func makeBody(configuration: Configuration) -> some View {
⋮----
static var pressable: PressableButtonStyle { PressableButtonStyle() }
```

## File: apps/factory-app-ios/FactoryApp/Utilities/GeocodeLocationSupport.swift
```swift
enum GeocodeLocationSupport {
static func hasValidCoordinates(lat: Double, lng: Double) -> Bool {
⋮----
static func resolveLocationValue(_ value: AddressLocationValue) async -> AddressLocationValue? {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/ManifestExceptions/ExceptionRow.swift
```swift
struct ExceptionRow: View {
let exception: ManifestException
var resolving: Bool = false
var onResolve: (() -> Void)? = nil
⋮----
var body: some View {
⋮----
private var attemptLabel: String {
⋮----
private func shortId(_ value: String) -> String {
⋮----
private static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
private static let isoStandardFormatter = ISO8601DateFormatter()
⋮----
private func formattedDate(_ raw: String) -> String {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/FactoryUiComponents.swift
```swift
struct KpiTile: View {
let title: String
let value: String
let systemImage: String
let tint: Color
var supporting: String? = nil
var chip: (text: String, tint: Color)? = nil
var staggerIndex: Int? = nil
var showChevron: Bool = true
⋮----
private var voiceOverSummary: String {
⋮----
var body: some View {
⋮----
private struct OptionalStaggerModifier: ViewModifier {
let index: Int?
⋮----
func body(content: Content) -> some View {
⋮----
func staggeredAppear(index: Int) -> some View {
⋮----
private struct StaggeredAppearModifier: ViewModifier {
let index: Int
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var appeared = false
⋮----
func body(content: Content) -> some View {
⋮----
struct FactorySectionHeader: View {
let title: String
var subtitle: String? = nil
var count: Int? = nil
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Dashboard/DashboardView.swift
```swift
struct DashboardView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
let onOpenSupplyRequests: () -> Void
let onOpenPayloadOverride: () -> Void
let onOpenManifestExceptions: () -> Void
let onOpenAnalytics: () -> Void
let onOpenInsights: () -> Void
var onOpenTransfers: (String) -> Void = { _ in }
var onOpenLoadingBay: () -> Void = {}
⋮----
@State private var vm = DashboardViewModel()
⋮----
private var gridMin: CGFloat {
⋮----
var body: some View {
⋮----
private func handleMetricTap(title: String) {
```

## File: apps/factory-app-ios/FactoryApp/Views/LoginView.swift
```swift
struct LoginView: View {
private enum Field: Hashable {
⋮----
@Environment(TokenStore.self) private var tokenStore
@State private var vm = LoginViewModel()
@FocusState private var focusedField: Field?
⋮----
var body: some View {
⋮----
private func storeAuth(_ auth: AuthResponse) {
```

## File: apps/payload-app-ios/payload-app-ios/Design/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/payload-app-ios/payload-app-ios/Design/CollapsibleSidebar.swift
```swift
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
let title: String
@Binding var isExpanded: Bool
@Binding var selection: Item?
let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
@ViewBuilder let detail: () -> Detail
let footer: (() -> AnyView)?
⋮----
init(
⋮----
var body: some View {
⋮----
private func sidebar(effectiveExpanded: Bool, sidebarWidth: CGFloat, canToggle: Bool) -> some View {
⋮----
private func sidebarRow(_ item: CollapsibleSidebarItem<Item>, effectiveExpanded: Bool) -> some View {
⋮----
struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
let id: String
let tag: Item
let label: String
let icon: String
⋮----
init(id: String? = nil, tag: Item, label: String, icon: String) {
```

## File: apps/payload-app-ios/payload-app-ios/Design/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/payload-app-ios/payload-app-ios/Design/DeliveryLiveActivityWidget.swift
```swift
public struct DeliveryLiveActivityWidget: Widget {
public init() {}
⋮----
public var body: some WidgetConfiguration {
⋮----
private func formatDistance(_ meters: Double) -> String {
⋮----
private struct LockScreenLiveActivityView: View {
let context: ActivityViewContext<DeliveryActivityAttributes>
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Empty pack prints the number only. Never invents UZS.
/// Uses strict integer division and modulo arithmetic (zero floating-point money math).
func formatPackMoney(_ minor: Int64, pack: MarketPack?, decimalPlaces: Int = 2) -> String {
⋮----
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/PegasusMonochromeTheme.swift
```swift
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum PegasusMonochromeTheme {
public static let readableMaxWidth: CGFloat = 960
⋮----
public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
⋮----
public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
⋮----
public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
⋮----
public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
⋮----
public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
⋮----
public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
⋮----
public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
⋮----
public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
⋮----
public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
⋮----
public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
⋮----
public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
⋮----
public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
⋮----
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)
⋮----
public static let spacingXS: CGFloat = 4
public static let spacingSM: CGFloat = 8
public static let spacingMD: CGFloat = 12
public static let spacingLG: CGFloat = 16
public static let spacingXL: CGFloat = 24
public static let spacingXXL: CGFloat = 32
⋮----
public static let radiusXS: CGFloat = 4
public static let radiusSM: CGFloat = 8
public static let radiusMD: CGFloat = 12
public static let radiusLG: CGFloat = 16
public static let radiusCard: CGFloat = 24
public static let radiusXL: CGFloat = 28
⋮----
public static func statusTint(for status: String) -> Color {
⋮----
public struct PegasusMonochromeCard: ViewModifier {
public init() {}
public func body(content: Content) -> some View {
⋮----
public struct MinimumHitTargetModifier: ViewModifier {
public var minSize: CGFloat
⋮----
public init(minSize: CGFloat = 44) {
⋮----
public func body(content: Content) -> some View {
⋮----
func pegasusMonochromeCard() -> some View {
⋮----
func pegasusReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
⋮----
public enum PegasusHaptics {
public static func light() {
⋮----
public static func medium() {
⋮----
public static func heavy() {
⋮----
public static func selection() {
⋮----
public static func success() {
⋮----
public static func warning() {
⋮----
public static func error() {
⋮----
public struct PegasusPressableButtonStyle: ButtonStyle {
public init() {}
public func makeBody(configuration: Configuration) -> some View {
⋮----
static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
```

## File: apps/payload-app-ios/payload-app-ios/Design/PegasusMotion.swift
```swift
public enum PegasusAnim {
public static let snappy = Animation.snappy(duration: 0.3)
public static let smooth = Animation.smooth(duration: 0.35)
public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
public static let quick = Animation.easeOut(duration: 0.15)
⋮----
public static func staggerDelay(index: Int) -> Double {
⋮----
public static func resolved(_ animation: Animation = snappy, reduceMotion: Bool) -> Animation? {
⋮----
public enum PegasusMotionDuration {
public static let short4: Double = 0.2
public static let medium2: Double = 0.3
public static let medium4: Double = 0.4
⋮----
func pegasusNumericMetric<V: Equatable>(value: V) -> some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/PegasusStateViews.swift
```swift
public struct PegasusLoadingView: View {
public let title: String
public var message: String = "Fetching the latest data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
public init(title: String, message: String = "Fetching the latest data.") {
⋮----
public var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Design/RealtimeRefresh.swift
```swift
public struct SilentRefreshModifier: ViewModifier {
private let refreshEpoch: Int
private let reconnectEpoch: Int
private let reload: (_ silent: Bool) -> Void
⋮----
@State private var coalesceTask: Task<Void, Never>?
⋮----
public init(
⋮----
public func body(content: Content) -> some View {
⋮----
private func scheduleCoalescedReload() {
⋮----
func silentRealtimeRefresh(
⋮----
public enum RealtimeLoadState {
public static func showFullScreenLoading(loading: Bool, hasData: Bool) -> Bool {
⋮----
public enum PulseHonesty {
public static let failed = "pulse_failed"
public static let commandFailed = "control_tower_pulse_failed"
⋮----
public struct Result<T> {
public let events: [T]
public let error: String?
⋮----
public struct ObjectResult<T> {
public let value: T?
public let error: String?
⋮----
public static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
⋮----
public static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
```

## File: apps/payload-app-ios/payload-app-ios/Design/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Models/Models.swift
```swift
struct LoginRequest: Encodable {
var phone: String = ""
var pin: String = ""
var idToken: String = ""
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct LoginResponse: Decodable {
let token: String
let refreshToken: String?
let workerId: String
let supplierId: String
let role: String
let name: String
let warehouseId: String
let warehouseName: String
let warehouseLat: Double
let warehouseLng: Double
let firebaseToken: String?
⋮----
init(
⋮----
struct RefreshTokenRequest: Encodable {
let refreshToken: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogBarcodeLookup: Decodable {
let productId: String?
let skuId: String?
let name: String?
let barcode: String?
⋮----
struct RefreshTokenResponse: Decodable {
let token: String
let refreshToken: String?
⋮----
struct Truck: Decodable, Identifiable, Equatable {
let id: String
let label: String?
let licensePlate: String?
let vehicleClass: String?
let truckStatus: String?
let usedVolumeVu: Double?
let maxVolumeVu: Double?
let stopCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
static func fixture(
⋮----
func withBoard(status: String, used: Double?, max: Double?, stops: Int?) -> Truck {
⋮----
struct LiveOrderItem: Decodable, Identifiable {
let lineItemId: String
let skuId: String
let skuName: String
let quantity: Int
let unitPrice: Int64?
let status: String?
var id: String { lineItemId }
⋮----
var totalTiyin: Int64 {
⋮----
struct LiveOrder: Decodable, Identifiable {
let orderId: String
let retailerId: String?
let amount: Int64?
let paymentGateway: String?
let state: String
let routeId: String?
let warehouseId: String?
let items: [LiveOrderItem]?
var id: String { orderId }
⋮----
var calculatedTotalTiyin: Int64 {
⋮----
public enum CurrencyFormat {
public static func formatTiyin(_ tiyin: Int64, currency: String = "UZS") -> String {
⋮----
struct Manifest: Decodable, Identifiable {
let manifestId: String
let truckId: String?
let vehicleId: String?
let driverId: String?
let state: String
let usedVolumeVu: Double?
let maxVolumeVu: Double?
let stopCount: Int?
let regionCode: String?
let sealedAt: String?
let dispatchedAt: String?
let createdAt: String?
⋮----
let orders: [LiveOrder]?
let overflowCount: Int?
⋮----
var source: String = "payloader"
var id: String { manifestId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func matchesTruck(_ truckId: String) -> Bool {
⋮----
func withSource(_ source: String) -> Manifest {
⋮----
init(
⋮----
struct ManifestsResponse: Decodable {
let manifests: [Manifest]
⋮----
struct SealOrderRequest: Encodable {
let manifestId: String
let orderId: String
let terminalId: String
let manifestCleared: Bool
enum CodingKeys: String, CodingKey {
⋮----
struct SealOrderResponse: Decodable {
let status: String?
let dispatchCode: String
let orderId: String
⋮----
struct SealManifestResponse: Decodable {
let status: String?
let stopCount: Int?
let volumeVu: Double?
let maxVu: Double?
⋮----
struct SealCompletedManifestsRequest: Encodable {
let manifestIds: [String]
enum CodingKeys: String, CodingKey { case manifestIds = "manifest_ids" }
⋮----
struct SealCompletedManifestsResponse: Decodable {
let status: String?
let sealedCount: Int?
let results: [SealCompletedManifestResult]?
enum CodingKeys: String, CodingKey {
⋮----
struct SealCompletedManifestResult: Decodable {
let manifestId: String?
let status: String?
let explain: StatusExplain?
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestExceptionRequest: Encodable {
let manifestId: String
let orderId: String
let reason: String
let metadata: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestExceptionResponse: Decodable {
let status: String?
let escalated: Bool?
let overflowCount: Int?
⋮----
struct ManifestExceptionRow: Decodable, Identifiable {
let exceptionId: String
let manifestId: String
let orderId: String
let reason: String
let attemptCount: Int
let escalated: Bool
let createdAt: String
var id: String { exceptionId }
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestExceptionsResponse: Decodable {
let exceptions: [ManifestExceptionRow]
⋮----
struct InjectOrderRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey { case orderId = "order_id" }
⋮----
struct RecommendReassignRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey { case orderId = "order_id" }
⋮----
struct TruckRecommendation: Decodable, Identifiable {
let driverId: String
let driverName: String?
let vehicleId: String?
let vehicleClass: String?
let licensePlate: String?
let maxVolumeVu: Double?
let usedVolumeVu: Double?
let freeVolumeVu: Double?
let distanceKm: Double?
let orderCount: Int?
let truckStatus: String?
let score: Double?
let recommendation: String?
var id: String { driverId }
⋮----
struct RecommendReassignResponse: Decodable {
let orderId: String?
let retailerName: String?
let orderVolumeVu: Double?
let currentDriver: String?
let recommendations: [TruckRecommendation]
⋮----
struct FleetReassignRequest: Encodable {
let orderIds: [String]
let newRouteId: String
enum CodingKeys: String, CodingKey {
⋮----
struct ReassignConflict: Decodable, Identifiable {
let orderId: String
let reason: String?
var id: String { orderId }
⋮----
struct FleetReassignResponse: Decodable {
let conflicts: [ReassignConflict]?
let total: Int?
let reassigned: Int?
let newRouteId: String?
⋮----
struct MissingItemEntry: Encodable {
let lineItemId: String
let quantity: Int
enum CodingKeys: String, CodingKey {
⋮----
struct MissingItemsRequest: Encodable {
let orderId: String
let missingItems: [MissingItemEntry]
enum CodingKeys: String, CodingKey {
⋮----
struct DeviceTokenRequest: Encodable {
let token: String
let platform: String
⋮----
struct PulseEvent: Decodable, Identifiable {
let id: String
let kind: String
let title: String
let description: String?
let occurredAt: String
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PulseResponse: Decodable {
let events: [PulseEvent]
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct StatusExplain: Decodable, Equatable {
let code: String
let title: String
let summary: String
let nextSteps: [String]?
let deepLink: String?
let recoverable: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct HandoffCardMetadata: Decodable, Equatable {
let kind: String
let title: String
let subtitle: String?
let primaryCta: String?
let primaryLink: String?
let entityType: String?
let entityId: String?
let fields: [String: String]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ApiErrorBody: Decodable {
let error: String?
let detail: String?
let message: String?
let explain: StatusExplain?
⋮----
enum ApiExplainParser {
static func parse(from data: Data) -> (message: String, explain: StatusExplain?)? {
⋮----
struct NotificationItem: Decodable, Identifiable {
let notificationId: String
let type: String
let title: String
let body: String
let payload: String?
let channel: String
let readAt: String?
let createdAt: String
let handoffMetadata: HandoffCardMetadata?
var id: String { notificationId }
var isUnread: Bool { (readAt ?? "").isEmpty }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NotificationsResponse: Decodable {
let notifications: [NotificationItem]
let unreadCount: Int
let total: Int
let limit: Int
⋮----
struct MarkReadRequest: Encodable {
let notificationIds: [String]?
let markAll: Bool?
enum CodingKeys: String, CodingKey {
⋮----
struct ClientPolicyResponse: Decodable {
let role: String
let outdated: Bool
let forceUpdate: Bool
let updateDeferred: Bool?
let minimumVersion: String
let recommendedVersion: String?
let updateURL: String?
let deferReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundReturnRow: Decodable, Identifiable {
var id: String { returnId }
let returnId: String
let orderId: String
let skuId: String
let productName: String
let barcode: String?
let expectedQty: Int
let receivedQty: Int
let reason: String
let physicalStatus: String
let driverName: String
let suggestedDisposition: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundReturnListResponse: Decodable {
let data: [InboundReturnRow]
⋮----
struct InboundSessionResponse: Decodable {
let sessionId: String
⋮----
struct InboundScanResponse: Decodable {
let matched: Bool
let returnId: String?
let variance: Bool
let message: String?
⋮----
struct InboundHistoryResponse: Decodable {
let data: [InboundReturnRow]
⋮----
struct WsMessage: Decodable {
let type: String?
let title: String?
let body: String?
let channel: String?
let manifestId: String?
let warehouseId: String?
let reason: String?
let timestamp: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct QueuedAction: Codable, Identifiable {
let id: String
let endpoint: String
let method: String
let body: String
let createdAt: Double
enum CodingKeys: String, CodingKey {
⋮----
struct StatusResponse: Decodable { let status: String? }
```

## File: apps/payload-app-ios/payload-app-ios/Services/APIClient.swift
```swift
enum APIError: Error {
⋮----
struct ProblemDetail: Decodable {
let type: String?
let title: String?
let status: Int
let detail: String?
let traceId: String?
let code: String?
let retryable: Bool?
⋮----
private actor PayloadHTTPCoalescer {
private struct CacheEntry {
let data: Data
let response: URLResponse
let timestamp: Date
⋮----
private var inFlightGets: [String: Task<(Data, URLResponse), Error>] = [:]
private var microCache: [String: CacheEntry] = [:]
private let microCacheTTL: TimeInterval = 2.0
private var refreshTask: Task<Bool, Error>?
⋮----
func fetchGet(
⋮----
func invalidateCache() {
⋮----
func refreshSingleFlight(
⋮----
final class APIClient: @unchecked Sendable {
static let shared = APIClient()
⋮----
let bootstrapURL: String = {
⋮----
if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
⋮----
if baseURL.hasPrefix("https://") {
⋮----
func login(phone: String, pin: String) async throws -> LoginResponse {
⋮----
func login(idToken: String) async throws -> LoginResponse {
⋮----
func lookupBarcode(ean: String) async throws -> CatalogBarcodeLookup {
⋮----
func trucks() async throws -> [Truck] { try await get("v1/payloader/trucks") }
func orders(vehicleId: String?, state: String? = nil) async throws -> [LiveOrder] {
⋮----
func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
⋮----
func reassignOrder(orderId: String, toDriverId: String, isPartial: Bool = false, reason: String = "payload-redispatch") async throws -> StatusResponse {
⋮----
func draftManifests(truckId: String?) async throws -> ManifestsResponse {
⋮----
func manifests(state: String, truckId: String?) async throws -> ManifestsResponse {
⋮----
func manifestDetail(_ manifestId: String) async throws -> Manifest {
⋮----
func startLoading(manifestId: String) async throws -> StatusResponse {
⋮----
func sealManifest(manifestId: String) async throws -> SealManifestResponse {
⋮----
func injectOrder(manifestId: String, orderId: String) async throws -> StatusResponse {
⋮----
func supplierManifests(state: String = "DRAFT") async throws -> ManifestsResponse {
⋮----
func supplierManifestDetail(_ manifestId: String) async throws -> Manifest {
⋮----
func supplierStartLoading(manifestId: String) async throws -> StatusResponse {
⋮----
func supplierSealManifest(manifestId: String) async throws -> SealManifestResponse {
⋮----
func factoryManifests(state: String = "DRAFT") async throws -> ManifestsResponse {
⋮----
func factoryManifestDetail(_ manifestId: String) async throws -> Manifest {
⋮----
func factoryStartLoading(manifestId: String) async throws -> StatusResponse {
⋮----
func factorySealManifest(manifestId: String) async throws -> SealManifestResponse {
⋮----
func listLoadingBayManifests(state: String = "DRAFT") async throws -> [Manifest] {
⋮----
func sealCompletedManifests(manifestIds: [String]) async throws -> SealCompletedManifestsResponse {
⋮----
func sealAllManifests() async throws -> SealCompletedManifestsResponse {
⋮----
func loadingManifests() async throws -> ManifestsResponse {
⋮----
func supplierInjectOrder(manifestId: String, orderId: String) async throws -> StatusResponse {
⋮----
func sealOrder(manifestId: String, orderId: String, terminalId: String) async throws -> SealOrderResponse {
⋮----
func manifestException(manifestId: String, orderId: String, reason: String, metadata: String = "") async throws -> ManifestExceptionResponse {
⋮----
func manifestExceptionsList(limit: Int = 50, offset: Int = 0) async throws -> ManifestExceptionsResponse {
⋮----
func reportMissingItems(orderId: String, items: [MissingItemEntry]) async throws -> StatusResponse {
⋮----
func reconcileSession() async {
⋮----
func fleetReassign(orderIds: [String], newRouteId: String) async throws -> FleetReassignResponse {
⋮----
func notifications(limit: Int = 100, offset: Int = 0) async throws -> NotificationsResponse {
⋮----
func pulse() async throws -> PulseResponse {
⋮----
func markRead(ids: [String]?, all: Bool? = nil) async throws -> StatusResponse {
⋮----
func clientPolicy(platform: String, version: String) async throws -> ClientPolicyResponse {
⋮----
func inboundReturns(physicalStatus: String = "ARRIVED", limit: Int = 100) async throws -> InboundReturnListResponse {
⋮----
func createInboundSession() async throws -> InboundSessionResponse {
⋮----
func scanInboundBarcode(barcode: String, qty: Int = 1, sessionId: String) async throws -> InboundScanResponse {
struct ScanBody: Encodable {
let barcode: String
let qty: Int
let sessionId: String
enum CodingKeys: String, CodingKey {
⋮----
func confirmInboundReturns(
⋮----
struct Line: Encodable {
let returnId: String
let disposition: String
let qty: Int?
enum CodingKeys: String, CodingKey {
⋮----
struct ConfirmBody: Encodable {
let lines: [Line]
let sessionId: String
enum CodingKeys: String, CodingKey {
⋮----
func returnsHistory(limit: Int = 50) async throws -> InboundHistoryResponse {
⋮----
func registerDeviceToken(_ token: String) async throws -> StatusResponse {
⋮----
func unregisterDeviceToken(_ token: String) async throws -> StatusResponse {
⋮----
func rawRequest(endpoint: String, method: String, body: String, idempotencyKey: String? = nil) async throws -> (Int, Data) {
⋮----
private struct EmptyBody: Encodable {}
⋮----
private func get<T: Decodable>(_ path: String) async throws -> T {
⋮----
private func post<B: Encodable, T: Decodable>(
⋮----
private func deterministicIdempotencyKey(action: String, entityId: String) -> String {
⋮----
private func buildRequest(path: String, method: String, authenticated: Bool = true) throws -> URLRequest {
⋮----
private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
⋮----
private func parseProblem(_ data: Data) -> ProblemDetail? {
⋮----
private func attemptRefresh() async throws -> Bool {
⋮----
private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
```

## File: apps/payload-app-ios/payload-app-ios/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.payload.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/payload-app-ios/payload-app-ios/Theme/TermTheme.swift
```swift
enum TermTheme {
⋮----
static let bg = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 14/255.0, green: 14/255.0, blue: 16/255.0, alpha: 1.0) : UIColor(red: 237/255.0, green: 237/255.0, blue: 240/255.0, alpha: 1.0) })
static let card = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 24/255.0, green: 24/255.0, blue: 27/255.0, alpha: 1.0) : UIColor.white })
static let accent = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 244/255.0, green: 244/255.0, blue: 245/255.0, alpha: 1.0) : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
static let secondary = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) : UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) })
static let tertiary = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) : UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) })
static let separator = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 228/255.0, green: 228/255.0, blue: 231/255.0, alpha: 1.0) })
static let accentWarm = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 210/255.0, green: 179/255.0, blue: 155/255.0, alpha: 1.0) : UIColor(red: 188/255.0, green: 127/255.0, blue: 112/255.0, alpha: 1.0) })
static let accentWarmSoft = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 54/255.0, green: 39/255.0, blue: 35/255.0, alpha: 1.0) : UIColor(red: 245/255.0, green: 236/255.0, blue: 233/255.0, alpha: 1.0) })
static let accentBronze = Color(UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 228/255.0, green: 191/255.0, blue: 162/255.0, alpha: 1.0) : UIColor(red: 98/255.0, green: 77/255.0, blue: 69/255.0, alpha: 1.0) })
⋮----
static let live = Color(UIColor.systemGreen)
static let alert = Color(UIColor.systemRed)
static let warn = Color(UIColor.systemOrange)
static let progress = Color(UIColor.systemBlue)
⋮----
static func statusTint(for status: String) -> Color {
⋮----
static let radiusLG: Double = 32
static let radiusMD: Double = 24
static let radiusSM: Double = 16
static let radiusXS: Double = 8
⋮----
static let s4: Double = 4
static let s8: Double = 8
static let s12: Double = 12
static let s16: Double = 16
static let s20: Double = 20
static let s24: Double = 24
static let s32: Double = 32
⋮----
func tacticalCard(radius: Double = TermTheme.radiusMD) -> some View {
⋮----
func tacticalButton(radius: Double = TermTheme.radiusSM) -> some View {
⋮----
enum TermAnim {
static let snappy = Animation.spring(response: 0.35, dampingFraction: 0.85, blendDuration: 0)
static let fluid = Animation.spring(response: 0.5, dampingFraction: 0.9, blendDuration: 0)
⋮----
private struct PressableModifier: ViewModifier {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var isPressed = false
⋮----
func body(content: Content) -> some View {
⋮----
func pressable() -> some View { modifier(PressableModifier()) }
⋮----
struct TacticalButtonStyle: ButtonStyle {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
⋮----
func makeBody(configuration: Configuration) -> some View {
⋮----
static var tactical: TacticalButtonStyle { TacticalButtonStyle() }
static var pressable: TacticalButtonStyle { TacticalButtonStyle() }
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/ManifestKpiGrid.swift
```swift
struct ManifestKpiGrid: View {
let manifest: Manifest
⋮----
private var progress: Double {
⋮----
private var volumeLabel: String {
⋮----
var body: some View {
⋮----
private var volumeKpiTile: some View {
⋮----
private var stopsKpiTile: some View {
⋮----
private var regionKpiTile: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/PayloadSectionHeader.swift
```swift
struct PayloadSectionHeader: View {
let title: String
var subtitle: String? = nil
var trailing: String? = nil
var trailingTint: Color = TermTheme.accent
⋮----
var body: some View {
⋮----
struct PayloadStatusBadge: View {
let text: String
var tint: Color? = nil
var large = false
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/HomeView.swift
```swift
struct HomeView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(\.scenePhase) private var scenePhase
@State private var viewModel = HomeViewModel()
@State private var isTruckSidebarExpanded = true
@State private var columnVisibility: NavigationSplitViewVisibility = .all
⋮----
var body: some View {
⋮----
private var navigationRoot: some View {
⋮----
private var sidebarToolbar: some ToolbarContent {
⋮----
private var detailToolbar: some ToolbarContent {
⋮----
private struct HomeSheetsModifier: ViewModifier {
@Bindable var viewModel: HomeViewModel
⋮----
func body(content: Content) -> some View {
⋮----
private struct HomeLifecycleModifier: ViewModifier {
@Bindable var viewModel: HomeViewModel
let tokenStore: TokenStore
let scenePhase: ScenePhase
⋮----
func body(content: Content) -> some View {
⋮----
func homeSheets(viewModel: HomeViewModel) -> some View {
⋮----
func homeLifecycle(
```

## File: apps/payload-app-ios/payload-app-ios/Views/LoginView.swift
```swift
struct LoginView: View {
@State private var viewModel = LoginViewModel()
@FocusState private var focus: Field?
⋮----
private enum Field: Hashable { case phone, otp, pin }
⋮----
var body: some View {
⋮----
private func primaryButtonLabel(_ title: String) -> some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/ManifestDetailPane.swift
```swift
struct ManifestDetailPane: View {
@Bindable var viewModel: HomeViewModel
let onShowException: (String) -> Void
let onShowReDispatch: (String) -> Void
let onScanProduct: () -> Void
⋮----
var body: some View {
⋮----
struct ManifestWorkflow: View {
let manifest: Manifest
let truck: Truck?
@Bindable var viewModel: HomeViewModel
let onShowException: (String) -> Void
let onShowReDispatch: (String) -> Void
let onScanProduct: () -> Void
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Views/TruckSidebar.swift
```swift
struct TruckSidebar: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Bindable var viewModel: HomeViewModel
@Binding var isExpanded: Bool
⋮----
var body: some View {
⋮----
private var expandedTruckList: some View {
⋮----
private var collapsedTruckRail: some View {
⋮----
struct TruckRow: View {
let truck: Truck
⋮----
var body: some View {
⋮----
private var displayLabel: String {
⋮----
private var subtitle: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Catalog/CatalogBentoGrid.swift
```swift
struct CatalogBentoGrid: View {
let categories: [ProductCategory]
⋮----
var body: some View {
⋮----
func bentoBig(_ cat: ProductCategory, height: Double) -> some View {
⋮----
func bentoWide(_ cat: ProductCategory, height: Double) -> some View {
⋮----
func bentoSmall(_ cat: ProductCategory) -> some View {
⋮----
func bentoCompact(_ cat: ProductCategory) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Dashboard/AiPredictionSection.swift
```swift
struct AiPredictionSection: View {
let predictions: [RetailerAIPrediction]
@Binding var actingId: String?
let onConfirm: (RetailerAIPrediction) async -> Void
let onReject: (RetailerAIPrediction) async -> Void
⋮----
var body: some View {
⋮----
struct PredictionCard: View {
let item: RetailerAIPrediction
@Binding var actingId: String?
let onConfirm: (RetailerAIPrediction) async -> Void
let onReject: (RetailerAIPrediction) async -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Dashboard/KpiGrid.swift
```swift
struct KpiGrid: View {
let activeOrdersCount: Int
let predictionsCount: Int
let reorderProductsCount: Int
let horizontalSizeClass: UserInterfaceSizeClass?
⋮----
private var kpiGridMin: CGFloat {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Dashboard/ServiceGrid.swift
```swift
struct ServiceGrid: View {
let activeOrdersCount: Int
let predictionsCount: Int
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/CreditProfileSection.swift
```swift
struct CreditProfileSection: View {
let profile: CreditProfile?
let isLoading: Bool
let missing: Bool
let error: String?
⋮----
private static let minorFormatter: NumberFormatter = {
⋮----
private func formatMinor(_ value: Int64) -> String {
⋮----
var body: some View {
⋮----
private func statusColor(_ status: String) -> Color {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Profile/StatsRowView.swift
```swift
struct StatsRowView: View {
var orderCount: Int
var totalSpent: Int64
var totalSpentCurrency: String
⋮----
private static let spentFormatter: NumberFormatter = {
⋮----
private func formatSpent(_ amount: Int64, currency: String) -> String {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/BottomSheetOverlay.swift
```swift
struct BottomSheetOverlay<Content: View>: View {
@Binding var isPresented: Bool
let snapFraction: CGFloat
@ViewBuilder var content: () -> Content
⋮----
@State private var dragOffset: CGFloat = 0
@State private var sheetHeight: CGFloat = 0
@GestureState private var isDragging = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/OrderCardView.swift
```swift
struct OrderCardView: View {
let order: Order
var onCancel: (() -> Void)?
var onConfirmAi: (() -> Void)? = nil
var onRejectAi: (() -> Void)? = nil
var onConfirmPreorder: (() -> Void)? = nil
var onEditPreorder: (() -> Void)? = nil
var onReviewDeliveryProposal: (() -> Void)? = nil
var onTap: (() -> Void)? = nil
⋮----
@State private var appeared = false
⋮----
var body: some View {
⋮----
private var statusBadge: some View {
⋮----
private var orderProgressBar: some View {
⋮----
private var statusColor: Color {
⋮----
private var statusIcon: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/QuantityStepper.swift
```swift
struct QuantityStepper: View {
@Binding var quantity: Int
var minimum: Int = 1
var maximum: Int = 99
var compact: Bool = false
⋮----
var body: some View {
⋮----
private func stepperButton(icon: String, enabled: Bool, action: @escaping () -> Void) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/SidebarMenu.swift
```swift
struct SidebarMenu: View {
@Binding var isOpen: Bool
var onNavigate: ((SidebarDestination) -> Void)?
⋮----
@State private var dragOffset: Double = 0
@State private var appeared = false
⋮----
var body: some View {
⋮----
private var sidebarHeader: some View {
⋮----
private func menuRow(_ destination: SidebarDestination) -> some View {
⋮----
enum SidebarDestination: String, CaseIterable {
⋮----
static var menuItems: [SidebarDestination] {
⋮----
var title: String {
⋮----
var icon: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Design/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct SourceChip: View {
let source: String
⋮----
var body: some View {
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
⋮----
func retailerOrderMatchesCommand(
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/Category.swift
```swift
struct ProductCategory: Codable, Identifiable, Hashable {
let id: String
let name: String
let icon: String
let productCount: Int?
let supplierCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(id: String, name: String, icon: String, productCount: Int?, supplierCount: Int? = nil) {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct SimpleAutoOrderSettings: Codable {
var globalEnabled: Bool
var supplierSettings: [String: Bool]
var productSettings: [String: Bool]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
static let `default` = SimpleAutoOrderSettings(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/Product.swift
```swift
enum RetailerMoney {
⋮----
static func formatTiyin(_ minor: Int64) -> String {
⋮----
static func decodeInt64<K: CodingKey>(from container: KeyedDecodingContainer<K>, forKey key: K) -> Int64? {
⋮----
struct Variant: Codable, Identifiable, Hashable {
let id: String
let size: String
let pack: String
let packCount: Int
let weightPerUnit: String
⋮----
let priceMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(id: String, size: String, pack: String, packCount: Int, weightPerUnit: String, priceMinor: Int64) {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct ProductOffer: Codable, Hashable {
let productID: String
let listPriceMinor: Int64
let salePriceMinor: Int64?
let discountBps: Int64?
let promotionID: String?
let promotionName: String?
let promotionLabel: String?
let promotionEndsAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct Product: Codable, Identifiable, Hashable {
let id: String
let name: String
let description: String
let nutrition: String
let imageURL: String?
let variants: [Variant]
let supplierID: String?
let supplierName: String?
let supplierCategory: String?
let categoryID: String?
let categoryName: String?
let sellByBlock: Bool
let unitsPerBlock: Int?
let priceMinor: Int64?
let availableStock: Int?
let isOutOfStockFlag: Bool
let acceptsBackorder: Bool
let showStockCounts: Bool
let maxQuantity: Int?
let offer: ProductOffer?
⋮----
var isOutOfStock: Bool {
⋮----
var isLowStock: Bool { availableStock.map { (1...5).contains($0) } ?? false }
var blocksAddToCart: Bool { isOutOfStock && !acceptsBackorder }
var cartMaxQuantity: Int? {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
var defaultVariant: Variant? { variants.first }
var hasSaleOffer: Bool {
⋮----
var displayListPrice: String? {
⋮----
var displayPrice: String {
⋮----
var promotionLabel: String? {
⋮----
var merchandisingLabel: String? {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/Supplier.swift
```swift
struct Supplier: Codable, Identifiable, Hashable {
let id: String
let name: String
let logoURL: String?
let category: String?
let orderCount: Int
let productCount: Int
let lastOrderDate: String?
let phone: String?
let email: String?
let address: String?
let primaryCategoryID: String?
let operatingCategoryIDs: [String]
let operatingCategoryNames: [String]
⋮----
let isActive: Bool
let manualOffShift: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
var catalogSubtitle: String {
⋮----
var displayCategory: String? {
⋮----
var initials: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/AutoOrderView.swift
```swift
struct AutoOrderView: View {
@Environment(\.dismiss) private var dismiss
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = AutoOrderViewModel()
⋮----
var body: some View {
⋮----
private var loadingState: some View {
⋮----
private var executionModeCard: some View {
⋮----
private var shadowInboxCard: some View {
⋮----
private func autoOrderWorkerCard(placeConfirmOpenBinding: Binding<Bool>) -> some View {
⋮----
private var reorderSuggestionsCard: some View {
⋮----
private func syncStatusBanner(_ message: String) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/DeliveriesHubView.swift
```swift
enum DeliveriesHubTab: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var title: String {
⋮----
struct DeliveriesHubView: View {
var initialTab: DeliveriesHubTab = .map
⋮----
@State private var selectedTab: DeliveriesHubTab
⋮----
init(initialTab: DeliveriesHubTab = .map) {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/FileClaimView.swift
```swift
struct FileClaimView: View {
let order: Order
var preferredSku: String? = nil
@Environment(\.dismiss) private var dismiss
@State private var vm: FileClaimViewModel
⋮----
init(order: Order, preferredSku: String? = nil) {
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/LocalSkusView.swift
```swift
struct LocalSkusView: View {
@State private var vm = LocalSkusViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/PosView.swift
```swift
struct PosView: View {
@State private var vm = PosViewModel()
@State private var network = NetworkLifecycleEngine.shared
private var isOnline: Bool { network.isOnline }
⋮----
private var sessionId: String? { vm.sessionId }
private var cart: [PosCartLine] { vm.cart }
private var banner: String? { vm.banner }
private var busy: Bool { vm.busy }
private var lastSaleId: String? { vm.lastSaleId }
private var pending: [PendingPosStore.Entry] { vm.pending }
private var holdsEnabled: Bool { vm.holdsEnabled }
private var serverHolds: [PosHoldWire] { vm.serverHolds }
private var totalMinor: Int64 { vm.totalMinor }
⋮----
var body: some View {
⋮----
private func refreshPending() {
⋮----
private func ensureRegister() async {
⋮----
private func loadServerHolds() async {
⋮----
private func parkHold() async {
⋮----
private func resumeHold(_ hold: PosHoldWire) async {
⋮----
private func voidHold(_ hold: PosHoldWire) async {
⋮----
private func openSession() async {
⋮----
private func closeSession() async {
⋮----
private func addLine() {
⋮----
private func completeSale() async {
⋮----
private func syncPending() async {
⋮----
private func voidLast() async {
⋮----
struct PosCartLine: Identifiable {
let id: String
let sku: String
let name: String
var qty: Int64
let unitPriceMinor: Int64
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ProductDetailView.swift
```swift
struct ProductDetailView: View {
let product: Product
@Environment(CartManager.self) private var cart
@Environment(\.dismiss) private var dismiss
⋮----
@State private var vm = ProductDetailViewModel()
⋮----
var body: some View {
⋮----
private var heroImage: some View {
⋮----
private var heroPlaceholder: some View {
⋮----
private var productInfo: some View {
⋮----
private var variantPicker: some View {
⋮----
private func quantitySection(quantityBinding: Binding<Int>) -> some View {
⋮----
private var nutritionSection: some View {
⋮----
private var addToCartBar: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ProfileView.swift
```swift
struct ProfileView: View {
@State private var vm = ProfileViewModel()
@State private var refreshCenter = RetailerRefreshCenter.shared
@AppStorage("aiAutoOrder") private var aiAutoOrder = false
@AppStorage("globalAutoOrder") private var globalAutoOrder = false
@AppStorage("notificationsEnabled") private var notificationsEnabled = true
⋮----
@Environment(AuthManager.self) private var auth
⋮----
private var user: User {
⋮----
private var displayName: String { vm.profileName.isEmpty ? user.name : vm.profileName }
private var displayCompany: String { vm.profileCompany.isEmpty ? user.company : vm.profileCompany }
⋮----
var body: some View {
⋮----
private func refreshAll() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ReportsProView.swift
```swift
struct ReportsProView: View {
@State private var salesMinor: Int64 = 0
@State private var saleCount = 0
@State private var onHand = 0
@State private var lowStock = 0
@State private var topLine = "—"
@State private var banner: String?
@State private var loadError: String?
@State private var summaryReady = false
@State private var exportURL: URL?
@State private var exporting = false
@State private var showShare = false
private let api = APIClient.shared
⋮----
var body: some View {
⋮----
private func load() async {
⋮----
private func exportCSV() async {
⋮----
private struct ActivityView: UIViewControllerRepresentable {
let activityItems: [Any]
⋮----
func makeUIViewController(context: Context) -> UIActivityViewController {
⋮----
func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SearchView.swift
```swift
struct SearchView: View {
@Environment(CartManager.self) private var cart
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = SearchViewModel()
@FocusState private var isSearchFocused: Bool
⋮----
private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]
⋮----
var body: some View {
⋮----
private var emptySearchState: some View {
⋮----
private var noResultsState: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ShopClosedBanner.swift
```swift
struct ShopClosedBanner: View {
let event: ShopClosedAlertEvent
let onRespond: (String) -> Void
@State private var socket = RetailerWebSocket.shared
@State private var vm = ShopClosedBannerViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.retailer.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/PendingPosStore.swift
```swift
final class PendingPosStore {
static let shared = PendingPosStore()
private let key = "retailer_pending_pos_sales_v1"
private let defaults = UserDefaults.standard
private let lock = NSLock()
⋮----
struct Entry: Codable, Identifiable {
var id: String { clientSaleId }
let clientSaleId: String
let clientReceipt: String
let sessionId: String
let payloadJson: String
let createdAt: TimeInterval
var retryCount: Int
var status: String
var lastError: String?
⋮----
private init() {}
⋮----
func list() -> [Entry] {
⋮----
func active() -> [Entry] {
⋮----
func countActive(sessionId: String) -> Int {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
func enqueue(sessionId: String, lines: [PosSaleLineWire], totalMinor: Int64) throws -> Entry {
⋮----
func update(_ entry: Entry) {
⋮----
func remove(clientSaleId: String) {
⋮----
private func save(_ rows: [Entry]) {
⋮----
func flush(using api: APIClient) async -> (flushed: Int, failed: Int) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerRefreshCenter.swift
```swift
final class RetailerRefreshCenter {
static let shared = RetailerRefreshCenter()
⋮----
private(set) var refreshToken = 0
private(set) var lastRefreshAt = Date.distantPast
private var trailingTask: Task<Void, Never>?
private let coalesceInterval: TimeInterval = 0.20
⋮----
private init() {}
⋮----
func trigger() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Utilities/CountdownTimer.swift
```swift
final class CountdownTimer {
var remainingString: String = "--:--:--"
var isExpired: Bool = false
⋮----
private var targetDate: Date?
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
func start(from isoString: String) {
⋮----
func start(from date: Date) {
⋮----
func update(now: Date) {
⋮----
struct CountdownText: View {
let targetISO: String
var font: Font = .caption.monospacedDigit()
var color: Color = AppTheme.textSecondary
⋮----
var body: some View {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private func countdownString(from now: Date) -> String {
```

## File: apps/retailer-app-ios/retailerapp/Views/Dock/ShelfAlertModal.swift
```swift
struct ShelfAlertModal: View {
let productId: String
let currentStock: Int64
let onDismiss: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Design/CellApi.swift
```swift
static func trimApiBase(_ url: String) -> String {
⋮----
static func isDevApiBootstrap(_ url: String) -> Bool {
⋮----
static func homeCellFromJwt(_ token: String?) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Design/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/supplier-app-ios/SupplierApp/Design/DeliveryLiveActivityWidget.swift
```swift
public struct DeliveryLiveActivityWidget: Widget {
public init() {}
⋮----
public var body: some WidgetConfiguration {
⋮----
private func formatDistance(_ meters: Double) -> String {
⋮----
private struct LockScreenLiveActivityView: View {
let context: ActivityViewContext<DeliveryActivityAttributes>
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Design/PegasusMonochromeTheme.swift
```swift
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum PegasusMonochromeTheme {
public static let readableMaxWidth: CGFloat = 960
⋮----
public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
⋮----
public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
⋮----
public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
⋮----
public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
⋮----
public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
⋮----
public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
⋮----
public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
⋮----
public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
⋮----
public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
⋮----
public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
⋮----
public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
⋮----
public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
⋮----
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)
⋮----
public static let spacingXS: CGFloat = 4
public static let spacingSM: CGFloat = 8
public static let spacingMD: CGFloat = 12
public static let spacingLG: CGFloat = 16
public static let spacingXL: CGFloat = 24
public static let spacingXXL: CGFloat = 32
⋮----
public static let radiusXS: CGFloat = 4
public static let radiusSM: CGFloat = 8
public static let radiusMD: CGFloat = 12
public static let radiusLG: CGFloat = 16
public static let radiusCard: CGFloat = 24
public static let radiusXL: CGFloat = 28
⋮----
public static func statusTint(for status: String) -> Color {
⋮----
public struct PegasusMonochromeCard: ViewModifier {
public init() {}
public func body(content: Content) -> some View {
⋮----
public struct MinimumHitTargetModifier: ViewModifier {
public var minSize: CGFloat
⋮----
public init(minSize: CGFloat = 44) {
⋮----
public func body(content: Content) -> some View {
⋮----
func pegasusMonochromeCard() -> some View {
⋮----
func pegasusReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
⋮----
public enum PegasusHaptics {
public static func light() {
⋮----
public static func medium() {
⋮----
public static func heavy() {
⋮----
public static func selection() {
⋮----
public static func success() {
⋮----
public static func warning() {
⋮----
public static func error() {
⋮----
public struct PegasusPressableButtonStyle: ButtonStyle {
public init() {}
public func makeBody(configuration: Configuration) -> some View {
⋮----
static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
```

## File: apps/supplier-app-ios/SupplierApp/Design/PegasusMotion.swift
```swift
public enum PegasusAnim {
public static let snappy = Animation.snappy(duration: 0.3)
public static let smooth = Animation.smooth(duration: 0.35)
public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
public static let quick = Animation.easeOut(duration: 0.15)
⋮----
public static func staggerDelay(index: Int) -> Double {
⋮----
public static func resolved(_ animation: Animation = snappy, reduceMotion: Bool) -> Animation? {
⋮----
public enum PegasusMotionDuration {
public static let short4: Double = 0.2
public static let medium2: Double = 0.3
public static let medium4: Double = 0.4
⋮----
func pegasusNumericMetric<V: Equatable>(value: V) -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Design/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct SourceChip: View {
let source: String
⋮----
var body: some View {
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Layout/SupplierSectionHeader.swift
```swift
struct SupplierSectionHeader: View {
let title: String
var subtitle: String? = nil
⋮----
var body: some View {
⋮----
struct SupplierStatusBadge: View {
let text: String
var tint: Color? = nil
⋮----
private var resolvedTint: Color {
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Models/FoundrySandboxModels.swift
```swift
public struct ScenarioPatchRequest: Codable, Sendable {
public let cellId: String
public let supplierId: String
public let objectType: String
public let objectId: String
public let operation: String
public let patchProperties: [String: String]
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct ScenarioWatermarkConflict: Codable, Sendable, Identifiable {
public var id: String { "\(objectType):\(objectId)" }
public let cellId: String
public let objectType: String
public let objectId: String
public let baseVersion: Int64
public let liveVersion: Int64
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct ScenarioCommitResult: Codable, Sendable {
public let scenarioId: String
public let committedObjects: Int
public let conflicts: [ScenarioWatermarkConflict]
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct LotRecallRequest: Codable, Sendable {
public let cellId: String
public let supplierId: String
public let rawMaterialLotId: String
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct RecallBlastRadiusReport: Codable, Sendable, Identifiable {
public var id: String { contaminatedRawMaterialLot }
public let cellId: String
public let supplierId: String
public let contaminatedRawMaterialLot: String
public let affectedWorkOrders: [String]
public let affectedFinishedLots: [String]
public let quarantinedSsccPallets: [String]
public let quarantinedWarehouseTotes: [String]
public let inTransitDriverDropBlocks: [String]
public let impactedSoliqFacturas: [String]
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct ActionExecutionRequest: Codable, Sendable {
public let actionName: String
public let actionType: String
public let scenarioId: String?
public let actorId: String
public let actorRole: String
public let targetObjectType: String
public let targetObjectId: String
public let idempotencyKey: String
public let cellId: String
public let supplierId: String
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct ActionExecutionRecord: Codable, Sendable, Identifiable {
public var id: String { executionId }
public let executionId: String
public let actionName: String
public let actionType: String
public let scenarioId: String?
public let actorId: String
public let actorRole: String
public let targetObjectType: String
public let targetObjectId: String
public let idempotencyKey: String
public let executedAt: String
public let status: String
public let cellId: String
public let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/supplier-app-ios/SupplierApp/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.supplier.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/supplier-app-ios/SupplierApp/Services/PushNotificationManager.swift
```swift
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
static let shared = PushNotificationManager()
⋮----
var isAuthorized = false
var deviceToken: String?
var errorMessage: String?
⋮----
private let api = APIClient.shared
private let storedTokenKey = "pegasus_push_token"
⋮----
private override init() {
⋮----
private var isRunningTests: Bool {
⋮----
func requestAuthorization() async {
⋮----
func didFailToRegisterForRemoteNotifications(error: Error) {
⋮----
func didReceiveFCMToken(_ token: String?) async {
⋮----
func uploadStoredTokenIfPossible() async {
⋮----
func uploadDeviceToken(_ token: String) async {
⋮----
func userNotificationCenter(
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierOperationsService.swift
```swift
enum SupplierOperationsService {
static func refreshToken(_ refreshToken: String) async throws -> LoginResponse {
⋮----
static func exceptions() async throws -> [SupplierExceptionRow] {
⋮----
static func shopClosedActive(limit: Int = 500, offset: Int = 0) async throws -> [ShopClosedAttemptRow] {
⋮----
static func complianceDashboard(limit: Int = 100) async throws -> ComplianceDashboardResponse {
⋮----
static func cashReconciliations() async throws -> CashReconciliationsResponse {
⋮----
static func acceptCashReconciliation(
⋮----
static func creditNotes() async throws -> CreditNotesResponse {
⋮----
static func issueCreditNote(id: String, idempotencyKey: String) async throws {
⋮----
static func creditProfiles(status: String? = nil, limit: Int = 100) async throws -> [CreditProfileRow] {
⋮----
static func patchRetailerCreditProfile(
⋮----
static func resolveException(
⋮----
static func routePerformance() async throws -> RoutePerformanceResponse {
⋮----
static func notificationPreferences() async throws -> NotificationPreferencesResponse {
⋮----
static func patchNotificationPreferences(_ preferences: [NotificationPreferenceRow]) async throws {
struct Body: Encodable {
let preferences: [NotificationPreferenceRow]
⋮----
static func negotiationsPending(limit: Int = 500, offset: Int = 0) async throws -> [NegotiationProposalRow] {
⋮----
static func resolveShopClosed(
⋮----
static func resolveNegotiation(
⋮----
static func manifests() async throws -> [SupplierManifestRow] {
⋮----
static func manifestDetail(_ manifestId: String) async throws -> SupplierManifestDetail {
⋮----
static func startManifestLoading(_ manifestId: String, idempotencyKey: String) async throws {
⋮----
static func injectManifestOrder(
⋮----
static func sealManifest(_ manifestId: String, idempotencyKey: String) async throws {
⋮----
static func manifestExceptions(escalatedOnly: Bool = false) async throws -> [SupplierManifestExceptionRow] {
⋮----
static func dispatchPreview(warehouseId: String? = nil) async throws -> SupplierDispatchPreview {
⋮----
static func createDispatchPreview(body: [String: String]) async throws -> SupplierDispatchPreview {
⋮----
static func executeDispatch(
⋮----
static func pricingRules() async throws -> SupplierPricingRule {
⋮----
static func listRetailerPriceOverrides(
⋮----
static func createRetailerPriceOverride(
⋮----
static func deleteRetailerPriceOverride(overrideId: String) async throws {
⋮----
static func topology() async throws -> SupplierTopologyResponse {
⋮----
static func updateTopology(_ request: SupplierTopologyUpdateRequest) async throws -> SupplierTopologyResponse {
⋮----
static func supplyLanes() async throws -> [SupplierSupplyLaneRow] {
⋮----
static func activity() async throws -> [SupplierActivityEvent] {
⋮----
static func fleetOrders() async throws -> [SupplierFleetOrderRow] {
⋮----
static func fleetLiveMap() async throws -> SupplierFleetLiveMapResponse {
⋮----
static func meiNetworkSummary() async throws -> SupplierMEIONetworkSummary {
⋮----
static func controlTowerZoneOverrides() async throws -> [ControlTowerZoneOverride] {
⋮----
static func planningSAndOP() async throws -> PlanningSAndOPSnapshot {
⋮----
static func runPlanningScenario(
⋮----
static func crmRetailers() async throws -> [SupplierCRMRetailer] {
⋮----
static func crmRetailer(_ retailerId: String) async throws -> SupplierCRMRetailerDetail {
⋮----
static func networkMode() async throws -> NetworkModeResponse {
⋮----
static func putNetworkMode(_ body: NetworkModeUpdateRequest, idempotencyKey: String) async throws -> NetworkModeUpdateResponse {
⋮----
static func planningPullMatrix(idempotencyKey: String) async throws -> PullMatrixResponse {
struct EmptyBody: Encodable {}
⋮----
static func planningSparsity(retailerId: String) async throws -> SparsityGateResult {
⋮----
static func planningPredictivePush(idempotencyKey: String) async throws -> PredictivePushResponse {
struct EmptyBody: Encodable {}
⋮----
static func loyaltyProgram() async throws -> LoyaltyProgram {
⋮----
static func patchLoyaltyProgram(_ body: LoyaltyProgram, idempotencyKey: String) async throws -> LoyaltyProgram {
⋮----
static func resolveEntity(_ body: EntityResolutionResolveRequest) async throws -> EntityResolutionResolveResponse {
⋮----
static func explainEntity(_ body: EntityResolutionExplainRequest) async throws -> EntityResolutionExplainResponse {
⋮----
static func planningKillSwitch(_ body: KillSwitchRequest, idempotencyKey: String) async throws -> KillSwitchResponse {
⋮----
static func payoutRail() async throws -> PayoutRailInfo {
⋮----
static func payoutPolicy() async throws -> SupplierPayoutPolicy {
⋮----
static func patchPayoutPolicy(_ body: SupplierPayoutPolicyPatch) async throws -> SupplierPayoutPolicy {
⋮----
static func scoredExceptions(limit: Int = 50) async throws -> [ScoredException] {
⋮----
static func playbooks() async throws -> [ControlTowerPlaybook] {
⋮----
static func payoutBatches() async throws -> [PayoutBatch] {
⋮----
static func generatePayoutBatch(_ body: PayoutBatchGenerateRequest, idempotencyKey: String) async throws -> PayoutBatchGenerateResponse {
⋮----
static func exportPayoutBatch(_ batchId: String, idempotencyKey: String) async throws -> String {
⋮----
static func markPayoutBatchPaid(_ batchId: String, idempotencyKey: String) async throws -> PayoutMarkPaidResponse {
⋮----
static func dispatchPayoutBatch(_ batchId: String, live: Bool, idempotencyKey: String) async throws -> PayoutDispatchResponse {
⋮----
static func seasonalOverrides() async throws -> SeasonalTemplatesResponse {
⋮----
static func createSeasonalOverride(_ request: SeasonalOverrideInput, idempotencyKey: String) async throws -> SeasonalOverrideRow {
⋮----
static func returnPolicy() async throws -> SupplierReturnPolicy {
⋮----
static func putReturnPolicy(_ body: SupplierReturnPolicy, idempotencyKey: String) async throws -> SupplierReturnPolicy {
⋮----
static func knowledgeGraph() async throws -> SupplierKnowledgeGraph {
⋮----
static func replenishmentPolicies() async throws -> SupplierReplenishmentPolicy {
⋮----
static func createControlTowerZoneOverride(
⋮----
static func wsSession() async throws -> SupplierWsSessionResponse {
⋮----
static func orders(
⋮----
static func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
⋮----
static func applyReassign(
⋮----
static func returns(status: String = "PENDING", limit: Int = 100, offset: Int = 0) async throws -> SupplierReturnsResponse {
⋮----
static func resolveReturn(
⋮----
static func paymentLedger(currency: String? = nil) async throws -> PaymentLedgerResponse {
⋮----
static func paymentSettlementAuthority() async throws -> SettlementAuthorityResponse {
⋮----
static func paymentReconciliationMismatches() async throws -> ReconciliationMismatchResponse {
⋮----
static func orgMembers() async throws -> [SupplierOrgMember] {
⋮----
static func createOrgMember(_ request: SupplierOrgMemberCreateRequest, idempotencyKey: String) async throws -> [SupplierOrgMember] {
⋮----
static func demandHistory() async throws -> SupplierDemandHistoryResponse {
⋮----
static func pulse() async throws -> SupplierPulseResponse {
⋮----
static func simulatePromotionPandL(_ request: PromoSimulateInput) async throws -> PromoSimulateResult {
⋮----
static func updateOrgMember(
⋮----
static func deactivateOrgMember(_ userId: String, idempotencyKey: String) async throws -> [SupplierOrgMember] {
⋮----
static func triggerReplenishment(idempotencyKey: String) async throws -> SupplierReplenishmentTriggerResponse {
⋮----
static func createFleetDriver(_ request: FleetDriverCreateRequest, idempotencyKey: String) async throws -> FleetDriversResponse {
⋮----
static func createFleetVehicle(_ request: FleetVehicleCreateRequest, idempotencyKey: String) async throws -> FleetVehiclesResponse {
⋮----
static func updateProfile(_ request: SupplierProfileUpdateRequest) async throws -> SupplierProfile {
⋮----
static func updateInventory(_ request: InventoryPatchRequest) async throws {
⋮----
static func importInventoryCSV(_ csv: String, idempotencyKey: String) async throws -> SupplierInventoryImportResult {
⋮----
static func createImportSession(fileName: String, fileSizeBytes: Int, idempotencyKey: String) async throws -> SupplierImportSessionCreateResponse {
struct CreateBody: Encodable {
let fileName: String
let fileSizeBytes: Int
enum CodingKeys: String, CodingKey {
⋮----
static func ingestImportSession(sessionId: String, csv: String, idempotencyKey: String) async throws -> SupplierImportIngestResponse {
⋮----
static func getImportSession(sessionId: String) async throws -> SupplierImportSession {
⋮----
static func getImportMapping(sessionId: String) async throws -> SupplierImportMappingResponse {
⋮----
static func approveImportSession(sessionId: String, idempotencyKey: String) async throws {
struct ApproveResponse: Decodable {
let sessionId: String
enum CodingKeys: String, CodingKey { case sessionId = "session_id" }
⋮----
static func applyImportSession(sessionId: String, idempotencyKey: String) async throws -> SupplierImportApplyResponse {
⋮----
static func recordChargeback(_ request: PaymentChargebackRequest, idempotencyKey: String) async throws -> PaymentChargebackResponse {
⋮----
static func recordChargebackReversal(_ request: PaymentChargebackReversalRequest, idempotencyKey: String) async throws -> PaymentChargebackReversalResponse {
⋮----
static func listSupplierClaims(status: String? = "OPEN", limit: Int = 50) async throws -> [SupplierClaim] {
⋮----
static func approveClaim(claimId: String, request: ApproveClaimRequest, idempotencyKey: String) async throws -> ApproveClaimResponse {
⋮----
static func rejectClaim(claimId: String, request: RejectClaimRequest, idempotencyKey: String) async throws -> SupplierClaim {
⋮----
static func listClaimChargebacks(limit: Int = 100, orderId: String? = nil) async throws -> ClaimChargebacksResponse {
⋮----
static func vetOrder(body: [String: String], idempotencyKey: String) async throws {
⋮----
static func aiRecommendations(status: String? = nil, limit: Int = 50) async throws -> SupplierAIRecommendationsResponse {
⋮----
static func recordAiRecommendationDecision(
⋮----
static func analyticsVelocity() async throws -> SupplierAnalyticsVelocityResponse {
⋮----
static func analyticsRevenue() async throws -> SupplierAnalyticsRevenueResponse {
⋮----
static func demandToday() async throws -> SupplierDemandSummaryResponse {
⋮----
static func issuePaymentBypass(
⋮----
static func approveEarlyComplete(driverId: String) async throws {
⋮----
static func empathyAdoption() async throws -> SupplierEmpathyAdoption {
⋮----
static func broadcast(
⋮----
static func exceptionMap(windowHours: Int = 24) async throws -> ExceptionMapResponse {
⋮----
static func previewRetailerPriceOverride(
⋮----
static func getWarehouseOrder(orderId: String, warehouseId: String) async throws -> WarehouseOrderDetail {
⋮----
static func proposeWarehouseOrder(
⋮----
static func rejectWarehouseOrder(
⋮----
/// Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
enum SupplierIdempotency {
static func resolveReturn(returnId: String, resolution: String) -> String {
⋮----
static func dispatch(
⋮----
static func broadcast(scopeId: String, role: String, title: String, body: String) -> String {
⋮----
static func paymentBypass(orderId: String, reason: String) -> String {
⋮----
private static func stableHash(_ input: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierService.swift
```swift
enum SupplierService {
static func login(phone: String, password: String) async throws -> LoginResponse {
⋮----
static func register(body: [String: Any]) async throws -> RegisterResponse {
⋮----
static func setupBusiness(_ request: BusinessSetupRequest) async throws -> BusinessSetupResponse {
⋮----
static func configure(body: [String: String]) async throws {
⋮----
static func updateInventory(_ request: InventoryPatchRequest) async throws {
⋮----
static func getDemandHistory() async throws -> SupplierDemandHistoryResponse {
⋮----
static func importInventoryCSV(_ csv: String, idempotencyKey: String) async throws -> SupplierInventoryImportResult {
⋮----
static func recordChargeback(_ request: PaymentChargebackRequest, idempotencyKey: String) async throws -> PaymentChargebackResponse {
⋮----
static func recordChargebackReversal(_ request: PaymentChargebackReversalRequest, idempotencyKey: String) async throws -> PaymentChargebackReversalResponse {
⋮----
static func dashboard() async throws -> SupplierDashboard {
⋮----
static func profile() async throws -> SupplierProfile {
⋮----
static func orders(
⋮----
static func fleetDrivers() async throws -> [FleetDriver] {
⋮----
static func fleetVehicles() async throws -> [FleetVehicle] {
⋮----
static func inventory() async throws -> [InventoryItem] {
⋮----
static func catalogProducts() async throws -> [CatalogProduct] {
⋮----
static func catalogCategories(supplierId: String? = nil) async throws -> [CatalogCategory] {
⋮----
static func catalogUploadTicket(ext: String) async throws -> CatalogUploadTicket {
⋮----
static func uploadCatalogImage(data: Data, ext: String) async throws -> String {
⋮----
static func createCatalogProduct(_ request: CatalogProductCreateRequest, idempotencyKey: String) async throws -> CatalogProduct {
⋮----
static func updateCatalogProduct(
⋮----
private static func mimeType(for ext: String) -> String {
⋮----
static func promotions() async throws -> [SupplierPromotion] {
⋮----
static func createPromotion(_ request: SupplierPromotionUpsertRequest, idempotencyKey: String) async throws -> SupplierPromotion {
⋮----
static func updatePromotion(
⋮----
static func deactivatePromotion(promotionId: String, idempotencyKey: String) async throws {
⋮----
static func earnings() async throws -> SupplierEarnings {
⋮----
static func configureBilling(_ request: BillingSetupRequest) async throws -> BillingSetupResponse {
⋮----
static func updatePricingRules(_ request: SupplierPricingRuleUpdateRequest) async throws -> SupplierPricingRule {
⋮----
static func updateTopology(body: [String: String]) async throws -> SupplierTopologyResponse {
```

## File: apps/supplier-app-ios/SupplierApp/Services/TokenStore.swift
```swift
final class TokenStore {
static let shared = TokenStore()
⋮----
private(set) var token: String?
private(set) var refreshToken: String?
private(set) var supplierId: String?
private(set) var isConfigured: Bool = false
private(set) var isRegistered: Bool = false
⋮----
private(set) var billingGateDismissed: Bool = false
⋮----
var isAuthenticated: Bool { token != nil }
⋮----
var needsBillingGate: Bool {
⋮----
var needsBusinessSetup: Bool {
⋮----
private let service = "com.pegasusx.supplier"
⋮----
private init() {
⋮----
func store(auth: LoginResponse) {
⋮----
func storeRegister(_ auth: RegisterResponse) {
⋮----
func updateTokens(token: String, refresh: String?) {
⋮----
func markRegistered(_ registered: Bool) {
⋮----
func markConfigured(_ configured: Bool) {
⋮----
func dismissBillingGate() {
⋮----
func showBillingGate() {
⋮----
func clear() {
⋮----
private func writeKeychain(account: String, value: String) {
⋮----
private func readKeychain(account: String) -> String? {
⋮----
private func deleteKeychain(account: String) {
```

## File: apps/supplier-app-ios/SupplierApp/supplierapp/Services/AutoUpdater.swift
```swift
final class AutoUpdater {
static let shared = AutoUpdater()
⋮----
struct Manifest: Sendable {
let versionCode: Int
let versionName: String
let manifestURL: URL
let notes: String
⋮----
struct UpdateState: Sendable {
let available: Bool
let force: Bool
let deferred: Bool
let message: String?
let manifest: Manifest?
let manifestFailed: Bool
⋮----
private(set) var lastManifest: Manifest?
⋮----
private init() {}
⋮----
static func failedPolicy(message: String, hadBanner: Bool) -> (clearBanner: Bool, force: Bool, notice: String) {
⋮----
func evaluate(
⋮----
func promptInstall(manifest: Manifest? = nil, force: Bool = false) -> Bool {
⋮----
func triggerOTAInstall(manifestURL: URL) {
⋮----
func openStoreListing(force: Bool = false) -> Bool {
⋮----
private func fetchManifest(from urlString: String) async -> Manifest? {
⋮----
enum AppBuildVersion {
/// A missing or non-numeric build is unknown. It is not build 0.
static func code(_ raw: String?) -> Int? {
⋮----
static func manifestIsNewer(_ manifestCode: Int, than current: Int?) -> Bool {
⋮----
static func marketingVersion(_ raw: String?) -> String? {
```

## File: apps/supplier-app-ios/SupplierApp/supplierapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/supplier-app-ios/SupplierApp/Theme/SupplierTheme.swift
```swift
static var pressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
⋮----
struct SupplierCardModifier: ViewModifier {
func body(content: Content) -> some View {
⋮----
func supplierCard() -> some View {
⋮----
func supplierReadableWidth() -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Utilities/ForecastConfidenceSupport.swift
```swift
enum ForecastConfidenceSupport {
private static let staleMinutes: Int64 = 30
private static let isoFormatter = ISO8601DateFormatter()
⋮----
static func isForecastStale(generatedAt: String?) -> Bool {
⋮----
static func formatForecastUpdatedAt(generatedAt: String?) -> String? {
⋮----
static func mapBaselineSource(_ src: String?) -> String? {
⋮----
static func formatBaselineSourceLabel(_ src: String?) -> String {
⋮----
static func fromDemand(_ summary: SupplierDemandSummaryResponse) -> ForecastConfidence {
⋮----
enum PlanBrainTab: String, CaseIterable {
⋮----
func planBrainTabFromQuery(_ raw: String?) -> PlanBrainTab {
⋮----
func brainForecastLine(confidence: ForecastConfidence?, accuracyPoints: [Double]) -> [Double]? {
⋮----
func factoryPlanningDisabledCode(status: Int, body: String) -> String? {
⋮----
var isBlocked: Bool {
⋮----
var confidenceTint: Color {
```

## File: apps/supplier-app-ios/SupplierApp/Utilities/SupplierIdempotencyKeys.swift
```swift
enum SupplierIdempotencyKeys {
⋮----
static func supplierScopeId() -> String {
⋮----
static func importDirect(scopeId: String, csvBody: String) -> String {
⋮----
static func importCreate(scopeId: String, fileName: String, fileSizeBytes: Int) -> String {
⋮----
static func importIngest(sessionId: String, csvBody: String) -> String {
⋮----
static func importApprove(sessionId: String) -> String {
⋮----
static func importApply(sessionId: String) -> String {
⋮----
static func orgMemberCreate(scopeId: String, phone: String) -> String {
⋮----
static func orgMemberUpdate(scopeId: String, userId: String, revision: String) -> String {
⋮----
static func orgMemberDeactivate(scopeId: String, userId: String) -> String {
⋮----
static func fleetDriverCreate(scopeId: String, phone: String) -> String {
⋮----
static func fleetVehicleCreate(scopeId: String, licensePlate: String) -> String {
⋮----
static func chargeback(orderId: String, reason: String) -> String {
⋮----
static func chargebackReversal(chargebackId: String, reason: String) -> String {
⋮----
static func profileUpdate(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func businessSetup(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func billingSetup(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func supplierConfigure(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func topologyPut(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func payoutPolicyPatch(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func exceptionResolve(scopeId: String, kind: String, id: String, creditNoteId: String) -> String {
⋮----
static func notificationsMarkAll(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func notificationPreferencesPatch(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func creditAdminDisable(mode: String, supplierId: String, retailerId: String, ticketId: String) -> String {
⋮----
static func encodedPayload<T: Encodable>(_ value: T) -> String {
⋮----
static func pricingRulePatch(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func inventoryAdjust(scopeId: String, skuId: String, quantityDelta: Int64, version: Int64) -> String {
⋮----
static func retailerPriceOverrideCreate(scopeId: String, retailerId: String, productId: String, priceMinor: Int64) -> String {
⋮----
static func retailerPriceOverrideDelete(scopeId: String, overrideId: String) -> String {
⋮----
static func promotionCreate(scopeId: String, payloadFingerprint: String) -> String {
⋮----
static func promotionUpdate(scopeId: String, promotionId: String, payloadFingerprint: String) -> String {
⋮----
static func promotionDeactivate(scopeId: String, promotionId: String) -> String {
⋮----
static func planningScenario(scopeId: String, factoryDowntimeHours: Int, demandDeltaPct: Double) -> String {
⋮----
static func seasonalOverrideCreate(scopeId: String, startDate: String, endDate: String) -> String {
⋮----
static func networkModePut(scopeId: String, mode: String) -> String {
⋮----
static func planningPullMatrix(scopeId: String) -> String {
⋮----
static func planningPredictivePush(scopeId: String) -> String {
⋮----
static func loyaltyProgramPatch(scopeId: String, reason: String) -> String {
⋮----
static func planningKillSwitch(scopeId: String, reason: String) -> String {
⋮----
static func catalogCreate(scopeId: String, name: String, priceMinor: Int64, barcode: String?) -> String {
⋮----
static func catalogUpdate(productId: String, priceMinor: Int64, unitVolumeVu: Double, barcode: String?) -> String {
⋮----
static func claimApprove(claimId: String, settlementMode: String) -> String {
⋮----
static func claimReject(claimId: String) -> String {
⋮----
static func deviceToken(platform: String, token: String) -> String {
⋮----
static func payoutExport(batchId: String) -> String {
⋮----
static func payoutMarkPaid(batchId: String) -> String {
⋮----
static func payoutDispatch(batchId: String, live: Bool) -> String {
⋮----
static func replenishmentTrigger(scopeId: String, day: String) -> String {
⋮----
static func utcDay(_ date: Date) -> String {
⋮----
static func payoutGenerate(scopeId: String, periodStart: String, periodEnd: String) -> String {
⋮----
static func returnPolicyPut(scopeId: String, hours: Int64) -> String {
⋮----
static func controlTowerZoneOverride(scopeId: String, action: String, polygonFingerprint: String) -> String {
⋮----
private static func stableHash(_ input: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Utilities/SupplierPortalLinks.swift
```swift
enum SupplierPortalLinks {
⋮----
static var baseURL: URL {
⋮----
static func url(for feature: SupplierPortalFeature) -> URL {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/InventoryViewModel.swift
```swift
final class InventoryViewModel {
var items: [InventoryItem] = []
var loading = true
private var loadGeneration = 0
var error: String?
var query = ""
var adjustingSku: String?
⋮----
var filtered: [InventoryItem] {
⋮----
func load(silent: Bool = false) async {
⋮----
func adjustQuantity(sku: String, delta: Int64) async {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/OnboardingViewModel.swift
```swift
enum RegisterStep: Int, CaseIterable {
⋮----
var title: String {
⋮----
struct RegisterCountry: Identifiable {
let id: String
let name: String
let dialCode: String
let currency: String
⋮----
final class OnboardingViewModel {
static let countries: [RegisterCountry] = [
⋮----
var step: RegisterStep = .identity
var countryCode = "UZ"
var phoneLocal = ""
var otpCode = ""
var legalName = ""
var contactName = ""
var email = ""
var password = ""
⋮----
var taxId = ""
var registrationNumber = ""
var headquartersAddress = ""
var city = ""
var postalCode = ""
⋮----
var loading = false
var error: String?
⋮----
var dialCode: String {
⋮----
var fullPhone: String { "\(dialCode)\(phoneLocal)" }
⋮----
func advanceStep() -> Bool {
⋮----
func retreatStep() {
⋮----
func register(tokenStore: TokenStore) async -> Bool {
⋮----
func setupBusiness(tokenStore: TokenStore) async -> Bool {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/TreasuryViewModel.swift
```swift
final class TreasuryViewModel {
var earnings: SupplierEarnings?
var ledgerCount = 0
var mismatchCount = 0
var settlementRows = 0
var monthLabel = "—"
var loading = true
private var loadGeneration = 0
var error: String?
⋮----
func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Activity/ActivityView.swift
```swift
struct ActivityView: View {
@State private var viewModel = ActivityViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/AI/AIRecommendationsView.swift
```swift
struct AIRecommendationsView: View {
private let statusFilters = ["PENDING", "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "ALL"]
private let decisions = ["ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED"]
⋮----
@State private var viewModel = AIRecommendationsViewModel()
⋮----
var body: some View {
⋮----
private func recommendationCard(_ rec: SupplierAIRecommendation) -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Analytics/AnalyticsView.swift
```swift
struct AnalyticsView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var loading = true
@State private var loadGeneration = 0
@State private var error: String?
@State private var hasSnapshot = false
@State private var pendingOrders: Int?
@State private var inventorySKUs: Int?
@State private var revenueLabel = "—"
@State private var predictionCount = 0
@State private var forecastUnits = 0
@State private var velocityCreated = 0
@State private var demandGeneratedAt: String?
@State private var demandConfidence: ForecastConfidence?
⋮----
private var gridMin: CGFloat {
⋮----
var body: some View {
⋮----
private func load(silent: Bool = false) async {
⋮----
private struct PlanningBrainSection: View {
@State private var sandop: PlanningSAndOPSnapshot?
@State private var scenario: PlanningScenarioResult?
@State private var downtimeHours = 8.0
@State private var demandDeltaPct = 10.0
@State private var loading = true
@State private var loadGeneration = 0
@State private var running = false
@State private var error: String?
⋮----
var body: some View {
⋮----
private func loadSandop(silent: Bool = false) async {
⋮----
private func runScenario() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Analytics/DemandHistoryView.swift
```swift
struct DemandHistoryView: View {
@State private var history: SupplierDemandHistoryResponse?
@State private var demandConfidence: ForecastConfidence?
@State private var demandGeneratedAt: String?
@State private var loading = true
@State private var loadGeneration = 0
@State private var error: String?
⋮----
var body: some View {
⋮----
private func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Auth/BusinessSetupView.swift
```swift
struct BusinessSetupView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var vm = OnboardingViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Auth/RegisterView.swift
```swift
struct RegisterView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(\.dismiss) private var dismiss
@State private var vm = OnboardingViewModel()
⋮----
var body: some View {
⋮----
private var identityStep: some View {
⋮----
private var verificationStep: some View {
⋮----
private var profileStep: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogBarcodeField.swift
```swift
struct CatalogBarcodeField: View {
@Binding var value: String
var enabled: Bool = true
⋮----
@State private var showScanner = false
@State private var scannerEnabled = true
@State private var validationMessage: String?
⋮----
private var normalized: String? {
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogCreateSheet.swift
```swift
struct CatalogCreateSheet: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = CatalogCreateViewModel()
⋮----
var onCreated: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogDetailView.swift
```swift
struct CatalogDetailView: View {
let productId: String?
@State private var vm = CatalogDetailViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogView.swift
```swift
struct CatalogView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = CatalogViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
⋮----
struct SupplierRefreshNotice: View {
let message: String
let dismiss: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Components/SupplierStateViews.swift
```swift
enum MoneyFormat {
static func minor(_ amount: Int64, currency: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Dashboard/NetworkPulseStrip.swift
```swift
struct NetworkPulseStrip: View {
let events: [SupplierPulseEvent]
let loading: Bool
var error: String? = nil
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Dispatch/DispatchPreviewView.swift
```swift
struct DispatchPreviewView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
var embeddedInHub: Bool = false
@State private var vm = DispatchPreviewViewModel()
⋮----
var body: some View {
⋮----
private func manualAssignmentSection(preview: SupplierDispatchPreview) -> some View {
⋮----
private struct DispatchKpiCard: View {
let title: String
let value: String
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Earnings/EarningsView.swift
```swift
struct EarningsView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var viewModel = EarningsViewModel()
⋮----
var body: some View {
⋮----
private func earningsLabel(_ minor: Int64, _ earnings: SupplierEarnings) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/ClaimsView.swift
```swift
struct ClaimsView: View {
@State private var viewModel = ClaimsViewModel()
⋮----
private let settlementModes: [(String, String)] = [
⋮----
var body: some View {
⋮----
static func claimAmountLabel(_ amount: Int64?, currency: String?) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/EarlyCompleteView.swift
```swift
struct EarlyCompleteView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = EarlyCompleteViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/ExceptionsView.swift
```swift
struct ExceptionsView: View {
@State private var viewModel = ExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/NegotiationsView.swift
```swift
struct NegotiationsView: View {
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Exceptions/ShopClosedView.swift
```swift
struct ShopClosedView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = ShopClosedViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Fleet/FleetOrdersView.swift
```swift
struct FleetOrdersView: View {
@State private var viewModel = FleetOrdersViewModel()
⋮----
var body: some View {
⋮----
static func formatOrderTotal(totalMinor: Int64?, currency: String?) -> String? {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Fleet/FleetView.swift
```swift
struct FleetView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = FleetViewModel()
⋮----
var body: some View {
⋮----
private func fleetList<Rows: View>(
```

## File: apps/supplier-app-ios/SupplierApp/Views/Inventory/InventoryList.swift
```swift
struct InventoryList: View {
let items: [InventoryItem]
let adjustingSku: String?
let onAdjustQuantity: (String, Int64) async -> Void
⋮----
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
⋮----
var body: some View {
⋮----
private func quantityCell(_ item: InventoryItem) -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Inventory/InventoryView.swift
```swift
struct InventoryView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = InventoryViewModel()
⋮----
var body: some View {
⋮----
private func loadSilent() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Manifests/ManifestDetailView.swift
```swift
struct ManifestDetailView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
let manifestId: String
@State private var viewModel = ManifestDetailViewModel()
⋮----
var body: some View {
⋮----
private func manifestContent(_ detail: SupplierManifestDetail, @Bindable vm: ManifestDetailViewModel) -> some View {
⋮----
static func connectionRestoredNotice() -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Manifests/ManifestExceptionsView.swift
```swift
struct ManifestExceptionsView: View {
@State private var viewModel = ManifestExceptionsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Manifests/ManifestsView.swift
```swift
struct ManifestsView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = ManifestsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/DeliveryZonesList.swift
```swift
struct DeliveryZonesList: View {
let warehouses: [SupplierTopologyWarehouse]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/DeliveryZonesView.swift
```swift
struct DeliveryZonesView: View {
@State private var vm = TopologyNodesListViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/FactoriesView.swift
```swift
struct FactoriesView: View {
@State private var vm = TopologyNodesListViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/GeoReportLanesList.swift
```swift
struct GeoReportLanesList: View {
let lanes: [SupplierSupplyLaneRow]
⋮----
private var totalCells: Int { lanes.reduce(0) { $0 + $1.h3Cells } }
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/GeoReportView.swift
```swift
struct GeoReportView: View {
@State private var vm = SupplyLanesViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/SupplyLanesList.swift
```swift
struct SupplyLanesList: View {
let lanes: [SupplierSupplyLaneRow]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/SupplyLanesView.swift
```swift
struct SupplyLanesView: View {
@State private var vm = SupplyLanesViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/TopologyNodeForms.swift
```swift
enum TopologyMutation {
static var defaultLat: Double { packMapCoordinate().lat }
static var defaultLng: Double { packMapCoordinate().lng }
⋮----
static func warehouseInput(
⋮----
static func warehouseInput(from node: SupplierTopologyWarehouse) -> SupplierTopologyWarehouseInput {
⋮----
static func factoryInput(
⋮----
static func factoryInput(from node: SupplierTopologyFactory) -> SupplierTopologyFactoryInput {
⋮----
static func appendWarehouse(
⋮----
static func appendFactory(name: String, location: AddressLocationValue) async throws -> SupplierTopologyResponse {
⋮----
final class TopologyNodeFormViewModel {
var name = ""
var location = AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng)
var coverageKm = "50"
var saving = false
var error: String?
⋮----
var canSave: Bool {
⋮----
func saveWarehouse() async -> Bool {
⋮----
func saveFactory() async -> Bool {
⋮----
struct AddWarehouseSheet: View {
@Environment(\.dismiss) private var dismiss
var onSaved: () -> Void
@State private var vm = TopologyNodeFormViewModel()
⋮----
var body: some View {
⋮----
struct AddFactorySheet: View {
@Environment(\.dismiss) private var dismiss
var onSaved: () -> Void
@State private var vm = TopologyNodeFormViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/TopologyView.swift
```swift
struct TopologyView: View {
@State private var vm = TopologyViewModel()
⋮----
var body: some View {
⋮----
private var readOnlyList: some View {
⋮----
private var editForm: some View {
⋮----
private struct TopologyNodeRow: View {
let name: String
let address: String
let meta: String
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/WarehouseList.swift
```swift
struct WarehouseList: View {
let warehouses: [SupplierTopologyWarehouse]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Network/WarehousesView.swift
```swift
struct WarehousesView: View {
@State private var vm = TopologyNodesListViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Notifications/NotificationInboxView.swift
```swift
struct NotificationInboxView: View {
@State private var vm = NotificationInboxViewModel()
⋮----
var body: some View {
⋮----
struct SupplierNotificationItem: Decodable, Identifiable, Sendable {
let id: String
let type: String
let title: String
let body: String
let payload: String
let channel: String
let readAt: String?
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Operations/OperationsView.swift
```swift
struct OperationsView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var vm = OperationsViewModel()
⋮----
var body: some View {
⋮----
private var operationsForm: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Operations/OperatorBroadcastView.swift
```swift
struct OperatorBroadcastView: View {
let broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"]
⋮----
@Binding var title: String
@Binding var bodyText: String
@Binding var broadcastRole: String
@Binding var templateDate: String
let broadcasting: Bool
let sendBroadcast: () -> Void
⋮----
var body: some View {
⋮----
private func applyTemplate(_ template: SupplierBroadcastTemplate) {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Operations/PaymentBypassView.swift
```swift
struct PaymentBypassView: View {
@Binding var orderId: String
@Binding var bypassReason: String
let bypassToken: String?
let bypassing: Bool
@Binding var showBypassConfirm: Bool
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Operations/ReplenishmentActionView.swift
```swift
struct ReplenishmentActionView: View {
let replenishing: Bool
let runReplenishment: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Orders/OrdersHubView.swift
```swift
private enum OrdersHubSurface: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
struct OrdersHubView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
var initialCommandStatus: String? = nil
@State private var surface: OrdersHubSurface = .queue
⋮----
var body: some View {
⋮----
private var hubChrome: some View {
⋮----
struct OrdersQueueView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(SupplierRealtimeHub.self) private var realtimeHub
var embeddedInHub: Bool = false
var initialCommandStatus: String? = nil
@State private var vm = OrdersViewModel()
⋮----
var body: some View {
⋮----
private var phoneContent: some View {
⋮----
private var splitContent: some View {
⋮----
private var filterTabs: some View {
⋮----
private var ordersList: some View {
⋮----
private var ordersToolbar: some ToolbarContent {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Orders/OrdersList.swift
```swift
struct OrdersList: View {
@Bindable var vm: OrdersViewModel
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Orders/OrdersView.swift
```swift
struct OrdersView: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = OrdersViewModel()
⋮----
var body: some View {
⋮----
private var phoneContent: some View {
⋮----
private var splitContent: some View {
⋮----
private var filterTabs: some View {
⋮----
private var ordersToolbar: some ToolbarContent {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/CreateDriverSheet.swift
```swift
struct CreateDriverSheet: View {
let topology: SupplierTopologyResponse
let vehicles: [FleetVehicle]
let onDone: () -> Void
@Environment(\.dismiss) private var dismiss
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var name = ""
@State private var phone = ""
@State private var pin = ""
@State private var nodeType = "WAREHOUSE"
@State private var nodeId = ""
@State private var vehicleId = ""
@State private var busy = false
@State private var error: String?
⋮----
var body: some View {
⋮----
private var nodeOptions: [(String, String)] {
⋮----
private var filteredVehicles: [FleetVehicle] {
⋮----
private func create() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/CreateOrgMemberSheet.swift
```swift
struct CreateOrgMemberSheet: View {
let topology: SupplierTopologyResponse
let onDone: () -> Void
@Environment(\.dismiss) private var dismiss
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var name = ""
@State private var email = ""
@State private var phone = ""
@State private var password = ""
@State private var role = "WAREHOUSE_ADMIN"
@State private var nodeType = "WAREHOUSE"
@State private var nodeId = ""
@State private var busy = false
@State private var error: String?
⋮----
var body: some View {
⋮----
private var nodeOptions: [(String, String)] {
⋮----
private func create() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/CreateVehicleSheet.swift
```swift
struct CreateVehicleSheet: View {
let topology: SupplierTopologyResponse
let onDone: () -> Void
@Environment(\.dismiss) private var dismiss
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var label = ""
@State private var plate = ""
@State private var nodeType = "WAREHOUSE"
@State private var nodeId = ""
@State private var busy = false
@State private var error: String?
⋮----
var body: some View {
⋮----
private var nodeOptions: [(String, String)] {
⋮----
private func create() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/DriverListView.swift
```swift
struct DriverListView: View {
let drivers: [FleetDriver]
let topology: SupplierTopologyResponse?
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/EditOrgMemberSheet.swift
```swift
struct EditOrgMemberSheet: View {
let member: SupplierOrgMember
let onDone: () -> Void
@Environment(\.dismiss) private var dismiss
@State private var name: String
@State private var role: String
@State private var isActive: Bool
@State private var busy = false
@State private var error: String?
⋮----
init(member: SupplierOrgMember, onDone: @escaping () -> Void) {
⋮----
var body: some View {
⋮----
private func save() async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/MetricsOverviewView.swift
```swift
struct MetricsOverviewView: View {
let driversCount: Int
let vehiclesCount: Int
let orgMembersCount: Int
let topology: SupplierTopologyResponse?
⋮----
var body: some View {
⋮----
private struct MetricCard: View {
let title: String
let count: Int
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/OrgFleetView.swift
```swift
struct OrgFleetView: View {
@State private var tab = 0
@State private var loading = true
@State private var loadGeneration = 0
@State private var error: String?
@State private var topology: SupplierTopologyResponse?
@State private var drivers: [FleetDriver] = []
@State private var vehicles: [FleetVehicle] = []
@State private var orgMembers: [SupplierOrgMember] = []
@State private var showDriverSheet = false
@State private var showVehicleSheet = false
@State private var showOrgSheet = false
@State private var showEditMemberSheet = false
@State private var editingMember: SupplierOrgMember?
@State private var memberActionId: String?
⋮----
var body: some View {
⋮----
private func reload(silent: Bool = false) async {
⋮----
private func deactivateMember(_ userId: String) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/OrgMemberListView.swift
```swift
struct OrgMemberListView: View {
let orgMembers: [SupplierOrgMember]
@Binding var editingMember: SupplierOrgMember?
@Binding var showEditMemberSheet: Bool
@Binding var memberActionId: String?
let deactivateAction: (String) async -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/OrgFleet/VehicleListView.swift
```swift
struct VehicleListView: View {
let vehicles: [FleetVehicle]
let topology: SupplierTopologyResponse?
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/CreateOverrideForm.swift
```swift
struct CreateOverrideForm: View {
let templates: [SeasonalBuiltinTemplate]
@Binding var templateId: String
@Binding var name: String
@Binding var startDate: String
@Binding var endDate: String
@Binding var multiplier: String
let formError: String?
let saving: Bool
let onSubmit: () -> Void
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/ForecastConfidenceView.swift
```swift
struct ForecastConfidenceView: View {
let confidence: ForecastConfidence
var updatedAt: String?
var stale: Bool = false
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/KnowledgeGraphView.swift
```swift
struct KnowledgeGraphView: View {
@State private var viewModel = KnowledgeGraphViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/PlanningBrainView.swift
```swift
struct PlanningBrainView: View {
@State private var viewModel = PlanningBrainViewModel()
⋮----
var body: some View {
⋮----
private func planningTab(@Bindable vm: PlanningBrainViewModel) -> some View {
⋮----
private func brainTab(@Bindable vm: PlanningBrainViewModel) -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/PlanningSettingsView.swift
```swift
struct PlanningSettingsView: View {
@State private var viewModel = PlanningSettingsViewModel()
⋮----
var body: some View {
⋮----
static func killSwitchReasonValidation(reason: String) -> String? {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/ReplenishmentPoliciesView.swift
```swift
struct ReplenishmentPoliciesView: View {
@State private var viewModel = ReplenishmentPoliciesViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Planning/SeasonalOverridesList.swift
```swift
struct SeasonalOverridesList: View {
let overrides: [SeasonalOverrideRow]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Pricing/PricingView.swift
```swift
struct PricingView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var products: [CatalogProduct] = []
@State private var loading = true
@State private var loadGeneration = 0
@State private var error: String?
@State private var query = ""
⋮----
private var filtered: [CatalogProduct] {
⋮----
var body: some View {
⋮----
static func formatPrice(_ product: CatalogProduct) -> String {
⋮----
private func load(silent: Bool = false) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Pricing/ProductPricingDetailView.swift
```swift
struct ProductPricingDetailView: View {
let product: CatalogProduct
var onUpdated: () -> Void
⋮----
@State private var priceMajor = ""
@State private var saleDiscountBps = ""
@State private var saleEnabled = false
@State private var saving = false
@State private var error: String?
@State private var activeSale: SupplierPromotion?
⋮----
var body: some View {
⋮----
private func bootstrap() async {
⋮----
private func save() async {
⋮----
static func parsePriceMinor(_ text: String) -> Int64? {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Pricing/RetailerOverridesView.swift
```swift
struct RetailerOverridesView: View {
@State private var overrides: [RetailerPriceOverride] = []
@State private var loading = true
@State private var loadGeneration = 0
@State private var error: String?
@State private var showCreate = false
@State private var retailerId = ""
@State private var productId = ""
@State private var price = ""
@State private var creating = false
@State private var preview: RetailerOverridePreview?
@State private var previewLoading = false
⋮----
var body: some View {
⋮----
private var createSheet: some View {
⋮----
private var previewKey: String {
⋮----
private func refreshPreview() async {
⋮----
private func load(silent: Bool = false) async {
⋮----
private func createOverride() async {
⋮----
private func deleteOverride(_ id: String) async {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Profile/ProfileView.swift
```swift
struct ProfileView: View {
@Environment(TokenStore.self) private var tokenStore
@State private var viewModel = ProfileViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Profile/SupplierIdentityCard.swift
```swift
struct SupplierIdentityCard: View {
let profile: SupplierProfile
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Promotions/PromotionsView.swift
```swift
struct PromotionsView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = PromotionsViewModel()
⋮----
var body: some View {
⋮----
static func formatPromoSummary(_ promo: SupplierPromotion) -> String {
⋮----
private func simSummary(_ sim: PromoSimulateResult) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Returns/ReturnsView.swift
```swift
struct ReturnsView: View {
@State private var viewModel = ReturnsViewModel()
⋮----
private let resolutions = ["RETURN_TO_STOCK", "WRITE_OFF"]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Settings/NotificationPreferencesView.swift
```swift
struct NotificationPreferencesView: View {
@State private var vm = NotificationPreferencesViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Settings/ReturnPolicySettingsView.swift
```swift
struct ReturnPolicySettingsView: View {
@State private var vm = ReturnPolicySettingsViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/ChargebacksView.swift
```swift
struct ChargebacksView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = ChargebacksViewModel()
⋮----
private let gateways = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/ClaimChargebacksView.swift
```swift
struct ClaimChargebacksView: View {
@State private var viewModel = ClaimChargebacksViewModel()
⋮----
var body: some View {
⋮----
static func formatChargebackRowAmount(_ amountMinor: Int64?, currency: String?) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/ComplianceAuditView.swift
```swift
struct ComplianceAuditView: View {
@State private var viewModel = ComplianceAuditViewModel()
⋮----
var body: some View {
⋮----
private func metricRow(_ label: String, value: Int) -> some View {
⋮----
enum ReceiptOpen {
static func shouldStayQuiet(_ error: Error) -> Bool {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/LedgerView.swift
```swift
struct LedgerView: View {
@State private var viewModel = LedgerViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/PaymentsView.swift
```swift
struct PaymentsView: View {
@State private var viewModel = PaymentsViewModel()
⋮----
var body: some View {
⋮----
private struct PaymentsKpiRow: View {
let label: String
let value: String
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/ReconciliationView.swift
```swift
struct ReconciliationView: View {
@State private var viewModel = ReconciliationViewModel()
⋮----
var body: some View {
⋮----
static func formatSettlementNet(netMinor: Int64?, currency: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Treasury/TreasuryHubView.swift
```swift
struct TreasuryHubView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = TreasuryViewModel()
⋮----
var body: some View {
⋮----
private struct TreasuryLinkRow<Destination: View>: View {
let title: LocalizedStringKey
let systemImage: String
@ViewBuilder let destination: () -> Destination
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/BillingGateView.swift
```swift
struct BillingGateView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var vm = BillingGateViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/L10n.swift
```swift
enum L10n {
private static let bundle: Bundle = {
⋮----
private static let englishBundle: Bundle? = {
⋮----
static func format(_ key: String, _ args: CVarArg...) -> String {
⋮----
private final class L10nBundleToken {}
```

## File: apps/supplier-app-ios/SupplierApp/RootView.swift
```swift
struct RootView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var realtime = SupplierRealtimeClient()
⋮----
var body: some View {
⋮----
private func syncRealtime(authenticated: Bool) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Inventory/InventoryStockList.swift
```swift
struct InventoryStockList: View {
let items: [InventoryItem]
let policySavingId: String?
let policies: [String]
var onAdjust: (InventoryItem) -> Void
var onPolicyChange: (InventoryItem, String) -> Void
⋮----
var body: some View {
⋮----
private struct InventoryStockRowView: View {
let item: InventoryItem
let isSavingPolicy: Bool
let policies: [String]
let onAdjust: () -> Void
let onPolicyChange: (String) -> Void
⋮----
@State private var selectedPolicy: String
⋮----
private static func resolvedPolicy(for item: InventoryItem) -> String {
⋮----
init(
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Components/Tactical/TacticalInspectorView.swift
```swift
struct TacticalInspectorView: View {
let title: String
let subtitle: String
var status: String = "ACTIVE"
var temperature: Double? = nil
var humidity: Int? = nil
var battery: Int? = nil
var boltSealCode: String? = nil
var timelineItems: [TacticalTimelineItem] = []
var onAction: ((String) -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/CollapsibleSidebar.swift
```swift
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
let title: String
@Binding var isExpanded: Bool
@Binding var selection: Item?
let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
@ViewBuilder let detail: () -> Detail
let footer: (() -> AnyView)?
⋮----
init(
⋮----
var body: some View {
⋮----
private func sidebar(effectiveExpanded: Bool, sidebarWidth: CGFloat, canToggle: Bool) -> some View {
⋮----
private func sidebarRow(_ item: CollapsibleSidebarItem<Item>, effectiveExpanded: Bool) -> some View {
⋮----
struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
let id: String
let tag: Item
let label: String
let icon: String
⋮----
init(id: String? = nil, tag: Item, label: String, icon: String) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/DeliveryActivityAttributes.swift
```swift
public struct DeliveryActivityAttributes: ActivityAttributes {
public struct ContentState: Codable, Hashable {
public var status: String
public var etaMinutes: Int
public var remainingDistanceMeters: Double
public var nextInstruction: String
public var nextManeuver: String?
public var progress: Double
public var currentStepIndex: Int
public var totalSteps: Int
public var deliveryToken: String?
public var lastUpdated: Date
⋮----
public init(
⋮----
public var orderId: String
public var routeId: String
public var supplierName: String
public var destinationName: String
public var totalItemCount: Int
public var totalAmountMinorUnits: Int64
public var isDriverPerspective: Bool
⋮----
public init(
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/DeliveryLiveActivityWidget.swift
```swift
public struct DeliveryLiveActivityWidget: Widget {
public init() {}
⋮----
public var body: some WidgetConfiguration {
⋮----
private func formatDistance(_ meters: Double) -> String {
⋮----
private struct LockScreenLiveActivityView: View {
let context: ActivityViewContext<DeliveryActivityAttributes>
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/PegasusMonochromeTheme.swift
```swift
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
convenience init(lightHex: UInt32, darkHex: UInt32) {
⋮----
public enum PegasusMonochromeTheme {
public static let readableMaxWidth: CGFloat = 960
⋮----
public static var background: Color { Color(uiColor: UIColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
⋮----
public static var card: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
⋮----
public static var secondaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
⋮----
public static var tertiaryBackground: Color { Color(uiColor: UIColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
⋮----
public static var iconTile: Color { Color(uiColor: UIColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
⋮----
public static var label: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
⋮----
public static var secondaryLabel: Color { Color(uiColor: UIColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
⋮----
public static var tertiaryLabel: Color { Color(uiColor: UIColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
⋮----
public static var separator: Color { Color(uiColor: UIColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
⋮----
public static var cardBorder: Color { Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) }) }
⋮----
public static var primaryAction: Color { Color(uiColor: UIColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(uiColor: UIColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
⋮----
public static var accentWarm: Color { Color(uiColor: UIColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(uiColor: UIColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(uiColor: UIColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
⋮----
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(uiColor: UIColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static var background: Color { Color(nsColor: NSColor(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)) }
public static var card: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x18181B)) }
public static var secondaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xF6F6F8, darkHex: 0x202024)) }
public static var tertiaryBackground: Color { Color(nsColor: NSColor(lightHex: 0xEAEAEB, darkHex: 0x27272A)) }
public static var iconTile: Color { Color(nsColor: NSColor(lightHex: 0xE2E2E6, darkHex: 0x27272A)) }
public static var fill: Color { iconTile }
public static var label: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xF4F4F5)) }
public static var secondaryLabel: Color { Color(nsColor: NSColor(lightHex: 0x71717A, darkHex: 0xA1A1AA)) }
public static var tertiaryLabel: Color { Color(nsColor: NSColor(lightHex: 0xA1A1AA, darkHex: 0x71717A)) }
public static var separator: Color { Color(nsColor: NSColor(lightHex: 0xE4E4E7, darkHex: 0x27272A)) }
public static var cardBorder: Color { Color(nsColor: NSColor(name: nil) { $0.bestMatch(from: [.darkAqua, .aqua]) == .darkAqua ? NSColor.white.withAlphaComponent(0.08) : NSColor.black.withAlphaComponent(0.04) }) }
public static var primaryAction: Color { Color(nsColor: NSColor(lightHex: 0x111113, darkHex: 0xFFFFFF)) }
public static var onPrimaryAction: Color { Color(nsColor: NSColor(lightHex: 0xFFFFFF, darkHex: 0x111113)) }
public static var accentWarm: Color { Color(nsColor: NSColor(lightHex: 0xBC7F70, darkHex: 0xD2B39B)) }
public static var accentWarmSoft: Color { Color(nsColor: NSColor(lightHex: 0xF5ECE9, darkHex: 0x362723)) }
public static var accentBronze: Color { Color(nsColor: NSColor(lightHex: 0x624D45, darkHex: 0xE4BFA2)) }
public static var barPrimary: Color { primaryAction }
public static var barMuted: Color { Color(nsColor: NSColor(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)) }
⋮----
public static let destructive = Color(red: 0.93, green: 0.27, blue: 0.27)
public static let success = Color(red: 0.13, green: 0.77, blue: 0.37)
public static let warning = Color(red: 0.96, green: 0.62, blue: 0.04)
public static let live = Color(red: 0.23, green: 0.51, blue: 0.96)
⋮----
public static let spacingXS: CGFloat = 4
public static let spacingSM: CGFloat = 8
public static let spacingMD: CGFloat = 12
public static let spacingLG: CGFloat = 16
public static let spacingXL: CGFloat = 24
public static let spacingXXL: CGFloat = 32
⋮----
public static let radiusXS: CGFloat = 4
public static let radiusSM: CGFloat = 8
public static let radiusMD: CGFloat = 12
public static let radiusLG: CGFloat = 16
public static let radiusCard: CGFloat = 24
public static let radiusXL: CGFloat = 28
⋮----
public static func statusTint(for status: String) -> Color {
⋮----
public struct PegasusMonochromeCard: ViewModifier {
public init() {}
public func body(content: Content) -> some View {
⋮----
public struct MinimumHitTargetModifier: ViewModifier {
public var minSize: CGFloat
⋮----
public init(minSize: CGFloat = 44) {
⋮----
public func body(content: Content) -> some View {
⋮----
func pegasusMonochromeCard() -> some View {
⋮----
func pegasusReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(_ minSize: CGFloat = 44) -> some View {
⋮----
public enum PegasusHaptics {
public static func light() {
⋮----
public static func medium() {
⋮----
public static func heavy() {
⋮----
public static func selection() {
⋮----
public static func success() {
⋮----
public static func warning() {
⋮----
public static func error() {
⋮----
public struct PegasusPressableButtonStyle: ButtonStyle {
public init() {}
public func makeBody(configuration: Configuration) -> some View {
⋮----
static var pegasusPressable: PegasusPressableButtonStyle { PegasusPressableButtonStyle() }
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/PegasusMotion.swift
```swift
public enum PegasusAnim {
public static let snappy = Animation.snappy(duration: 0.3)
public static let smooth = Animation.smooth(duration: 0.35)
public static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
public static let quick = Animation.easeOut(duration: 0.15)
⋮----
public static func staggerDelay(index: Int) -> Double {
⋮----
public static func resolved(_ animation: Animation = snappy, reduceMotion: Bool) -> Animation? {
⋮----
public enum PegasusMotionDuration {
public static let short4: Double = 0.2
public static let medium2: Double = 0.3
public static let medium4: Double = 0.4
⋮----
func pegasusNumericMetric<V: Equatable>(value: V) -> some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/PegasusStateViews.swift
```swift
public struct PegasusLoadingView: View {
public let title: String
public var message: String = "Fetching the latest data."
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
public init(title: String, message: String = "Fetching the latest data.") {
⋮----
public var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Design/RealtimeRefresh.swift
```swift
public struct SilentRefreshModifier: ViewModifier {
private let refreshEpoch: Int
private let reconnectEpoch: Int
private let reload: (_ silent: Bool) -> Void
⋮----
@State private var coalesceTask: Task<Void, Never>?
⋮----
public init(
⋮----
public func body(content: Content) -> some View {
⋮----
private func scheduleCoalescedReload() {
⋮----
func silentRealtimeRefresh(
⋮----
public enum RealtimeLoadState {
public static func showFullScreenLoading(loading: Bool, hasData: Bool) -> Bool {
⋮----
public enum PulseHonesty {
public static let failed = "pulse_failed"
public static let commandFailed = "control_tower_pulse_failed"
⋮----
public struct Result<T> {
public let events: [T]
public let error: String?
⋮----
public struct ObjectResult<T> {
public let value: T?
public let error: String?
⋮----
public static func apply<T>(ok: Bool, incoming: [T]?, previous: [T]) -> Result<T> {
⋮----
public static func applyObject<T>(ok: Bool, incoming: T?, previous: T?) -> ObjectResult<T> {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/StatusStack.swift
```swift
enum StatusStackMode: String {
⋮----
struct StatusStackRow: Equatable {
let key: String
let count: Int?
let share: Double
⋮----
struct StatusStackModel: Equatable {
let mode: StatusStackMode
let rows: [StatusStackRow]
let total: Int
⋮----
let orderStatusFunnel = [
⋮----
let manifestStates = [
⋮----
let truckDutyStatuses = [
⋮----
let factoryTransferStates = [
⋮----
let factoryVehicleStates = ["READY", "AVAILABLE", "UNAVAILABLE"]
⋮----
let factoryDriverDuty = ["ON_SHIFT", "OFF_SHIFT"]
⋮----
func canonicalizeOrderStatus(_ status: String) -> String {
⋮----
func incrementOrderStatusCount(_ counts: [String: Int], status: String) -> [String: Int] {
⋮----
func statusStackModel(
⋮----
struct SourceChip: View {
let source: String
⋮----
var body: some View {
⋮----
struct CommandStatusJump: Hashable, Identifiable {
let id: UUID
let status: String
var supplierId: String?
⋮----
init(status: String, supplierId: String? = nil) {
⋮----
struct StatusStackView: View {
let dictionary: [String]
let counts: [String: Int]?
var available: Bool = true
var source: String? = nil
var onSelect: ((String) -> Void)? = nil
⋮----
private var model: StatusStackModel {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/WarehouseSection.swift
```swift
enum WarehouseSection: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var icon: String {
⋮----
static var compactTabs: [WarehouseSection] {
⋮----
static var primarySections: [WarehouseSection] {
⋮----
static var fulfillmentSections: [WarehouseSection] {
⋮----
static var inventorySections: [WarehouseSection] {
⋮----
static var operationsSections: [WarehouseSection] {
⋮----
static var portalSections: [WarehouseSection] {
⋮----
static var sidebarSections: [WarehouseSection] {
⋮----
var portalFeature: WarehousePortalFeature? {
⋮----
enum WarehouseCompactTab: Hashable {
```

## File: apps/warehouse-app-ios/WarehouseApp/Models/FoundryCashierModels.swift
```swift
public struct StartCashierSessionRequest: Codable, Sendable {
public let cellId: String
public let supplierId: String
public let depotId: String
public let driverId: String
public let routeId: String
public let bagSealBarcode: String
public let expectedCashMinor: Int64
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct CashierSessionView: Codable, Sendable, Identifiable {
public var id: String { sessionId }
public let sessionId: String
public let supplierId: String
public let depotId: String
public let driverId: String
public let routeId: String
public let bagSealBarcode: String
public let attemptNumber: Int64
public let countedCashMinor: Int64
public let expectedCashMinor: Int64?
public let varianceMinor: Int64?
public let status: String
public let clearanceTokenHash: String?
public let microSkimmingFlagged: Bool?
public let createdAt: String
public let resolvedAt: String?
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct BlindCountRequest: Codable, Sendable {
public let denominationsUzs: [String: Int64]
public let toleranceThresholdMinor: Int64
⋮----
public init(denominationsUzs: [String: Int64], toleranceThresholdMinor: Int64 = 500000) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct RRNTenderReclassificationRequest: Codable, Sendable {
public let sessionId: String
public let orderId: String
public let disputedCashMinor: Int64
public let gatewayRrn: String
public let terminalId: String
public let cardLastFour: String
public let authCode: String
public let paymentNetwork: String
public let rrnCapturedAt: String
public let podGeofenceTimestamp: String
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct CITHandoverRequest: Codable, Sendable {
public let depotId: String
public let supplierId: String
public let sessionIds: [String]
public let courierBadgeId: String
public let citSealBarcode: String
public let cashMinor: Int64
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct DepotVaultLedgerEntry: Codable, Sendable, Identifiable {
public var id: String { vaultEntryId }
public let vaultEntryId: String
public let depotId: String
public let supplierId: String
public let sessionId: String?
public let citManifestId: String?
public let entryType: String
public let amountMinor: Int64
public let runningBalanceMinor: Int64
public let courierBadgeId: String?
public let sealBarcode: String?
public let createdAt: String
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct AMLCheckRequest: Codable, Sendable {
public let legalEntityTin: String
public let supplierId: String
public let calendarDate: String
public let cashMinor: Int64
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct AMLCheckResponse: Codable, Sendable {
public let legalEntityTin: String
public let accumulatedCashMinor: Int64
public let ceilingMinor: Int64
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct YardGateCheckEvent: Codable, Sendable, Identifiable {
public var id: String { eventId ?? UUID().uuidString }
public let eventId: String?
public let depotId: String
public let supplierId: String
public let vehicleId: String
public let licensePlate: String
public let direction: String
public let vehiclePurpose: String
public let boltSealBarcode: String?
public let sealIntact: Bool
public let assignedBaySlot: String?
public let checkedAt: String?
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/FirebaseAuthHelper.swift
```swift

```

## File: apps/warehouse-app-ios/WarehouseApp/Services/GeocodeService.swift
```swift
struct GeocodePrediction: Decodable, Equatable {
let placeId: String
let description: String
⋮----
init(placeId: String, description: String) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ResolvedLocation: Decodable, Equatable {
let address: String
let lat: Double
let lng: Double
let placeId: String?
⋮----
init(address: String, lat: Double, lng: Double, placeId: String? = nil) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum GeocodeService {
static func autocomplete(_ input: String) async throws -> [GeocodePrediction] {
⋮----
static func resolvePlace(_ placeId: String) async throws -> ResolvedLocation {
⋮----
static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
⋮----
static func forward(address: String) async throws -> ResolvedLocation {
struct ForwardRequest: Encodable { let address: String }
⋮----
private struct AutocompleteResponse: Decodable {
let predictions: [GeocodePrediction]
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Empty pack prints the number only. Never invents UZS.
func formatPackMoney(_ minor: Int64, pack: MarketPack?, decimalPlaces: Int = 2) -> String {
⋮----
/// Stored/event currency, else session pack. Empty pack does not invent UZS.
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/NetworkLifecycleEngine.swift
```swift
public enum NetworkState: String, Sendable {
⋮----
public final class NetworkLifecycleEngine {
public static let shared = NetworkLifecycleEngine()
⋮----
public private(set) var isOnline: Bool = true
public private(set) var state: NetworkState = .connected
public private(set) var syncEpoch: Int = 0
⋮----
private let monitor: NWPathMonitor
private let queue = DispatchQueue(label: "com.pegasusx.warehouse.network-lifecycle", qos: .utility)
private var transitionTimestamps: [Date] = []
private var flappingTask: Task<Void, Never>?
private var isTesting = false
⋮----
private init() {
⋮----
private func handlePathUpdate(_ path: NWPath) {
⋮----
private func recordTransition(online: Bool) {
⋮----
public func triggerSyncEpoch() async {
⋮----
public func waitForOnline() async {
⋮----
public func resetForTesting() {
⋮----
public func setOnlineForTesting(_ online: Bool) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseOperationsService.swift
```swift
enum WarehouseOperationsService {
private static let api = APIClient.shared
⋮----
static func emergencyTransfer(totalVolumeVu: Double, notes: String? = nil) async throws -> TransferMutationResponse {
⋮----
static func forceReceive(
⋮----
static func receiveTransfer(transferId: String) async throws -> TransferMutationResponse {
⋮----
static func replenishmentInsights() async throws -> [ReplenishmentInsight] {
⋮----
static func replenishmentInsightAction(
⋮----
static func opsFinancials(period: String? = nil) async throws -> OpsFinancialsResponse {
⋮----
static func delayOrder(orderId: String, reason: String? = nil) async throws -> WarehouseOrderMutationResponse {
⋮----
static func rejectOrder(orderId: String, reason: String) async throws -> WarehouseOrderMutationResponse {
⋮----
static func proposeOrderDelivery(
⋮----
static func proposePreorderDelivery(
⋮----
static func rejectPreorder(orderId: String, reason: String) async throws -> WarehouseOrderMutationResponse {
⋮----
static func overflowOrder(orderId: String, reason: String? = nil) async throws -> WarehouseOrderMutationResponse {
⋮----
static func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
⋮----
static func reassignOrder(_ request: ReassignOrderRequest, idempotencyKey: String) async throws {
⋮----
static func refreshToken(_ refreshToken: String) async throws -> AuthResponse {
⋮----
static func issuePaymentBypass(orderId: String) async throws -> PaymentBypassResponse {
⋮----
static func getEarlyCompleteRequest(driverId: String) async throws -> [String: String] {
⋮----
static func approveEarlyComplete(driverId: String, action: String, newWindowStart: String? = nil, newWindowEnd: String? = nil) async throws -> [String: String] {
⋮----
struct PaymentBypassRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentBypassResponse: Decodable {
let status: String?
let bypassToken: String?
let orderId: String?
enum CodingKeys: String, CodingKey {
⋮----
struct EarlyCompleteRequest: Encodable {
let truckId: String
enum CodingKeys: String, CodingKey {
⋮----
struct EarlyCompleteResponse: Decodable {
let status: String?
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseRealtimeClient.swift
```swift
enum WarehouseRealtimeStatus: Equatable {
⋮----
final class WarehouseRealtimeClient {
private let session: URLSession
private let decoder = JSONDecoder()
private let pathMonitor = NWPathMonitor()
private let workQueue = DispatchQueue(label: "warehouse.realtime.client")
⋮----
private var task: URLSessionWebSocketTask?
private var pingTask: Task<Void, Never>?
private var reconnectWorkItem: DispatchWorkItem?
private var reconnectAttempt = 0
private var closed = true
private var networkAvailable = true
private var stateHandler: (@MainActor (WarehouseRealtimeStatus) -> Void)?
private var eventHandler: (@MainActor (WarehouseLiveEvent) -> Void)?
private var reconnectHandler: (@MainActor () -> Void)?
private var hasConnectedOnce = false
private var lastEventId: String?
private static let pingIntervalNs: UInt64 = 25_000_000_000
⋮----
init(session: URLSession = .shared) {
⋮----
deinit {
⋮----
func connect(
⋮----
func disconnect() {
⋮----
private func openSocket(isReconnect: Bool) {
⋮----
private func startHeartbeatLoop(for socketTask: URLSessionWebSocketTask) {
⋮----
private func receiveLoop() {
⋮----
private func handleSocketDrop() {
⋮----
private func scheduleReconnect() {
⋮----
private func publish(_ status: WarehouseRealtimeStatus) {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseRealtimeHub.swift
```swift
final class WarehouseRealtimeHub {
var refreshEpoch: Int = 0
var reconnectEpoch: Int = 0
⋮----
@ObservationIgnored private var lastBumpAt: Date = .distantPast
@ObservationIgnored private var trailingTask: Task<Void, Never>?
@ObservationIgnored private let coalesceWindow: TimeInterval = 0.20
⋮----
func bump() {
⋮----
func bumpReconnect() {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseService.swift
```swift
enum WarehouseService {
private static let api = APIClient.shared
⋮----
static func login(phone: String, pin: String) async throws -> AuthResponse {
⋮----
static func login(idToken: String) async throws -> AuthResponse {
⋮----
static func setup(
⋮----
static func dashboard() async throws -> DashboardData {
⋮----
static func orders(state: String? = nil) async throws -> OrderListResponse {
⋮----
static func order(id: String) async throws -> Order {
⋮----
static func drivers() async throws -> DriverListResponse {
⋮----
static func createDriver(name: String, phone: String) async throws -> CreateDriverResponse {
⋮----
static func assignDriver(driverId: String, vehicleId: String?) async throws -> AssignDriverVehicleResponse {
⋮----
static func vehicles() async throws -> VehicleListResponse {
⋮----
static func vehicle(id: String) async throws -> VehicleDetailResponse {
⋮----
static func createVehicle(label: String, licensePlate: String, vehicleClass: String) async throws -> Vehicle {
⋮----
static func updateVehicleAvailability(
⋮----
static func inventory(lowStock: Bool = false) async throws -> InventoryListResponse {
⋮----
static func adjustInventory(productId: String, quantity: Int) async throws {
⋮----
static func createBin(locationId: String?, zone: String, locationType: String = "STAGE") async throws -> WarehouseBinLocation {
⋮----
static func putawayLot(
⋮----
static func listPickWaves(manifestId: String? = nil) async throws -> PickWaveListResponse {
⋮----
static func createPickWave(manifestId: String) async throws -> PickWave {
⋮----
static func getPickWave(waveId: String) async throws -> PickWave {
⋮----
static func confirmPickTask(waveId: String, taskId: String, quantityPicked: Int? = nil) async throws -> PickWave {
⋮----
static func listCycleCounts() async throws -> CycleCountListResponse {
⋮----
static func createCycleCount(locationId: String, productId: String, expectedQty: Int? = nil) async throws -> CycleCount {
⋮----
static func submitCycleCount(countId: String, countedQty: Int) async throws -> CycleCount {
⋮----
static func patchInventoryPolicy(productId: String, policy: String) async throws {
⋮----
static func opsSettings() async throws -> WarehouseOpsSettingsResponse {
⋮----
static func getReturnPolicy(
⋮----
static func putReturnPolicy(
⋮----
static func preorders() async throws -> WarehousePreordersResponse {
⋮----
static func stockCommitments() async throws -> StockCommitmentsResponse {
⋮----
static func patchOpsSettings(_ body: WarehouseOpsSettingsPatchRequest) async throws {
⋮----
static func patchOpsSettings(policy: String, operatingSchedule: [String: AnyCodable]) async throws {
⋮----
static func products() async throws -> ProductListResponse {
⋮----
static func manifests() async throws -> ManifestListResponse {
⋮----
static func analytics(period: String = "30d") async throws -> AnalyticsData {
⋮----
static func retailers() async throws -> RetailerListResponse {
⋮----
static func returns() async throws -> ReturnListResponse {
⋮----
static func inboundReturns(physicalStatus: String = "OPEN", limit: Int = 100) async throws -> InboundReturnListResponse {
⋮----
static func returnsHistory(limit: Int = 50) async throws -> InboundReturnListResponse {
⋮----
static func reverseLogistics(status: String = "OPEN", warehouseId: String? = nil) async throws -> ReverseLogisticsListResponse {
⋮----
static func receiveReverseLogistics(
⋮----
static func opsExceptions() async throws -> WarehouseOpsExceptionsResponse {
⋮----
static func laborZoneCapacity(date: String) async throws -> LaborZoneCapacityListResponse {
⋮----
static func laborDriverScore(driverId: String) async throws -> LaborDriverScore {
⋮----
static func setLaborDriverAvailability(_ body: LaborDriverAvailabilityRequest) async throws -> LaborDriverAvailabilityResponse {
⋮----
static func supplierClaims(status: String? = "OPEN", limit: Int = 50) async throws -> WarehouseClaimsResponse {
⋮----
static func createInboundSession() async throws -> String {
⋮----
static func scanInboundBarcode(barcode: String, qty: Int = 1, sessionId: String? = nil, idempotencyKey: String) async throws -> InboundScanResponse {
⋮----
static func scanInboundReturn(returnId: String, qty: Int = 1, sessionId: String? = nil, idempotencyKey: String) async throws -> InboundScanResponse {
struct ScanByReturnBody: Encodable {
let returnId: String
let qty: Int
let sessionId: String
enum CodingKeys: String, CodingKey {
⋮----
static func confirmInboundReturns(returnIds: [String], disposition: String, sessionId: String, idempotencyKey: String) async throws -> InboundConfirmResponse {
⋮----
static func treasuryOverview() async throws -> TreasuryOverview {
⋮----
static func treasuryInvoices() async throws -> InvoiceListResponse {
⋮----
static func dispatchPreview() async throws -> DispatchPreview {
⋮----
static func createDispatchPreview(body: [String: String]) async throws -> DispatchPreview {
⋮----
static func executeDispatch(body: DispatchExecuteRequest, idempotencyKey: String) async throws -> DispatchExecuteResponse {
⋮----
static func previewRescue(brokenDriverId: String) async throws -> RescuePreviewResponse {
⋮----
static func proposeRescue(
⋮----
static func supplyRequests(state: String? = nil) async throws -> [WarehouseSupplyRequest] {
⋮----
static func supplyRequest(id: String) async throws -> WarehouseSupplyRequest {
⋮----
static func supplyRequestQC(id: String) async throws -> SupplyRequestQCResponse {
⋮----
static func demandForecast(days: Int = 7) async throws -> DemandForecastResponse {
⋮----
static func dispatchLocks() async throws -> [WarehouseDispatchLock] {
⋮----
static func createSupplyRequest(form: SupplyRequestFormData) async throws -> CreateWarehouseSupplyRequestResponse {
⋮----
static func cancelSupplyRequest(id: String) async throws -> WarehouseSupplyRequestTransitionResponse {
⋮----
static func acquireDispatchLock(lockType: String = "MANUAL_DISPATCH") async throws -> CreateWarehouseDispatchLockResponse {
⋮----
static func releaseDispatchLock(lockId: String) async throws -> ReleaseWarehouseDispatchLockResponse {
⋮----
static func staff() async throws -> StaffListResponse {
⋮----
static func createStaff(name: String, phone: String, role: String) async throws -> CreateStaffResponse {
⋮----
static func paymentConfig() async throws -> PaymentConfigResponse {
⋮----
static func opsCoverage() async throws -> WarehouseCoverageResponse {
⋮----
static func opsSupplyFactory() async throws -> WarehouseSupplyFactoryResponse {
⋮----
static func replenishmentInsights() async throws -> ReplenishmentInsightsResponse {
⋮----
static func opsBoard(date: String) async throws -> WarehouseOpsBoardResponse {
⋮----
static func replenishmentInsightAction(insightId: String, action: String) async throws -> ReplenishmentInsightActionResponse {
⋮----
static func emergencyTransfer(body: EmergencyTransferRequest) async throws -> TransferMutationResponse {
⋮----
static func forceReceive(body: ForceReceiveRequest) async throws -> TransferMutationResponse {
⋮----
static func receiveTransfer(transferId: String) async throws -> TransferMutationResponse {
⋮----
static func opsFinancials(period: String? = nil) async throws -> OpsFinancialsResponse {
⋮----
static func broadcastTemplates() async throws -> BroadcastTemplatesResponse {
⋮----
static func createBroadcastTemplate(
⋮----
static func deleteBroadcastTemplate(templateId: String, idempotencyKey: String) async throws -> BroadcastTemplateDeleteResponse {
⋮----
static func postBroadcast(_ request: WarehouseBroadcastRequest, idempotencyKey: String) async throws -> WarehouseBroadcastResponse {
⋮----
static func previewRetailerPriceOverride(_ request: RetailerOverridePreviewRequest) async throws -> RetailerOverridePreview {
⋮----
static func delayOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
⋮----
static func rejectOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
⋮----
static func overflowOrder(orderId: String, body: WarehouseOrderMutationRequest) async throws -> WarehouseOrderMutationResponse {
⋮----
static func fleetLiveMap(warehouseId: String? = nil) async throws -> WarehouseFleetLiveMapResponse {
⋮----
static func pulse() async throws -> WarehousePulseResponse {
⋮----
static func dispatchSettings() async throws -> DispatchSettingsResponse {
⋮----
static func patchDispatchSettings(enabled: Bool) async throws {
⋮----
static func warehouseLocation() async throws -> WarehouseLocationResponse {
⋮----
static func patchWarehouseLocation(address: String, placeId: String?, lat: Double, lng: Double) async throws -> WarehouseLocationResponse {
⋮----
static func recommendReassign(orderId: String) async throws -> RecommendReassignResponse {
⋮----
static func reassignOrder(_ request: ReassignOrderRequest, idempotencyKey: String) async throws {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseService+Foundry.swift
```swift
static func startCashierSession(req: StartCashierSessionRequest) async throws -> CashierSessionView {
⋮----
static func submitBlindCount(sessionId: String, req: BlindCountRequest) async throws -> CashierSessionView {
⋮----
static func reclassifyTender(req: RRNTenderReclassificationRequest) async throws -> CashierSessionView {
⋮----
static func executeCITHandover(req: CITHandoverRequest) async throws -> DepotVaultLedgerEntry {
⋮----
static func checkAML(req: AMLCheckRequest) async throws -> AMLCheckResponse {
⋮----
static func recordGateEvent(event: YardGateCheckEvent) async throws -> YardGateCheckEvent {
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/WarehouseSessionReconcile.swift
```swift
enum WarehouseSessionReconcile {
private static let utcDateFormatter: DateFormatter = {
⋮----
static func run() async {
```

## File: apps/warehouse-app-ios/WarehouseApp/Theme/PegasusTheme.swift
```swift
enum Anim {
static let snappy = Animation.snappy(duration: 0.3)
static let smooth = Animation.smooth(duration: 0.35)
static let spring = Animation.spring(response: 0.4, dampingFraction: 0.85)
static let quick = Animation.easeOut(duration: 0.15)
⋮----
struct LabCardModifier: ViewModifier {
func body(content: Content) -> some View {
⋮----
func labCard() -> some View {
⋮----
func labReadableWidth() -> some View {
⋮----
struct StaggeredAppearModifier: ViewModifier {
let index: Int
@State private var appeared = false
⋮----
func body(content: Content) -> some View {
⋮----
func staggeredAppear(index: Int) -> some View {
⋮----
struct PressableButtonStyle: ButtonStyle {
func makeBody(configuration: Configuration) -> some View {
⋮----
static var pressable: PressableButtonStyle { PressableButtonStyle() }
```

## File: apps/warehouse-app-ios/WarehouseApp/Theme/WarehouseTacticalTheme.swift
```swift
init(hex: UInt32, opacity: Double = 1.0) {
⋮----
init(lightHex: UInt32, darkHex: UInt32) {
⋮----
enum TacticalTheme {
⋮----
static let canvas = Color(lightHex: 0xEDEDF0, darkHex: 0x0E0E10)
static let surface = Color(lightHex: 0xFFFFFF, darkHex: 0x18181B)
static let surfaceSubtle = Color(lightHex: 0xF6F6F8, darkHex: 0x202024)
static let surfaceSunken = Color(lightHex: 0xEAEAEB, darkHex: 0x131316)
static let surfaceRaised = Color(lightHex: 0xFFFFFF, darkHex: 0x1F1F23)
static let iconTile = Color(lightHex: 0xE2E2E6, darkHex: 0x27272A)
⋮----
static let border = Color(lightHex: 0xE4E4E7, darkHex: 0x27272A)
static let borderStrong = Color(lightHex: 0xD4D4D8, darkHex: 0x3F3F46)
⋮----
static let textPrimary = Color(lightHex: 0x111113, darkHex: 0xF4F4F5)
static let textSecondary = Color(lightHex: 0x71717A, darkHex: 0xA1A1AA)
static let textTertiary = Color(lightHex: 0xA1A1AA, darkHex: 0x71717A)
⋮----
static let accentCobalt = Color(lightHex: 0x2563EB, darkHex: 0x3B82F6)
static let accentOrange = Color(lightHex: 0xF97316, darkHex: 0xFF7A1A)
static let accentLime = Color(lightHex: 0x65A30D, darkHex: 0xE2FD52)
static let accentCyan = Color(lightHex: 0x0284C7, darkHex: 0x06B6D4)
static let accentPurple = Color(lightHex: 0x7C3AED, darkHex: 0x8B5CF6)
⋮----
static let statusSuccess = Color(lightHex: 0x059669, darkHex: 0x10B981)
static let statusWarning = Color(lightHex: 0xD97706, darkHex: 0xF59E0B)
static let statusDanger = Color(lightHex: 0xDC2626, darkHex: 0xEF4444)
⋮----
static let radiusSM: CGFloat = 8
static let radiusMD: CGFloat = 12
static let radiusLG: CGFloat = 16
static let radiusCard: CGFloat = 24
static let radiusXL: CGFloat = 28
⋮----
struct TacticalCardModifier: ViewModifier {
var isRaised: Bool = false
var padding: CGFloat = 16
⋮----
func body(content: Content) -> some View {
⋮----
func tacticalCard(isRaised: Bool = false, padding: CGFloat = 16) -> some View {
⋮----
func tacticalMicroLabel() -> some View {
⋮----
func tacticalMonospaceValue(_ style: Font.TextStyle = .title) -> some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/GeocodeLocationSupport.swift
```swift
enum GeocodeLocationSupport {
static func hasValidCoordinates(lat: Double, lng: Double) -> Bool {
⋮----
static func resolveLocationValue(_ value: AddressLocationValue) async -> AddressLocationValue? {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/WarehouseIdempotency.swift
```swift
enum WarehouseIdempotency {
private static func warehouseId() -> String {
⋮----
private static func stableHash(_ input: String) -> String {
⋮----
static func emergencyTransfer(volumeVu: Double, notes: String?) -> String {
⋮----
static func forceReceive(volumeVu: Double, notes: String?, factoryId: String? = nil) -> String {
⋮----
static func receiveTransfer(transferId: String) -> String {
⋮----
static func createDriver(phone: String) -> String {
⋮----
static func createStaff(phone: String) -> String {
⋮----
static func createVehicle(licensePlate: String) -> String {
⋮----
static func adjustInventory(productId: String, quantity: Int) -> String {
⋮----
static func assignDriverVehicle(driverId: String, vehicleId: String?) -> String {
⋮----
static func updateVehicle(vehicleId: String, isActive: Bool, unavailableReason: String? = nil) -> String {
⋮----
static func inventoryPolicy(productId: String, policy: String) -> String {
⋮----
static func replenishmentInsightAction(insightId: String, action: String) -> String {
⋮----
static func dispatchSettings(autoDispatchEnabled: Bool) -> String {
⋮----
static func dispatchLockAcquire(entityType: String = "WAREHOUSE", entityId: String? = nil) -> String {
⋮----
static func dispatchLockRelease(lockId: String) -> String {
⋮----
static func dispatch(actorId: String, routeFingerprint: String) -> String {
⋮----
static func rescuePropose(rescueId: String, brokenDriverId: String, rescueDriverId: String) -> String {
⋮----
static func inboundScan(barcode: String, sessionId: String) -> String {
⋮----
static func inboundConfirm(returnIds: [String], disposition: String) -> String {
⋮----
static func createSupplyRequest(factoryId: String, mode: String, notes: String) -> String {
⋮----
static func opsSettings() -> String {
⋮----
static func returnPolicyPut(supplierId: String = "") -> String {
⋮----
static func opsLocation(lat: Double, lng: Double, placeId: String? = nil) -> String {
⋮----
static func supplyRequestTransition(requestId: String, action: String) -> String {
⋮----
static func orderDelay(orderId: String) -> String {
⋮----
static func orderReject(orderId: String, reason: String) -> String {
⋮----
static func orderProposeDelivery(orderId: String, proposedDate: String, reason: String) -> String {
⋮----
static func orderOverflow(orderId: String) -> String {
⋮----
static func broadcast(role: String, title: String, body: String) -> String {
⋮----
static func broadcastTemplateCreate(title: String, body: String) -> String {
⋮----
static func broadcastTemplateDelete(templateId: String) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/Utilities/WarehousePortalLinks.swift
```swift
enum WarehousePortalLinks {
private static let fallbackURL = URL(string: "http://localhost:3002") ?? URL(fileURLWithPath: "/")
⋮----
static var baseURL: URL {
⋮----
static func url(for feature: WarehousePortalFeature) -> URL {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/ClientPolicyBanner.swift
```swift
struct ClientPolicyBanner: View {
let message: String?
var force: Bool = false
var onUpdate: (() -> Void)? = nil
var onDismiss: (() -> Void)? = nil
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/ForecastConfidenceView.swift
```swift
struct ForecastConfidenceData {
let lowUnits: Int64?
let highUnits: Int64?
let confidencePct: Int?
let baselineSource: String?
let blockedReason: String?
let label: String?
⋮----
var blocked: Bool {
⋮----
var seasonalActive: Bool {
⋮----
func parseForecastConfidence(_ breakdown: [String: AnyCodable]?) -> ForecastConfidenceData? {
⋮----
func formatSourceBadge(_ source: String?) -> String {
⋮----
private func number(from codable: AnyCodable?) -> Double? {
⋮----
private func int64(from codable: AnyCodable?) -> Int64? {
⋮----
private func int(from codable: AnyCodable?) -> Int? {
⋮----
private func string(from codable: AnyCodable?) -> String? {
⋮----
struct ForecastConfidenceView: View {
let confidence: ForecastConfidenceData
var compact: Bool = false
⋮----
var body: some View {
⋮----
private func confidenceColor(_ pct: Int) -> Color {
⋮----
private static let decimalFormatter: NumberFormatter = {
⋮----
private func formatUnits(_ value: Int64) -> String {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dashboard/DashboardView.swift
```swift
struct DashboardView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var vm = DashboardViewModel()
⋮----
private var gridMin: CGFloat {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/DemandForecast/ForecastSkuTable.swift
```swift
struct ForecastSkuTable: View {
let products: [DemandForecastProduct]
let generatedAt: String?
let forecastDays: Int
⋮----
var body: some View {
⋮----
private static let isoFractionalFormatter: ISO8601DateFormatter = {
⋮----
private static let isoStandardFormatter = ISO8601DateFormatter()
⋮----
private func formattedGeneratedAt(_ raw: String) -> String {
⋮----
private struct ForecastProductRow: View {
let product: DemandForecastProduct
⋮----
var body: some View {
⋮----
private func metricColumn(title: String, value: String, valueColor: Color = LabTheme.label) -> some View {
⋮----
private func sourceChip(label: String, value: String) -> some View {
⋮----
private func priorityColor(_ priority: String) -> Color {
⋮----
private func stockoutColor(_ days: Double) -> Color {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Inventory/InventoryView.swift
```swift
struct InventoryView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = InventoryViewModel()
⋮----
var body: some View {
⋮----
private struct AdjustInventorySheet: View {
let onAdjusted: () -> Void
@Environment(\.dismiss) private var dismiss
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm: AdjustInventoryViewModel
⋮----
init(item: InventoryItem, onAdjusted: @escaping () -> Void) {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/LaborCapacity/LaborCapacityView.swift
```swift
struct LaborCapacityView: View {
@State private var vm = LaborCapacityViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Orders/OrdersView.swift
```swift
struct OrdersView: View {
var initialState: String? = nil
@State private var vm: OrdersViewModel
⋮----
init(initialState: String? = nil) {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Preorders/PreordersView.swift
```swift
struct PreordersView: View {
@State private var vm = PreordersViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Returns/ReturnsList.swift
```swift
struct ReturnsList: View {
let items: [InboundReturnRow]
let isQueueTab: Bool
@Binding var selected: Set<String>
⋮----
var body: some View {
⋮----
private struct ReturnItemRowView: View {
let item: InboundReturnRow
let selectable: Bool
@Binding var selected: Set<String>
⋮----
@State private var isSelected: Bool
⋮----
init(
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Returns/ReturnsView.swift
```swift
struct ReturnsView: View {
@State private var vm = ReturnsViewModel()
⋮----
var body: some View {
⋮----
private var scanSection: some View {
⋮----
private var queueList: some View {
⋮----
private var historyList: some View {
⋮----
private var reverseList: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Staff/StaffList.swift
```swift
struct StaffList: View {
let staff: [StaffMember]
let loading: Bool
let error: String?
let onRetry: () -> Void
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/SupplyRequests/CreateSupplyRequestSheet.swift
```swift
struct SupplyRequestFormData {
let factoryId: String
let priority: String
let notes: String
let useDemandForecast: Bool
let requestedDeliveryDate: String?
let items: [CreateWarehouseSupplyRequestItem]
⋮----
struct CreateSupplyRequestSheet: View {
let onCreate: (SupplyRequestFormData) -> Void
⋮----
@Environment(\.dismiss) private var dismiss
@State private var vm = CreateSupplyRequestViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/TomorrowBoard/TomorrowBoardView.swift
```swift
struct TomorrowBoardView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = TomorrowBoardViewModel()
⋮----
var body: some View {
⋮----
private func boardRow(_ row: WarehouseOpsBoardOrder, lane: String) -> some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Vehicles/VehicleDetailView.swift
```swift
struct VehicleDetailView: View {
let vehicleId: String
⋮----
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = VehicleDetailViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/LoginView.swift
```swift
struct LoginView: View {
private enum Field: Hashable {
⋮----
@Environment(TokenStore.self) private var tokenStore
@Environment(\.openURL) private var openURL
@State private var vm = LoginViewModel()
@FocusState private var focusedField: Field?
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/warehouseapp/Services/EnterpriseUpdateConfig.swift
```swift

```

## File: apps/control-tower-macos/Sources/Services/ControlTowerAPIClient.swift
```swift
public actor ControlTowerAPIClient {
public static let shared = ControlTowerAPIClient()
⋮----
public static var defaultBaseURL: URL {
⋮----
public nonisolated let baseURL: URL
private let session: URLSession
private var authToken: String?
⋮----
public init(baseURL: URL? = nil) {
⋮----
public func setAuthToken(_ token: String?) {
⋮----
public func currentAuthToken() -> String? {
⋮----
public func fetchCellHealth() async throws -> [CellHealth] {
⋮----
public func fetchSpannerMetrics() async throws -> SpannerClusterMetrics {
⋮----
public func fetchKafkaLags() async throws -> [KafkaPartitionLag] {
⋮----
public func fetchRedisPools() async throws -> [RedisPermitPool] {
⋮----
public func fetchWaterfall(for role: RolePerspective) async throws -> [TenderWaterfallSlice] {
⋮----
public func fetchAttributedCosts() async throws -> [AttributedCostItem] {
⋮----
public func fetchScorecards() async throws -> [RolePerformanceScorecard] {
```

## File: apps/control-tower-macos/Sources/ViewModels/ControlTowerViewModel.swift
```swift
public enum NavigationSection: String, CaseIterable, Identifiable, Sendable {
⋮----
public var id: String { rawValue }
⋮----
public var icon: String {
⋮----
public final class ControlTowerViewModel {
public var selectedSection: NavigationSection = .overview {
⋮----
public var selectedRole: RolePerspective = .overall {
⋮----
public var searchQuery: String = ""
public var selectedSeverity: LogSeverity? = nil
public var isCheatsheetPresented: Bool = false
public var isInspectorPresented: Bool = true
⋮----
public var telemetryHUDString: String {
⋮----
public private(set) var cells: [CellHealth] = []
public private(set) var spanner: SpannerClusterMetrics?
public private(set) var kafkaPartitions: [KafkaPartitionLag] = []
public private(set) var redisPools: [RedisPermitPool] = []
public private(set) var waterfallSlices: [TenderWaterfallSlice] = []
public private(set) var costs: [AttributedCostItem] = []
public private(set) var scorecards: [RolePerformanceScorecard] = []
public private(set) var isLoading: Bool = false
public private(set) var lastRefreshed: Date = Date()
public var copiedToastMessage: String?
⋮----
public let streamClient: WebSocketStreamClient
private let apiClient: ControlTowerAPIClient
private var toastGeneration: UInt64 = 0
⋮----
public init(
⋮----
public convenience init(apiClient: ControlTowerAPIClient = .shared) {
⋮----
public func refresh() {
⋮----
public func initialLoad() async {
⋮----
public func loadRoleSpecificData() async {
⋮----
public func triageDLQInTelemetry() {
⋮----
public func copyToClipboard(_ text: String, label: String = "ID") {
⋮----
public var filteredTelemetryEvents: [TelemetryLogEvent] {
⋮----
public func exportDiagnosticBundle() {
⋮----
public func exportWaterfallCSV() {
```

## File: apps/control-tower-macos/Sources/Views/AnalyticsScorecardView.swift
```swift
public struct AnalyticsScorecardView: View {
var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
```

## File: apps/control-tower-macos/Sources/Views/FinOpsView.swift
```swift
public struct FinOpsView: View {
@Bindable var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
// Itemized Breakdown Table
⋮----
// Attributed Infrastructure Cost-to-Serve Table
```

## File: apps/control-tower-macos/Sources/Views/InfraHealthView.swift
```swift
public struct InfraHealthView: View {
var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
private func latencyStatPill(label: String, value: String, color: Color) -> some View {
```

## File: apps/control-tower-macos/Sources/Views/InspectorView.swift
```swift
public struct InspectorView: View {
var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
private func inspectorRow(label: String, value: String, isAlert: Bool = false) -> some View {
```

## File: apps/control-tower-macos/Sources/Views/MainSplitView.swift
```swift
public struct MainSplitView: View {
@Bindable var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
```

## File: apps/control-tower-macos/Sources/Views/MenuBarStatusView.swift
```swift
public struct MenuBarStatusView: View {
var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
```

## File: apps/control-tower-macos/Sources/Views/SidebarView.swift
```swift
public struct SidebarView: View {
@Bindable var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
private func badgeForSection(_ section: NavigationSection) -> some View {
⋮----
private func colorForSection(_ section: NavigationSection) -> Color {
```

## File: apps/control-tower-macos/Sources/Views/TelemetryStreamView.swift
```swift
public struct TelemetryStreamView: View {
@Bindable var viewModel: ControlTowerViewModel
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
private func severityBadge(_ severity: LogSeverity) -> some View {
```

## File: apps/control-tower-macos/Sources/PegasusControlTowerApp.swift
```swift
struct PegasusControlTowerApp: App {
@State private var viewModel = ControlTowerViewModel()
⋮----
var body: some Scene {
```

## File: apps/driver-app-ios/driverappios/driverappios/Models/FoundryFleetModels.swift
```swift
public struct DVIRChecklist8Point: Codable, Sendable, Equatable {
public var steeringAndHornOk: Bool
public var serviceAndParkingBrakesOk: Bool
public var tiresTreadAndPressureOk: Bool
public var lightsAndHazardsOk: Bool
public var wipersAndFluidsOk: Bool
public var emergencyKitAndExtinguisherOk: Bool
public var cargoBoxSanitationOk: Bool
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct FleetDVIRInspection: Codable, Sendable, Identifiable {
public var id: String { inspectionId ?? UUID().uuidString }
public let inspectionId: String?
public let supplierId: String
public let vehicleId: String
public let driverId: String
public let odometerKm: Int64
public let checklist: DVIRChecklist8Point
public let failedPoints: [String]
public let criticalDefectFound: Bool
public let redTagApplied: Bool
public let status: String
public let maintenanceWorkOrderId: String?
public let createdAt: String?
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct YardGateCheckEvent: Codable, Sendable, Identifiable {
public var id: String { eventId ?? UUID().uuidString }
public let eventId: String?
public let depotId: String
public let supplierId: String
public let vehicleId: String
public let licensePlate: String
public let direction: String
public let vehiclePurpose: String
public let boltSealBarcode: String?
public let sealIntact: Bool
public let assignedBaySlot: String?
public let checkedAt: String?
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct RescueToteItem: Codable, Sendable, Identifiable {
public var id: String { toteId }
public let toteId: String
public let stopSequence: Int64
public let isColdChain: Bool
public let expectedCashMinor: Int64
public let thermalExcursionMinutes: Int
public let maxAllowedExcursionMins: Int
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct MidRouteRescueRequest: Codable, Sendable {
public let cellId: String
public let supplierId: String
public let originalRouteId: String
public let brokenVehicleId: String
public let originalDriverId: String
public let rescueVehicleId: String
public let rescueDriverId: String
public let lastCompletedStopSeq: Int64
public let newRescueBoltSeal: String
public let totes: [RescueToteItem]
⋮----
public init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct MidRouteRescueResult: Codable, Sendable {
public let rescueRouteId: String
public let originalDriverLiabilityMinor: Int64
public let rescueDriverLiabilityMinor: Int64
public let transferredToteIds: [String]
public let quarantinedSpoiledToteIds: [String]
public let newRescueBoltSeal: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
public struct DispatchCheckResponse: Codable, Sendable {
public let vehicleId: String
public let isDispatchAllowed: Bool
public let redTagQuarantined: Bool
public let reason: String?
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/APIClient.swift
```swift
enum APIError: Error {
⋮----
struct ProblemDetail: Codable {
let type: String?
let title: String?
let status: Int
let detail: String?
let traceId: String?
let instance: String?
let code: String?
let messageKey: String?
let retryable: Bool?
let action: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
⋮----
func getAssignedOrders() async throws -> [Order] {
⋮----
func getRouteGeometry(
⋮----
func getManifest(date: String) async throws -> RouteManifest {
⋮----
// MARK: - Driver Earnings & Cash
⋮----
/// GET /v1/driver/earnings — lifetime totals + per-day breakdown for last 30 days
func getEarnings() async throws -> DriverEarningsResponse {
⋮----
func getPendingCollections() async throws -> [PendingCollection] {
⋮----
func getOrder(id: String) async throws -> Order {
⋮----
func submitDelivery(orderId: String, qrToken: String, latitude: Double, longitude: Double) async throws -> DeliverySubmitResponse {
⋮----
func amendOrder(request: AmendOrderRequest) async throws -> AmendOrderResponse {
⋮----
func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
⋮----
func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
⋮----
func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse {
struct Req: Encodable {
let order_id: String
let qr_token: String
⋮----
func completeOrder(orderId: String) async throws {
struct Resp: Decodable { let status: String }
⋮----
func collectCash(orderId: String, latitude: Double, longitude: Double, amountReceivedMinor: Int64? = nil) async throws -> CollectCashResponse {
⋮----
func retryFiscal(orderId: String) async throws -> CollectCashResponse {
struct Empty: Encodable {}
⋮----
func transitionState(orderId: String, newState: String) async throws -> Order {
⋮----
func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse {
⋮----
func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse {
⋮----
func markArrived(orderId: String) async throws {
struct Resp: Decodable { let status: String; let orderId: String }
⋮----
func reportShopClosed(
⋮----
struct Req: Encodable {
let order_id: String
let latitude: Double?
let longitude: Double?
let reason: String?
let photo_url: String?
let client_timestamp: String?
⋮----
func proximityUnlock(
⋮----
struct Req: Encodable {
let order_id: String
let latitude: Double
let longitude: Double
let client_timestamp: String?
let force_bypass_token: String?
⋮----
func partialOffload(
⋮----
struct Req: Encodable {
let order_id: String
let lines: [PartialOffloadLineRequest]
let client_timestamp: String?
let signed_nonce: String?
let note: String?
⋮----
func bypassOffload(orderId: String, token: String) async throws -> [String: String] {
⋮----
func confirmPaymentBypass(orderId: String, token: String) async throws -> [String: String] {
⋮----
func ackWebSocketCommand(commandId: String, traceId: String?, eventType: String) async throws {
struct AckRequest: Encodable {
let commandId: String
let traceId: String?
let eventType: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AckResponse: Decodable {
let status: String
let commandId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func depart(truckId: String) async throws -> [String: String] {
⋮----
func checkManifestGate(manifestId: String) async throws -> ManifestGateResult {
⋮----
func getFleetManifest() async throws -> [String: AnyDecodable] {
⋮----
struct OpenFiscalResponse: Decodable {
let openFiscalCount: Int64
let orderIds: [String]?
let cashBagFrozen: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func getOpenFiscal() async throws -> OpenFiscalResponse {
⋮----
func returnComplete(truckId: String) async throws -> [String: String] {
⋮----
struct CashReconciliationRow: Decodable {
let reconciliationId: String
let expectedCashMinor: Int64
let declaredCashMinor: Int64
let differenceMinor: Int64
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CashReconciliationsResponse: Decodable {
let reconciliations: [CashReconciliationRow]
⋮----
func listCashReconciliations() async throws -> CashReconciliationsResponse {
⋮----
func submitCashReconciliation(declaredCashMinor: Int64, driverNote: String?) async throws -> CashReconciliationRow {
struct Body: Encodable {
let declaredCashMinor: Int64
let driverNote: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func getReturnGoods() async throws -> ReturnGoodsResponse {
⋮----
func setAvailability(available: Bool, reason: String? = nil, note: String? = nil) async throws {
struct Req: Encodable { let available: Bool; let reason: String?; let note: String? }
struct Resp: Decodable { let status: String }
⋮----
func getAvailability() async throws -> [String: AnyDecodable] {
⋮----
func updateAvailability(payload: [String: AnyEncodable]) async throws -> [String: String] {
⋮----
func reorderStops(routeId: String, orderSequence: [String]) async throws -> RouteReorderResponse {
⋮----
func requestEarlyComplete(reason: String, note: String) async throws -> EarlyCompleteRequestResponse {
struct Req: Encodable { let reason: String; let note: String }
⋮----
func requestRescue(reason: String, note: String) async throws -> [String: String] {
struct Req: Encodable { let reason: String; let note: String }
⋮----
func respondRescue(rescueId: String, brokenDriverId: String, accept: Bool) async throws -> [String: String] {
struct Req: Encodable {
let rescue_id: String
let broken_driver_id: String
let accept: Bool
⋮----
func reassignHandshake(orderId: String) async throws -> [String: String] {
⋮----
func markCreditDelivery(
⋮----
func reportMissingItems(
⋮----
struct Req: Encodable {
let order_id: String
let missing_items: [MissingItemRequest]
let photo_url: String?
let note: String?
⋮----
func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String? = nil) async throws -> SplitPaymentResponse {
struct Req: Encodable {
let order_id: String
let cash_minor: Int64
let card_minor: Int64
let currency: String?
⋮----
func getSupplyTransfers() async throws -> SupplyTransfersResponse {
⋮----
func arriveSupplyTransfer(transferId: String, latitude: Double, longitude: Double) async throws -> ArriveSupplyTransferResponse {
struct Body: Encodable {
let latitude: Double
let longitude: Double
⋮----
func getDriverProfile() async throws -> DriverProfileResponse {
⋮----
func getDriverHistory() async throws -> DriverHistoryResponse {
⋮----
func getPulse() async throws -> PulseResponse {
⋮----
private struct EmptyBody: Encodable {}
⋮----
func get<T: Decodable>(_ path: String) async throws -> T {
⋮----
func post<B: Encodable, T: Decodable>(
⋮----
private func patch<B: Encodable, T: Decodable>(
⋮----
private func deterministicIdempotencyKey(action: String, orderId: String) -> String {
⋮----
func rawRequest(
⋮----
private func buildRequest(path: String, method: String, authenticated: Bool = true) throws -> URLRequest {
⋮----
private func dataWithFallback(for request: URLRequest) async throws -> (Data, URLResponse) {
⋮----
private func execute<T: Decodable>(_ request: URLRequest, isRetry: Bool = false) async throws -> T {
⋮----
private static func parseProblemDetail(data: Data, response: HTTPURLResponse) -> ProblemDetail? {
⋮----
private func attemptTokenRefresh() async -> String? {
⋮----
struct RefreshResponse: Decodable { let token: String }
⋮----
private actor DriverHTTPCoalescer {
private var inFlightGETs: [String: Task<(Data, URLResponse), Error>] = [:]
private var getCache: [String: (data: Data, response: URLResponse, expiresAt: Date)] = [:]
private var refreshTask: Task<String?, Never>?
private let cacheTTL: TimeInterval = 2.0
⋮----
func fetchOrCoalesce(
⋮----
func invalidateCache() {
⋮----
func refreshToken(using operation: @Sendable @escaping () async -> String?) async -> String? {
⋮----
struct AnyEncodable: Encodable {
private let _encode: (Encoder) throws -> Void
let value: Any
⋮----
init(_ wrapped: any Encodable) {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct AnyDecodable: Decodable {
let value: Any
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ManifestGateResponse: Decodable {
let cleared: Bool
let state: String?
let manifestId: String?
let error: String?
let message: String?
let explain: StatusExplain?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ManifestGateResult {
let gate: ManifestGateResponse
let explain: StatusExplain?
⋮----
struct EarlyCompleteRequestResponse: Decodable {
let status: String
let orderCount: Int
let orderIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationItemRequest: Encodable {
let skuId: String
let originalQty: Int64
let proposedQty: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationProposalResponse: Decodable {
let status: String
let proposalId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RouteReorderResponse: Decodable {
let status: String
let routeId: String
let stopCount: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PartialOffloadLineRequest: Encodable {
let sku: String
let delivered_qty: Int64
let remaining_qty: Int64
let reason: String?
⋮----
private struct DepartRequest: Encodable {
let truckId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
private struct ReturnCompleteRequest: Encodable {
let truckId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyTransferRow: Decodable, Identifiable {
let transferId: String
let warehouseId: String
let supplyRequestId: String?
let state: String
let totalVolumeVu: Double
⋮----
var id: String { transferId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyTransfersResponse: Decodable {
let transfers: [SupplyTransferRow]
⋮----
struct ArriveSupplyTransferResponse: Decodable {
let transferId: String
let state: String
let eventType: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReturnGoodsLine: Decodable, Identifiable {
var id: String { returnId }
let returnId: String
let orderId: String
let skuId: String
let productName: String
let quantity: Int64
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReturnGoodsResponse: Decodable {
let items: [ReturnGoodsLine]
let totalUnits: Int64
let lineCount: Int
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/APIClient+FoundryFleet.swift
```swift
func submitDVIR(inspection: FleetDVIRInspection) async throws -> FleetDVIRInspection {
⋮----
func recordGateEvent(event: YardGateCheckEvent) async throws -> YardGateCheckEvent {
⋮----
func executeMidRouteRescue(request: MidRouteRescueRequest) async throws -> MidRouteRescueResult {
⋮----
func checkVehicleDispatch(vehicleId: String) async throws -> DispatchCheckResponse {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/AuthService.swift
```swift
private enum AuthNamespace {
static let primaryService = "com.pegasusx.driver"
static let primaryPrefix = "com.pegasusx.driver"
⋮----
private enum KeychainHelper {
static func save(_ value: String, forKey key: String) {
⋮----
static func load(forKey key: String) -> String? {
⋮----
static func saveDouble(_ value: Double, forKey key: String) {
⋮----
static func loadDouble(forKey key: String) -> Double {
⋮----
static func delete(forKey key: String) {
⋮----
private static func loadFromService(_ service: String, forKey key: String) -> String? {
⋮----
private static func deleteFromService(_ service: String, forKey key: String) {
⋮----
final class TokenStore {
static let shared = TokenStore()
⋮----
var token: String?
var userId: String?
var driverName: String?
var vehicleType: String?
var licensePlate: String?
var supplierId: String?
var vehicleId: String?
var vehicleClass: String?
var maxVolumeVU: Double = 0
var warehouseId: String?
var warehouseName: String?
var warehouseLat: Double = 0
var warehouseLng: Double = 0
var homeNodeType: String?
var homeNodeId: String?
var driverMode: String?
var factoryId: String?
var factoryName: String?
var factoryLat: Double = 0
var factoryLng: Double = 0
⋮----
var isAuthenticated: Bool { token != nil }
⋮----
var isFactoryScopedDriver: Bool {
⋮----
private let tokenKey = "\(AuthNamespace.primaryPrefix).token"
private let userKey  = "\(AuthNamespace.primaryPrefix).userId"
private let nameKey  = "\(AuthNamespace.primaryPrefix).driverName"
private let vehicleKey = "\(AuthNamespace.primaryPrefix).vehicleType"
private let plateKey = "\(AuthNamespace.primaryPrefix).licensePlate"
private let vehicleIdKey = "\(AuthNamespace.primaryPrefix).vehicleId"
private let vehicleClassKey = "\(AuthNamespace.primaryPrefix).vehicleClass"
private let maxVolumeVUKey = "\(AuthNamespace.primaryPrefix).maxVolumeVU"
private let warehouseIdKey = "\(AuthNamespace.primaryPrefix).warehouseId"
private let warehouseNameKey = "\(AuthNamespace.primaryPrefix).warehouseName"
private let warehouseLatKey = "\(AuthNamespace.primaryPrefix).warehouseLat"
private let warehouseLngKey = "\(AuthNamespace.primaryPrefix).warehouseLng"
private let homeNodeTypeKey = "\(AuthNamespace.primaryPrefix).homeNodeType"
private let homeNodeIdKey = "\(AuthNamespace.primaryPrefix).homeNodeId"
private let driverModeKey = "\(AuthNamespace.primaryPrefix).driverMode"
private let factoryIdKey = "\(AuthNamespace.primaryPrefix).factoryId"
private let factoryNameKey = "\(AuthNamespace.primaryPrefix).factoryName"
private let factoryLatKey = "\(AuthNamespace.primaryPrefix).factoryLat"
private let factoryLngKey = "\(AuthNamespace.primaryPrefix).factoryLng"
⋮----
private init() {
⋮----
private func migrateString(_ key: String) -> String? {
⋮----
private func loadString(key: String) -> String? {
⋮----
private func loadDouble(key: String) -> Double {
⋮----
private func saveString(_ value: String, key: String) {
⋮----
private func saveDouble(_ value: Double, key: String) {
⋮----
func save(response: AuthResponse) {
⋮----
func persistVehicleInfo() {
⋮----
/// Update only the token (used after silent refresh).
func updateToken(_ newToken: String) {
⋮----
func logout() {
⋮----
final class DriverSocketState {
struct OutdatedNotice {
let message: String
let blockedEventType: String?
let requiredSchemaVersion: Int?
let clientSchemaVersion: Int?
⋮----
struct DriverEvent {
let type: String
let orderId: String?
let response: String?
let bypassToken: String?
let attemptId: String?
let status: String?
let state: String?
let rescueId: String?
let brokenDriverId: String?
⋮----
init(
⋮----
private struct DriverEnvelope: Decodable {
let type: String
let event_id: String?
let order_id: String?
let response: String?
let bypass_token: String?
let attempt_id: String?
let command_id: String?
let trace_id: String?
let required_schema_version: Int?
let client_schema_version: Int?
let blocked_event_type: String?
let message: String?
let status: String?
let state: String?
let rescue_id: String?
let broken_driver_id: String?
⋮----
static let shared = DriverSocketState()
⋮----
enum SocketConnectionState: Equatable {
⋮----
var outdatedNotice: OutdatedNotice?
var lastEvent: DriverEvent?
var eventSequence = 0
var reconnectEpoch = 0
var connectionState: SocketConnectionState = .disconnected
⋮----
private let wsSchemaVersion = 2
private var lastEventID: String?
private var webSocketTask: URLSessionWebSocketTask?
private var reconnectWorkItem: DispatchWorkItem?
private var shouldReconnect = false
private var reconnectAttempt = 0
private var hasConnectedOnce = false
private var currentBaseURL: String?
private var currentToken: String?
⋮----
private init() {}
⋮----
func startMonitoring(baseURL: String, token: String) {
⋮----
func stopMonitoring() {
⋮----
func publishOutdatedEvent(
⋮----
func clearOutdatedNotice() {
⋮----
private func establishConnection() {
⋮----
private func listenForMessages() {
⋮----
private func handleIncoming(_ message: URLSessionWebSocketTask.Message) {
⋮----
private func publishDriverEvent(_ envelope: DriverEnvelope) {
⋮----
private func scheduleReconnect() {
⋮----
private func reconnectDelaySeconds(attempt: Int, base: TimeInterval, maxDelay: TimeInterval, retryAfter: TimeInterval = 0) -> TimeInterval {
⋮----
struct LoginRequest: Encodable {
var phone: String = ""
var pin: String = ""
var idToken: String = ""
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct AuthResponse: Codable {
let token: String
let userId: String
let driverId: String
let name: String
let vehicleType: String
let licensePlate: String
let supplierId: String
let vehicleId: String
let vehicleClass: String
let maxVolumeVU: Double
let firebaseToken: String?
let warehouseId: String?
let warehouseName: String?
let warehouseLat: Double?
let warehouseLng: Double?
let homeNodeType: String?
let homeNodeId: String?
let driverMode: String?
let factoryId: String?
let factoryName: String?
let factoryLat: Double?
let factoryLng: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DriverProfileResponse: Codable {
let driverId: String
let name: String
let phone: String
let driverType: String
let vehicleType: String
let licensePlate: String
let isActive: Bool
let supplierId: String
let vehicleId: String
let vehicleClass: String
let maxVolumeVU: Double
let warehouseId: String?
let warehouseName: String?
let warehouseLat: Double?
let warehouseLng: Double?
let homeNodeType: String?
let homeNodeId: String?
let driverMode: String?
let factoryId: String?
let factoryName: String?
let factoryLat: Double?
let factoryLng: Double?
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/driver-app-ios/driverappios/driverappios/Services/FleetServiceLive.swift
```swift
final class FleetServiceLive: FleetServiceProtocol {
⋮----
static let shared = FleetServiceLive()
⋮----
private let api = APIClient.shared
⋮----
private lazy var modelContainer: ModelContainer? = {
⋮----
func fetchActiveMissions(routeId: String) async throws -> [Mission] {
⋮----
func deliverOrder(orderId: String, scannedToken: String) async throws {
⋮----
func validateQR(orderId: String, scannedToken: String) async throws -> ValidateQRResponse {
⋮----
func confirmOffload(orderId: String) async throws -> ConfirmOffloadResponse {
⋮----
func scanDeliveryQR(orderId: String, qrToken: String) async throws -> DeliveryScanQRResponse {
⋮----
func completeOrder(orderId: String) async throws {
⋮----
func collectCash(orderId: String, amountReceivedMinor: Int64? = nil) async throws -> CollectCashResponse {
⋮----
func retryFiscal(orderId: String) async throws -> CollectCashResponse {
⋮----
func fetchOrderLineItems(orderId: String) async throws -> [LineItem] {
⋮----
func amendOrder(
⋮----
func verifyHandshake(orderId: String, token: String, latitude: Double, longitude: Double) async throws -> VerifyHandshakeResponse {
⋮----
func updateOrderDuringDelivery(orderId: String, latitude: Double, longitude: Double) async throws -> UpdateOrderDuringDeliveryResponse {
// G1.C: do not hit network — backend has no durable writer. Use amend / missing-items.
struct MidDeliveryDisabled: LocalizedError {
var errorDescription: String? {
⋮----
func markCreditDelivery(orderId: String, photoProofUrl: String? = nil, signatureUrl: String? = nil) async throws -> [String: String] {
⋮----
func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64, currency: String? = nil) async throws -> SplitPaymentResponse {
⋮----
func requestRescue(reason: String, note: String) async throws {
⋮----
func respondRescue(rescueId: String, brokenDriverId: String, accept: Bool) async throws {
⋮----
func reassignHandshake(orderId: String) async throws {
⋮----
private func enqueueOfflineDelivery(
⋮----
func flushOfflineQueue() async {
⋮----
private func requireCurrentLocation() async throws -> CLLocationCoordinate2D {
⋮----
enum FleetServiceError: LocalizedError {
⋮----
var errorDescription: String? {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/FleetDVIRViewModel.swift
```swift
final class FleetDVIRViewModel {
⋮----
var supplierId: String = "supplier-tashkent-1"
var vehicleId: String = "vh-isuzu-01"
var driverId: String = "driver-uz-09"
var odometerKm: Int64 = 125400
⋮----
var checklist = DVIRChecklist8Point()
⋮----
var isSubmitting: Bool = false
var isRedTagQuarantined: Bool = false
var redTagReason: String? = nil
var workOrderId: String? = nil
var successMessage: String? = nil
var errorMessage: String? = nil
⋮----
private let apiClient: APIClient
⋮----
init(apiClient: APIClient = .shared) {
⋮----
func toggleSteeringAndHorn() {
⋮----
func toggleBrakes() {
⋮----
func toggleTires() {
⋮----
func toggleLights() {
⋮----
func toggleWipers() {
⋮----
func toggleEmergencyKit() {
⋮----
func toggleCargoSanitation() {
⋮----
var failedPoints: [String] {
⋮----
var hasCriticalDefect: Bool {
⋮----
func submitInspection() async {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/TelemetryViewModel.swift
```swift
final class TelemetryViewModel {
⋮----
var isLive = false
var isCommandSocketLive = false
⋮----
private let service: TelemetryServiceProtocol
private let driverId: String
private var isTrackingNetwork = false
private var lastSentLocation: CLLocationCoordinate2D?
⋮----
convenience init() {
⋮----
init(
⋮----
// MARK: - Connect / Disconnect
⋮----
func start() async {
⋮----
.replacingOccurrences(of: "http://", with: "ws://")
⋮----
private func trackNetworkState() {
⋮----
func stop() {
⋮----
func syncLiveFlags(networkSatisfied: Bool = true) {
⋮----
func sendLocation(_ coordinate: CLLocationCoordinate2D, accuracy: Double?) {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/HomeView.swift
```swift
struct HomeView: View {
@Bindable var vm: FleetViewModel
let onOpenMap: () -> Void
var onHandoffNavigate: ((HandoffDestination) -> Void)? = nil
@Environment(\.modelContext) private var modelContext
@Environment(\.horizontalSizeClass) private var sizeClass
⋮----
@State private var homeVM = DriverHomeViewModel()
@State private var driverSocketState = DriverSocketState.shared
⋮----
var body: some View {
⋮----
private var fieldEarningsRow: some View {
⋮----
private var fieldHistoryPeek: some View {
⋮----
private func statusChip(icon: String, label: String, active: Bool, tint: Color? = nil) -> some View {
⋮----
struct PulseModifier: ViewModifier {
@State private var isPulsing = false
⋮----
func body(content: Content) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/ProfileView.swift
```swift
struct ProfileView: View {
@Environment(\.modelContext) private var modelContext
@Bindable var vm: FleetViewModel
@State private var profileVM = ProfileViewModel()
⋮----
var body: some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/RidesListView.swift
```swift
struct RidesListView: View {
@Bindable var vm: FleetViewModel
@State private var ridesVM = RidesListViewModel()
@State private var driverSocketState = DriverSocketState.shared
var onRequestEarlyComplete: ((String, String) -> Void)?
⋮----
var body: some View {
```

## File: apps/factory-app-ios/FactoryApp/Layout/FactoryAdaptiveShell.swift
```swift
struct FocusedFactorySectionKey: FocusedValueKey {
⋮----
var factorySection: Binding<FactorySection?>? {
⋮----
struct FactoryAdaptiveShell: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var sidebarSelection: FactorySection? = .dashboard
@State private var isSidebarExpanded = true
@State private var compactTab: FactoryCompactTab = .dashboard
@State private var pendingTransferFilter: String?
@State private var morePath: [FactorySection] = []
⋮----
private var network = NetworkLifecycleEngine.shared
⋮----
var body: some View {
⋮----
private var regularShell: some View {
⋮----
private var sidebarGroups: [(title: String, items: [CollapsibleSidebarItem<FactorySection>])] {
⋮----
private func items(for sections: [FactorySection]) -> [CollapsibleSidebarItem<FactorySection>] {
⋮----
private var compactShell: some View {
⋮----
private func openSecondarySection(_ section: FactorySection) {
⋮----
private func sectionView(_ section: FactorySection) -> some View {
```

## File: apps/factory-app-ios/FactoryApp/Views/Components/AddressLocationField.swift
```swift
struct AddressLocationValue: Equatable {
var address: String = ""
var lat: Double = 0
var lng: Double = 0
var placeId: String?
⋮----
struct AddressLocationField: View {
@Binding var value: AddressLocationValue
var label: String = "Address"
⋮----
@State private var vm = AddressLocationViewModel()
@FocusState private var focused: Bool
⋮----
private var pinned: Bool {
⋮----
var body: some View {
```

## File: apps/payload-app-ios/payload-app-ios/Services/WebSocketClient.swift
```swift
final class WebSocketClient {
private(set) var online = false
⋮----
var onFrame: ((WsMessage) -> Void)?
⋮----
var onReconnect: (() -> Void)?
⋮----
private var task: URLSessionWebSocketTask?
private var session: URLSession
private var token: String?
private var lastEventId: String?
private var receiveTask: Task<Void, Never>?
private var heartbeatTask: Task<Void, Never>?
private var syncCoalesceTask: Task<Void, Never>?
private var reconnectTask: Task<Void, Never>?
private var reconnectAttempt = 0
private let decoder = JSONDecoder()
⋮----
init() {
⋮----
func connect(token: String) {
⋮----
func disconnect() {
⋮----
private func openSocket() {
⋮----
private func startHeartbeat(_ t: URLSessionWebSocketTask) {
⋮----
private func startReceiving(_ t: URLSessionWebSocketTask) {
⋮----
private func emitCoalescedSync(_ frame: WsMessage) {
⋮----
private func handleDisconnect() {
⋮----
private func reconnectDelaySeconds(attempt: Int, base: TimeInterval, maxDelay: TimeInterval) -> TimeInterval {
```

## File: apps/payload-app-ios/payload-app-ios/ViewModels/HomeViewModel.swift
```swift
struct ExceptionSheetTarget: Identifiable, Equatable {
let id: String
⋮----
struct ReDispatchSheetTarget: Identifiable, Equatable {
let id: String
⋮----
final class HomeViewModel {
private func deterministicQueueActionId(action: String, entityId: String) -> String {
⋮----
private(set) var trucks: [Truck] = []
private(set) var selectedTruckId: String?
var sidebarSelectionId: String?
private(set) var loadingTrucks = false
⋮----
private(set) var manifest: Manifest?
private(set) var orders: [LiveOrder] = []
private(set) var selectedOrderId: String?
⋮----
private(set) var checkedItems: Set<String> = []
private(set) var sealedOrderIds: Set<String> = []
private(set) var dispatchCodes: [String: String] = [:]
private(set) var postSealOrderId: String?
private(set) var postSealCountdown: Int = 0
private(set) var manifestSealed = false
⋮----
private(set) var loadingManifest = false
private(set) var loadingOrders = false
private(set) var startingLoading = false
private(set) var sealingOrderId: String?
private(set) var sealingManifest = false
⋮----
private(set) var error: String?
private(set) var errorExplain: StatusExplain?
private(set) var barcodeScanMessage: String?
⋮----
private(set) var pulseEvents: [PulseEvent] = []
private(set) var pulseLoading = false
private(set) var pulseError: String?
⋮----
var showInjectSheet = false
var showProductScanner = false
var showInboundReturns = false
var exceptionTarget: ExceptionSheetTarget?
var reDispatchTarget: ReDispatchSheetTarget?
var isShowingEscalatedAlert = false
var isShowingHandoffAlert = false
⋮----
private(set) var notifications: [NotificationItem] = []
private(set) var unreadCount: Int = 0
private(set) var online: Bool = false
private(set) var queuedActions: Int = 0
var showNotificationsPanel: Bool = false
private(set) var queuedNoticeMessage: String?
private(set) var syncCompleteMessage: String?
⋮----
private(set) var exceptionLoadingOrderId: String?
⋮----
private(set) var escalatedMessage: String?
⋮----
private(set) var injectingOrder = false
⋮----
private(set) var reDispatchOrderId: String?
private(set) var loadingRecommendations = false
private(set) var recommendations: RecommendReassignResponse?
private(set) var reassigning = false
private(set) var reportingMissingItems = false
private(set) var missingItemsReportedMessage: String?
var showExceptionsPanel = false
private(set) var loadingExceptions = false
private(set) var manifestExceptions: [ManifestExceptionRow] = []
private var sealedOrdersByTruck: [String: Set<String>] = [:]
private(set) var batchReadyManifestIds: [String] = []
private(set) var batchSealing = false
private(set) var batchSealFailures: [SealCompletedManifestResult] = []
private(set) var handoffNavigationMessage: String?
⋮----
let bannerAutoDismissNanoseconds: UInt64
⋮----
private let api: APIClient
private let ws: WebSocketClient
private let queue: OfflineQueue
private var countdownTask: Task<Void, Never>?
private var barcodeBannerTask: Task<Void, Never>?
private var missingItemsBannerTask: Task<Void, Never>?
private var queuedNoticeBannerTask: Task<Void, Never>?
private var syncCompleteBannerTask: Task<Void, Never>?
⋮----
init(
⋮----
deinit {
⋮----
func presentExceptionSheet(for orderId: String) {
⋮----
func dismissExceptionSheet() {
⋮----
func postBarcodeScanMessage(_ message: String) {
⋮----
func postMissingItemsReportedMessage(_ message: String) {
⋮----
func postQueuedNoticeMessage(_ message: String) {
⋮----
func postSyncCompleteMessage(_ message: String) {
⋮----
private func setEscalatedAlert(_ message: String) {
⋮----
private func setHandoffAlert(_ message: String) {
⋮----
func hydrateState(
⋮----
func refreshTrucks(silent: Bool = false) async {
⋮----
func selectTruck(_ truckId: String) async {
⋮----
func refreshManifest(silent: Bool = false) async {
⋮----
private func reloadSelectedTruck(silent: Bool = false) async {
⋮----
func refreshBatchReadyManifests() async {
⋮----
func finalizeBatchSeal() async {
⋮----
func handleHandoffLink(_ link: String) async {
⋮----
func clearHandoffNavigationMessage() {
⋮----
private func loadManifest(for truckId: String, silent: Bool = false) async {
⋮----
private func loadOrders(for truckId: String, silent: Bool = false) async {
⋮----
func selectOrder(_ orderId: String) {
⋮----
func toggleItem(_ lineItemId: String) {
⋮----
func clearBarcodeScanMessage() {
⋮----
func onBarcodeScanned(_ ean: String) async {
⋮----
func canSealOrder(_ orderId: String) -> Bool {
⋮----
func sealSelectedOrder() async {
⋮----
private func startCountdown() {
⋮----
private func advanceAfterCountdown() {
⋮----
func dismissCountdown() {
⋮----
private func cancelCountdown() {
⋮----
func startLoading() async {
⋮----
func sealManifest() async {
⋮----
func startNewManifest() async {
⋮----
func clearError() {
⋮----
func clearEscalatedMessage() {
⋮----
func clearMissingItemsReportedMessage() {
⋮----
func toggleExceptionsPanel() {
⋮----
func loadManifestExceptions() async {
⋮----
func reportMissingItems(orderId: String) async {
⋮----
func reportException(orderId: String, reason: String) async {
⋮----
func injectOrder(_ orderId: String) async {
⋮----
func openReDispatch(orderId: String) async {
⋮----
func closeReDispatch() {
⋮----
func reassignTo(_ newDriverId: String, isPartial: Bool = false) async {
⋮----
func bootstrapPhase6(token: String) async {
⋮----
func refreshPulse() async {
⋮----
func disconnectPhase6() {
⋮----
func loadNotifications() async {
⋮----
func toggleNotificationsPanel() {
⋮----
func markNotificationRead(_ id: String) {
⋮----
func markAllNotificationsRead() {
⋮----
func clearSyncCompleteMessage() {
⋮----
func clearQueuedNoticeMessage() {
⋮----
func registerFcmToken(_ token: String) {
⋮----
private func handleFrame(_ frame: WsMessage) {
⋮----
private func copyNotification(_ item: NotificationItem, readAt: String?) -> NotificationItem {
⋮----
private func nowIso() -> String {
⋮----
private func reconcileSession() async {
⋮----
private func recoverInFlightMutations() {
⋮----
private func flushQueue() async {
⋮----
// MARK: - Helpers
⋮----
/// Manifest is a struct of `let` constants; rebuild it with the new state.
private func mutateState(_ m: Manifest, to state: String) -> Manifest {
⋮----
private func describe(_ e: Error) -> String {
```

## File: apps/payload-app-ios/payload-app-ios/Views/Components/PulseStrip.swift
```swift
struct PulseStrip: View {
let events: [PulseEvent]
let loading: Bool
var error: String? = nil
⋮----
var body: some View {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private func formatTimestamp(_ iso: String) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/Dashboard/QuickReorderSection.swift
```swift
struct QuickReorderSection: View {
let reorderProducts: [Product]
⋮----
var body: some View {
⋮----
struct QuickReorderItem: View {
let product: Product
@Environment(CartManager.self) private var cart
@State private var addedToast = false
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/FloatingActiveOrdersBar.swift
```swift
struct FloatingActiveOrdersBar: View {
let activeOrders: [Order]
let onTap: () -> Void
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var pulse = false
⋮----
var body: some View {
⋮----
// ETA of first order
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Components/ProductCardView.swift
```swift
struct ProductCardView: View {
let product: Product
var onTap: (() -> Void)?
⋮----
@State private var appeared = false
⋮----
var body: some View {
⋮----
private var productPlaceholder: some View {
⋮----
private func tagPill(_ text: String, color: Color, textColor: Color) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/AppModels.swift
```swift
struct User: Codable, Identifiable {
let id: String
let name: String
let company: String
let email: String?
let avatarURL: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerMembershipDTO: Codable, Identifiable, Hashable {
var id: String { "\(retailerId)|\(userId)" }
let userId: String
let retailerId: String
let retailerRole: String
let name: String?
let phone: String?
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AuthResponse: Codable {
let token: String
let tokenType: String?
let user: User?
let firebaseToken: String?
let isConfigured: Bool?
let memberships: [RetailerMembershipDTO]?
let expiresInSec: Int?
let retailerId: String?
let refreshToken: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var isPendingOrgSelect: Bool {
⋮----
struct SelectOrgRequest: Encodable {
let retailerId: String
enum CodingKeys: String, CodingKey { case retailerId = "retailer_id" }
⋮----
struct DemandForecast: Codable, Identifiable, Hashable {
let id: String
let productId: String
let productName: String
let predictedQuantity: Int
let confidence: Double
let reasoning: String
let suggestedOrderDate: String
let blocked: Bool?
let blockedReason: String?
let label: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var confidencePercent: String {
⋮----
var isBlocked: Bool {
⋮----
struct RetailerAILineItem: Codable, Hashable {
let sku: String
let name: String?
let quantity: Int64
let unitPriceMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerAIPrediction: Codable, Identifiable, Hashable {
var id: String { orderId }
let orderId: String
let supplierId: String?
let orderSource: String?
let confirmationStatus: String?
let requestedDeliveryDate: String?
let autoConfirmAt: String?
let totalMinor: Int64
let currency: String
let derivedFromOrderId: String?
let updatedAt: String
let lineItems: [RetailerAILineItem]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
var title: String {
⋮----
var quantity: Int64 {
⋮----
var statusLabel: String {
⋮----
var deliveryLabel: String {
⋮----
var formattedTotal: String {
⋮----
struct RetailerAIPredictionsResponse: Codable {
let items: [RetailerAIPrediction]
⋮----
struct CartItem: Identifiable, Hashable {
let id: String
let product: Product
let variant: Variant
var quantity: Int
⋮----
var totalPriceMinor: Int64 {
⋮----
struct APIResponse<T: Codable>: Codable {
let data: T?
let error: String?
let message: String?
⋮----
struct DeviceTokenPayload: Codable, Equatable {
let token: String
let platform: String
⋮----
struct RetailerProfileRequest: Encodable {
let name: String?
let company: String?
let phone: String?
let location: String?
let regionId: String?
let receivingWindowOpen: String?
let receivingWindowClose: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerProfileResponse: Decodable, Identifiable {
let id: String
let name: String
let phone: String
let company: String
let location: String?
let regionId: String?
let taxId: String?
let status: String
let receivingWindowOpen: String?
let receivingWindowClose: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RetailerCapabilityPack: Codable, Identifiable {
let id: String
let name: String
let description: String
let hardDeps: [String]?
let softDeps: [String]?
let alwaysOn: Bool?
let enabled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerCapabilitiesResponse: Codable {
let retailerId: String
let capabilities: [String]
let packs: [RetailerCapabilityPack]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerMeResponse: Codable {
let userId: String
let retailerId: String
let retailerOrgId: String
let retailerRole: String
let name: String
let permissions: [String]
let capabilities: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerCapabilityMutationResponse: Codable {
let status: String?
let packId: String?
let enabled: Bool?
let message: String?
let capabilities: [String]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerLocationDTO: Codable, Identifiable {
let locationId: String
let retailerId: String?
let name: String
let deliveryAddress: String?
let lat: Double?
let lng: Double?
let isPrimary: Bool
let isActive: Bool
⋮----
var id: String { locationId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerLocationsResponse: Codable {
let retailerId: String?
let activeLocationId: String?
let items: [RetailerLocationDTO]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOrgMember: Codable, Identifiable {
let userId: String
let retailerId: String
let name: String
let phone: String
let retailerRole: String
let isOwner: Bool
let isActive: Bool
⋮----
var id: String { userId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOrgMembersResponse: Codable {
let retailerId: String
let items: [RetailerOrgMember]
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FamilyMemberRequest: Encodable {
let name: String
let phone: String?
let photoUrl: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FamilyMemberResponse: Codable, Identifiable {
let memberId: String
let retailerId: String?
let name: String
let phone: String?
let photoUrl: String?
let createdAt: String?
⋮----
var id: String { memberId }
⋮----
var nickname: String { name }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct FamilyMembersListResponse: Codable {
let members: [FamilyMemberResponse]
let familyWrites: String?
let migrate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FamilyMigrateItem: Codable, Identifiable {
let memberId: String
let userId: String
let phone: String
let name: String
let retailerRole: String
let tempPassword: String?
⋮----
var id: String { userId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FamilyMigrateSkipped: Codable, Identifiable {
let memberId: String
let phone: String?
let reason: String
⋮----
var id: String { memberId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FamilyMigrateResult: Codable {
let retailerId: String
let migrated: [FamilyMigrateItem]
let skipped: [FamilyMigrateSkipped]
let familyRemaining: Int
let familyWrites: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderSkip: Codable {
let sku: String?
let reason: String
⋮----
struct AutoOrderPlacedOrder: Codable, Identifiable {
let orderId: String
let supplierId: String?
let lineCount: Int
let totalMinor: Int64?
let skus: [String]?
⋮----
var id: String { orderId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderRun: Codable, Identifiable {
let runId: String
let retailerId: String
let startedAt: String
let finishedAt: String?
let mode: String
let draftLines: Int
let placedLines: Int?
let placedOrders: [AutoOrderPlacedOrder]?
let skipped: [AutoOrderSkip]?
let status: String
let message: String?
let suggestionsSeen: Int?
let scheduleBucket: String?
let candidateSource: String?
⋮----
var id: String { runId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AutoOrderRunsResponse: Codable {
let items: [AutoOrderRun]
⋮----
struct RetailerReorderSuggestion: Codable, Identifiable {
let sku: String
let suggestedQty: Int64
let adjustedDemandPerDay: Double?
let currentStock: Int64?
let inFlightQty: Int64?
let safetyStock: Double?
let status: String?
let sources: [String]?
let sellThroughVelocity: Double?
let baseDemandPerDay: Double?
⋮----
var id: String { sku }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerReorderSuggestionsResponse: Codable {
let items: [RetailerReorderSuggestion]
⋮----
struct CartSyncItem: Codable {
let cartId: String?
let skuId: String
let supplierId: String
let quantity: Int64
let unitPrice: Int64
let currency: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CartSyncRequest: Encodable {
let items: [CartSyncItem]
⋮----
struct CartSyncResponse: Codable {
let items: [CartSyncItem]
let total: Int
⋮----
struct RetailerSupplierResponse: Codable, Identifiable {
let id: String
let name: String
let logoUrl: String
let category: String
let primaryCategoryId: String?
let operatingCategoryIds: [String]?
let operatingCategoryNames: [String]?
let orderCount: Int64
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct OrderTimelineEntry: Codable, Identifiable {
var id: String { transitionId }
let transitionId: String
let orderId: String
let previousStatus: String?
let newStatus: String
let reason: String?
let actorRole: String?
let actorId: String?
let eventKind: String?
let metadata: [String: String]?
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct OrderTimelineResponse: Codable {
let orderId: String
let items: [OrderTimelineEntry]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreditProfile: Decodable {
let retailerId: String
let supplierId: String
let creditLimitMinor: Int64
let currentBalanceMinor: Int64
let availableCreditMinor: Int64
let riskScore: Int64?
let riskTier: String?
let delinquencyCount: Int64?
let status: String
let version: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct LoyaltyTierView: Decodable {
let enrolled: Bool
let tier: String
let lifetimePoints: Int64
let availablePoints: Int64
let nextTier: String
let pointsToNext: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ActiveOrderView.swift
```swift
struct ActiveOrderView: View {
@State private var vm = ActiveOrderViewModel()
⋮----
var body: some View {
⋮----
private func activeOrderCard(_ order: Order) -> some View {
⋮----
private var emptyState: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CartView.swift
```swift
struct CartView: View {
@Environment(CartManager.self) private var cart
@Environment(\.dismiss) private var dismiss
⋮----
@State private var showCheckout = false
⋮----
var body: some View {
⋮----
private var cartBottomBar: some View {
⋮----
private var emptyCartView: some View {
⋮----
private struct CartItemRowView: View {
let item: CartItem
let onUpdateQuantity: (Int) -> Void
let onRemove: () -> Void
⋮----
@State private var quantity: Int
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CategoryProductsView.swift
```swift
struct CategoryProductsView: View {
let category: ProductCategory
@Environment(CartManager.self) private var cart
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = CategoryProductsViewModel()
⋮----
var body: some View {
⋮----
private var emptyState: some View {
⋮----
private struct CategoryProductRowView: View {
let product: Product
let externalEnabled: Bool
let isToggling: Bool
let onToggleAutoOrder: (Bool) -> Void
let onSelect: () -> Void
⋮----
@State private var isAutoOrderEnabled: Bool
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/FutureDemandView.swift
```swift
struct FutureDemandView: View {
@Environment(\.dismiss) private var dismiss
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = FutureDemandViewModel()
⋮----
var body: some View {
⋮----
private var headerCard: some View {
⋮----
private func miniStat(value: String, label: String) -> some View {
⋮----
private func predictionCard(_ item: RetailerAIPrediction) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/HistoryView.swift
```swift
struct HistoryView: View {
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = HistoryViewModel()
⋮----
var body: some View {
⋮----
private var statusFilters: some View {
⋮----
private func filterChip(_ title: String, status: OrderStatus?) -> some View {
⋮----
private var emptyState: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/HqView.swift
```swift
struct HqView: View {
@State private var vm = HqViewModel()
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ProcurementView.swift
```swift
struct ProcurementView: View {
@Environment(CartManager.self) private var cart
@Environment(\.modelContext) private var modelContext
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = ProcurementViewModel()
⋮----
var body: some View {
⋮----
private var headerSection: some View {
⋮----
private var suggestionsSection: some View {
⋮----
private var selectedSummary: some View {
⋮----
private var actionBar: some View {
⋮----
private struct ProcurementSuggestionCardView: View {
let forecast: DemandForecast
let isSelected: Bool
let externalQuantity: Int
let onToggleSelection: () -> Void
let onUpdateQuantity: (Int) -> Void
⋮----
@State private var localQuantity: Int
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/AuthManager.swift
```swift
func clearOrgScopedState() {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/GeocodeService.swift
```swift
struct ResolvedLocationResponse: Decodable {
let address: String
let lat: Double
let lng: Double
let placeId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum GeocodeService {
static func reverse(lat: Double, lng: Double) async -> String? {
```

## File: apps/supplier-app-ios/SupplierApp/Design/CollapsibleSidebar.swift
```swift
struct CollapsibleSidebar<Item: Hashable, Detail: View>: View {
let title: String
@Binding var isExpanded: Bool
@Binding var selection: Item?
let groups: [(title: String, items: [CollapsibleSidebarItem<Item>])]
@ViewBuilder let detail: () -> Detail
let footer: (() -> AnyView)?
⋮----
init(
⋮----
var body: some View {
⋮----
private func sidebar(effectiveExpanded: Bool, sidebarWidth: CGFloat, canToggle: Bool) -> some View {
⋮----
private func sidebarRow(_ item: CollapsibleSidebarItem<Item>, effectiveExpanded: Bool) -> some View {
⋮----
struct CollapsibleSidebarItem<Item: Hashable>: Identifiable {
let id: String
let tag: Item
let label: String
let icon: String
⋮----
init(id: String? = nil, tag: Item, label: String, icon: String) {
```

## File: apps/supplier-app-ios/SupplierApp/Design/MarketPack.swift
```swift
struct MarketPack: Sendable, Equatable {
var code: String
var name: String
var timezone: String
var currencyCode: String
var fiscalAdapter: String
var mapsAdapter: String
var mapCenterLat: Double
var mapCenterLng: Double
var checkoutReadsThis: Bool
⋮----
var receiptLabel: String { fiscalReceiptLabel(fiscalAdapter) }
⋮----
struct AuthSession: Sendable, Equatable {
var marketCode: String
var homeCell: String
var apiUrl: String
var pack: MarketPack?
var checkoutReadsThis: Bool
⋮----
func fiscalReceiptLabel(_ adapter: String?) -> String {
⋮----
func packCurrency(_ pack: MarketPack?, fallback: String = "") -> String {
⋮----
/// Empty pack prints the number only. Never invents UZS.
func formatPackMoney(_ minor: Int64, pack: MarketPack?, decimalPlaces: Int = 2) -> String {
⋮----
/// Integer minor units. Double division drops digits past the 53-bit mantissa.
func formatMinorUnits(_ amount: Int64, fractionDigits: Int, trimTrailingZeros: Bool) -> String {
⋮----
func displayPackCurrency(_ raw: String?, pack: MarketPack? = MarketPackStore.pack) -> String {
⋮----
/// Shipped pack camera. Empty/planned pack does not invent Tashkent.
func packMapCenter(_ pack: MarketPack?) -> (lat: Double, lng: Double)? {
⋮----
func packMapCoordinate(_ pack: MarketPack? = MarketPackStore.pack) -> (lat: Double, lng: Double) {
⋮----
func pinnedAPIBaseURL(bootstrap: URL) -> URL {
⋮----
enum CellTokenCache {
private static let lock = NSLock()
private static var value = ""
⋮----
static var token: String {
⋮----
enum MarketPackStore {
private static let lock = NSLock()
private static var session: AuthSession?
⋮----
static var current: AuthSession? {
⋮----
static var pack: MarketPack? { current?.pack }
⋮----
static var sessionApiUrl: String? {
⋮----
static func set(_ next: AuthSession?) {
⋮----
static func clear() {
⋮----
enum MarketPackBinder {
static func fetch(baseUrl: String, token: String) async -> AuthSession? {
⋮----
static func parse(_ data: Data) -> AuthSession? {
⋮----
struct PackBanner: View {
let pack: MarketPack?
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Design/PegasusStateViews.swift
```swift
public struct PegasusLoadingView: View {
public let title: String
public var message: String = "mobile_supplier.loading.latest"
⋮----
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var animating = false
⋮----
public init(title: String, message: String = "mobile_supplier.loading.latest") {
⋮----
public var body: some View {
⋮----
public struct PegasusErrorView: View {
public let message: String
public let retry: () -> Void
⋮----
public init(message: String, retry: @escaping () -> Void) {
⋮----
public var body: some View {
⋮----
final class SupplierStatusCenter {
static let shared = SupplierStatusCenter()
⋮----
var notice: String?
⋮----
func clear() {
⋮----
enum SupplierScreenLoad {
static func begin(silent: Bool, loading: inout Bool, error: inout String?) {
⋮----
static func succeed(error: inout String?) {
⋮----
static func fail(
⋮----
static func fail(
⋮----
enum SupplierActionFailure {
static func record(underlying: Error, hasContent: Bool, error: inout String?) {
```

## File: apps/supplier-app-ios/SupplierApp/Design/RealtimeRefresh.swift
```swift
public struct SilentRefreshModifier: ViewModifier {
private let refreshEpoch: Int
private let reconnectEpoch: Int
private let reload: (_ silent: Bool) -> Void
⋮----
@State private var coalesceTask: Task<Void, Never>?
⋮----
public init(
⋮----
public func body(content: Content) -> some View {
⋮----
private func scheduleCoalescedReload() {
⋮----
func silentRealtimeRefresh(
⋮----
public enum RealtimeLoadState {
public static func showFullScreenLoading(loading: Bool, hasData: Bool) -> Bool {
⋮----
public enum PulseHonesty {
public static let failed = "pulse_failed"
public static let commandFailed = "control_tower_pulse_failed"
⋮----
public struct Result<T> {
public let events: [T]
public let error: String?
⋮----
public struct ObjectResult<T> {
public let value: T?
public let error: String?
⋮----
public static func apply<T>(ok: Bool, incoming: [T]?, previous: [T], failure: String = failed) -> Result<T> {
⋮----
public static func applyObject<T>(
```

## File: apps/supplier-app-ios/SupplierApp/Models/SupplierModels.swift
```swift
struct LoginRequest: Encodable {
let phone: String
let password: String
⋮----
struct LoginResponse: Decodable {
let supplierId: String
let isRegistered: Bool
let isConfigured: Bool
let nextStep: String
let token: String?
let refreshToken: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct DeviceTokenRequest: Encodable, Equatable {
let token: String
let platform: String
⋮----
struct RegisterResponse: Decodable {
let supplierId: String
let legalName: String
let isRegistered: Bool
let isConfigured: Bool
let nextStep: String
let token: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RefreshTokenRequest: Encodable {
let refreshToken: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDashboard: Decodable {
let supplierId: String
let isConfigured: Bool
let inventorySKUs: Int?
let pendingOrders: Int?
let updatedAt: String
let ordersByStatus: [String: Int]
let todayRevenueMinor: Int64?
let deliveriesCompletedToday: Int?
let deliveriesAttemptedToday: Int?
let manifestsByState: [String: Int]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct SupplierProfile: Decodable {
let supplierId: String
let legalName: String
let contactName: String
let email: String
let phone: String
let country: String
let currency: String
let categories: [String]
let isRegistered: Bool
let isConfigured: Bool
let selectedGateways: [String]
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierOrdersResponse: Decodable {
let orders: [SupplierOrder]
let total: Int?
let limit: Int?
let offset: Int?
⋮----
init(orders: [SupplierOrder] = [], total: Int? = nil, limit: Int? = nil, offset: Int? = nil) {
⋮----
struct SupplierOrder: Decodable, Identifiable, Hashable {
var id: String { orderId }
let orderId: String
let retailerId: String
let warehouseId: String?
let status: String
let decision: String?
let note: String?
let totalMinor: Int64
let currency: String
let updatedAt: String
⋮----
init(
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOrderLineItem: Decodable, Identifiable, Hashable {
var id: String {
⋮----
let productId: String?
let productName: String?
let quantity: Double?
let unitPrice: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOrderDetail: Decodable {
let orderId: String
let retailerName: String?
let state: String?
let status: String?
let totalUzs: Int64?
let totalMinor: Int64?
let lineItems: [WarehouseOrderLineItem]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseProposeDeliveryRequest: Encodable {
let proposedDeliveryDate: String
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOrderMutationRequest: Encodable {
let reason: String?
⋮----
struct WarehouseOrderMutationResponse: Decodable {
let status: String?
⋮----
struct SupplierReturnRow: Decodable, Identifiable, Hashable {
var id: String { returnId }
let returnId: String
let orderId: String
let skuId: String
let productName: String
let quantity: Int64
let unitPrice: Int64
let status: String
let physicalStatus: String
let receivedQty: Int64
let reason: String
let driverName: String
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierReturnsResponse: Decodable {
let data: [SupplierReturnRow]
⋮----
struct ResolveReturnRequest: Encodable {
let returnId: String
let lineItemId: String
let resolution: String
let notes: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FleetDriversResponse: Decodable {
let supplierId: String
let items: [FleetDriver]
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FleetDriver: Decodable, Identifiable {
var id: String { driverId }
let driverId: String
let name: String
let phone: String
let homeNodeType: String
let homeNodeId: String
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FleetVehiclesResponse: Decodable {
let supplierId: String
let items: [FleetVehicle]
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FleetVehicle: Decodable, Identifiable {
var id: String { vehicleId }
let vehicleId: String
let label: String?
let licensePlate: String
let homeNodeType: String
let homeNodeId: String
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InventoryListResponse: Decodable {
let items: [InventoryItem]
⋮----
struct InventoryItem: Decodable, Identifiable {
var id: String { sku }
let sku: String
let productName: String
let quantity: Int64
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InventoryPatchRequest: Encodable {
let skuId: String?
let sku: String?
let quantityDelta: Int64?
let quantity: Int64?
let reason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogProduct: Decodable, Identifiable {
let productId: String
let name: String
let categoryId: String
let priceMinor: Int64
let currency: String
let unit: String
let unitVolumeVu: Double
let imageUrl: String?
let barcode: String?
let isActive: Bool
let version: Int64
⋮----
var id: String { productId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogCategory: Decodable, Identifiable {
let categoryId: String
let name: String
⋮----
var id: String { categoryId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogUploadTicket: Decodable {
let uploadUrl: String
let imageUrl: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogProductCreateRequest: Encodable {
let categoryId: String
let name: String
let description: String
let priceMinor: Int64
let currency: String
let unitVolumeVu: Double
let stockQuantity: Int64
let unit: String
let imageUrl: String?
let barcode: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CatalogProductUpdateRequest: Encodable {
let name: String
let priceMinor: Int64
let currency: String
let unit: String
let unitVolumeVu: Double
let imageUrl: String?
let barcode: String?
let isActive: Bool
let version: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierEarnings: Decodable {
let currency: String
let todayMinor: Int64
let weekMinor: Int64
let monthMinor: Int64
let authoritative: Bool
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum EarningsHonesty {
⋮----
static func minor(_ value: Int64, authoritative: Bool) -> Int64? {
⋮----
enum MissingNumber {
static func label<T: CustomStringConvertible>(_ value: T?) -> String {
⋮----
enum ActionResultText {
static func nonEmpty(status: String?, error: String?) -> String? {
⋮----
// MARK: - Promotions
⋮----
struct SupplierPromotion: Decodable, Identifiable {
let promotionId: String
let supplierId: String
let name: String
let description: String?
let discountBps: Int64
let scopeType: String
let scopeProductId: String?
let retailerScope: String
let isActive: Bool
let priority: Int64
⋮----
var id: String { promotionId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierPromotionsResponse: Decodable {
let promotions: [SupplierPromotion]
⋮----
struct SupplierPromotionUpsertRequest: Encodable {
let name: String
let description: String
let discountBps: Int64
let scopeType: String
let retailerScope: String
let scopeProductId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct BillingSetupRequest: Encodable {
let bankName: String
let accountHolder: String
let accountNumber: String
let swiftBic: String
let iban: String?
let selectedGateways: [String]
⋮----
struct BillingSetupResponse: Decodable {
let supplierId: String
let isConfigured: Bool
let selectedGateways: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DriverRecommendation: Decodable, Identifiable {
let driverId: String
let driverName: String
let currentLat: Double
let currentLon: Double
let distanceKm: Double
let score: Double
let vehicleClass: String
let licensePlate: String
⋮----
var id: String { driverId }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RecommendReassignResponse: Decodable {
let orderId: String
let retailerName: String
let orderVolumeVu: Double
let recommendations: [DriverRecommendation]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ApplyReassignRequest: Encodable {
let driverId: String
let isPartial: Bool
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/supplier-app-ios/SupplierApp/Models/SupplierOpsModels.swift
```swift
struct SupplierExceptionRow: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let kind: String
let status: String
let retailerId: String?
let note: String?
let manifestId: String?
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierExceptionsResponse: Decodable {
let exceptions: [SupplierExceptionRow]
⋮----
struct SupplierManifestRow: Decodable, Identifiable {
var id: String { manifestId }
let manifestId: String
let status: String
let state: String
let ordersCount: Int
let driverId: String?
let driverName: String
let vehicleId: String?
let vehiclePlate: String?
let truckId: String?
let totalVu: Int64
let totalVolumeVu: Double?
let maxVolumeVu: Double?
let stopCount: Int
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierManifestsResponse: Decodable {
let manifests: [SupplierManifestRow]
⋮----
struct SupplierManifestOrderWire: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let retailerId: String?
let amount: Int64
let state: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierManifestDetail: Decodable {
let manifestId: String
let status: String
let state: String
let ordersCount: Int
let driverId: String?
let driverName: String
let vehiclePlate: String?
let totalVu: Int64
let totalVolumeVu: Double?
let maxVolumeVu: Double?
let updatedAt: String
let orders: [SupplierManifestOrderWire]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierManifestExceptionRow: Decodable, Identifiable {
var id: String { exceptionId }
let exceptionId: String
let manifestId: String
let orderId: String
let reason: String
let metadata: String?
let attemptCount: Int64
let escalated: Bool
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierManifestExceptionsResponse: Decodable {
let exceptions: [SupplierManifestExceptionRow]
⋮----
struct SupplierManifestInjectOrderRequest: Encodable {
let orderId: String
let volumeVu: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RouteCoordinateWire: Decodable {
let lat: Double
let lng: Double
⋮----
struct RouteGeometryWire: Decodable {
let routeId: String?
let encodedPolyline: String?
let coordinates: [RouteCoordinateWire]
let source: String
let stopCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchProposedRoute: Decodable, Identifiable {
var id: String { driverId ?? driverName ?? "\(orderIds.joined(separator: "-"))" }
let driverId: String?
let driverName: String?
let vehicleId: String?
let orderIds: [String]
let volumeVu: Double?
let maxVolumeVu: Double?
let stopCount: Int?
let routeGeometry: RouteGeometryWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDispatchOrderRow: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let volumeVu: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierDispatchDriverRow: Decodable, Identifiable {
var id: String { driverId }
let driverId: String
let name: String
let maxVolumeVu: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDispatchManualRoutePayload: Encodable {
let driverId: String
let orderIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDispatchExecuteBody: Encodable {
let mode: String
let forceCapacity: Bool?
let routes: [SupplierDispatchManualRoutePayload]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDispatchExecuteResponse: Decodable {
let status: String?
⋮----
struct SupplierDispatchPreview: Decodable {
let pendingCount: Int?
let availableDriverCount: Int?
let undispatchedOrderCount: Int?
let undispatchedOrders: [SupplierDispatchOrderRow]
let availableDrivers: [SupplierDispatchDriverRow]
let proposedRoutes: [DispatchProposedRoute]
let optimizerSource: String?
let optimizerWarnings: [String]
let planFingerprint: String?
let warehousePlanFingerprint: String?
let planFingerprintMismatch: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierPricingRule: Decodable {
let supplierId: String
let baseMarkupBps: Int
let retailerDiscountBps: Int
let minMarginBps: Int
let currency: String
let ruleVersion: Int
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierPricingRuleUpdateRequest: Encodable {
let baseMarkupBps: Int
let retailerDiscountBps: Int
let minMarginBps: Int
let currency: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerPriceOverride: Decodable, Identifiable {
var id: String { overrideId }
let overrideId: String
let supplierId: String
let retailerId: String
let productId: String
let price: Int
let setBy: String
let setByRole: String
let isActive: Bool
let notes: String?
let expiresAt: String?
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerPriceOverridesResponse: Decodable {
let overrides: [RetailerPriceOverride]
let total: Int
⋮----
struct CreateRetailerPriceOverrideRequest: Encodable {
let retailerId: String
let productId: String
let price: Int
let notes: String?
let expiresAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateRetailerPriceOverrideResponse: Decodable {
let status: String
let overrideId: String
let retailerId: String
let productId: String
let price: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierTopologyWarehouse: Decodable, Identifiable {
var id: String { warehouseId }
let warehouseId: String
let name: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
let coverageRadiusKm: Double
let isActive: Bool
let isOnShift: Bool
let transferMode: String?
let coLocateWithFactoryId: String?
let primaryFactoryId: String?
let secondaryFactoryId: String?
let assignedFactoryIds: [String]
let countryCode: String
let coverageCities: [SupplierCoverageCity]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierCoverageCity: Codable {
let name: String
let lat: Double
let lng: Double
⋮----
struct SupplierTopologyFactory: Decodable, Identifiable {
var id: String { factoryId }
let factoryId: String
let name: String
let address: String
let placeId: String?
let lat: Double
let lng: Double
let isActive: Bool
let countryCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierTopologyWarehouseInput: Encodable {
let warehouseId: String?
let name: String
let address: String?
let placeId: String?
let lat: Double
let lng: Double
let coverageRadiusKm: Double?
let isActive: Bool?
let isOnShift: Bool?
let transferMode: String?
var primaryFactoryId: String? = nil
var countryCode: String? = nil
var coverageCities: [SupplierCoverageCity]? = nil
var assignedFactoryIds: [String]? = nil
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct SupplierTopologyFactoryInput: Encodable {
let factoryId: String?
let name: String
let address: String?
let placeId: String?
let lat: Double
let lng: Double
let isActive: Bool?
var countryCode: String? = nil
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct SupplierTopologyUpdateRequest: Encodable {
let warehouses: [SupplierTopologyWarehouseInput]
let factories: [SupplierTopologyFactoryInput]
⋮----
struct SupplierTopologyResponse: Decodable {
let supplierId: String
let warehouses: [SupplierTopologyWarehouse]
let factories: [SupplierTopologyFactory]
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierSupplyLaneRow: Decodable, Identifiable {
var id: String { laneId }
let laneId: String
let name: String
let warehouseId: String
let h3Cells: Int
let drivers: Int
let ordersToday: Int
let capacity: Int
let utilizationPct: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierSupplyLanesResponse: Decodable {
let lanes: [SupplierSupplyLaneRow]
⋮----
struct ShopClosedAttemptRow: Decodable, Identifiable {
var id: String { attemptId }
let attemptId: String
let orderId: String
let driverId: String
let retailerId: String
let resolution: String
let shopClosedReason: String?
let shopClosedResolution: String?
let graceEndsAt: String?
let shopClosedAt: String?
let createdAt: String
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ShopClosedActiveResponse: Decodable {
let data: [ShopClosedAttemptRow]
⋮----
struct NegotiationProposalItem: Decodable {
let skuId: String
let originalQty: Int
let proposedQty: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationProposalRow: Decodable, Identifiable {
var id: String { proposalId }
let proposalId: String
let orderId: String
let driverId: String
let items: [NegotiationProposalItem]
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationPendingResponse: Decodable {
let data: [NegotiationProposalRow]
⋮----
struct ShopClosedResolveRequest: Encodable {
let attemptId: String
let action: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationResolveRequest: Encodable {
let proposalId: String
let action: String
let resolution: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NegotiationResolveResponse: Decodable {
let status: String
let proposalId: String
let orderId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierFleetOrderRow: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let retailerId: String?
let driverId: String?
let status: String
let state: String?
let routeId: String?
let totalMinor: Int64?
let currency: String?
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - Fleet live map
⋮----
struct SupplierDriverLocationWire: Decodable {
let driverId: String
let supplierId: String?
let lat: Double
let lng: Double
let latitude: Double
let longitude: Double
let reportedAt: String
let receivedAt: String
let staleAfterSeconds: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierFleetLiveRoute: Decodable, Identifiable {
var id: String { manifestId }
let manifestId: String
let routeId: String
let driverId: String
let driverName: String?
let manifestState: String
let routeGeometry: RouteGeometryWire?
let driverLocation: SupplierDriverLocationWire?
let liveLocationAvailable: Bool
let locationStale: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierFleetLiveMapResponse: Decodable {
let routes: [SupplierFleetLiveRoute]
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierMEIONetworkSummary: Decodable {
let supplierId: String
let warehousesScanned: Int
let skusAnalyzed: Int
let insightsGenerated: Int
let transferRecommendations: Int
let generatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ControlTowerZoneOverride: Decodable, Identifiable {
var id: String { overrideId }
let overrideId: String
let supplierId: String
let warehouseId: String?
let action: String
let ttlExpiresAt: String
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ControlTowerZoneOverridesResponse: Decodable {
let overrides: [ControlTowerZoneOverride]
⋮----
struct PlanningSAndOPSnapshot: Decodable {
let supplierId: String
let horizonDays: Int
let factoryCapacityUnits: Int64
let warehouseInboundCapUnits: Int64
let warehouseOutboundCapUnits: Int64
let utilizationPct: Double
let capacityAlert: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PlanningScenarioInput: Encodable {
let factoryDowntimeHours: Int
let demandDeltaPct: Double
let horizonDays: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PlanningScenarioResult: Decodable {
let scenarioId: String
let slaRiskPct: Double
let fleetVolumeOrders: Int64
let stockoutSkus: [String]
let capacityBreach: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SparsityGateResult: Decodable {
let allowed: Bool
let completedOrders: Int
let blockedReason: String?
let label: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ForecastConfidence: Decodable {
let lowUnits: Int64?
let highUnits: Int64?
let confidencePct: Int?
let baselineSource: String?
let blockedReason: String?
let label: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SeasonalOverrideInput: Encodable {
let templateId: String?
let startDate: String
let endDate: String
let name: String?
let multiplier: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SeasonalOverrideRow: Decodable, Identifiable {
var id: String { overrideId }
let overrideId: String
let supplierId: String
let templateId: String
let name: String?
let startDate: String
let endDate: String
let isActive: Bool
let multiplier: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SeasonalBuiltinTemplate: Decodable, Identifiable {
var id: String
let name: String
let multiplier: Double?
⋮----
struct SeasonalTemplatesResponse: Decodable {
let builtinTemplates: [SeasonalBuiltinTemplate]
let overrides: [SeasonalOverrideRow]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierReturnPolicy: Codable {
var defaultWindowHours: Int64
var concealedDamageWindowHours: Int64?
var requirePhoto: Bool
var allowExpiredClaims: Bool
var policySourceHint: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct KnowledgeGraphNode: Decodable, Identifiable {
var id: String
let type: String
let name: String?
⋮----
struct KnowledgeGraphEdge: Decodable, Identifiable {
var id: String { "\(from)-\(to)-\(relation)" }
let from: String
let to: String
let relation: String
⋮----
struct SupplierKnowledgeGraph: Decodable {
let supplierId: String
let nodes: [KnowledgeGraphNode]
let edges: [KnowledgeGraphEdge]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierReplenishmentPolicy: Decodable {
let supplierId: String
let autoApproveStable: Bool
let autoApprovePredictivePush: Bool
let maxDailyTransferUnits: Int64
let minConfidenceScore: Double
let targetServiceLevel: Double
let leadTimeDays: Int64
let leadTimeSigmaDays: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct GeoJSONPolygonPayload: Encodable {
let type: String = "Polygon"
let coordinates: [[[Double]]]
⋮----
struct ControlTowerZoneOverrideCreateRequest: Encodable {
let action: String
let ttlSeconds: Int
let polygonGeojson: GeoJSONPolygonPayload
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierWsSessionResponse: Decodable {
let token: String
let expiresAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentLedgerEntry: Decodable, Identifiable {
var id: String { ledgerEntryId.isEmpty ? (referenceId ?? occurredAt) : ledgerEntryId }
let ledgerEntryId: String
let orderId: String?
let retailerId: String?
let gateway: String
let entryType: String
let amountMinor: Int64?
let currency: String
let referenceId: String?
let source: String?
let occurredAt: String
let createdAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PaymentLedgerResponse: Decodable {
let items: [PaymentLedgerEntry]
let count: Int
let limit: Int
let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierReplenishmentTriggerResponse: Decodable {
let status: String
let requestId: String
let warehouseId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierActivityEvent: Decodable, Identifiable {
let id: String
let type: String
let timestamp: String
let description: String
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierActivityResponse: Decodable {
let events: [SupplierActivityEvent]
⋮----
struct FleetDriverCreateRequest: Encodable {
let name: String
let phone: String
let pin: String
let homeNodeType: String
let homeNodeId: String
let vehicleId: String?
let isActive: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct FleetVehicleCreateRequest: Encodable {
let label: String?
let licensePlate: String
let homeNodeType: String
let homeNodeId: String
let isActive: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierProfileUpdateRequest: Encodable {
let legalName: String?
let contactName: String?
let email: String?
let phone: String?
let country: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SettlementCurrencyTotal: Decodable {
let currency: String
let entryCount: Int
let amountMinorTotal: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SettlementAuthorityRow: Decodable, Identifiable {
var id: String { "\(gateway)-\(entryType)-\(currency)" }
let gateway: String
let entryType: String
let currency: String
let entryCount: Int
let amountMinorTotal: Int64
let firstOccurredAt: String
let lastOccurredAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SettlementAuthorityResponse: Decodable {
let items: [SettlementAuthorityRow]
let count: Int
let supplierId: String
let entryCountTotal: Int
let totalsByCurrency: [SettlementCurrencyTotal]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ReconciliationMismatchRow: Decodable, Identifiable {
var id: String { "\(gateway)-\(currency)" }
let gateway: String
let currency: String
let netAmountMinor: Int64?
let creditAmountMinorTotal: Int64?
let debitAmountMinorTotal: Int64?
let entryCountTotal: Int
let lastOccurredAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - AI recommendations
⋮----
struct SupplierAIRecommendationEvidence: Decodable, Identifiable {
var id: String { "\(label)-\(value)" }
let label: String
let value: String
let href: String?
⋮----
struct SupplierAIRecommendation: Decodable, Identifiable {
var id: String { recommendationId }
let recommendationId: String
let aggregateId: String
let aggregateType: String
let action: String
let status: String
let score: Double
let confidence: Double
let source: String
let explanation: String
let reasonCodes: [String]
let evidence: [SupplierAIRecommendationEvidence]
let decision: String?
let decisionNote: String?
let decidedBy: String?
let decidedAt: String?
let generatedAt: String
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierAIRecommendationsResponse: Decodable {
let items: [SupplierAIRecommendation]
let count: Int
let limit: Int
let status: String?
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAIRecommendationDecisionRequest: Encodable {
let recommendationId: String
let decision: String
let note: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAIRecommendationDecisionResponse: Decodable {
let recommendation: SupplierAIRecommendation
⋮----
struct ReconciliationMismatchResponse: Decodable {
let items: [ReconciliationMismatchRow]
⋮----
struct SupplierOrgMember: Decodable, Identifiable {
var id: String { userId }
let userId: String
let name: String
let phone: String
let supplierRole: String
let assignedWarehouseId: String?
let assignedFactoryId: String?
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierOrgMembersResponse: Decodable {
let items: [SupplierOrgMember]
⋮----
struct SupplierOrgMemberCreateRequest: Encodable {
let name: String
let email: String?
let phone: String
let password: String
let supplierRole: String
let assignedWarehouseId: String?
let assignedFactoryId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierOrgMemberUpdateRequest: Encodable {
let name: String?
let supplierRole: String?
let assignedWarehouseId: String?
let assignedFactoryId: String?
let isActive: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ApproveEarlyCompleteRequest: Encodable {
let driverId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAnalyticsVelocityPoint: Decodable {
let date: String
let ordersCreated: Int
let ordersCompleted: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAnalyticsVelocityResponse: Decodable {
let periodDays: Int
let points: [SupplierAnalyticsVelocityPoint]
let generatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAnalyticsRevenueResponse: Decodable {
let currency: String
let totalMinor: Int64
let series: [SupplierAnalyticsRevenuePoint]
let generatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierAnalyticsRevenuePoint: Decodable {
let date: String
let revenueMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDemandSummaryResponse: Decodable {
let totalRetailers: Int
let totalPallets: Int
let totalValue: Int64
let predictionCount: Int
let generatedAt: String
let baselineSource: String?
let granularity: String?
let confidence: ForecastConfidence?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierEmpathyAdoption: Decodable {
let totalPredictions: Int64
let predictionsDormant: Int64
let predictionsWaiting: Int64
let predictionsFired: Int64
let predictionsRejected: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierBroadcastRequest: Encodable {
let title: String
let body: String
let role: String
⋮----
struct SupplierBroadcastResponse: Decodable {
let status: String
let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ExceptionMapCell: Decodable, Identifiable {
var id: String { h3Cell }
let h3Cell: String
let lat: Double
let lng: Double
let severity: String
let counts: [String: Int]
let sampleOrderIds: [String]?
let deepLink: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ExceptionMapResponse: Decodable {
let cells: [ExceptionMapCell]
let windowHours: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOverridePreview: Decodable {
let retailersOnSkuCount: Int
let activeOverrideCount: Int
let catalogListPrice: Int64
let marginDeltaPerUnit: Int64
let marginEstimateLabel: String
let affectedRetailerIds: [String]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerOverridePreviewRequest: Encodable {
let retailerId: String?
let productId: String?
let skuId: String?
let proposedPrice: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierBroadcastTemplate: Identifiable {
let id: String
let title: String
let body: String
let defaultRole: String
⋮----
let supplierBroadcastTemplates: [SupplierBroadcastTemplate] = [
⋮----
struct PaymentBypassRequest: Encodable {
let orderId: String
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentBypassResponse: Decodable {
let status: String
let bypassToken: String
let orderId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDemandHistoryPoint: Decodable, Identifiable {
var id: String { date }
let date: String
let predicted: Double
let actual: Double
let predictedQty: Double
let actualQty: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDemandUpcomingRow: Decodable, Identifiable {
var id: String { "\(date)-\(skuId)-\(retailerName)" }
let date: String
let retailerName: String
let skuId: String
let productName: String
let predictedQty: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierDemandHistoryResponse: Decodable {
let timeSeries: [SupplierDemandHistoryPoint]
let upcoming: [SupplierDemandUpcomingRow]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierInventoryImportResult: Decodable {
let sessionId: String?
let applied: Int
let skipped: Int
let errors: [String]?
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportSessionCreateResponse: Decodable {
let sessionId: String
let status: String
let fileName: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportSession: Decodable {
let sessionId: String
let status: String
let fileName: String
let totalRows: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportMappingCandidate: Decodable, Identifiable {
var id: String { sourceColumn }
let sourceColumn: String
let targetField: String
let confidence: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportMappingResponse: Decodable {
let sessionId: String
let mappingJson: SupplierImportMappingJson?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportMappingJson: Decodable {
let mappings: [SupplierImportMappingCandidate]?
⋮----
struct SupplierImportIngestResponse: Decodable {
let sessionId: String
let status: String
let rowsStaged: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierImportApplyResponse: Decodable {
let sessionId: String
let status: String
let appliedRows: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentChargebackRequest: Encodable {
let orderId: String
let retailerId: String
let gateway: String
let amount: Int64
let currency: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentChargebackResponse: Decodable {
let status: String
⋮----
struct PaymentChargebackReversalRequest: Encodable {
let sessionId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentChargebackReversalResponse: Decodable {
let status: String
⋮----
struct SupplierClaimLine: Decodable {
let sku: String
let quantity: Int64
let reason: String?
let amountMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierClaimEvidence: Decodable {
let evidenceType: String
let uri: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierClaim: Decodable, Identifiable {
var id: String { claimId }
let claimId: String
let orderId: String
let retailerId: String
let claimType: String
let status: String
let amountMinor: Int64?
let currency: String?
let description: String?
let lineItems: [SupplierClaimLine]
let evidences: [SupplierClaimEvidence]
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierClaimsListResponse: Decodable {
let claims: [SupplierClaim]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case claims }
⋮----
struct ApproveClaimRequest: Encodable {
let resolutionNote: String
let settlementMode: String
let skipGatewayRefund: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RejectClaimRequest: Encodable {
let resolutionNote: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ClaimSettlementResult: Decodable {
let chargebackId: String?
let amountMinor: Int64?
let mode: String
let gatewayRefunded: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ApproveClaimResponse: Decodable {
let claim: SupplierClaim?
let settlement: ClaimSettlementResult?
⋮----
struct ClaimChargebacksResponse: Decodable {
let items: [PaymentLedgerEntry]
let count: Int
let limit: Int
let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct BusinessSetupRequest: Encodable {
let taxId: String
let registrationNumber: String
let headquartersAddress: String
let city: String
let postalCode: String
⋮----
struct BusinessSetupResponse: Decodable {
let supplierId: String
let isRegistered: Bool
let nextStep: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierPulseEvent: Decodable, Identifiable {
let id: String
let kind: String
let title: String
let description: String?
let occurredAt: String
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierPulseResponse: Decodable {
let events: [SupplierPulseEvent]
let fetchedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PromoSimulateInput: Encodable {
let promotionId: String?
let discountPct: Double?
let expectedUnits: Int64?
let avgUnitMarginMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PromoSimulateResult: Decodable {
let simulationId: String
let promotionId: String?
let projectedVolume: Int64
let projectedRevenueMinor: Int64
let projectedMarginMinor: Int64
let marginDeltaPct: Double
let sandboxOnly: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ComplianceSummary: Decodable {
let openFiscalCount: Int
let forceCompleteCount: Int
let claimMismatchCount: Int
let creditFreezeCount: Int
let generatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(openFiscalCount: Int = 0, forceCompleteCount: Int = 0, claimMismatchCount: Int = 0, creditFreezeCount: Int = 0, generatedAt: String = "") {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ComplianceFiscalOpenRow: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let status: String
let fiscalStatus: String
let totalMinor: Int64
let currency: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ComplianceForceCompleteRow: Decodable, Identifiable {
var id: String { orderId + completedAt }
let orderId: String
let reasonCode: String
let actorId: String
let completedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ComplianceClaimMismatchRow: Decodable, Identifiable {
var id: String { claimId }
let claimId: String
let orderId: String
let mismatchReason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ComplianceCreditFreezeRow: Decodable, Identifiable {
var id: String { retailerId + status }
let retailerId: String
let status: String
let currentBalanceMinor: Int64?
let creditLimitMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct OrderReceiptMeta: Decodable {
let receiptId: String
let htmlUrl: String
let pdfUrl: String
let qrUrl: String
let partyCopy: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ComplianceDashboardResponse: Decodable {
let summary: ComplianceSummary
let openFiscal: [ComplianceFiscalOpenRow]
let forceCompletes: [ComplianceForceCompleteRow]
let claimMismatches: [ComplianceClaimMismatchRow]
let creditFreezes: [ComplianceCreditFreezeRow]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CashReconciliationRow: Decodable, Identifiable {
var id: String { reconciliationId }
let reconciliationId: String
let driverId: String
let status: String
let differenceMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CashReconciliationsResponse: Decodable {
let reconciliations: [CashReconciliationRow]
⋮----
struct CreditNoteRow: Decodable, Identifiable {
var id: String { creditNoteId }
let creditNoteId: String
let orderId: String
let status: String
let totalGrossMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CreditNotesResponse: Decodable {
let creditNotes: [CreditNoteRow]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CreditProfileRow: Decodable, Identifiable {
var id: String { retailerId }
let retailerId: String
let supplierId: String
let creditLimitMinor: Int64?
let currentBalanceMinor: Int64?
let availableCreditMinor: Int64?
let riskScore: Int64?
let riskTier: String
let delinquencyCount: Int64
let status: String
let utilizationBps: Int64?
let needsAttention: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct CreditProfilesResponse: Decodable {
let profiles: [CreditProfileRow]
let count: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RetailerCreditProfilePatchRequest: Encodable {
let retailerId: String
let creditLimitMinor: Int64
let status: String
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RoutePerformanceRow: Decodable, Identifiable {
var id: String { routeId }
let routeId: String
let driverId: String
let ordersCompleted: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RoutePerformanceResponse: Decodable {
let routes: [RoutePerformanceRow]
⋮----
struct NotificationPreferencesResponse: Decodable {
let preferences: [NotificationPreferenceRow]
⋮----
struct NotificationPreferenceRow: Codable {
let eventType: String
let channel: String
let enabled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplierCRMRetailer: Decodable, Identifiable {
var id: String { retailerId }
let retailerId: String
let retailerName: String
let phone: String
let email: String
let lifetime: Int64?
let orderCount: Int64
let lastOrderDate: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierCRMOrder: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let state: String
let amount: Int64?
let itemCount: Int64
let createdAt: String
let lines: [SupplierCRMOrderLine]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierCRMOrderLine: Decodable, Identifiable {
var id: String { "\(sku)-\(productName)-\(qty)" }
let sku: String
let productName: String
let qty: Int64
let amountMinor: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierCRMRetailerDetail: Decodable {
let retailerId: String
let retailerName: String
let phone: String
let email: String
let lifetime: Int64?
let orderCount: Int64
let lastOrderDate: String
let status: String
let orders: [SupplierCRMOrder]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierCRMListResponse: Decodable {
let retailers: [SupplierCRMRetailer]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case retailers }
⋮----
struct NetworkModeResponse: Decodable {
let mode: String
let supplierId: String
let planningEnabled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct NetworkModeUpdateRequest: Encodable {
let mode: String
let reason: String?
⋮----
struct NetworkModeUpdateResponse: Decodable {
let oldMode: String
let newMode: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PullMatrixResponse: Decodable {
let status: String
let transfers: Int
let skus: Int
let source: String
let grain: String
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case status, transfers, skus, source, grain }
⋮----
struct PredictivePushResponse: Decodable {
let transfers: Int
let skus: Int
let source: String
let grain: String
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case transfers, skus, source, grain }
⋮----
struct LoyaltyProgram: Codable {
var supplierId: String
var earnBps: Int64
var tiers: [LoyaltyTier]
var reason: String?
var source: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct LoyaltyTier: Codable {
let name: String
let minPoints: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct EntityResolutionResolveRequest: Encodable {
let entityType: String
let query: String?
let entityId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct EntityResolutionExplainRequest: Encodable {
let entityType: String
let entityId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct EntityResolutionCandidate: Decodable, Identifiable {
var id: String { nodeId }
let nodeId: String
let entityType: String
let entityId: String
let label: String
let score: Double
let confidenceClass: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct EntityResolutionResolveResponse: Decodable {
let requestedType: String
let resolved: EntityResolutionCandidate?
let candidates: [EntityResolutionCandidate]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct EntityResolutionExplainResponse: Decodable {
let source: EntityResolutionCandidate
let projection: EntityResolutionProjection
⋮----
struct EntityResolutionProjection: Decodable {
let edges: [EntityResolutionEdge]
⋮----
struct EntityResolutionEdge: Decodable {
let from: String
let to: String
let relation: String
⋮----
struct KillSwitchRequest: Encodable {
let reason: String
⋮----
struct KillSwitchResponse: Decodable {
let status: String
let cancelledTransfers: Int
let mode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PayoutRailInfo: Decodable {
let name: String
let isLive: Bool
let workflow: String
let steps: [String]
let message: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PayoutBatch: Decodable, Identifiable {
var id: String { batchId }
let batchId: String
let supplierId: String
let periodStart: String
let periodEnd: String
let netPayoutMinor: Int64?
let currency: String
let status: String
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PayoutBatchListResponse: Decodable {
let batches: [PayoutBatch]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case batches }
⋮----
struct PayoutBatchGenerateRequest: Encodable {
let periodStart: String
let periodEnd: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PayoutBatchGenerateResponse: Decodable {
let batch: PayoutBatch?
let rail: PayoutRailInfo?
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case batch, rail }
⋮----
struct PayoutMarkPaidResponse: Decodable {
let status: String
let batchId: String
let message: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PayoutDispatchRequest: Encodable {
let live: Bool
⋮----
struct PayoutDispatchResponse: Decodable {
let error: String
let code: String
let message: String
let rail: PayoutRailInfo?
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case error, code, message, rail }
⋮----
struct SupplierPayoutPolicy: Decodable {
let supplierId: String
let payoutMode: String
let feePolicyVersion: String
let effectiveAt: String?
let updatedBy: String
let updatedByType: String
let reason: String
let isActive: Bool
let source: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplierPayoutPolicyPatch: Encodable {
let payoutMode: String
let feePolicyVersion: String?
let reason: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ScoredException: Decodable, Identifiable {
var id: String { exceptionId }
let exceptionId: String
let type: String
let severity: String
let orderId: String
let score: Int64
let ageMinutes: Int64
let topPlaybookName: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ScoredExceptionsResponse: Decodable {
let exceptions: [ScoredException]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case exceptions }
⋮----
struct ControlTowerPlaybook: Decodable, Identifiable {
var id: String { playbookId }
let playbookId: String
let name: String
let description: String
let isActive: Bool
let priority: Int64
let autoExecute: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ControlTowerPlaybooksResponse: Decodable {
let playbooks: [ControlTowerPlaybook]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case playbooks }
```

## File: apps/supplier-app-ios/SupplierApp/Services/APIClient.swift
```swift
else if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
⋮----
func get<T: Decodable>(_ path: String, query: [String: String] = [:]) async throws -> T {
⋮----
func getJSONString(_ path: String, query: [String: String] = [:]) async throws -> String {
⋮----
func post<B: Encodable, T: Decodable>(
⋮----
func postEmpty<T: Decodable>(_ path: String, idempotencyKey: String? = nil, authenticated: Bool = true) async throws -> T {
⋮----
func postText(
⋮----
func put<B: Encodable, T: Decodable>(
⋮----
func put<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
⋮----
func patch<B: Encodable, T: Decodable>(
⋮----
func patch<B: Encodable, T: Decodable>(_ path: String, body: B, authenticated: Bool = true) async throws -> T {
⋮----
func postJSON<T: Decodable>(
⋮----
func postRaw<T: Decodable>(
⋮----
func patchVoid<B: Encodable>(
⋮----
func postVoid<B: Encodable>(
⋮----
func deleteVoid(
⋮----
func delete<T: Decodable>(
⋮----
private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
⋮----
private func executeVoid(_ request: URLRequest) async throws {
⋮----
private func stampIdempotency(_ key: String?, on request: inout URLRequest) {
⋮----
private func dataForRequestWithRetry(_ request: URLRequest) async throws -> (Data, URLResponse) {
⋮----
private struct SupplierTokenRefreshResponse: Decodable {
let token: String
let refreshToken: String?
enum CodingKeys: String, CodingKey {
⋮----
private func attemptRefresh() async throws -> Bool {
⋮----
private func requestURL(path: String, query: [String: String] = [:]) throws -> URL {
⋮----
private func attachToken(_ request: inout URLRequest) async {
⋮----
private actor SupplierHTTPCoalescer {
private var inFlightGETs: [String: Task<(Data, URLResponse), Error>] = [:]
private var getCache: [String: (data: Data, response: URLResponse, expiresAt: Date)] = [:]
private var refreshTask: Task<Bool, Error>?
private let cacheTTL: TimeInterval = 2.0
⋮----
func fetchOrCoalesce(
⋮----
func invalidateCache() {
⋮----
func refreshToken(using operation: @Sendable @escaping () async throws -> Bool) async throws -> Bool {
⋮----
struct APIErrorBody: Decodable {
let error: String
⋮----
enum APIError: LocalizedError {
⋮----
var errorDescription: String? {
⋮----
enum SupplierURL {
static let devBootstrap = literal(scheme: "http", host: "localhost", port: 8180)
static let releaseBootstrap = literal(scheme: "https", host: "api.pegasusx.app")
static let portalDefault = literal(scheme: "http", host: "localhost", port: 3000)
⋮----
static func literal(scheme: String, host: String, port: Int? = nil, path: String = "/") -> URL {
⋮----
static func parse(_ raw: String) -> URL? {
```

## File: apps/supplier-app-ios/SupplierApp/Services/GeocodeService.swift
```swift
struct GeocodePrediction: Decodable, Equatable {
let placeId: String
let description: String
⋮----
init(placeId: String, description: String) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ResolvedLocation: Decodable, Equatable {
let address: String
let lat: Double
let lng: Double
let placeId: String?
⋮----
init(address: String, lat: Double, lng: Double, placeId: String? = nil) {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum GeocodeService {
static func autocomplete(_ input: String) async throws -> [GeocodePrediction] {
⋮----
static func resolvePlace(_ placeId: String) async throws -> ResolvedLocation {
⋮----
static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
⋮----
static func forward(address: String) async throws -> ResolvedLocation {
struct ForwardRequest: Encodable { let address: String }
⋮----
private struct AutocompleteResponse: Decodable {
let predictions: [GeocodePrediction]
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierRealtimeClient.swift
```swift
enum SupplierRealtimeStatus: Equatable {
⋮----
struct SupplierLiveEvent: Decodable {
let type: String
let minimum_version: String?
⋮----
func connect(
⋮----
func disconnect() {
⋮----
private func openSocket(onEvent: @escaping (SupplierLiveEvent) -> Void) async {
⋮----
private func startHeartbeatLoop(for socketTask: URLSessionWebSocketTask, onEvent: @escaping (SupplierLiveEvent) -> Void) {
⋮----
private func scheduleReconnect(onEvent: @escaping (SupplierLiveEvent) -> Void) {
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierRealtimeHub.swift
```swift
final class SupplierRealtimeHub {
var refreshEpoch: Int = 0
var reconnectEpoch: Int = 0
⋮----
@ObservationIgnored private var lastBumpAt: Date = .distantPast
@ObservationIgnored private var trailingTask: Task<Void, Never>?
@ObservationIgnored private let coalesceWindow: TimeInterval = 0.20
⋮----
func bump() {
⋮----
func bumpReconnect() {
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierService+Foundry.swift
```swift
static func applyScenarioPatch(scenarioId: String, req: ScenarioPatchRequest, idempotencyKey: String) async throws -> [String: String] {
⋮----
static func commitScenario(scenarioId: String, idempotencyKey: String) async throws -> ScenarioCommitResult {
⋮----
static func triggerLotRecall(req: LotRecallRequest, idempotencyKey: String) async throws -> RecallBlastRadiusReport {
⋮----
static func executeAction(req: ActionExecutionRequest, idempotencyKey: String) async throws -> ActionExecutionRecord {
```

## File: apps/supplier-app-ios/SupplierApp/Services/SupplierSessionReconcile.swift
```swift
enum SupplierSessionReconcile {
static func notice(previewError: String?, manifestsError: String?) -> String? {
⋮----
static func run() async {
⋮----
private static func failure<T>(_ work: () async throws -> T) async -> String? {
```

## File: apps/supplier-app-ios/SupplierApp/ViewModels/OrdersViewModel.swift
```swift
final class OrdersViewModel {
var orders: [SupplierOrder] = []
var loading = true
var error: String?
var statusFilter = "ACTIVE"
var commandStatus: String?
var selection: SupplierOrder?
var vettingOrderId: String?
⋮----
var reassignTarget: String? {
⋮----
var reassignRecommendations: RecommendReassignResponse?
var isReassigning = false
var reassignMessage: String? {
⋮----
var showReassignSheet = false
var showReassignMessageAlert = false
var notice: String?
⋮----
private var loadGeneration = 0
⋮----
let filters: [(id: String, label: String)] = [
⋮----
var loadIdentity: String { "\(commandStatus ?? "")|\(statusFilter)" }
⋮----
func applyCommandStatus(_ status: String) {
⋮----
func setCoarseFilter(_ id: String) {
⋮----
func load(silent: Bool = false) async {
⋮----
func canWarehouseOps(for order: SupplierOrder) -> Bool {
⋮----
func proposeWarehouseOrder(_ order: SupplierOrder, proposedDeliveryDate: String, reason: String) async {
⋮----
func rejectWarehouseOrder(_ order: SupplierOrder, reason: String) async {
⋮----
func openReassignDialog(orderId: String) async {
⋮----
func issuePaymentBypass(orderId: String, reason: String = "OPERATIONS_BYPASS") async {
⋮----
func closeReassignDialog() {
⋮----
func applyReassign(orderId: String, driverId: String, isPartial: Bool) async {
⋮----
enum OrdersLoadFailure {
static func message(
⋮----
func resolveSupplierOrdersQuery(commandStatus: String?, coarseFilter: String) -> (status: String?, filter: String?) {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Catalog/CatalogList.swift
```swift
struct CatalogList: View {
let products: [CatalogProduct]
let query: String
@Binding var draftVU: [String: String]
@Binding var draftBarcode: [String: String]
let savingId: String?
let imageSavingId: String?
let onSaveUnitVolume: (CatalogProduct) async -> Void
let onSaveProductImage: (CatalogProduct, Data) async -> Void
⋮----
private var filtered: [CatalogProduct] {
⋮----
var body: some View {
⋮----
struct CatalogRowView: View {
let product: CatalogProduct
@Binding var draftVU: [String: String]
@Binding var draftBarcode: [String: String]
let isSaving: Bool
let isImageSaving: Bool
let onSaveUnitVolume: (CatalogProduct) async -> Void
let onSaveProductImage: (CatalogProduct, Data) async -> Void
⋮----
@State private var vuInput = ""
@State private var barcodeInput = ""
⋮----
private var isDirty: Bool {
⋮----
var body: some View {
⋮----
private func syncInputsFromModel() {
⋮----
private static let decimalFormatter: NumberFormatter = {
⋮----
func formatted() -> String {
⋮----
enum PhotoLoad {
static func imageData(_ data: Data?) throws -> Data {
⋮----
enum PhotoLoadError: LocalizedError {
⋮----
var errorDescription: String? {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Components/KpiTile.swift
```swift
struct KpiTile: View {
let title: String
let value: String
let systemImage: String
let tint: Color
var supporting: String? = nil
var chip: (text: String, tint: Color)? = nil
var showChevron: Bool = true
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Dashboard/DashboardView.swift
```swift
struct DashboardView: View {
@Environment(TokenStore.self) private var tokenStore
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var vm = DashboardViewModel()
⋮----
private var gridMin: CGFloat {
⋮----
var body: some View {
⋮----
private var billingBanner: some View {
⋮----
private var signOutToolbar: some ToolbarContent {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Dispatch/DispatchPreviewMapView.swift
```swift
struct DispatchPreviewMapView: View {
let routes: [DispatchProposedRoute]
⋮----
@State private var cameraPosition: MapCameraPosition = .region(
⋮----
private var routableRoutes: [DispatchProposedRoute] {
⋮----
var body: some View {
⋮----
private func routeColor(_ index: Int) -> Color {
⋮----
private func fitCamera() {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Fleet/FleetLiveMapView.swift
```swift
struct FleetLiveMapView: View {
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var viewModel = FleetLiveMapViewModel()
⋮----
private let publishActions = ["REROUTE", "FREEZE_DISPATCH", "PRIORITY_BOOST"]
⋮----
var body: some View {
⋮----
private func routeColor(_ index: Int) -> Color {
⋮----
private func routeColor(for route: SupplierFleetLiveRoute) -> Color {
⋮----
private func exceptionColor(_ severity: String) -> Color {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Foundry/FoundrySandboxView.swift
```swift
struct FoundrySandboxView: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = FoundrySandboxViewModel()
@State private var selectedTab: Int = 0
var onBack: (() -> Void)? = nil
⋮----
init(onBack: (() -> Void)? = nil) {
⋮----
var body: some View {
⋮----
private var scenarioSandboxTab: some View {
⋮----
private var lotRecallTab: some View {
⋮----
private struct MetricCard: View {
let title: String
let count: Int
let items: [String]
var isWarning: Bool = false
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Foundry/FoundrySandboxViewModel.swift
```swift
public struct ScenarioPatchSummaryItem: Identifiable, Sendable {
public var id: String { "\(objectType):\(objectId):\(key)" }
public let objectType: String
public let objectId: String
public let key: String
public let value: String
⋮----
final class FoundrySandboxViewModel {
⋮----
var supplierId: String = "supplier-tashkent-1"
var cellId: String = "cell-uz"
var scenarioId: String = "scenario-surge-plan-01"
var objectType: String = "SKU"
var objectId: String = "sku-uz-cola-1l"
var patchKey: String = "price_minor"
var patchValue: String = "1250000"
⋮----
var appliedPatches: [ScenarioPatchSummaryItem] = []
var isCommitting: Bool = false
var commitResult: ScenarioCommitResult? = nil
var hasConflict: Bool = false
var conflicts: [ScenarioWatermarkConflict] = []
⋮----
var lotBarcode: String = "LOT-SUGAR-2026-X8"
var isAnalyzingRecall: Bool = false
var recallReport: RecallBlastRadiusReport? = nil
var isQuarantining: Bool = false
var quarantineSuccess: Bool = false
⋮----
var statusMessage: String? = nil
var errorMessage: String? = nil
⋮----
func applyCowPatch() async {
⋮----
func commitScenario() async {
⋮----
func analyzeLotRecall() async {
⋮----
func executeQuarantineBlastRadius() async {
⋮----
private func pinnedSupplierId() -> String? {
⋮----
enum FoundryIdempotency {
static func scenarioPatch(
⋮----
static func scenarioCommit(scenarioId: String) -> String {
⋮----
static func lotRecall(supplierId: String, lot: String) -> String {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Inventory/InventoryImportView.swift
```swift
struct InventoryImportView: View {
@State private var viewModel = InventoryImportViewModel()
⋮----
private let stepLabels = ["mobile_supplier.chrome.upload", "mobile_supplier.chrome.mapping", "mobile_supplier.chrome.review", "mobile_supplier.chrome.done"]
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/More/MoreHubView.swift
```swift
struct MoreHubView: View {
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Orders/OrderDetailPanel.swift
```swift
struct OrderDetailPanel: View {
let order: SupplierOrder
@Bindable var vm: OrdersViewModel
@State private var detailVM = OrderDetailViewModel()
⋮----
var body: some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/LoginView.swift
```swift
struct LoginView: View {
private enum Field: Hashable {
⋮----
@Environment(TokenStore.self) private var tokenStore
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var vm = SupplierLoginViewModel()
@FocusState private var focusedField: Field?
⋮----
private var formMaxWidth: CGFloat {
⋮----
var body: some View {
⋮----
private func submitLogin() {
```

## File: apps/warehouse-app-ios/WarehouseApp/Models/WarehouseModels.swift
```swift
struct LoginRequest: Encodable {
let phone: String
let pin: String
let idToken: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(phone: String = "", pin: String = "", idToken: String = "") {
⋮----
struct AuthResponse: Decodable {
let token: String
let refreshToken: String
let warehouseId: String
let isConfigured: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
// MARK: - Dashboard
⋮----
struct FleetStatusEntry: Decodable, Hashable {
let status: String
let count: Int64
⋮----
struct HoldReasonEntry: Decodable, Hashable {
let code: String
let count: Int
⋮----
struct DashboardData: Decodable {
let activeOrders: Int64
let completedToday: Int64
let pendingDispatch: Int64
let driversOnRoute: Int64
let driversIdle: Int64
let totalDrivers: Int64
let totalVehicles: Int64
let todayRevenue: Int64
let completedTodayAvailable: Bool
let todayRevenueAvailable: Bool
let lowStockCount: Int64
let totalStaff: Int64
let fleetStatus: [FleetStatusEntry]
let ordersByStatus: [String: Int]
let truckDuty: [String: Int]
let holdReasons: [HoldReasonEntry]
let demandSource: String
let historyAvailable: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
static let empty = DashboardData()
⋮----
struct Order: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let retailerName: String
let state: String
let totalUzs: Int
let lineItems: [LineItem]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct OrderReceiptMeta: Decodable {
let receiptId: String
let htmlUrl: String
let pdfUrl: String
let qrUrl: String
let partyCopy: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct LineItem: Decodable, Identifiable {
var id: String { productId }
let productId: String
let productName: String
let quantity: Int
let unitPrice: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct OrderListResponse: Decodable {
let orders: [Order]
⋮----
// MARK: - Driver
⋮----
struct Driver: Decodable, Identifiable {
var id: String { driverId }
let driverId: String
let name: String
let phone: String
let truckStatus: String
let isActive: Bool
let vehicleId: String?
let vehicleClass: String?
let vehicleIsActive: Bool
let vehicleUnavailableReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DriverListResponse: Decodable {
let drivers: [Driver]
⋮----
struct CreateDriverRequest: Encodable {
let name: String
let phone: String
⋮----
struct CreateDriverResponse: Decodable {
let driverId: String
let pin: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AssignDriverVehicleRequest: Encodable {
let vehicleId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AssignDriverVehicleResponse: Decodable {
let status: String
let driverId: String
let vehicleId: String?
let previouslyAssignedDriver: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum VehicleUnavailableReasonOption: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
var title: String {
⋮----
func formatUnavailableReason(_ reason: String?, note: String? = nil) -> String {
⋮----
func vehicleUnavailableReasonLabel(_ reason: String) -> String {
⋮----
struct Vehicle: Decodable, Identifiable {
var id: String { vehicleId }
let vehicleId: String
let label: String
let licensePlate: String
let vehicleClass: String
let capacityVu: Int
let status: String
let isActive: Bool
let unavailableReason: String?
let unavailableNote: String?
let assignedDriverId: String?
let assignedDriverName: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct VehicleListResponse: Decodable {
let vehicles: [Vehicle]
⋮----
struct VehicleDetailResponse: Decodable {
let vehicle: Vehicle
⋮----
struct CreateVehicleRequest: Encodable {
let label: String
let licensePlate: String
let vehicleClass: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct UpdateVehicleRequest: Encodable {
let isActive: Bool?
let unavailableReason: String?
let unavailableNote: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct VehicleMutationResponse: Decodable {
let status: String
let vehicleId: String
let unavailableReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InventoryItem: Decodable, Identifiable {
var id: String { productId }
let productId: String
let productName: String
let quantity: Int
let reorderThreshold: Int
let outOfStockPolicy: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct InventoryPolicyPatchRequest: Encodable {
let outOfStockPolicy: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DeliveryFeeTier: Codable, Equatable {
let maxKm: Double?
let feeMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DeliveryFeeRules: Codable, Equatable {
let currency: String
let baseFeeMinor: Int64
let tiers: [DeliveryFeeTier]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOpsSettingsResponse: Decodable {
let warehouseId: String
let name: String
let defaultOutOfStockPolicy: String
let showStockCountsToRetailers: Bool
let operatingSchedule: [String: AnyCodable]?
let opsAlwaysAvailable: Bool
let expressEnabled: Bool
let expressStockFloor: Int64
let preorderMinLeadDays: Int64
let preorderMaxLeadDays: Int64
let orderLineMinQuantity: Int64?
let orderLineMaxQuantity: Int64?
let deliveryFeeRules: DeliveryFeeRules?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseReturnPolicy: Codable {
var supplierId: String
var reverseDockSlaHours: Int64?
var retailerFileWindowHours: Int64?
var canOverrideRetailerWindow: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOpsSettingsPatchRequest: Encodable {
let defaultOutOfStockPolicy: String
let showStockCountsToRetailers: Bool?
let operatingSchedule: [String: AnyCodable]
let preorderMinLeadDays: Int64?
let preorderMaxLeadDays: Int64?
let orderLineMinQuantity: Int64?
let orderLineMaxQuantity: Int64?
let clearOrderLineMinQuantity: Bool?
let clearOrderLineMaxQuantity: Bool?
let expressEnabled: Bool?
let expressStockFloor: Int64?
let deliveryFeeRules: DeliveryFeeRules?
let clearDeliveryFeeRules: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehousePreorderRow: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let status: String
let orderSource: String?
let confirmationStatus: String?
let requestedDeliveryDate: String?
let proposedDeliveryDate: String?
let deliveryProposalReason: String?
let preorderBadge: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehousePreordersResponse: Decodable {
let preorders: [WarehousePreorderRow]
let items: [WarehousePreorderRow]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case preorders, items }
⋮----
struct StockCommitmentRow: Decodable, Identifiable {
var id: String { skuId }
let skuId: String
let name: String?
let imageUrl: String?
let onHand: Int64
let availableQty: Int64
let reservedAsap: Int64
let reservedScheduled: Int64
let deficitQty: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StockCommitmentsResponse: Decodable {
let items: [StockCommitmentRow]
let skus: [StockCommitmentRow]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case items, skus }
⋮----
struct InventoryListResponse: Decodable {
let items: [InventoryItem]
⋮----
struct InventoryAdjustRequest: Encodable {
let productId: String
let quantity: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseBinCreateRequest: Encodable {
var locationId: String?
var zone: String?
var locationType: String
var pickSequence: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseBinLocation: Decodable {
let warehouseId: String?
let locationId: String
let zone: String?
let locationType: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StockLotPutawayRequest: Encodable {
let productId: String
let locationId: String
let quantity: Int
var lotCode: String?
var expiryDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StockLotPutawayResponse: Decodable {
let lotId: String
let productId: String?
let locationId: String?
let quantityOnHand: Int?
let status: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PickTask: Decodable, Identifiable {
var id: String { taskId }
let taskId: String
let orderId: String?
let productId: String
let lotId: String
let locationId: String?
let quantityRequested: Int
let quantityPicked: Int?
let status: String
let pickSequence: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PickWave: Decodable {
let waveId: String
let warehouseId: String?
let manifestId: String
let status: String
let tasks: [PickTask]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PickWaveListResponse: Decodable {
let waves: [PickWave]
let pickWavesEnabled: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PickWaveCreateRequest: Encodable {
let manifestId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PickTaskConfirmRequest: Encodable {
var quantityPicked: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CycleCount: Decodable, Identifiable {
var id: String { countId }
let countId: String
let locationId: String?
let productId: String
let expectedQty: Int
let countedQty: Int?
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CycleCountListResponse: Decodable {
let counts: [CycleCount]
⋮----
struct CycleCountCreateRequest: Encodable {
let locationId: String
let productId: String
var expectedQty: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CycleCountSubmitRequest: Encodable {
let countedQty: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct Product: Decodable, Identifiable {
var id: String { productId }
let productId: String
let name: String
let skuId: String
let category: String
let priceUzs: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ProductListResponse: Decodable {
let products: [Product]
⋮----
// MARK: - Manifest
⋮----
struct Manifest: Decodable, Identifiable {
var id: String { manifestId }
let manifestId: String
let driverName: String
let vehicleLabel: String
let stopCount: Int
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ManifestListResponse: Decodable {
let manifests: [Manifest]
⋮----
// MARK: - Analytics
⋮----
struct DailyMetric: Decodable, Identifiable {
var id: String { date }
let date: String
let revenue: Int
let orders: Int
let completed: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(date: String = "", revenue: Int = 0, orders: Int = 0, completed: Int = 0) {
⋮----
struct AnalyticsData: Decodable {
let period: String
let totalOrders: Int
let totalRevenue: Int
let completedOrders: Int
let cancelledOrders: Int
let avgOrderValue: Double
let fleetUtilizationPct: Double
let topProducts: [TopProduct]
let dailyBreakdown: [DailyMetric]
let daily: [DailyMetric]
let importFreshness: ImportFreshness
let importAnomalyQueue: ImportAnomalyQueue
⋮----
var chartDaily: [DailyMetric] {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
static let empty = AnalyticsData(
⋮----
init(
⋮----
struct FleetUtilization: Decodable {
let utilizationPct: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
static let empty = FleetUtilization(utilizationPct: 0)
⋮----
struct ImportFreshness: Decodable {
let appliedRows30d: Int
let appliedSkus30d: Int
let quantityDelta30d: Int
let lastSessionId: String
let lastAppliedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
static let empty = ImportFreshness(
⋮----
struct ImportAnomalyQueue: Decodable {
let openRows30d: Int
let affectedSessions30d: Int
let lastSessionId: String
let lastDetectedAt: String
let lastDetail: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
static let empty = ImportAnomalyQueue(
⋮----
struct TopProduct: Decodable, Identifiable {
var id: String { productName }
let productName: String
let unitsSold: Int
let revenue: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - Retailer (CRM)
⋮----
struct Retailer: Decodable, Identifiable {
var id: String { retailerId }
let retailerId: String
let name: String
let totalOrders: Int
let totalRevenue: Int
let lastOrderDate: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RetailerListResponse: Decodable {
let retailers: [Retailer]
⋮----
// MARK: - Return
⋮----
struct ReturnItem: Decodable, Identifiable {
var id: String { lineItemId }
let lineItemId: String
let orderId: String
let productName: String
let quantity: Int
let status: String
let updatedAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ReturnListResponse: Decodable {
let items: [ReturnItem]
⋮----
init(from decoder: Decoder) throws {
⋮----
private enum CodingKeys: String, CodingKey {
⋮----
struct InboundReturnRow: Decodable, Identifiable {
var id: String { returnId }
let returnId: String
let orderId: String
let productName: String
let expectedQty: Int
let receivedQty: Int
let reason: String
let physicalStatus: String
let driverName: String
let driverNotes: String
let suggestedDisposition: String
let barcode: String?
⋮----
var isClaimTicket: Bool {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - Credit-note reverse logistics
⋮----
struct ReverseLogisticsTask: Decodable, Identifiable {
var id: String { taskId }
let taskId: String
let orderId: String
let status: String
let warehouseId: String
let expectedQtyJson: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ReverseLogisticsListResponse: Decodable {
let tasks: [ReverseLogisticsTask]
⋮----
init(from decoder: Decoder) throws {
⋮----
private enum CodingKeys: String, CodingKey { case tasks }
⋮----
struct ReverseLogisticsReceiveRequest: Encodable {
let warehouseId: String
let receivedQty: [String: Int]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DeliveryExpectationWire: Decodable {
let targetLabel: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseOpsException: Decodable, Identifiable {
var id: String {
⋮----
let exceptionId: String
let kind: String
let orderId: String
let manifestId: String
let reason: String
let status: String
let updatedAt: String
let deliveryExpectation: DeliveryExpectationWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseOpsExceptionsResponse: Decodable {
let exceptions: [WarehouseOpsException]
⋮----
init(from decoder: Decoder) throws {
⋮----
private enum CodingKeys: String, CodingKey { case exceptions }
⋮----
// MARK: - Claims (read-only)
⋮----
struct WarehouseClaimLine: Decodable, Identifiable {
var id: String { "\(sku)-\(quantity)" }
let sku: String
let quantity: Int
let reason: String
let amountMinor: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseClaim: Decodable, Identifiable {
var id: String { claimId }
let claimId: String
let orderId: String
let retailerId: String
let claimType: String
let status: String
let amountMinor: Int
let currency: String
let description: String
let lineItems: [WarehouseClaimLine]
let createdAt: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseClaimsResponse: Decodable {
let claims: [WarehouseClaim]
⋮----
init(from decoder: Decoder) throws {
⋮----
private enum CodingKeys: String, CodingKey { case claims }
⋮----
// MARK: - Fleet rescue
⋮----
struct RescuePreviewRequest: Encodable {
let brokenDriverId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RescueOption: Decodable, Identifiable {
var id: String { driverId }
let driverId: String
let name: String
let licensePlate: String
let truckStatus: String
let effectiveCapacityVu: Double
let isCapacityExceeded: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RescuePreviewResponse: Decodable {
let brokenDriverId: String
let pendingVolumeVu: Double
let rescueOptions: [RescueOption]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RescueProposeRequest: Encodable {
let rescueId: String
let brokenDriverId: String
let rescueDriverId: String
let forceCapacity: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundReturnListResponse: Decodable {
let data: [InboundReturnRow]
⋮----
struct InboundSessionResponse: Decodable {
let sessionId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundScanBody: Encodable {
let barcode: String
let qty: Int
let sessionId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundConfirmLine: Encodable {
let returnId: String
let disposition: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundConfirmBody: Encodable {
let lines: [InboundConfirmLine]
let sessionId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct InboundConfirmResponse: Decodable {
let status: String?
⋮----
struct InboundScanResponse: Decodable {
let matched: Bool
let returnId: String?
let variance: Bool
let message: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TreasuryOverview: Decodable {
let balance: Int
let totalReceivable: Int
let totalCollected: Int
let overdueAmount: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
static let empty = TreasuryOverview(balance: 0, totalReceivable: 0, totalCollected: 0, overdueAmount: 0)
⋮----
init(balance: Int, totalReceivable: Int, totalCollected: Int, overdueAmount: Int) {
⋮----
struct Invoice: Decodable, Identifiable {
var id: String { invoiceId }
let invoiceId: String
let retailerName: String
let amountUzs: Int
let currency: String
let status: String
let dueDate: String
let feeAmount: Int
let netPayoutAmount: Int
let payoutOwnerType: String
let payoutOwnerId: String
let feePolicyVersion: String
let settlementTarget: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct InvoiceListResponse: Decodable {
let invoices: [Invoice]
⋮----
// MARK: - Dispatch
⋮----
struct DispatchPreview: Decodable {
let undispatchedOrders: [DispatchOrder]
let availableDrivers: [AvailableDriver]
let unavailableDrivers: [AvailableDriver]
let proposedRoutes: [DispatchProposedRoute]
let optimizerSource: String?
let optimizerWarnings: [String]
let windowConstrainedCount: Int
let fleetEffectiveCapacityVu: Double
let planFingerprint: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DispatchProposedRoute: Decodable, Identifiable {
var id: String { "\(driverId ?? "route")-\(orderIds.joined(separator: "-"))" }
let driverId: String?
let driverName: String?
let vehicleId: String?
let orderIds: [String]
let stops: [DispatchProposedStop]
let volumeVu: Double?
let loadedVolume: Double?
let maxVolumeVu: Double?
let stopCount: Int?
let routeGeometry: RouteGeometryWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DispatchProposedStop: Decodable {
let orderId: String
let retailerId: String?
let retailerName: String?
let lat: Double?
let lng: Double?
let volumeVu: Double?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchOrder: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let retailerName: String
let totalUzs: Int
let itemCount: Int
let volumeVu: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct AvailableDriver: Decodable, Identifiable {
var id: String { driverId }
let driverId: String
let name: String
let phone: String
let vehicleLabel: String
let truckStatus: String
let maxVolumeVu: Double
let usedVolumeVu: Double?
let freeVolumeVu: Double?
let activeManifestId: String?
let unavailableReason: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DispatchCapacityWarning: Decodable {
let driverId: String
let loadedVu: Double
let maxVolumeVu: Double
let effectiveMaxVu: Double
let excessVu: Double
let suggestedUnselectOrderIds: [String]
let suggestedDeferOrderIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DispatchExecuteResponse: Decodable {
let status: String
let ordersAssigned: Int
let warnings: [String]
let capacityWarnings: [DispatchCapacityWarning]
let orphanOrderIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DispatchExecuteRouteRequest: Encodable {
let driverId: String
let orderIds: [String]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DispatchExecuteRequest: Encodable {
let mode: String
let forceCapacity: Bool
let acceptPartial: Bool?
let orderIds: [String]?
let planFingerprint: String?
let routes: [DispatchExecuteRouteRequest]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SupplyRequestListResponse: Decodable {
let requests: [WarehouseSupplyRequest]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DemandForecastDay: Decodable, Identifiable {
var id: String { date }
let date: String
let projectedUnits: Int64
let projectedRevenue: Int64
let committedUnits: Int64
let pendingConfirmationUnits: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct DemandForecastSources: Decodable {
let incomingOrders: Int64
let aiPrediction: Int64
let preOrders: Int64
let burnRate: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(incomingOrders: Int64 = 0, aiPrediction: Int64 = 0, preOrders: Int64 = 0, burnRate: Double = 0) {
⋮----
struct DemandForecastProduct: Decodable, Identifiable {
var id: String { productId }
let productId: String
let productName: String
let currentStock: Int64
let recommendedQty: Int64
let daysUntilStockout: Double
let priority: String
let unit: String
let sources: DemandForecastSources
let demandBreakdown: [String: AnyCodable]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
struct DemandForecastResponse: Decodable {
let warehouseId: String
let forecastDays: Int
let generatedAt: String?
let series: [DemandForecastDay]
let products: [DemandForecastProduct]
let source: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseSupplyRequest: Decodable, Identifiable {
var id: String { requestId }
let requestId: String
let warehouseId: String
let factoryId: String
let supplierId: String
let state: String
let priority: String
let requestedDeliveryDate: String?
let totalVolumeVu: Double
let notes: String
let transferOrderId: String?
let createdBy: String
let createdAt: String
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct SupplyRequestQCResponse: Decodable {
let requestId: String
let result: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseDispatchLock: Decodable, Identifiable {
var id: String { lockId }
let lockId: String
let supplierId: String
let warehouseId: String
let factoryId: String
let lockType: String
let lockedAt: String
let unlockedAt: String?
let lockedBy: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseLiveEvent: Decodable {
let type: String
let warehouseId: String
let requestId: String?
let state: String?
let lockId: String?
let action: String?
let timestamp: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateWarehouseSupplyRequestItem: Encodable {
let productId: String
let requestedQuantity: Int
let recommendedQty: Int
let unitVolumeVu: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateWarehouseSupplyRequestRequest: Encodable {
let factoryId: String
let priority: String
let notes: String
let items: [CreateWarehouseSupplyRequestItem]
let useDemandForecast: Bool
let requestedDeliveryDate: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateWarehouseSupplyRequestResponse: Decodable {
let requestId: String
let state: String
let priority: String
let totalVolumeVu: Double
let itemsCount: Int
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseSupplyRequestTransitionRequest: Encodable {
let action: String
let transferOrderId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseSupplyRequestTransitionResponse: Decodable {
let requestId: String
let state: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateWarehouseDispatchLockRequest: Encodable {
let lockType: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CreateWarehouseDispatchLockResponse: Decodable {
let lockId: String
let lockType: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReleaseWarehouseDispatchLockResponse: Decodable {
let lockId: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StaffMember: Decodable, Identifiable {
var id: String { workerId }
let workerId: String
let name: String
let phone: String
let role: String
let isActive: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct StaffListResponse: Decodable {
let staff: [StaffMember]
⋮----
struct CreateStaffRequest: Encodable {
let name: String
let phone: String
let role: String
⋮----
struct CreateStaffResponse: Decodable {
let workerId: String
let pin: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PaymentGateway: Decodable, Identifiable {
var id: String { code }
let code: String
let name: String
let provider: String
let isActive: Bool
let status: String
let selectable: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PSPListing: Decodable, Identifiable {
var id: String { code }
let code: String
let status: String
let selectable: Bool
⋮----
struct PaymentConfigResponse: Decodable {
let gateways: [PaymentGateway]
let catalog: [PSPListing]
let currencyCode: String
let marketCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseCoverageResponse: Decodable {
let warehouseId: String
let mode: String
let cities: [WarehouseCoverageCity]
let pins: [WarehouseServicePin]
let countryCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseCoverageCity: Decodable, Identifiable {
var id: String { "\(name):\(lat):\(lng)" }
let name: String
let lat: Double
let lng: Double
⋮----
struct WarehouseServicePin: Decodable, Identifiable {
var id: String { "\(targetType):\(targetId)" }
let targetType: String
let targetId: String
let priority: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct WarehouseSupplyFactoryResponse: Decodable {
let warehouseId: String
let factoryId: String
let transferMode: String
let source: String
let countryCode: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
// MARK: - JSON helpers
⋮----
struct AnyCodable: Codable {
let value: Any
⋮----
init(_ value: Any) {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
func prettyJSONString() -> String {
⋮----
struct WarehouseOpsBoardDeliveryExpectation: Decodable {
let targetLabel: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOpsBoardOrder: Decodable, Identifiable {
var id: String { orderId }
let orderId: String
let status: String
let retailerId: String?
let totalMinor: Int64
let deliveryExpectation: WarehouseOpsBoardDeliveryExpectation?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct WarehouseOpsBoardResponse: Decodable {
let date: String
let warehouseId: String
let preorders: [WarehouseOpsBoardOrder]
let deliverBefore: [WarehouseOpsBoardOrder]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct LaborZoneCapacity: Decodable, Identifiable {
var id: String { "\(zoneH3)-\(date)" }
let zoneH3: String
let date: String
let totalCapacity: Double
let usedCapacity: Double
let computedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(zoneH3: String, date: String, totalCapacity: Double, usedCapacity: Double, computedAt: String?) {
⋮----
struct LaborZoneCapacityListResponse: Decodable {
let zones: [LaborZoneCapacity]
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct LaborDriverScore: Decodable {
let driverId: String
let score: Double
let onTimeRate: Double
let completionRate: Double
let damageRate: Double
let shopClosedRate: Double
let feedbackScore: Double
let stopsPerHour: Double
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct LaborDriverAvailabilityRequest: Encodable {
let driverId: String
let date: String
let availableHours: Double
let status: String
let zoneH3: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct LaborDriverAvailabilityResponse: Decodable {
let status: String
⋮----
init(from decoder: Decoder) throws {
⋮----
enum CodingKeys: String, CodingKey { case status }
```

## File: apps/warehouse-app-ios/WarehouseApp/Services/APIClient.swift
```swift
final class APIClient: Sendable {
static let shared = APIClient()
⋮----
private struct TokenRefreshResponse: Decodable {
let token: String
let refreshToken: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
private struct EmptyBody: Encodable {}
⋮----
private let bootstrapURL: URL = {
⋮----
else if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
⋮----
private let bootstrapURL: URL = {
⋮----
return URL(string: s) ?? URL(string: "https://api.pegasusx.app/") ?? URL(fileURLWithPath: "/")
⋮----
private func buildURL(path: String, query: [String: String] = [:]) throws -> URL {
⋮----
func get<T: Decodable>(_ path: String, query: [String: String] = [:]) async throws -> T {
⋮----
func getJSONString(_ path: String, query: [String: String] = [:]) async throws -> String {
⋮----
func post<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
⋮----
func postEmpty<T: Decodable>(_ path: String, idempotencyKey: String? = nil) async throws -> T {
⋮----
func postVoid<B: Encodable>(_ path: String, body: B) async throws {
⋮----
func patch<B: Encodable, T: Decodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws -> T {
⋮----
func put<B: Encodable, T: Decodable>(
⋮----
func patchVoid<B: Encodable>(_ path: String, body: B, idempotencyKey: String? = nil) async throws {
⋮----
func delete<T: Decodable>(_ path: String, query: [String: String] = [:], idempotencyKey: String? = nil) async throws -> T {
⋮----
func registerDeviceToken(token: String, platform: String = "ios") async throws {
⋮----
private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
⋮----
private func attachToken(_ request: inout URLRequest) async {
⋮----
private func attemptRefresh() async throws -> Bool {
⋮----
private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
⋮----
func warehouseWebSocketURL(token: String, lastEventId: String? = nil) -> URL {
⋮----
private actor WarehouseHTTPCoalescer {
private var inFlightGETs: [String: Task<(Data, URLResponse), Error>] = [:]
private var getCache: [String: (data: Data, response: URLResponse, expiresAt: Date)] = [:]
private var refreshTask: Task<Bool, Error>?
private let cacheTTL: TimeInterval = 2.0
⋮----
func fetchOrCoalesce(
⋮----
func invalidateCache() {
⋮----
func refreshToken(using operation: @Sendable @escaping () async throws -> Bool) async throws -> Bool {
⋮----
enum APIError: LocalizedError {
⋮----
var errorDescription: String? {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Components/AddressLocationField.swift
```swift
struct AddressLocationValue: Equatable {
var address: String = ""
var lat: Double = 0
var lng: Double = 0
var placeId: String?
⋮----
struct AddressLocationField: View {
@Binding var value: AddressLocationValue
var label: String = "Address"
⋮----
@State private var vm = AddressLocationFieldViewModel()
@FocusState private var focused: Bool
⋮----
private var pinned: Bool {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchPreviewMapView.swift
```swift
struct DispatchPreviewMapView: View {
let routes: [DispatchProposedRoute]
⋮----
@State private var cameraPosition: MapCameraPosition = .region(
⋮----
private var routesWithGeometry: [DispatchProposedRoute] {
⋮----
var body: some View {
⋮----
private func fitCamera() {
⋮----
private func dispatchRouteColor(_ index: Int) -> Color {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Dispatch/DispatchView.swift
```swift
struct DispatchView: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@Environment(\.scenePhase) private var scenePhase
@State private var vm = DispatchViewModel()
⋮----
var body: some View {
⋮----
private var dispatchBody: some View {
⋮----
private var dispatchDialogs: some View {
⋮----
private func dispatchContent(preview: DispatchPreview) -> some View {
⋮----
private func ordersSegment(preview: DispatchPreview) -> some View {
⋮----
private func driversSegment(preview: DispatchPreview) -> some View {
⋮----
private var supplySegment: some View {
⋮----
private var locksSegment: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Fleet/FleetLiveMapSection.swift
```swift
struct FleetLiveMapSection: View {
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@State private var vm = FleetLiveMapViewModel()
⋮----
var mapHeight: CGFloat = 280
var showsExpand: Bool = true
var onExpand: (() -> Void)?
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/More/MoreHubView.swift
```swift
struct MoreHubView: View {
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Orders/OrderDetailView.swift
```swift
struct OrderDetailView: View {
let orderId: String
@Environment(\.openURL) private var openURL
@State private var vm: OrderDetailViewModel
⋮----
init(orderId: String) {
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Scanner/BarcodeScannerScreen.swift
```swift
struct BarcodeScannerScreen: View {
@State private var vm = BarcodeScannerViewModel()
@FocusState private var isManualFieldFocused: Bool
⋮----
var body: some View {
⋮----
private var cameraSection: some View {
⋮----
private var scanBorderColor: Color {
⋮----
private var statusIndicatorSection: some View {
⋮----
private var manualInputSection: some View {
⋮----
private var historyQueueSection: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Yard/YardGateView.swift
```swift
struct YardGateView: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = YardGateViewModel()
⋮----
var onBack: (() -> Void)? = nil
⋮----
init(onBack: (() -> Void)? = nil) {
⋮----
var body: some View {
```

## File: apps/control-tower-macos/Sources/Views/OverviewView.swift
```swift
public struct OverviewView: View {
var viewModel: ControlTowerViewModel
@State private var isPulsing: Bool = false
@Environment(\.accessibilityReduceMotion) private var reduceMotion
⋮----
public init(viewModel: ControlTowerViewModel) {
⋮----
public var body: some View {
⋮----
private func metricCard(title: String, value: String, subtext: String, icon: String, color: Color, trend: String) -> some View {
```

## File: apps/driver-app-ios/driverappios/driverappios/Offline/DriverOfflineActionCatalog.swift
```swift
enum DriverOfflineActionStatus: String {
⋮----
enum DriverOfflineActionCatalog {
static let maxAttempts = OfflineHttpSemantics.maxAttemptsDefault
static let proximityMaxAge: TimeInterval = 120
⋮----
static let proximity = "v1/delivery/proximity-unlock"
static let shopClosed = "v1/delivery/shop-closed"
static let partial = "v1/delivery/partial-offload"
static let deliver = "v1/order/deliver"
static let submitOfflinePoD = "v1/delivery/submit-offline-pod"
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
⋮----
static func normalize(_ endpoint: String) -> String {
⋮----
static func priority(for endpoint: String) -> Int {
⋮----
static func isOfflineEligible(_ endpoint: String) -> Bool {
⋮----
static func isRetryableHTTP(_ code: Int) -> Bool {
⋮----
static func isSuccessHTTP(_ code: Int) -> Bool {
⋮----
static func isNetworkEnqueueable(_ error: Error) -> Bool {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
static func nowIso() -> String {
```

## File: apps/driver-app-ios/driverappios/driverappios/ViewModels/FleetViewModel.swift
```swift
// MARK: - Delivery-edge APIs
```

## File: apps/driver-app-ios/driverappios/driverappios/Views/FleetDVIRView.swift
```swift
struct FleetDVIRView: View {
⋮----
@State private var vm = FleetDVIRViewModel()
let onBack: () -> Void
⋮----
var body: some View {
⋮----
private struct ChecklistRow: View {
let number: Int
let title: String
let description: String
let isOk: Bool
let onToggle: () -> Void
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ActiveDeliveriesView.swift
```swift
struct ActiveDeliveriesView: View {
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = ActiveDeliveriesViewModel()
⋮----
private var orders: [Order] { vm.orders }
private var isLoading: Bool { vm.isLoading }
private var approachingOrderIds: Set<String> { vm.approachingOrderIds }
⋮----
var body: some View {
⋮----
private func listenForApproaching() async {
⋮----
private var emptyState: some View {
⋮----
private func loadOrders() async {
⋮----
struct DeliveryCard: View {
let order: Order
var isApproaching: Bool = false
let onTap: () -> Void
let onShowQR: () -> Void
⋮----
var body: some View {
⋮----
private var statusColor: Color {
⋮----
private var statusIcon: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/ArrivalView.swift
```swift
struct DockView: View {
@State private var vm = ArrivalViewModel()
⋮----
var body: some View {
⋮----
private var summaryRow: some View {
⋮----
private func dockMetric(title: String, value: String) -> some View {
⋮----
private func supplierSection(_ group: SupplierDockGroup) -> some View {
⋮----
private func dockOrderRow(_ order: TrackingOrder) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CatalogView.swift
```swift
struct CatalogView: View {
@Environment(CartManager.self) private var cart
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = CatalogViewModel()
⋮----
var onNavigateToSuppliers: () -> Void = {}
⋮----
private var categories: [ProductCategory] { vm.categories }
private var products: [Product] { vm.products }
private var isLoading: Bool { vm.isLoading }
private var loadError: String? { vm.loadError }
private var isSearching: Bool { vm.isSearching }
private var filteredProducts: [Product] { vm.filteredProducts }
⋮----
var body: some View {
⋮----
private func loadCategories() async {
⋮----
private func loadProducts() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CheckoutView.swift
```swift
struct CheckoutView: View {
var supplierIsActive: Bool = true
⋮----
@Environment(CartManager.self) private var cart
@Environment(\.dismiss) private var dismiss
@Environment(\.modelContext) private var modelContext
⋮----
@State private var vm = CheckoutViewModel()
⋮----
var body: some View {
⋮----
private func checkoutContent(
⋮----
private var stockWarningsSection: some View {
⋮----
private var oosItemsSection: some View {
⋮----
private func deliverySection(
⋮----
private func deliveryModeButton(_ title: String, mode: String, subtitle: String) -> some View {
⋮----
private var cartItemsSection: some View {
⋮----
private var paymentSection: some View {
⋮----
private var summarySection: some View {
⋮----
private func summaryRow(_ title: String, value: String, valueColor: Color = AppTheme.textPrimary) -> some View {
⋮----
private var submitButton: some View {
⋮----
private var successView: some View {
⋮----
private struct CheckoutCartItemRowView: View {
let item: CartItem
let isOos: Bool
let maximumQuantity: Int
let onUpdateQuantity: (Int) -> Void
⋮----
@State private var quantity: Int
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/LocationPickerView.swift
```swift
struct LocationPickerView: View {
@Environment(\.dismiss) private var dismiss
@State private var vm: LocationPickerViewModel
⋮----
var initialLatitude: Double
var initialLongitude: Double
var onConfirm: (Double, Double, String) -> Void
⋮----
init(
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/MySuppliersView.swift
```swift
struct MySuppliersView: View {
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = MySuppliersViewModel()
⋮----
private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]
⋮----
var body: some View {
⋮----
private func supplierCard(_ supplier: Supplier) -> some View {
⋮----
private func syncStatusBanner(_ message: String) -> some View {
⋮----
private var emptyState: some View {
⋮----
private func skeletonGrid(cardCount: Int) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/SupplierProductsView.swift
```swift
struct SupplierProductsView: View {
let supplier: Supplier
@Environment(CartManager.self) private var cart
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = SupplierProductsViewModel()
⋮----
var body: some View {
⋮----
private var supplierHeader: some View {
⋮----
private func syncStatusBanner(_ message: String) -> some View {
⋮----
private var productsSection: some View {
⋮----
private var loadingProductsSection: some View {
⋮----
private var emptyProductsSection: some View {
⋮----
private func productRow(_ product: Product) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/LocationManager.swift
```swift
final class LocationManager: NSObject, CLLocationManagerDelegate {
static let defaultLocation = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
⋮----
private let manager = CLLocationManager()
⋮----
var lastLocation: CLLocationCoordinate2D?
var authorizationStatus: CLAuthorizationStatus
var isLocating: Bool = false
var locationError: String?
⋮----
override init() {
⋮----
func requestPermission() {
⋮----
func requestLocation() {
⋮----
func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
⋮----
func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
⋮----
func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/RetailerWebSocket.swift
```swift
struct PaymentRequiredEvent: Decodable, Identifiable {
var id: String { orderId }
let type: String
let orderId: String
let invoiceId: String
let sessionId: String
let amountUzs: Int
let originalAmountUzs: Int
let availableCardGateways: [String]
let message: String
let paymentMethod: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct OrderCompletedEvent: Decodable {
let type: String
let orderId: String
let amountUzs: Int
let message: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct PaymentFailureEvent: Decodable {
let type: String
let orderId: String
let sessionId: String
let gateway: String
let message: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ShopClosedAlertEvent: Decodable, Identifiable {
let type: String
let orderId: String
let driverName: String
let options: [String]
let attemptId: String
⋮----
var id: String { "\(orderId)-\(attemptId)" }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PromotionChangedEvent: Decodable {
let type: String
let supplierId: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct CartSyncUpdatedEvent: Decodable {
let type: String
let retailerId: String
let itemCount: Int?
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
enum RetailerWSEvent {
⋮----
final class RetailerWebSocket {
static let shared = RetailerWebSocket()
⋮----
private(set) var isConnected = false
private(set) var reconnectEpoch = 0
private var task: URLSessionWebSocketTask?
private var session: URLSession?
private var retailerId: String?
private var subscriberContinuations: [UUID: AsyncStream<RetailerWSEvent>.Continuation] = [:]
private var shouldReconnect = false
private var reconnectWorkItem: DispatchWorkItem?
private let subscriberQueue = DispatchQueue(label: "RetailerWebSocket.subscribers")
⋮----
private var reconnectAttempts = 0
private let maxReconnectDelay: TimeInterval = 60.0
private let initialReconnectDelay: TimeInterval = 2.0
private var lastEventId: String? = nil
private var heartbeatTask: Task<Void, Never>? = nil
⋮----
private init() {}
⋮----
func eventStream() -> AsyncStream<RetailerWSEvent> {
⋮----
func connect(retailerId: String) {
⋮----
func disconnect() {
⋮----
private func startHeartbeat() {
⋮----
private func receiveNext() {
⋮----
private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
⋮----
private func scheduleReconnect() {
⋮----
private func emit(_ event: RetailerWSEvent) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/InsightsViewModel.swift
```swift
final class InsightsViewModel {
var analytics: RetailerAnalytics?
var detailed: RetailerDetailedAnalytics?
var predictions: [RetailerAIPrediction] = []
var isLoading = false
var selectedRange: DateRange = .month
var correctingId: String?
⋮----
private let api = APIClient.shared
⋮----
var delta: Int {
⋮----
func loadAnalytics() async {
⋮----
private static let fullDateFormatter: ISO8601DateFormatter = {
⋮----
func loadDetailedAnalytics() async {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ViewModels/OrdersViewModel.swift
```swift
final class OrdersViewModel {
var allOrders: [Order] = []
var predictions: [RetailerAIPrediction] = []
var isLoading = false
var loadError: String?
var orderActionPending = false
var commandStatus: String?
var commandSupplierId: String?
⋮----
var selectedTab: OrderTab = .active
var selectedOrder: Order?
var qrOverlayOrder: Order?
var deliveryProposalOrder: Order?
⋮----
private let api = APIClient.shared
⋮----
var showLoadErrorAlert: Bool {
⋮----
var commandFilteredOrders: [Order] {
⋮----
func applyCommandFilter(status: String, supplierId: String? = nil) {
⋮----
func clearCommandFilter() {
⋮----
var activeOrders: [Order] {
⋮----
var pendingOrders: [Order] {
⋮----
func loadData(silent: Bool = false) async {
⋮----
func listenWebSocket(modelContext: ModelContext) async {
⋮----
func flushPendingOrders(modelContext: ModelContext) async {
⋮----
func cancelOrder(_ order: Order) async {
⋮----
func confirmAiOrder(_ orderId: String) async {
⋮----
func rejectAiOrder(_ orderId: String) async {
⋮----
func confirmPreorder(_ orderId: String) async {
⋮----
func acceptDeliveryProposal(_ orderId: String) async {
⋮----
func rejectDeliveryProposal(_ orderId: String, reason: String? = "Retailer rejected proposed date") async {
⋮----
func editPreorder(_ order: Order) async {
⋮----
private func replayPendingOrder(_ order: PendingOrder) async throws -> Bool {
⋮----
private func pendingOrderIdempotencyKey(_ order: PendingOrder) -> String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/ContentView.swift
```swift
enum AppTab: String, CaseIterable {
⋮----
var title: String {
⋮----
var icon: String {
⋮----
enum SideMenuTab: String, Hashable, CaseIterable {
⋮----
var title: String {
⋮----
var icon: String {
⋮----
struct ContentView: View {
@Environment(CartManager.self) private var cart
@Environment(AuthManager.self) private var auth
@Environment(\.scenePhase) private var scenePhase
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(\.modelContext) private var modelContext
⋮----
@State private var vm = AppShellViewModel()
@State private var refreshCenter = RetailerRefreshCenter.shared
@Namespace private var namespace
⋮----
private let ws = RetailerWebSocket.shared
⋮----
var body: some View {
⋮----
private var ipadLayout: some View {
⋮----
private func sidebarItem(for tab: SideMenuTab) -> some View {
⋮----
private var ipadDetailContent: some View {
⋮----
private var iphoneLayout: some View {
⋮----
private func tabContent(_ tab: AppTab) -> some View {
⋮----
private var standardToolbar: some ToolbarContent {
```

## File: apps/supplier-app-ios/SupplierApp/Layout/SupplierAdaptiveShell.swift
```swift
enum CompactTab: Hashable {
⋮----
struct FocusedSupplierSectionKey: FocusedValueKey {
⋮----
var supplierSection: Binding<SupplierSection?>? {
⋮----
struct SupplierAdaptiveShell: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(TokenStore.self) private var tokenStore
@Environment(SupplierRealtimeHub.self) private var realtimeHub
@State private var vm = SupplierShellViewModel()
private var network = NetworkLifecycleEngine.shared
private var statusCenter = SupplierStatusCenter.shared
⋮----
var body: some View {
⋮----
private var regularShell: some View {
⋮----
private var sidebarGroups: [(title: String, items: [CollapsibleSidebarItem<SupplierSection>])] {
⋮----
private func items(for sections: [SupplierSection]) -> [CollapsibleSidebarItem<SupplierSection>] {
⋮----
private var compactShell: some View {
⋮----
private func sectionView(_ section: SupplierSection) -> some View {
```

## File: apps/supplier-app-ios/SupplierApp/Views/Components/AddressLocationField.swift
```swift
struct AddressLocationValue: Equatable {
var address: String = ""
var lat: Double = 0
var lng: Double = 0
var placeId: String?
⋮----
struct AddressLocationField: View {
@Binding var value: AddressLocationValue
var label: String = "mobile_supplier.chrome.address"
⋮----
@State private var vm = AddressLocationViewModel()
⋮----
var body: some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Layout/WarehouseAdaptiveShell.swift
```swift
struct FocusedWarehouseSectionKey: FocusedValueKey {
⋮----
var warehouseSection: Binding<WarehouseSection?>? {
⋮----
struct WarehouseAdaptiveShell: View {
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@Environment(WarehouseRealtimeHub.self) private var realtimeHub
@Environment(TokenStore.self) private var tokenStore
⋮----
@State private var sidebarSelection: WarehouseSection? = .dashboard
@State private var columnVisibility: NavigationSplitViewVisibility = .all
@State private var compactTab: WarehouseCompactTab = .dashboard
@State private var showInspector: Bool = false
@State private var searchFilter: String = ""
@FocusState private var isSidebarFilterFocused: Bool
private var network = NetworkLifecycleEngine.shared
private var wasOffline: Bool { !network.isOnline }
⋮----
var body: some View {
⋮----
// MARK: - iPad Control Tower 3-Column Split View
private var regularControlTowerShell: some View {
⋮----
private var sidebarView: some View {
⋮----
private var contentColumnView: some View {
⋮----
private var detailWorkbenchView: some View {
⋮----
private var compactFloorSuiteShell: some View {
⋮----
private var sidebarGroups: [(title: String, items: [WarehouseSection])] {
⋮----
private var filteredSidebarGroups: [(title: String, items: [WarehouseSection])] {
⋮----
private func sectionView(_ section: WarehouseSection) -> some View {
```

## File: apps/warehouse-app-ios/WarehouseApp/Views/Cashier/CashierBlindCountView.swift
```swift
struct CashierBlindCountView: View {
@Environment(\.dismiss) private var dismiss
@State private var vm = CashierViewModel()
var onBack: (() -> Void)? = nil
⋮----
init(onBack: (() -> Void)? = nil) {
⋮----
private let banknoteDenominations: [Int64] = [200000, 100000, 50000, 20000, 10000, 5000, 2000, 1000]
⋮----
var body: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Models/Order.swift
```swift
private let retailerMoneyFormatter: NumberFormatter = {
⋮----
private func formatRetailerMoney(_ amount: Int64, currency: String) -> String {
⋮----
func decodeInt64Lossy(forKey key: Key) throws -> Int64 {
⋮----
enum OrderStatus: String, Codable, CaseIterable {
⋮----
var displayName: String {
⋮----
var color: String {
⋮----
var isActive: Bool {
⋮----
var hasDeliveryToken: Bool {
⋮----
var timelineStepIndex: Int {
⋮----
static let timelineSteps = ["Placed", "Approved", "Dispatched", "Active", "Arrived", "Delivered"]
⋮----
var canCancel: Bool {
⋮----
var customerStage: CustomerStage {
⋮----
enum CustomerStage: String, Codable, CaseIterable {
⋮----
var displayName: String {
⋮----
var progressPercent: Int {
⋮----
struct OrderLineItem: Codable, Identifiable, Hashable {
let id: String
let productId: String
let productName: String
let variantId: String
let variantSize: String
let quantity: Int
let unitPrice: Int64
let totalPrice: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(id: String, productId: String, productName: String, variantId: String, variantSize: String, quantity: Int, unitPrice: Int64, totalPrice: Int64) {
⋮----
init(from decoder: Decoder) throws {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
// MARK: - Order
⋮----
struct Order: Codable, Identifiable, Hashable {
let id: String
let retailerId: String
let supplierId: String?
let supplierName: String?
let status: OrderStatus
let items: [OrderLineItem]
let totalAmount: Int64
let currency: String
let paymentGateway: String
let paymentStatus: String?
let routeId: String?
let autoConfirmAt: String?
let deliverBefore: String?
let orderSource: String?
let confirmationStatus: String?
let deliveryPriority: String?
let preorderBadge: String?
let proposedDeliveryDate: String?
let deliveryProposalReason: String?
let createdAt: String
let updatedAt: String
let estimatedDelivery: String?
let qrCode: String?
let version: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(id: String, retailerId: String, supplierId: String?, supplierName: String?, status: OrderStatus, items: [OrderLineItem], totalAmount: Int64, currency: String = packCurrency(MarketPackStore.pack), paymentGateway: String = "", paymentStatus: String? = nil, routeId: String? = nil, autoConfirmAt: String? = nil, deliverBefore: String? = nil, orderSource: String?, confirmationStatus: String? = nil, deliveryPriority: String? = nil, preorderBadge: String? = nil, proposedDeliveryDate: String? = nil, deliveryProposalReason: String? = nil, createdAt: String, updatedAt: String, estimatedDelivery: String?, qrCode: String?, version: Int64 = 0) {
⋮----
init(from decoder: Decoder) throws {
⋮----
// Handle Status Safely
⋮----
var isAiGenerated: Bool {
⋮----
var needsManualPreorderAction: Bool {
⋮----
var needsDeliveryProposalReview: Bool {
⋮----
var displayTotal: String {
⋮----
var itemCount: Int {
⋮----
var deliveryQRCodePayload: String? {
⋮----
private struct RetailerDeliveryQRPayload: Encodable {
let order_id: String
let token: String
⋮----
struct TrackingOrderItem: Codable, Identifiable, Hashable {
var id: String { productId }
let productId: String
let productName: String
let quantity: Int
let unitPrice: Int64
let lineTotal: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(productId: String, productName: String, quantity: Int, unitPrice: Int64, lineTotal: Int64) {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RouteGeometryCoordinate: Codable, Hashable {
let lat: Double
let lng: Double
⋮----
struct RouteGeometryWire: Codable, Hashable {
let routeId: String?
let encodedPolyline: String?
let coordinates: [RouteGeometryCoordinate]
let source: String
let stopCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TrackingOrder: Codable, Identifiable, Hashable {
var id: String { orderId }
let orderId: String
let supplierId: String
let supplierName: String
let warehouseId: String?
let warehouseName: String?
let driverId: String
let state: String
let totalAmount: Int
let orderSource: String
let driverLatitude: Double?
let driverLongitude: Double?
let liveLocationAvailable: Bool
let isApproaching: Bool
let deliveryToken: String
let createdAt: String
let fiscalStatus: String
let fiscalQr: String
let latestFiscalReceiptId: String
let items: [TrackingOrderItem]
let routeGeometry: RouteGeometryWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(
⋮----
init(from decoder: Decoder) throws {
⋮----
var displayTotal: String {
⋮----
/// Receipt row label for tracking / recent receipts.
var fiscalReceiptLabel: String {
⋮----
var isGreen: Bool {
⋮----
var hasDriverLocation: Bool {
⋮----
var deliveryQRCodePayload: String? {
⋮----
struct TrackingResponse: Codable {
let status: String?
let orders: [TrackingOrder]
let recentReceipts: [TrackingOrder]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ActiveFulfillmentItem: Codable, Identifiable, Hashable {
var id: String { orderId }
let orderId: String
let supplierId: String
let supplierName: String
let state: String
let adjustedAmount: Int
let itemCount: Int
let liveLocationAvailable: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ActiveFulfillmentsResponse: Codable {
let fulfillments: [ActiveFulfillmentItem]
let count: Int
⋮----
struct PendingPaymentSession: Codable, Identifiable, Hashable {
var id: String { sessionId ?? orderId }
let sessionId: String?
let orderId: String
let retailerId: String
let supplierId: String
let gateway: String?
let lockedAmount: Int
let currency: String
let status: String
let currentAttemptNo: Int
let invoiceId: String?
let redirectURL: String?
let expiresAt: String?
let createdAt: String
let updatedAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct PendingPaymentsResponse: Codable {
let pendingPayments: [PendingPaymentSession]
let count: Int
⋮----
enum CodingKeys: String, CodingKey {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/CategorySuppliersView.swift
```swift
struct CategorySuppliersView: View {
let category: ProductCategory
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = CategorySuppliersViewModel()
⋮----
private let columns = [GridItem(.adaptive(minimum: 160), spacing: 14)]
⋮----
var body: some View {
⋮----
private func supplierCard(_ supplier: Supplier) -> some View {
⋮----
private func syncStatusBanner(_ message: String) -> some View {
⋮----
private var emptyState: some View {
⋮----
private var headerSkeleton: some View {
⋮----
private func skeletonGrid(cardCount: Int) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/DeliveryMapView.swift
```swift
struct DeliveryMapView: View {
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var network = NetworkLifecycleEngine.shared
@State private var vm = DeliveryMapViewModel()
⋮----
var body: some View {
⋮----
// Supplier filter chips
⋮----
// Active count badge
⋮----
// Loading overlay
⋮----
// Empty state
⋮----
// Selected order info card
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/LoginView.swift
```swift
struct LoginView: View {
@Environment(AuthManager.self) private var auth
@State private var vm = LoginViewModel()
@FocusState private var focusedField: Field?
⋮----
enum Field: Hashable {
⋮----
var body: some View {
⋮----
private func advanceFocusOrSubmit() {
⋮----
private func formField(
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/NotificationInboxView.swift
```swift
struct NotificationItem: Identifiable, Decodable {
let id: String
let type: String
let title: String
let body: String
let payload: String
let channel: String
let readAt: String?
let createdAt: String
⋮----
var isUnread: Bool { readAt == nil }
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct NotificationsResponse: Decodable {
let notifications: [NotificationItem]
let unreadCount: Int
let hasMore: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
final class NotificationInboxViewModel {
var items: [NotificationItem] = []
var unreadCount: Int = 0
var isLoading = true
var isLoadingMore = false
var hasMore = false
⋮----
private let api = APIClient.shared
private let pageSize = 50
private var nextOffset = 0
⋮----
func load() async {
⋮----
func loadMore() async {
⋮----
func markRead(_ id: String) async {
struct Payload: Encodable { let notification_ids: [String] }
⋮----
func markAllRead() async {
struct Payload: Encodable { let mark_all: Bool }
⋮----
private struct EmptyOK: Decodable {}
⋮----
struct NotificationInboxView: View {
@State private var vm = NotificationInboxViewModel()
@State private var refreshCenter = RetailerRefreshCenter.shared
@Environment(\.dismiss) private var dismiss
⋮----
var body: some View {
⋮----
private struct NotificationRow: View {
let notification: NotificationItem
⋮----
var body: some View {
⋮----
private var typeIcon: String {
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private var timeAgo: String {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/StoreStockView.swift
```swift
struct AggregatedStockItem: Identifiable, Equatable {
var id: String { sku }
let sku: String
var floorOnHand: Int64 = 0
var floorAvailable: Int64 = 0
var backroomOnHand: Int64 = 0
var backroomAvailable: Int64 = 0
⋮----
var totalOnHand: Int64 { floorOnHand + backroomOnHand }
var totalAvailable: Int64 { floorAvailable + backroomAvailable }
var isLowShelf: Bool { floorOnHand < 10 && backroomOnHand > 0 }
var hasBackroom: Bool { backroomOnHand > 0 }
⋮----
var shelfRatio: Double {
⋮----
enum StockFilterTab: String, CaseIterable, Identifiable {
⋮----
var id: String { rawValue }
⋮----
struct StoreStockRow: Identifiable, Equatable {
let id: String
let sku: String
let bin: String
let onHand: Int64
let available: Int64
⋮----
func asClaimOrder() -> Order {
⋮----
// MARK: - Main StoreStockView
⋮----
struct StoreStockView: View {
@State private var vm = StoreStockViewModel()
⋮----
var body: some View {
⋮----
// MARK: - Subviews
⋮----
private func bannerView(_ message: String) -> some View {
⋮----
private var kpiOverviewGrid: some View {
⋮----
private func kpiCard(title: String, value: String, subtitle: String, icon: String, color: Color) -> some View {
⋮----
private var actionShortcutsBar: some View {
⋮----
private var filterAndSearchBar: some View {
⋮----
private func stockItemCard(_ item: AggregatedStockItem) -> some View {
⋮----
private var emptyStateView: some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Services/APIClient.swift
```swift
private actor RetailerHTTPCoalescer {
private struct CacheEntry {
let data: Data
let response: URLResponse
let timestamp: Date
⋮----
private var inFlightGets: [String: Task<(Data, URLResponse), Error>] = [:]
private var microCache: [String: CacheEntry] = [:]
private let microCacheTTL: TimeInterval = 2.0
private var refreshTask: Task<String?, Never>?
⋮----
func fetchGet(
⋮----
func invalidateCache() {
⋮----
func refreshSingleFlight(
⋮----
final class APIClient {
static let shared = APIClient()
⋮----
private static let isoFormatter = ISO8601DateFormatter()
⋮----
private let session: URLSession
private let decoder: JSONDecoder
private let encoder: JSONEncoder
private let coalescer = RetailerHTTPCoalescer()
⋮----
var bootstrapURL: String = {
⋮----
if raw.hasPrefix("http://") || raw.hasPrefix("https://") { return raw }
⋮----
// MARK: - Generic Request
⋮----
func request<T: Decodable>(
⋮----
private func attemptTokenRefresh() async -> String? {
⋮----
private func performTransport(_ request: URLRequest, isRetry: Bool) async throws -> (Data, URLResponse) {
⋮----
private func dataForRequestWithFallback(_ request: URLRequest) async throws -> (Data, URLResponse) {
⋮----
func get<T: Decodable>(path: String) async throws -> T {
⋮----
func requestData(
⋮----
func post<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
⋮----
func patch<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
⋮----
func put<T: Decodable>(path: String, body: (any Encodable)? = nil, headers: [String: String] = [:]) async throws -> T {
⋮----
func getCards() async throws -> [RetailerCardToken] {
⋮----
func getOrderCurrencies() async throws -> OrderCurrencyOptions {
⋮----
func getPaymentCatalog() async throws -> RetailerPaymentCatalogResponse {
⋮----
func initiateCardSave(gateway: String = "GLOBAL_PAY") async throws -> CardInitiateResponse {
⋮----
func confirmCardSave(cardToken: String, otpCode: String) async throws -> CardConfirmResponse {
⋮----
func getClaimEligibility(orderId: String) async throws -> ClaimEligibility {
⋮----
func listOrderClaims(orderId: String) async throws -> [RetailerClaim] {
⋮----
func fileOrderClaim(
⋮----
func deactivateCard(tokenId: String) async throws {
⋮----
func setDefaultCard(tokenId: String) async throws {
⋮----
func respondToShopClosed(orderId: String, response: String, photoURL: String? = nil) async throws {
struct ShopClosedRequest: Encodable {
let orderId: String
let response: String
let photoURL: String?
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct ConfirmAiOrderRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey { case orderId = "order_id" }
⋮----
struct RejectAiOrderRequest: Encodable {
let orderId: String
let reason: String
enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
⋮----
struct ConfirmPreorderRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey { case orderId = "order_id" }
⋮----
struct AcceptDeliveryProposalRequest: Encodable {
let orderId: String
enum CodingKeys: String, CodingKey { case orderId = "order_id" }
⋮----
struct RejectDeliveryProposalRequest: Encodable {
let orderId: String
let reason: String?
enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
⋮----
struct RejectPreorderRequest: Encodable {
let orderId: String
let reason: String?
enum CodingKeys: String, CodingKey { case orderId = "order_id"; case reason }
⋮----
struct EditPreorderItem: Encodable {
let sku: String
let name: String
let quantity: Int64
let unitPriceMinor: Int64
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct EditPreorderRequest: Encodable {
let orderId: String
let requestedDeliveryDate: String
let lineItems: [EditPreorderItem]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func getRetailerAIPredictions(limit: Int? = nil) async throws -> [RetailerAIPrediction] {
⋮----
func confirmAiOrder(orderId: String) async throws {
⋮----
func rejectAiOrder(orderId: String, reason: String) async throws {
⋮----
func confirmPreorder(orderId: String) async throws {
⋮----
func acceptDeliveryProposal(orderId: String) async throws {
⋮----
func rejectDeliveryProposal(orderId: String, reason: String? = nil) async throws {
⋮----
func rejectPreorder(orderId: String, reason: String? = nil) async throws {
⋮----
func editPreorder(orderId: String, deliveryDate: String, items: [EditPreorderItem]) async throws {
⋮----
func setupRetailer(payload: [String: AnyEncodable], idempotencyKey: String? = nil) async throws {
⋮----
func requestCancelOrder(orderId: String, retailerId: String, reason: String = "Retailer requested cancellation") async throws {
⋮----
func getPricingRules() async throws -> [String: AnyDecodable] {
⋮----
func getProfile() async throws -> RetailerProfileResponse {
⋮----
func getCreditProfile() async throws -> CreditProfile {
⋮----
func getLoyaltyTier() async throws -> LoyaltyTierView {
⋮----
func getCreditRelationships() async throws -> [CreditRelationshipWire] {
⋮----
func getArInvoices(status: String = "OPEN") async throws -> [ArInvoiceWire] {
⋮----
func getHqSummary(day: String) async throws -> HqSummaryWire {
⋮----
func getHqSalesByLocation(day: String) async throws -> [HqLocationWire] {
⋮----
func updateProfile(request: RetailerProfileRequest, idempotencyKey: String) async throws {
⋮----
func getRetailerMe() async throws -> RetailerMeResponse {
⋮----
func getCapabilities() async throws -> RetailerCapabilitiesResponse {
⋮----
func enableCapability(packId: String, acceptSoft: Bool = true, enableDeps: Bool = true) async throws -> RetailerCapabilityMutationResponse {
struct Body: Encodable {
let accept_soft_deps: Bool
let enable_deps: Bool
⋮----
func disableCapability(packId: String) async throws -> RetailerCapabilityMutationResponse {
struct Empty: Encodable {}
⋮----
func getLocations() async throws -> RetailerLocationsResponse {
⋮----
func createLocation(name: String, address: String, lat: Double = 0, lng: Double = 0) async throws -> RetailerLocationsResponse {
struct Body: Encodable {
let name: String
let delivery_address: String
let lat: Double
let lng: Double
⋮----
func setPrimaryLocation(locationId: String) async throws -> RetailerLocationsResponse {
struct Empty: Encodable {}
⋮----
func switchLocation(locationId: String) async throws -> [String: AnyDecodable] {
struct Body: Encodable { let location_id: String }
⋮----
func getStoreStock(locationId: String?) async throws -> StoreStockListWire {
⋮----
func receiveStoreStock(orderId: String, locationId: String) async throws -> [String: AnyDecodable] {
struct Body: Encodable {
let order_id: String
let location_id: String
let confirm: Bool
let stock_bin: String
⋮----
func transferStoreStock(locationId: String, sku: String, qty: Int64) async throws -> [String: AnyDecodable] {
struct Body: Encodable {
let location_id: String
let sku: String
let qty: Int64
let from_bin: String
let to_bin: String
⋮----
func adjustStoreStock(locationId: String, sku: String, qtyDelta: Int64) async throws -> [String: AnyDecodable] {
struct Body: Encodable {
let location_id: String
let sku: String
let qty_delta: Int64
let stock_bin: String
let note: String
⋮----
func getLocalSkus() async throws -> LocalSkuListWire {
⋮----
func createLocalSku(name: String, barcode: String, priceMinor: Int64) async throws -> LocalSkuWire {
struct Body: Encodable {
let name: String
let barcode: String
let default_price_minor: Int64
let currency: String
⋮----
func patchLocalSku(id: String, isActive: Bool) async throws -> LocalSkuWire {
struct Body: Encodable { let is_active: Bool }
⋮----
func getRegisters() async throws -> RetailerRegistersWire {
⋮----
func createRegister(label: String) async throws -> RetailerRegisterWire {
struct Body: Encodable { let label: String }
⋮----
func openPosSession(registerId: String) async throws -> PosSessionWire {
struct Body: Encodable {
let register_id: String
let opening_float_minor: Int64
let currency: String
⋮----
func closePosSession(sessionId: String, closingCashMinor: Int64) async throws -> PosSessionWire {
struct Body: Encodable { let closing_cash_minor: Int64 }
⋮----
func createPosSale(
⋮----
struct Body: Encodable {
let session_id: String
let stock_bin: String
let lines: [PosSaleLineWire]
let tenders: [PosTenderWire]
let client_sale_id: String
let origin: String
let client_created_at: String?
⋮----
func voidPosSale(saleId: String) async throws -> PosSaleWire {
struct Body: Encodable { let reason: String }
⋮----
func listPosHolds(locationId: String?) async throws -> PosHoldsListWire {
⋮----
func parkPosHold(
⋮----
struct Cart: Encodable { let lines: [PosSaleLineWire] }
struct Body: Encodable {
let location_id: String
let register_id: String
let cart: Cart
let note: String?
⋮----
func resumePosHold(holdId: String, locationId: String) async throws -> PosHoldWire {
struct Body: Encodable { let location_id: String }
⋮----
func voidPosHold(holdId: String) async throws -> PosHoldWire {
struct EmptyBody: Encodable {}
⋮----
func clockIn(locationId: String? = nil) async throws -> TimeEntryWire {
struct Body: Encodable { let location_id: String? }
⋮----
func clockOut() async throws -> TimeEntryWire {
struct Empty: Encodable {}
⋮----
func getTimeEntries() async throws -> TimeEntriesWire {
⋮----
func getShifts(locationId: String? = nil) async throws -> ShiftsListWire {
⋮----
func openShift(registerId: String?, openingFloatMinor: Int64) async throws -> ShiftWire {
struct Body: Encodable {
let register_id: String?
let opening_float_minor: Int64
let currency: String
⋮----
func closeShift(shiftId: String, closingCashMinor: Int64) async throws -> ShiftWire {
struct Body: Encodable { let closing_cash_minor: Int64 }
⋮----
func getSections(locationId: String? = nil) async throws -> SectionsListWire {
⋮----
func createSection(name: String, aisleTag: String?) async throws -> SectionWire {
struct Body: Encodable {
let name: String
let aisle_tag: String?
⋮----
func putSectionSkus(sectionId: String, skus: [String]) async throws -> [String: AnyDecodable] {
struct Body: Encodable { let skus: [String] }
⋮----
func getReportsSummary() async throws -> ReportsSummaryWire {
⋮----
func exportReportsCSV(report: String = "sales") async throws -> Data {
⋮----
func getRetailerPulse() async throws -> RetailerPulseResponse {
⋮----
func getControlTowerPulse() async throws -> ControlTowerPulseWire {
⋮----
func getAssistTickets() async throws -> AssistTicketsWire {
⋮----
func createAssistTicket(sectionId: String, note: String) async throws -> AssistTicketWire {
struct Body: Encodable {
let section_id: String
let note: String
⋮----
func claimAssistTicket(ticketId: String) async throws -> AssistTicketWire {
struct Empty: Encodable {}
⋮----
func completeAssistTicket(ticketId: String) async throws -> AssistTicketWire {
struct Empty: Encodable {}
⋮----
func getOrgMembers() async throws -> RetailerOrgMembersResponse {
⋮----
func createOrgMember(name: String, phone: String, password: String, role: String) async throws -> RetailerOrgMembersResponse {
struct Body: Encodable {
let name: String
let phone: String
let password: String
let retailer_role: String
⋮----
func deactivateOrgMember(userId: String) async throws -> RetailerOrgMembersResponse {
struct Empty: Encodable {}
⋮----
func getFamilyMembersList() async throws -> FamilyMembersListResponse {
⋮----
func getFamilyMembers() async throws -> [FamilyMemberResponse] {
⋮----
func addFamilyMember(request: FamilyMemberRequest) async throws {
struct Empty: Decodable {}
⋮----
func migrateFamilyToTeam(role: String = "RECEIVER") async throws -> FamilyMigrateResult {
struct Body: Encodable {
let retailerRole: String
enum CodingKeys: String, CodingKey { case retailerRole = "retailer_role" }
⋮----
func removeFamilyMember(memberId: String) async throws {
⋮----
func getCartSync() async throws -> CartSyncResponse {
⋮----
func syncCart(request: CartSyncRequest) async throws -> CartSyncResponse {
⋮----
func checkoutQuote(supplierID: String, lines: [CheckoutQuoteLine]) async throws -> CheckoutQuoteResponse {
⋮----
func getSuppliers() async throws -> [RetailerSupplierResponse] {
⋮----
func favoriteSupplier(supplierId: String) async throws {
⋮----
func unfavoriteSupplier(supplierId: String) async throws {
⋮----
struct GlobalAutoOrderRequest: Encodable {
let globalAutoOrderEnabled: Bool?
let globalEnabled: Bool?
let executionMode: String?
let useHistory: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct ScopedAutoOrderRequest: Encodable {
let autoOrderEnabled: Bool
let useHistory: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
func getAutoOrderSettings() async throws -> AutoOrderSettings {
⋮----
func setGlobalAutoOrder(enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
⋮----
func setAutoOrderExecutionMode(mode: String) async throws -> [String: Bool] {
⋮----
func getAutoOrderShadowProposals() async throws -> AutoOrderShadowProposalsResponse {
⋮----
func getAutoOrderShadowStats() async throws -> AutoOrderShadowStats {
⋮----
func getAutoOrderSoakGate() async throws -> AutoOrderSoakGate {
⋮----
func setSupplierAutoOrder(supplierId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
⋮----
func setCategoryAutoOrder(categoryId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
⋮----
func setProductAutoOrder(productId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
⋮----
func setVariantAutoOrder(skuId: String, enabled: Bool, useHistory: Bool? = nil) async throws -> [String: Bool] {
⋮----
func runAutoOrder(mode: String = "draft") async throws -> AutoOrderRun {
⋮----
func getAutoOrderRuns() async throws -> AutoOrderRunsResponse {
⋮----
func getReorderSuggestions(source: String? = nil) async throws -> RetailerReorderSuggestionsResponse {
⋮----
func getTracking() async throws -> TrackingResponse {
⋮----
func getTrackingOrders() async throws -> [TrackingOrder] {
⋮----
func getPendingPayments() async throws -> PendingPaymentsResponse {
⋮----
func getActiveFulfillments() async throws -> ActiveFulfillmentsResponse {
⋮----
func getOrderTimeline(orderId: String) async throws -> OrderTimelineResponse {
⋮----
enum APIError: LocalizedError {
⋮----
var errorDescription: String? {
⋮----
var problem: ProblemDetail? {
⋮----
struct ProblemDetail: Codable {
let type: String?
let title: String?
let status: Int
let detail: String?
let traceId: String?
let instance: String?
let code: String?
let messageKey: String?
let retryable: Bool?
let action: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AnyEncodable: Encodable {
private let _encode: (Encoder) throws -> Void
⋮----
init(_ wrapped: any Encodable) {
⋮----
func encode(to encoder: Encoder) throws {
⋮----
struct RetailerRegistersWire: Codable {
let items: [RetailerRegisterWire]
⋮----
struct RetailerRegisterWire: Codable {
let registerId: String
let label: String?
let locationId: String?
enum CodingKeys: String, CodingKey {
⋮----
struct PosHoldCartWire: Codable {
let lines: [PosHoldCartLineWire]?
⋮----
struct PosHoldCartLineWire: Codable {
let sku: String
let name: String?
let qty: Int64?
let unitPriceMinor: Int64?
enum CodingKeys: String, CodingKey {
⋮----
struct PosHoldWire: Codable {
let holdId: String
let locationId: String?
let status: String?
let note: String?
let cart: PosHoldCartWire?
enum CodingKeys: String, CodingKey {
⋮----
struct PosHoldsListWire: Codable {
let items: [PosHoldWire]
⋮----
struct PosSessionWire: Codable {
let sessionId: String
let status: String?
enum CodingKeys: String, CodingKey {
⋮----
struct PosSaleLineWire: Encodable {
let sku: String
let name: String
let qty: Int64
let unit_price_minor: Int64
init(sku: String, name: String, qty: Int64, unitPriceMinor: Int64) {
⋮----
struct PosTenderWire: Encodable {
let method: String
let amount_minor: Int64
⋮----
struct PosSaleWire: Codable {
let saleId: String
let receiptNumber: String
let totalMinor: Int64?
let status: String?
enum CodingKeys: String, CodingKey {
⋮----
struct TimeEntriesWire: Codable {
let items: [TimeEntryWire]?
let clockedIn: Bool
let openEntry: TimeEntryWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct TimeEntryWire: Codable {
let entryId: String?
let userId: String?
let locationId: String?
let status: String?
let clockInAt: String?
let clockOutAt: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ShiftsListWire: Codable {
let items: [ShiftWire]
⋮----
struct ShiftWire: Codable {
let shiftId: String
let status: String
let openingFloatMinor: Int64
let varianceMinor: Int64?
let currency: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct SectionsListWire: Codable {
let items: [SectionWire]
⋮----
struct SectionWire: Codable {
let sectionId: String
let name: String
let aisleTag: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct ReportsSummaryWire: Codable {
let salesMinor: Int64?
let saleCount: Int?
let onHandSkuCount: Int?
let lowStockCount: Int?
let topSkus: [ReportsTopSkuWire]?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct RetailerPulseEvent: Codable, Identifiable {
let id: String
let kind: String?
let title: String
let description: String?
let occurredAt: String?
let deepLink: String?
let orderId: String?
let manifestId: String?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct RetailerPulseResponse: Codable {
let events: [RetailerPulseEvent]
let fetchedAt: String?
let unreadCount: Int?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RetailerSupplierOrderFacetWire: Decodable {
let supplierId: String
let ordersByStatus: [String: Int]
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct RetailerPulseLoyaltyWire: Decodable {
let enrolled: Bool
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
struct ControlTowerPulseWire: Decodable {
let retailerId: String?
let generatedAt: String?
let openOrders: Int
let activeFulfillments: Int
let dockPending: Int
let posOpenSessions: Int
let openShifts: Int
let openAssistTickets: Int
let lowStockSkuBins: Int
let shiftVariances7d: Int
let salesMinor7d: Int64
let capabilities: [String]?
let empty: Bool
let source: String?
let ordersByStatus: [String: Int]
let ordersBySupplier: [RetailerSupplierOrderFacetWire]
let loyalty: RetailerPulseLoyaltyWire?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
init(from decoder: Decoder) throws {
⋮----
init(
⋮----
struct ReportsTopSkuWire: Codable {
let sku: String?
let salesMinor: Int64?
let units: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AssistTicketsWire: Codable {
let items: [AssistTicketWire]
⋮----
struct AssistTicketWire: Codable {
let ticketId: String
let note: String
let status: String
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct LocalSkuListWire: Codable {
let items: [LocalSkuWire]
⋮----
struct LocalSkuWire: Codable {
let localSkuId: String
let name: String
let barcode: String?
let defaultPriceMinor: Int64?
let currency: String?
let isActive: Bool?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct StoreStockListWire: Codable {
let items: [StoreStockBalanceWire]
⋮----
struct StoreStockBalanceWire: Codable {
let locationId: String?
let stockBin: String
let sku: String
let onHand: Int64
let available: Int64?
⋮----
enum CodingKeys: String, CodingKey {
⋮----
struct AnyDecodable: Decodable {
let value: Any
⋮----
init(from decoder: Decoder) throws {
⋮----
private enum AuthNamespace {
static let primaryService = "com.pegasus.retailerapp"
⋮----
enum KeychainHelper {
static func save(key: String, value: String) {
⋮----
static func read(key: String) -> String? {
⋮----
static func delete(key: String) {
⋮----
private static func readFromService(_ service: String, key: String) -> String? {
⋮----
private static func deleteFromService(_ service: String, key: String) {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Theme/Theme.swift
```swift
static let background = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 14/255.0, green: 14/255.0, blue: 16/255.0, alpha: 1.0) : UIColor(red: 237/255.0, green: 237/255.0, blue: 240/255.0, alpha: 1.0) })
static let cardBackground = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 24/255.0, green: 24/255.0, blue: 27/255.0, alpha: 1.0) : UIColor.white })
static let secondaryBackground = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 32/255.0, green: 32/255.0, blue: 36/255.0, alpha: 1.0) : UIColor(red: 246/255.0, green: 246/255.0, blue: 248/255.0, alpha: 1.0) })
static let separator = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 228/255.0, green: 228/255.0, blue: 231/255.0, alpha: 1.0) })
static let cardBorder = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white.withAlphaComponent(0.08) : UIColor.black.withAlphaComponent(0.04) })
static let accent = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor.white : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
static let accentSoft = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 39/255.0, green: 39/255.0, blue: 42/255.0, alpha: 1.0) : UIColor(red: 226/255.0, green: 226/255.0, blue: 230/255.0, alpha: 1.0) })
static let accentWarm = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 210/255.0, green: 179/255.0, blue: 155/255.0, alpha: 1.0) : UIColor(red: 188/255.0, green: 127/255.0, blue: 112/255.0, alpha: 1.0) })
static let accentWarmSoft = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 54/255.0, green: 39/255.0, blue: 35/255.0, alpha: 1.0) : UIColor(red: 245/255.0, green: 236/255.0, blue: 233/255.0, alpha: 1.0) })
static let accentBronze = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 228/255.0, green: 191/255.0, blue: 162/255.0, alpha: 1.0) : UIColor(red: 98/255.0, green: 77/255.0, blue: 69/255.0, alpha: 1.0) })
static let iconTile = accentSoft
static let accentDark = accent
static let destructive = Color(uiColor: .systemRed)
static let destructiveSoft = Color(uiColor: .systemRed).opacity(0.1)
⋮----
static let error = destructive
static let success = Color(uiColor: .systemGreen)
static let successSoft = Color(uiColor: .systemGreen).opacity(0.1)
static let warning = Color(uiColor: .systemOrange)
static let warningSoft = Color(uiColor: .systemOrange).opacity(0.1)
static let info = Color(uiColor: .systemBlue)
static let infoSoft = Color(uiColor: .systemBlue).opacity(0.1)
static let textPrimary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 244/255.0, green: 244/255.0, blue: 245/255.0, alpha: 1.0) : UIColor(red: 17/255.0, green: 17/255.0, blue: 19/255.0, alpha: 1.0) })
static let textSecondary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) : UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) })
static let textTertiary = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 113/255.0, green: 113/255.0, blue: 122/255.0, alpha: 1.0) : UIColor(red: 161/255.0, green: 161/255.0, blue: 170/255.0, alpha: 1.0) })
static let surfaceElevated = Color(uiColor: UIColor { $0.userInterfaceStyle == .dark ? UIColor(red: 31/255.0, green: 31/255.0, blue: 35/255.0, alpha: 1.0) : UIColor.white })
⋮----
static let background = Color.gray.opacity(0.15)
static let cardBackground = Color.white
static let secondaryBackground = Color.gray.opacity(0.1)
static let separator = Color.gray.opacity(0.2)
static let cardBorder = Color.black.opacity(0.04)
static let accent = Color.black
static let accentSoft = Color.gray.opacity(0.15)
static let iconTile = accentSoft
static let accentDark = accent
static let destructive = Color.red
static let destructiveSoft = Color.red.opacity(0.1)
static let error = destructive
static let success = Color.green
static let successSoft = Color.green.opacity(0.1)
static let warning = Color.orange
static let warningSoft = Color.orange.opacity(0.1)
static let info = Color.blue
static let infoSoft = Color.blue.opacity(0.1)
static let textPrimary = Color.black
static let textSecondary = Color.gray
static let textTertiary = Color.gray.opacity(0.6)
static let surfaceElevated = Color.white
⋮----
static let accentGradient = LinearGradient(
⋮----
static let heroGradient = LinearGradient(
⋮----
static let warmGradient = LinearGradient(
⋮----
static let coolGradient = LinearGradient(
⋮----
static let meshBackground = LinearGradient(
⋮----
static let radiusXS: Double = 4
static let radiusSM: Double = 8
static let radiusMD: Double = 12
static let radiusLG: Double = 20
static let radiusXL: Double = 24
static let radiusXXL: Double = 32
static let radiusCard: Double = 24
static let radiusButton: Double = 16
static let radiusPill: Double = 100
⋮----
static let spacingXS: Double = 4
static let spacingSM: Double = 8
static let spacingMD: Double = 12
static let spacingLG: Double = 16
static let spacingXL: Double = 24
static let spacingXXL: Double = 32
static let spacingHuge: Double = 48
⋮----
static let shadowRadius: Double = 4
static let shadowColor = Color.black.opacity(0.04)
static let shadowOffsetY: Double = 2
static let shadowRadiusLG: Double = 8
static let shadowColorLG = Color.black.opacity(0.08)
⋮----
static let separatorHeight: Double = 0.33
⋮----
static let iconSM: Double = 16
static let iconMD: Double = 20
static let iconLG: Double = 24
static let iconXL: Double = 32
static let iconHuge: Double = 48
⋮----
static let readableMaxWidth: CGFloat = 960
⋮----
static let minTouchTarget: CGFloat = 44
⋮----
static func statusTint(for status: String) -> Color {
⋮----
struct ShimmerModifier: ViewModifier {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var phase: Double = 0
⋮----
func body(content: Content) -> some View {
⋮----
struct SkeletonModifier: ViewModifier {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var opacity: Double = 0.3
⋮----
func body(content: Content) -> some View {
⋮----
struct GlassCardModifier: ViewModifier {
var cornerRadius: Double = AppTheme.radiusCard
⋮----
func body(content: Content) -> some View {
⋮----
struct ElevatedCardModifier: ViewModifier {
var cornerRadius: Double = AppTheme.radiusCard
⋮----
func body(content: Content) -> some View {
⋮----
struct PressableButtonStyle: ButtonStyle {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
⋮----
func makeBody(configuration: Configuration) -> some View {
⋮----
static var pressable: PressableButtonStyle { PressableButtonStyle() }
⋮----
struct PressableModifier: ViewModifier {
func body(content: Content) -> some View {
⋮----
struct FloatingModifier: ViewModifier {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var yOffset: Double = 0
⋮----
func body(content: Content) -> some View {
⋮----
struct SlideInModifier: ViewModifier {
@Environment(\.accessibilityReduceMotion) private var reduceMotion
@State private var appeared = false
let delay: Double
⋮----
func body(content: Content) -> some View {
⋮----
func cardStyle() -> some View {
⋮----
func glassCard(cornerRadius: Double = AppTheme.radiusCard) -> some View {
⋮----
func elevatedCard(cornerRadius: Double = AppTheme.radiusCard) -> some View {
⋮----
func shimmer() -> some View {
⋮----
func skeleton() -> some View {
⋮----
func pressable() -> some View {
⋮----
func floating() -> some View {
⋮----
func slideIn(delay: Double = 0) -> some View {
⋮----
func staggeredSlideIn(index: Int, baseDelay: Double = 0.05) -> some View {
⋮----
func retailerCard() -> some View {
⋮----
func retailerReadableWidth() -> some View {
⋮----
func pegasusMinimumHitTarget(minWidth: CGFloat = AppTheme.minTouchTarget, minHeight: CGFloat = AppTheme.minTouchTarget) -> some View {
```

## File: apps/retailer-app-ios/retailerapp/retailerapp/Screens/DashboardView.swift
```swift
struct DashboardView: View {
@Environment(CartManager.self) private var cart
@Environment(\.horizontalSizeClass) private var horizontalSizeClass
@State private var refreshCenter = RetailerRefreshCenter.shared
@State private var vm = DashboardViewModel()
@State private var socket = RetailerWebSocket.shared
⋮----
private var activeOrders: [Order] { vm.activeOrders }
private var predictions: [RetailerAIPrediction] { vm.predictions }
private var reorderProducts: [Product] { vm.reorderProducts }
private var isLoading: Bool { vm.isLoading }
private var actingId: String? { vm.actingId }
private var orderActionPending: Bool {
⋮----
private var loadError: String? {
⋮----
private var pulseEvents: [RetailerPulseEvent] { vm.pulseEvents }
private var pulseLoading: Bool { vm.pulseLoading }
private var pulseError: String? { vm.pulseError }
private var commandPulse: ControlTowerPulseWire? { vm.commandPulse }
private var commandPulseError: String? { vm.commandPulseError }
⋮----
private let api = APIClient.shared
⋮----
private var kpiGridMin: CGFloat {
⋮----
var body: some View {
⋮----
private var retailerCommandBoard: some View {
⋮----
private func commandTile(_ label: String, _ value: String) -> some View {
⋮----
private var activeDeliveriesSection: some View {
⋮----
private func emptyState(icon: String, title: String, message: String) -> some View {
⋮----
private func confidenceColor(_ confidence: Double) -> Color {
⋮----
private var hasCachedDashboardData: Bool {
⋮----
private func loadData(silent: Bool = false) async {
⋮----
private func loadCommandPulse() async {
⋮----
private func cancelOrder(_ orderId: String) async {
⋮----
private func confirmAiOrder(_ orderId: String) async {
⋮----
private func rejectAiOrder(_ orderId: String) async {
⋮----
private func confirmPreorder(_ orderId: String) async {
⋮----
private func editPreorder(_ orderId: String) async {
⋮----
private func findOrder(_ orderId: String) async -> Order? {
⋮----
private struct ServiceTile {
enum Size { case small, regular, large }
let title: String
let icon: String
let subtitle: String?
let size: Size
```
