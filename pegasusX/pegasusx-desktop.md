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
- Only files matching these patterns are included: apps/*portal*/**/*.ts, apps/*portal*/**/*.tsx, apps/*desktop*/**/*.ts, apps/*desktop*/**/*.tsx, apps/**/src-tauri/src/**/*.rs, apps/**/src-tauri/Cargo.toml, apps/**/src-tauri/tauri.conf.json
- Files matching these patterns are excluded: **/node_modules/**, **/dist/**, **/build/**, **/target/**, **/src-tauri/gen/**, **/src-tauri/icons/**, **/i18n/**, **/locales/**, **/*.test.*, **/*.spec.*, **/*.d.ts, repomix*
- Files matching patterns in .gitignore are excluded
- Files matching default ignore patterns are excluded
- Code comments have been removed from supported file types
- Empty lines have been removed from all files
- Content has been compressed - code blocks are separated by ⋮---- delimiter
- Files are sorted by Git change count (files with more changes are at the bottom)

# Directory Structure
```
apps/
  admin-portal/
    app/
      analytics/
        page.tsx
      api/
        auth/
          login/
            route.ts
          logout/
            route.ts
          session/
            route.ts
      compliance/
        aml/
          page.tsx
        cashier/
          page.tsx
        fiscal/
          page.tsx
      finops/
        page.tsx
      governance/
        audit/
          page.tsx
        flags/
          page.tsx
        tenants/
          page.tsx
      infra/
        page.tsx
      logistics/
        dvir/
          page.tsx
        gate/
          page.tsx
      overview/
        page.tsx
      telemetry/
        page.tsx
      traceability/
        actions/
          page.tsx
        recall/
          page.tsx
        sandbox/
          page.tsx
      error.tsx
      global-error.tsx
      layout.tsx
      loading.tsx
      page.tsx
    components/
      auth/
        AdminAuthGate.tsx
      layout/
        AdminAppShell.tsx
        AdminHeader.tsx
        AdminSidebar.tsx
      ui/
        BentoGrid.tsx
        CopyableId.tsx
        CountUp.tsx
        DesktopKeyboardCheatsheet.tsx
        EmptyState.tsx
        Icon.tsx
        motion.ts
        PageTransition.tsx
        PortalPrimitives.tsx
        RouteSkeleton.tsx
        Skeleton.tsx
        StatsCard.tsx
        Toast.tsx
      AccuracyPanel.tsx
      AuditPanel.tsx
      BillingPanel.tsx
      CashierVaultMonitor.tsx
      CommandBoard.tsx
      FlagsPanel.tsx
      FleetYardMonitor.tsx
      FoundryTriadPanel.tsx
      MatchQueuePanel.tsx
      OntologyBlastRadiusGraph.tsx
      OpsPanel.tsx
      OrderStateMachineGraph.tsx
      PartnerPanel.tsx
      ScenarioSandboxManager.tsx
      TenantsPanel.tsx
    lib/
      api.ts
      deadLetterHealth.ts
      enterprise-swr.ts
      errors.ts
      foundry-api.ts
      money.ts
      session.tsx
      use-admin-ws-refresh.ts
      ws-provider.tsx
    src-tauri/
      src/
        commands/
          diagnostics.rs
          mod.rs
          security.rs
          telemetry.rs
        lib.rs
        main.rs
      Cargo.toml
      tauri.conf.json
    middleware.ts
    next.config.ts
    vitest.config.ts
  factory-portal/
    app/
      analytics/
        page.tsx
        ProductionForecastChart.tsx
      auth/
        login/
          page.tsx
        register/
          page.tsx
          wizard-state.ts
        layout.tsx
      fleet/
        page.tsx
      insights/
        page.tsx
      loading-bay/
        page.tsx
      manifest-exceptions/
        page.tsx
      manifests/
        [id]/
          layout.tsx
          page.tsx
        page.tsx
      notifications/
        page.tsx
      payload/
        page.tsx
      payload-override/
        page.tsx
      settings/
        location/
          page.tsx
      setup/
        factory/
          page.tsx
        layout.tsx
      staff/
        [id]/
          layout.tsx
          page.tsx
        page.tsx
      supply-requests/
        page.tsx
      transfers/
        [id]/
          layout.tsx
          page.tsx
        create/
          page.tsx
        page.tsx
      error.tsx
      layout.tsx
      page.tsx
    components/
      dashboard/
        DashboardActionGrid.tsx
        DashboardAlerts.tsx
        DashboardMetrics.tsx
        types.ts
      loading-bay/
        FactoryYardGateSection.tsx
        LoadingBayControls.tsx
        LoadingBayGrid.tsx
      manifest-exceptions/
        ManifestExceptionsList.tsx
      payload-override/
        PayloadList.tsx
        PayloadOverrideForm.tsx
      portal/
        index.ts
        PortalPrimitives.tsx
      setup/
        constants.ts
        SetupWizardShell.tsx
      staff/
        StaffList.tsx
      supply-requests/
        constants.ts
        SupplyRequestBoard.tsx
        SupplyRequestHeader.tsx
        SupplyRequestList.tsx
        types.ts
      transfers/
        TransferFilters.tsx
        TransferList.tsx
      ui/
        AuthLoginCard.tsx
        AuthRegisterStepper.tsx
        AuthRegisterSteps.tsx
        countries.ts
        DesktopKeyboardCheatsheet.tsx
        DesktopOfflineTray.tsx
        ExplainStatusBanner.tsx
        firebase-emulator-phone.ts
        GenericFleetLiveMap.tsx
        HealthStrip.tsx
        index.ts
        map-chrome.tsx
        motion.ts
        PackChip.tsx
        PageChrome.tsx
        PortalPrimitives.tsx
        PulseTimeline.tsx
        SessionPackChip.tsx
        SourceChip.tsx
        StatusStack.tsx
        use-desktop-connectivity.ts
        useAnimatedDriverMarkers.ts
      AuthGuard.tsx
      ClientPolicyBanner.tsx
      EmptyState.tsx
      FactoryPageState.tsx
      FactoryRuntimeBanner.tsx
      FactoryShell.tsx
      FleetLiveMap.tsx
      HandoffTimelinePanel.tsx
      Icon.tsx
      KpiStatCard.tsx
      LanguageSwitcher.tsx
      ListToolbar.tsx
      LocaleBootstrap.tsx
      LocationPicker.tsx
      LotRecallBanner.tsx
      NetworkPulsePanel.tsx
      NotificationPanel.tsx
      PageChrome.tsx
      PageSection.tsx
      PageTransition.tsx
      Providers.tsx
      SessionPackChip.tsx
      Skeleton.tsx
      ThemeProvider.tsx
      Toast.tsx
    lib/
      api.ts
      auth.ts
      bridge.ts
      client-policy.ts
      csv.ts
      desktop-deep-link.tsx
      desktop-updater.tsx
      enterprise-swr.ts
      factory-reconnect.ts
      factory-scope.ts
      firebase.ts
      foundry-api.ts
      geocode.ts
      i18n.ts
      manifest-lifecycle.ts
      money.ts
      nav.ts
      portal-offline-tray.tsx
      session-reconcile.ts
      use-factory-fleet-live-map.ts
      use-factory-session-reconcile.ts
      use-pagination.ts
      useNotifications.ts
    src/
      components/
        production/
          ProductionKioskScreen.tsx
    src-tauri/
      src/
        commands/
          mod.rs
          security.rs
        lib.rs
        main.rs
      Cargo.toml
      tauri.conf.json
    middleware.ts
    next.config.ts
    vitest.config.ts
    vitest.setup.ts
  retailer-app-desktop/
    app/
      (dashboard)/
        assist/
          page.tsx
        auto-order/
          page.tsx
        catalog/
          page.tsx
        control-tower/
          page.tsx
        credit/
          page.tsx
        dashboard/
          page.tsx
        dock/
          page.tsx
        hq/
          page.tsx
        insights/
          page.tsx
          SpendAnalytics.tsx
        my-suppliers/
          page.tsx
        notifications/
          page.tsx
        orders/
          page.tsx
        pos/
          page.tsx
        procurement/
          page.tsx
        reports/
          page.tsx
        sections/
          page.tsx
        settings/
          capabilities/
            page.tsx
          family/
            page.tsx
          locations/
            page.tsx
          team/
            page.tsx
          page.tsx
        shifts/
          page.tsx
        stock/
          local-skus/
            page.tsx
          page.tsx
        tracking/
          page.tsx
        layout.tsx
      auth/
        login/
          page.tsx
        register/
          page.tsx
          wizard-state.ts
        select-org/
          page.tsx
        layout.tsx
      setup/
        address/
          page.tsx
        tax/
          page.tsx
        layout.tsx
        page.tsx
      error.tsx
      layout.tsx
    components/
      auto-order/
        AutoOrderList.tsx
        AutoOrderRules.tsx
      catalog/
        CatalogFilters.tsx
        ProductGrid.tsx
      dashboard/
        AiPredictionSection.tsx
        CommandBoard.tsx
        KpiGrid.tsx
        QuickReorderSection.tsx
      insights/
        InsightsSidebar.tsx
        InsightsSummary.tsx
        SellThroughPanel.tsx
      orders/
        OrderFilters.tsx
        OrderList.tsx
      portal/
        index.ts
        PortalPrimitives.tsx
      settings/
        OverrideComponents.tsx
        ProfileFields.tsx
      setup/
        constants.ts
        SetupWizardShell.tsx
      tracking/
        TrackingMap.tsx
        TrackingStatus.tsx
      ui/
        AuthLoginCard.tsx
        AuthRegisterStepper.tsx
        AuthRegisterSteps.tsx
        countries.ts
        DemandSourceChips.tsx
        DesktopKeyboardCheatsheet.tsx
        DesktopOfflineTray.tsx
        ExplainStatusBanner.tsx
        firebase-emulator-phone.ts
        GenericFleetLiveMap.tsx
        HandoffCard.tsx
        HealthStrip.tsx
        index.ts
        map-chrome.tsx
        motion.ts
        PackChip.tsx
        PageChrome.tsx
        PageSkeletons.tsx
        PortalPrimitives.tsx
        PulseTimeline.tsx
        SessionPackChip.tsx
        SourceChip.tsx
        StatusStack.tsx
        use-desktop-connectivity.ts
        useAnimatedDriverMarkers.ts
        VirtualScrollList.tsx
      BentoGrid.tsx
      CartDrawer.tsx
      CheckoutModal.tsx
      ClientPolicyBanner.tsx
      CountUp.tsx
      CreditProfileCard.tsx
      EmptyState.tsx
      FileClaimPanel.tsx
      Icon.tsx
      KpiStatCard.tsx
      LanguageSwitcher.tsx
      LocaleBootstrap.tsx
      LocationPicker.tsx
      LoyaltyCard.tsx
      MiniSparkline.tsx
      NetworkPulsePanel.tsx
      OrderTimelinePanel.tsx
      OrgSwitcher.tsx
      PageChrome.tsx
      PageSection.tsx
      PageTransition.tsx
      PaymentModal.tsx
      ProductDetailDrawer.tsx
      Providers.tsx
      RetailerShell.tsx
      SessionPackChip.tsx
      ShopClosedModal.tsx
      Skeleton.tsx
      StockRequestReturnModal.tsx
      ThemeProvider.tsx
    lib/
      api.ts
      auth.ts
      bridge.ts
      cart.tsx
      clear-org-scoped-state.ts
      desktop-cache-bootstrap.tsx
      desktop-deep-link.tsx
      desktop-updater.tsx
      dock-pending-patches.ts
      enterprise-swr.ts
      firebase.ts
      geocode.ts
      hooks.ts
      i18n.ts
      money.ts
      multi-org-auth.ts
      nav.ts
      notifications-core.ts
      notifications.tsx
      order-receipt.ts
      payment-catalog.ts
      pending-checkout-flusher.tsx
      pending-checkout.ts
      pending-pos-flusher.tsx
      pending-pos-sales.ts
      receiving-window.ts
      retailer-offline-tray.tsx
      retailer-profile.ts
      session-reconcile-listener.tsx
      session-reconcile.ts
      stock-policy.ts
      types.ts
      use-payment-catalog.ts
      use-retailer-session-reconcile.ts
      ws.tsx
    src/
      components/
        dock/
          ShelfAlertModal.tsx
    src-tauri/
      src/
        commands/
          mod.rs
          security.rs
        lib.rs
        main.rs
      Cargo.toml
      tauri.conf.json
    clean.tsx
    middleware.ts
    next.config.ts
    vitest.config.ts
    vitest.setup.ts
  supplier-portal/
    app/
      (dashboard)/
        activity/
          page.tsx
        chargebacks/
          page.tsx
        compliance/
          page.tsx
        ledger/
          page.tsx
        settings/
          fx-rates/
            page.tsx
          tax-regimes/
            page.tsx
      (portal)/
        _components/
          PortalSurface.tsx
        analytics/
          demand/
            flywheel/
              page.tsx
            signals/
              page.tsx
            page.tsx
          knowledge-graph/
            page.tsx
          route-performance/
            page.tsx
          page.tsx
          RevenueHeatmap.tsx
        catalog/
          [productId]/
            layout.tsx
            page.tsx
          components/
            CatalogTable.tsx
            CreateProductForm.tsx
            types.ts
          page.tsx
        chargebacks/
          claims/
            page.tsx
        control-tower/
          page.tsx
          use-control-tower-telemetry.ts
        credit/
          admin-disable/
            page.tsx
          collections/
            page.tsx
          policy/
            page.tsx
        crm/
          page.tsx
        dashboard/
          page.tsx
          use-dashboard-data.ts
          use-dashboard-history.ts
        delivery-zones/
          page.tsx
        dispatch/
          page.tsx
          use-dispatch-data.ts
        entity-resolution/
          page.tsx
        exceptions/
          claims/
            page.tsx
          early-complete/
            page.tsx
          negotiations/
            page.tsx
          shop-closed/
            page.tsx
          page.tsx
        factories/
          components/
            FactoryForm.tsx
            FactoryList.tsx
          page.tsx
        finance/
          aml-exposure/
            page.tsx
          credit-notes/
            page.tsx
          payouts/
            page.tsx
        fleet/
          orders/
            page.tsx
          page.tsx
        foundry/
          page.tsx
        geo-report/
          page.tsx
        inventory/
          import/
            page.tsx
          page.tsx
        labor-capacity/
          page.tsx
        loyalty/
          page.tsx
        manifest-exceptions/
          page.tsx
        manifests/
          [id]/
            layout.tsx
            page.tsx
          page.tsx
        operations/
          replenishment-policies/
            page.tsx
          page.tsx
        ops/
          map/
            page.tsx
        orders/
          [id]/
            layout.tsx
            page.tsx
          components/
            OrdersList.tsx
          page.tsx
        planning/
          page.tsx
        pricing/
          [productId]/
            layout.tsx
            page.tsx
          retailer-overrides/
            page.tsx
          page.tsx
        profile/
          page.tsx
        promotions/
          page.tsx
        reconciliation/
          page.tsx
        replenishment/
          suggestions/
            page.tsx
        returns/
          page.tsx
        settings/
          integrations/
            page.tsx
          notification-preferences/
            page.tsx
          planning/
            page.tsx
          playbooks/
            page.tsx
          return-policy/
            page.tsx
          segmentation/
            page.tsx
        supply-lanes/
          page.tsx
        topology/
          page.tsx
        treasury/
          cash-reconciliations/
            page.tsx
          page.tsx
        warehouses/
          [warehouseId]/
            coverage/
              page.tsx
            layout.tsx
          components/
            WarehouseForm.tsx
            WarehouseList.tsx
          page.tsx
        error.tsx
        layout.tsx
        page.tsx
      ai/
        recommendations/
          layout.tsx
          page.tsx
      api/
        api/
          [...path]/
            route.ts
          ws-session/
            route.ts
      auth/
        login/
          page.tsx
        oidc/
          callback/
            page.tsx
        register/
          page.tsx
          wizard-state.ts
        layout.tsx
      demand/
        payday-calendar/
          page.tsx
        signals/
          page.tsx
      earnings/
        layout.tsx
        page.tsx
      org-fleet/
        components/
          DriverForm.tsx
          DriverTable.tsx
          MetricsOverview.tsx
          OrgMemberForm.tsx
          OrgMemberTable.tsx
          utils.tsx
          VehicleForm.tsx
          VehicleTable.tsx
        layout.tsx
        page.tsx
      payments/
        _shared/
          finance.ts
        layout.tsx
        page.tsx
      setup/
        billing/
          page.tsx
        business/
          page.tsx
        layout.tsx
      tracking/
        page.tsx
      error.tsx
      layout.tsx
      page.tsx
    components/
      compliance/
        MetricCard.tsx
      control-tower/
        ScoredExceptionsPanel.tsx
      delivery-zones/
        DeliveryZonesList.tsx
        index.ts
      dispatch/
        CargoPhotoCarousel.tsx
        DispatchSidebarNav.tsx
        PartnerFilterBar.tsx
        RouteTelemetryMap.tsx
        ShipmentCard.tsx
        ShipmentCardGrid.tsx
        VehicleCapacityGauge.tsx
      exceptions/
        ExceptionsList.tsx
        PlaybookRunsPanel.tsx
      foundry/
        LotRecallBlastRadiusView.tsx
      geo-report/
        GeoReportLanesList.tsx
        index.ts
      inventory/
        index.ts
        InventoryTable.tsx
      manifests/
        index.ts
        ManifestsTable.tsx
      operations/
        index.ts
        OperatorBroadcast.tsx
        PaymentBypass.tsx
        ReplenishmentAction.tsx
      orders/
        index.ts
        OrderActionDialog.tsx
        OrderKebabMenu.tsx
        OrderLineItems.tsx
        OrderOpsActions.tsx
        OrderOpsCard.tsx
        OrderStateChip.tsx
        ProposeDelayDialog.tsx
        ReDispatchDialog.tsx
      portal/
        index.ts
        PortalPrimitives.tsx
      profile/
        ContactDetailsForm.tsx
        index.ts
        SupplierIdentityCard.tsx
      settings/
        planning/
          CreateOverrideForm.tsx
          FactoryPlanningOpsPanel.tsx
          ForecastAccuracyPanel.tsx
          index.ts
          SeasonalOverridesTable.tsx
        OIDCAttachCard.tsx
        SegmentationPanel.tsx
      setup/
        constants.ts
        SetupPrimitives.tsx
        SetupWizardShell.tsx
      supply-lanes/
        index.ts
        SupplyLanesList.tsx
      ui/
        AuthLoginCard.tsx
        AuthRegisterStepper.tsx
        AuthRegisterSteps.tsx
        countries.ts
        DemandSourceChips.tsx
        DesktopKeyboardCheatsheet.tsx
        DesktopOfflineTray.tsx
        ExplainStatusBanner.tsx
        firebase-emulator-phone.ts
        GenericFleetLiveMap.tsx
        GlassmorphismPanel.tsx
        HandoffCard.tsx
        HealthStrip.tsx
        HexagonalControlTowerMap.tsx
        index.ts
        KpiStat.tsx
        LiveEKGNetworkGraph.tsx
        map-chrome.tsx
        motion.ts
        PackChip.tsx
        PageChrome.tsx
        PageSkeletons.tsx
        PlanBrainTabs.tsx
        PortalPrimitives.tsx
        PulseTimeline.tsx
        RangeToggle.tsx
        SessionPackChip.tsx
        SourceChip.tsx
        StatusStack.preview.tsx
        StatusStack.tsx
        use-desktop-connectivity.ts
        useAnimatedDriverMarkers.ts
        VirtualScrollList.tsx
      AdminOrderOpsPanel.tsx
      AuthGuard.tsx
      BentoGrid.tsx
      BulkImportWizard.tsx
      ClientPolicyBanner.tsx
      ControlTowerCommandPanel.tsx
      CoverageCityChips.tsx
      CreditEnableModal.tsx
      DigitalBrainPanel.tsx
      DispatchPreviewMap.tsx
      EmptyState.tsx
      FleetLiveMap.tsx
      FleetLiveMapPanel.tsx
      ForecastConfidenceCard.tsx
      HandoffInboxCard.tsx
      Icon.tsx
      KpiStatCard.tsx
      LanguageSwitcher.tsx
      ListToolbar.tsx
      LiveOpsMap.tsx
      LocaleBootstrap.tsx
      LocationPicker.tsx
      MeioNetworkPanel.tsx
      NetworkPulsePanel.tsx
      NotificationPanel.tsx
      OrderTimelinePanel.tsx
      PageChrome.tsx
      PageSection.tsx
      PlanningBrainPanel.tsx
      PlanningOutcomesPanel.tsx
      Providers.tsx
      ReplenishmentTraceabilityPanel.tsx
      SessionPackChip.tsx
      SignalIngestOpsPanel.tsx
      Skeleton.tsx
      StatusBadge.tsx
      StatusChip.tsx
      SupplierShell.tsx
      ThemeProvider.tsx
      Toast.tsx
      TopologyEditor.tsx
    lib/
      admin-scope.ts
      api.ts
      auth.ts
      bridge.ts
      client-policy.ts
      coverage.ts
      csv.ts
      dashboard-command.ts
      desktop-cache-bootstrap.tsx
      desktop-deep-link.tsx
      desktop-updater.tsx
      enterprise-swr.ts
      firebase.ts
      forecast-confidence.ts
      foundry-api.ts
      geocode.ts
      hooks.ts
      i18n.ts
      money.ts
      nav.ts
      oidc.ts
      order-actions.ts
      order-receipt.ts
      portal-offline-tray.tsx
      scenario-chart.ts
      session-reconcile.ts
      supplier-cache-keys.ts
      supplier-reconnect.ts
      supplier-scope.ts
      supplier-warehouse-ops.ts
      supplier-ws-events.ts
      topology.ts
      use-fleet-live-map.ts
      use-pagination.ts
      use-payment-catalog.ts
      use-supplier-session-reconcile.ts
      use-supplier-ws-refresh.ts
      useNotifications.ts
    src-tauri/
      src/
        commands/
          mod.rs
          security.rs
        lib.rs
        main.rs
      Cargo.toml
      tauri.conf.json
    middleware.ts
    vitest.config.ts
    vitest.setup.ts
  warehouse-portal/
    app/
      analytics/
        page.tsx
        VelocityGauge.tsx
      auth/
        login/
          page.tsx
        register/
          page.tsx
          wizard-state.ts
        layout.tsx
      bins/
        page.tsx
      cashier/
        page.tsx
      claims/
        page.tsx
      control-tower/
        page.tsx
        use-control-tower-telemetry.ts
      coverage/
        page.tsx
      credit/
        policy/
          page.tsx
      crm/
        page.tsx
      cycle-counts/
        page.tsx
      demand-forecast/
        page.tsx
      dispatch/
        rescues/
          page.tsx
        page.tsx
      dispatch-locks/
        page.tsx
      dispatch-settings/
        page.tsx
      drivers/
        page.tsx
      exceptions/
        page.tsx
      fleet/
        page.tsx
      fleet-live-map/
        page.tsx
      foundry/
        page.tsx
      inventory/
        page.tsx
      labor-capacity/
        page.tsx
      manifests/
        page.tsx
      operations/
        page.tsx
      orders/
        [id]/
          layout.tsx
          page.tsx
        components/
          OrdersList.tsx
        page.tsx
      payment-config/
        page.tsx
      pick-waves/
        page.tsx
      preorders/
        page.tsx
      products/
        page.tsx
      profile/
        page.tsx
      replenishment/
        page.tsx
      returns/
        page.tsx
      settings/
        page.tsx
      setup/
        location/
          page.tsx
        layout.tsx
      staff/
        page.tsx
      stock-commitments/
        page.tsx
      supply-requests/
        [id]/
          layout.tsx
          page.tsx
        new/
          page.tsx
        page.tsx
      tomorrow-board/
        page.tsx
      tracking/
        page.tsx
      transfers/
        page.tsx
      treasury/
        page.tsx
      vehicles/
        [vehicleId]/
          layout.tsx
          page.tsx
        page.tsx
      error.tsx
      layout.tsx
      page.tsx
    components/
      analytics/
        AnalyticsChartGrid.tsx
      cashier/
        AMLExposureCheck.tsx
        BlindCountDenominationGrid.tsx
        CashierSessionView.tsx
        CITHandoverManifest.tsx
        RRNReclassificationModal.tsx
      dispatch/
        DispatchDriverList.tsx
        DispatchOrderList.tsx
        DispatchPanelsAndModals.tsx
      drivers/
        DriversList.tsx
      fleet/
        FleetDVIRScreen.tsx
        MidRouteRescueModal.tsx
        YardGateCheckScreen.tsx
      forecast/
        ForecastChartPanel.tsx
        ForecastSkuTable.tsx
      foundry/
        OntologyBlastRadiusViewer.tsx
        ScenarioSandboxWorkbench.tsx
      inventory/
        InventoryStockList.tsx
      operations/
        OperationsBroadcastForm.tsx
        OperationsPricingPreview.tsx
      orders/
        index.ts
        OrderActionDialog.tsx
        OrderKebabMenu.tsx
        OrderLineItems.tsx
        OrderOpsCard.tsx
        OrderProposeDateDialog.tsx
        OrderStateChip.tsx
      portal/
        index.ts
        PortalPrimitives.tsx
      preorders/
        PreordersList.tsx
      replenishment/
        ReplenishmentList.tsx
      returns/
        ReturnsList.tsx
        ReverseLogisticsPanel.tsx
      settings/
        OpsSettingsForm.tsx
        ReturnPolicySettingsSection.tsx
      setup/
        constants.ts
        SetupWizardShell.tsx
      staff/
        StaffList.tsx
      treasury/
        TreasuryTransactionList.tsx
      ui/
        AuthLoginCard.tsx
        AuthRegisterStepper.tsx
        AuthRegisterSteps.tsx
        countries.ts
        DemandSourceChips.tsx
        DesktopKeyboardCheatsheet.tsx
        DesktopOfflineTray.tsx
        ExplainStatusBanner.tsx
        firebase-emulator-phone.ts
        GenericFleetLiveMap.tsx
        GlassmorphismPanel.tsx
        HandoffCard.tsx
        HealthStrip.tsx
        HexagonalControlTowerMap.tsx
        index.ts
        KpiStat.tsx
        LiveEKGNetworkGraph.tsx
        map-chrome.tsx
        motion.ts
        PackChip.tsx
        PageChrome.tsx
        PageSkeletons.tsx
        PlanBrainTabs.tsx
        PortalPrimitives.tsx
        PulseTimeline.tsx
        RangeToggle.tsx
        SessionPackChip.tsx
        SourceChip.tsx
        StatusStack.preview.tsx
        StatusStack.tsx
        use-desktop-connectivity.ts
        useAnimatedDriverMarkers.ts
        VirtualScrollList.tsx
      vehicles/
        VehiclesList.tsx
      AuthGuard.tsx
      ClientPolicyBanner.tsx
      DispatchPreviewMap.tsx
      EmptyState.tsx
      FleetLiveMap.tsx
      FleetLiveMapPanel.tsx
      ForecastConfidenceView.tsx
      HandoffTimelinePanel.tsx
      Icon.tsx
      KpiStatCard.tsx
      LanguageSwitcher.tsx
      ListToolbar.tsx
      LocaleBootstrap.tsx
      LocationPicker.tsx
      NetworkPulsePanel.tsx
      NotificationPanel.tsx
      OrderTimelinePanel.tsx
      PageChrome.tsx
      PageSection.tsx
      PageTransition.tsx
      Providers.tsx
      SessionPackChip.tsx
      Skeleton.tsx
      ThemeProvider.tsx
      Toast.tsx
      VehicleAvailabilityPanel.tsx
      WarehouseShell.tsx
    lib/
      api.ts
      auth.ts
      bridge.ts
      client-policy.ts
      coverage.ts
      csv.ts
      desktop-cache-bootstrap.tsx
      desktop-deep-link.tsx
      desktop-updater.tsx
      dispatch-preview-cache.ts
      dispatch-runs-cache.ts
      enterprise-swr.ts
      firebase.ts
      fleet-ws-events.ts
      forecast-confidence.ts
      foundry-api.ts
      geocode.ts
      i18n.ts
      money.ts
      nav.ts
      order-actions.ts
      order-receipt.ts
      portal-offline-tray.tsx
      session-reconcile.ts
      use-pagination.ts
      use-payment-catalog.ts
      use-warehouse-fleet-live-map.ts
      use-warehouse-session-reconcile.ts
      use-warehouse-vehicles-live.ts
      use-warehouse-ws-refresh.ts
      useNotifications.ts
      useStableCallback.ts
      vehicle-fleet.ts
      warehouse-api.ts
      warehouse-ops.ts
      warehouse-reconnect.ts
      warehouse-scope.ts
    src-tauri/
      src/
        commands/
          mod.rs
          security.rs
        lib.rs
        main.rs
      Cargo.toml
      tauri.conf.json
    middleware.ts
    vitest.config.ts
    vitest.setup.ts
```

# Files

## File: apps/admin-portal/components/AuditPanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { api, type AuditRow } from "@/lib/api";
⋮----
<button onClick=
```

## File: apps/admin-portal/components/BillingPanel.tsx
```typescript
import { useEffect, useState } from "react";
import { api, type BillingInvoice, type BillingFeeSchedule } from "@/lib/api";
⋮----
export default function BillingPanel({ token }: { token: string })
⋮----
onClick={async () =>
```

## File: apps/admin-portal/components/CommandBoard.tsx
```typescript
import { useEffect, useState } from "react";
import { api, type FlagOverride, type Tenant } from "@/lib/api";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";
⋮----
export default function CommandBoard({
  token,
  refreshKey = 0,
  onOpenTab,
}: {
  token: string;
  refreshKey?: number;
  onOpenTab: (tab: string) => void;
})
```

## File: apps/admin-portal/components/MatchQueuePanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { api, type MatchQueueItem } from "@/lib/api";
⋮----
const resolve = async (id: string, decision: "ACCEPT" | "REJECT") =>
```

## File: apps/admin-portal/components/OpsPanel.tsx
```typescript
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";
⋮----
export default function OpsPanel({ token }: { token: string })
⋮----
<tr key=
```

## File: apps/admin-portal/components/TenantsPanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { api, type Tenant } from "@/lib/api";
⋮----
const transition = async (t: Tenant, to: string) =>
```

## File: apps/admin-portal/lib/deadLetterHealth.ts
```typescript
export type DeadLetterSummary = {
  dead_letter_count?: number;
  dead_letter_available?: boolean;
  available?: boolean;
  page_count?: number;
  count?: number;
  items?: unknown[];
};
⋮----
export type DeadLetterHealth =
  | { kind: "unavailable" }
  | { kind: "zero" }
  | { kind: "count"; count: number };
⋮----
export function deadLetterHealth(summary: DeadLetterSummary | null | undefined): DeadLetterHealth
⋮----
export function deadLetterLabel(h: DeadLetterHealth): string
```

## File: apps/admin-portal/lib/money.ts
```typescript
export interface ParseMoneyOptions {
  allowNegative?: boolean;
}
⋮----
export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}
⋮----
function toSafeNumberOrNull(value: bigint): number | null
⋮----
function toMinorBigInt(value: number | bigint | null | undefined): bigint
⋮----
function groupAsciiDigits(digits: string, separator: string): string
⋮----
function stripSpaceGrouping(raw: string): string | null
⋮----
export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null
⋮----
export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function majorIntToMinorBigInt(major: number | bigint | string): bigint
⋮----
export function majorIntToMinor(major: number | bigint | string): number
⋮----
export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeTrunc(minor: number | bigint | null | undefined): number
⋮----
export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeRound(minor: number | bigint | null | undefined): number
⋮----
export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint
⋮----
export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number
⋮----
export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint
⋮----
export function applyBpsFee(amount: number, bps: number): number
⋮----
export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string
⋮----
export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string
⋮----
export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string
⋮----
export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string
⋮----
export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number
⋮----
export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string
```

## File: apps/admin-portal/vitest.config.ts
```typescript
import { defineConfig } from "vitest/config";
import path from "path";
```

## File: apps/factory-portal/app/analytics/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
⋮----
interface AnalyticsOverview {
  daily_activity: unknown[];
  transfers_total: number;
  manifests_active: number;
  exception_queue: number;
  avg_lead_time_mins: number;
  product_drill_down: {
    product_id: string;
    requested: number;
    shipped: number;
    fulfillment_ratio: number;
  }[];
}
⋮----
description=
⋮----
setLoading(true);
void load();
⋮----
<KpiStatCard label=
⋮----
label=
⋮----
<PageSection title=
```

## File: apps/factory-portal/app/analytics/ProductionForecastChart.tsx
```typescript
type ProductionForecastChartProps = {
  className?: string;
  series?: { day: string; rawMaterials: number; burnRate: number }[];
};
⋮----
export default function ProductionForecastChart({
  className,
  series = [],
}: ProductionForecastChartProps)
```

## File: apps/factory-portal/app/auth/register/wizard-state.ts
```typescript
export type StepId = "identity" | "verification" | "profile";
⋮----
export interface Country {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export interface IdentityStep {
  countryCode: string;
  phoneLocal: string;
}
⋮----
export interface VerificationStep {
  otpCode: string;
  idToken: string;
}
⋮----
export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
}
⋮----
export interface WizardState {
  step: StepId;
  identity: IdentityStep;
  verification: VerificationStep;
  profile: ProfileStep;
}
⋮----
export function validateIdentity(s: IdentityStep): Record<string, string>
⋮----
export function validateVerification(s: VerificationStep): Record<string, string>
⋮----
export function validateProfile(s: ProfileStep): Record<string, string>
```

## File: apps/factory-portal/app/auth/layout.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';
```

## File: apps/factory-portal/app/fleet/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import FleetLiveMap from '@/components/FleetLiveMap';
import { useFactoryFleetLiveMap } from '@/lib/use-factory-fleet-live-map';
import { motion } from 'framer-motion';
⋮----
interface Vehicle {
  id: string;
  plate_number: string;
  capacity_m3: number;
  status: string;
  driver_name: string;
  current_route_id: string;
}
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/factory-portal/app/insights/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';
⋮----
interface InsightRow {
  id?: string;
  warehouse_id?: string;
  warehouse_name?: string;
  product_id?: string;
  product_name?: string;
  urgency?: string;
  current_stock?: number;
  avg_daily_velocity?: number;
  daily_velocity?: number;
  reorder_quantity?: number;
  reorder_qty?: number;
  days_until_stockout?: number;
  days_to_empty?: number;
  status?: string;
  created_at?: string;
  reason_code?: string;
  demand_breakdown?: Record<string, unknown> | null;
}
⋮----
interface Insight {
  id: string;
  warehouse_id: string;
  warehouse_name: string;
  product_name: string;
  urgency: string;
  current_stock: number;
  daily_velocity: number;
  reorder_qty: number;
  days_to_empty: number;
  status: string;
  created_at: string;
  reason_code?: string;
  demand_breakdown?: Record<string, unknown> | null;
}
⋮----
function normalizeInsight(row: InsightRow): Insight
⋮----
function formatDemandWhy(
  breakdown: Insight['demand_breakdown'],
  reasonCode?: string,
): string
⋮----
const urgencyClass = (u: string) =>
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/factory-portal/app/manifests/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/factory-portal/app/manifests/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
⋮----
interface ManifestRow {
  manifest_id: string;
  state: string;
  transfer_count?: number;
  total_volume_vu?: number;
  max_volume_vu?: number;
  driver_id?: string;
  vehicle_id?: string;
  updated_at?: string;
}
⋮----
function stateClass(state: string): string
⋮----
description=
⋮----
<KpiStatCard label=
⋮----
<PageSection title=
```

## File: apps/factory-portal/app/notifications/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useNotifications } from '@/lib/useNotifications';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import Icon from '@/components/Icon';
import { motion } from 'framer-motion';
⋮----
function timeAgo(iso: string): string
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/factory-portal/app/payload/page.tsx
```typescript
import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { manifestTransitionIdempotencyKey, nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
⋮----
interface ManifestRow {
  manifest_id: string;
  state: string;
  transfer_count?: number;
  total_volume_vu?: number;
}
⋮----
async function runAction(manifest: ManifestRow)
```

## File: apps/factory-portal/app/payload-override/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { PayloadList } from '../../components/payload-override/PayloadList';
import { PayloadOverrideForm } from '../../components/payload-override/PayloadOverrideForm';
⋮----
export interface Transfer {
  transfer_id: string;
  product_name: string;
  quantity: number;
  volume_vu: number;
  state: string;
}
⋮----
export interface Manifest {
  manifest_id: string;
  truck_id: string;
  truck_plate?: string;
  state: string;
  total_volume_vu: number;
  max_capacity_vu: number;
  transfers: Transfer[];
  created_at: string;
}
⋮----
function manifestSignature(items: Manifest[])
⋮----
function formatSyncTime(value: number | null)
⋮----
const handleOnline = () =>
⋮----
const handleOffline = () =>
⋮----
const handleRebalance = async () =>
⋮----
const handleCancelTransfer = async (transferId: string, manifestId: string) =>
⋮----
const handleCancelManifest = async (manifestId: string) =>
⋮----
description=
⋮----
onCancelTransfer={(transferId, manifestId) =>
⋮----
onCancelManifest={(manifestId) =>
```

## File: apps/factory-portal/app/settings/location/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { factoryOpsLocationKey } from "@pegasusx/api-core";
import { apiFetch } from "@/lib/auth";
import { useFactorySessionReconcile } from "@/lib/use-factory-session-reconcile";
import { factoryOperatorId } from "@/lib/factory-scope";
import PageTransition from "@/components/PageTransition";
import { PageChrome } from "@/components/PageChrome";
import { PortalSection } from "@/components/portal";
import { useToast } from "@/components/Toast";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";
⋮----
type FactoryLocation = {
  factory_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};
⋮----
async function saveLocation()
⋮----
description=
```

## File: apps/factory-portal/app/setup/factory/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import Icon from "@/components/Icon";
import { PortalField, PortalInput, PortalSelect, PortalSection, FormAlert } from "@/components/portal";
import { factoryOpsLocationKey } from "@pegasusx/api-core";
import { apiFetch, decodeJwtPayload, persistSession, readTokenFromCookie, refreshFactorySession } from "@/lib/auth";
import { factoryOperatorId } from "@/lib/factory-scope";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";
⋮----
interface FactorySetupState {
  factoryName: string;
  facilityType: string;
}
⋮----
type FactoryLocation = {
  factory_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};
⋮----
function validate(): Record<string, string>
⋮----
async function submit()
⋮----
onChange=
```

## File: apps/factory-portal/app/setup/layout.tsx
```typescript
import SetupWizardShell from "@/components/setup/SetupWizardShell";
⋮----
export default function SetupLayout({ children }: { children: React.ReactNode })
```

## File: apps/factory-portal/app/staff/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/factory-portal/app/staff/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PortalField, PortalInput, PortalSelect } from '@/components/portal';
import { StaffList, type StaffMember } from '@/components/staff/StaffList';
⋮----
async function handleCreate(e: React.FormEvent)
⋮----
description=
⋮----
emptyMessage=
⋮----
onSubmit=
```

## File: apps/factory-portal/app/supply-requests/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useCallback, useMemo, useRef, Fragment } from 'react';
import { factorySupplyRequestTransitionKey, factorySupplyRequestQCKey } from '@pegasusx/api-core';
import type { SupplyFulfillOptions } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { exportCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { ListToolbar } from '@/components/ListToolbar';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
⋮----
import type { SupplyRequest } from '@/components/supply-requests/types';
import { ACTIONS } from '@/components/supply-requests/constants';
import { SupplyRequestHeader } from '@/components/supply-requests/SupplyRequestHeader';
import { SupplyRequestList } from '@/components/supply-requests/SupplyRequestList';
import { SupplyRequestBoard } from '@/components/supply-requests/SupplyRequestBoard';
⋮----
type FilterState = 'ALL' | 'SUBMITTED' | 'ACKNOWLEDGED' | 'IN_PRODUCTION' | 'READY' | 'FULFILLED' | 'CANCELLED';
⋮----
function requestSignature(items: SupplyRequest[])
⋮----
function formatSyncTime(value: number | null)
⋮----
/* missing QC stays blank — GET "" is not PASS */
⋮----
const handleOnline = () =>
⋮----
const handleOffline = () =>
⋮----
const exportCsvHandler = async () =>
⋮----
const handleTransition = async (request: SupplyRequest, action: string) =>
⋮----
const handleQC = async (request: SupplyRequest, result: "PASS" | "FAIL") =>
⋮----
const runTransition = async (request: SupplyRequest, action: string) =>
⋮----
const handleToggleAll = () =>
⋮----
const handleToggleOne = (id: string, e: React.ChangeEvent<HTMLInputElement>) =>
⋮----
const runBatchTransition = async (action: string) =>
⋮----
description=
⋮----
<PageSection title=
⋮----
<button type="button" className=
⋮----
<p className="text-xs opacity-70">Linked transfer updated:
```

## File: apps/factory-portal/app/transfers/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/factory-portal/app/transfers/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { factoryTransferTransitionKey } from '@pegasusx/api-core';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
⋮----
interface TransferItem {
  sku_id: string;
  product_name: string;
  quantity: number;
  volume_m3: number;
}
⋮----
interface FleetDriver {
  driver_id: string;
  name: string;
  on_shift: boolean;
}
⋮----
interface TransferDetail {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  driver_id?: string;
  notes: string;
  created_at: string;
  updated_at: string;
  items: TransferItem[];
}
⋮----
function stateClass(state: string): string
⋮----
async function handleAssignDriver(newDriver: string)
⋮----
async function handleProgress()
⋮----
description=
⋮----
<button type="button" onClick=
```

## File: apps/factory-portal/app/transfers/create/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { factoryTransferCreateKey } from '@pegasusx/api-core';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PortalField, PortalInput, PortalSelect } from '@/components/portal';
⋮----
interface FleetDriver {
  driver_id: string;
  name: string;
  on_shift?: boolean;
}
⋮----
interface FleetVehicle {
  vehicle_id: string;
  plate_no: string;
  state: string;
}
⋮----
interface CreateTransferResponse {
  transfer_id: string;
  state: string;
}
⋮----
const handleSubmit = async (event: React.FormEvent) =>
⋮----
<PageChrome icon="transfers" title=
⋮----
description=
```

## File: apps/factory-portal/app/error.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
```

## File: apps/factory-portal/app/layout.tsx
```typescript
import type { Metadata } from "next";
import { Plus_Jakarta_Sans, EB_Garamond } from "next/font/google";
import LocaleBootstrap from "../components/LocaleBootstrap";
import Providers from "../components/Providers";
⋮----
export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>)
```

## File: apps/factory-portal/components/dashboard/DashboardActionGrid.tsx
```typescript
import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';
```

## File: apps/factory-portal/components/dashboard/DashboardAlerts.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';
```

## File: apps/factory-portal/components/dashboard/DashboardMetrics.tsx
```typescript
import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';
```

## File: apps/factory-portal/components/dashboard/types.ts
```typescript
import type { FactoryDashboardException } from '@pegasusx/types';
⋮----
export interface FactoryStats {
  source: string;
  plane: string;
  pending_transfers: number;
  loading_transfers: number;
  active_manifests: number;
  dispatched_today: number;
  vehicles_total: number;
  vehicles_available: number;
  staff_on_shift: number;
  critical_insights: number;
  transfers_by_state: Record<string, number>;
  manifests_by_state: Record<string, number>;
  vehicles_by_state: Record<string, number>;
  driver_duty: Record<string, number>;
  sla_by_status: Record<string, number>;
  qc_by_result: Record<string, number>;
  qc_available: boolean;
  bay_loading_transfers: number;
  bay_loading_manifests: number;
  exceptions: FactoryDashboardException[];
}
```

## File: apps/factory-portal/components/loading-bay/LoadingBayControls.tsx
```typescript
import Icon from '@/components/Icon';
⋮----
interface LoadingBayControlsProps {
  onRefresh: () => void;
  onDispatch: () => void;
  dispatching: boolean;
}
⋮----
export default function LoadingBayControls({ onRefresh, onDispatch, dispatching }: LoadingBayControlsProps)
```

## File: apps/factory-portal/components/loading-bay/LoadingBayGrid.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { motion } from 'framer-motion';
import { PageSection } from '@/components/PageSection';
import Icon from '@/components/Icon';
⋮----
type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';
⋮----
interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}
⋮----
interface LoadingBayGridProps {
  grouped: {
    key: TransferState;
    label: string;
    css: string;
    items: Transfer[];
  }[];
}
```

## File: apps/factory-portal/components/manifest-exceptions/ManifestExceptionsList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { PageSection } from '@/components/PageSection';
⋮----
export interface ManifestException {
  exception_id: string;
  manifest_id: string;
  transfer_id: string;
  reason: string;
  metadata?: string;
  attempt_count: number;
  escalated: boolean;
  created_at: string;
  correlation_id?: string;
}
⋮----
export function shortId(id: string): string
⋮----
export function reasonBadge(reason: string)
⋮----
interface ManifestExceptionsListProps {
  exceptions: ManifestException[];
  resolvingId?: string | null;
  onResolve?: (exceptionId: string) => void;
}
⋮----
<PageSection title=
⋮----
onClick=
```

## File: apps/factory-portal/components/payload-override/PayloadList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Icon from '@/components/Icon';
import type { Manifest, Transfer } from '../../app/payload-override/page';
```

## File: apps/factory-portal/components/portal/index.ts
```typescript

```

## File: apps/factory-portal/components/setup/constants.ts
```typescript
export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];
⋮----
export function setupStepIndex(pathname: string): number
```

## File: apps/factory-portal/components/setup/SetupWizardShell.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useTheme } from "@/components/ThemeProvider";
import Icon from "@/components/Icon";
import { SETUP_STEPS, setupStepIndex } from "./constants";
⋮----
function SetupThemeToggle()
```

## File: apps/factory-portal/components/staff/StaffList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { motion } from 'framer-motion';
⋮----
export interface StaffMember {
  id: string;
  name: string;
  phone: string;
  role: string;
  status: string;
  created_at: string;
}
⋮----
interface StaffListProps {
  staff: StaffMember[];
}
⋮----
export function StaffList({ staff }: StaffListProps)
```

## File: apps/factory-portal/components/supply-requests/constants.ts
```typescript

```

## File: apps/factory-portal/components/supply-requests/SupplyRequestBoard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { PageSection } from '@/components/PageSection';
import type { SupplyRequest } from './types';
import { ACTIONS } from './constants';
⋮----
interface SupplyRequestBoardProps {
  filtered: SupplyRequest[];
  transitioning: string | null;
  handleTransition: (request: SupplyRequest, action: string) => void;
  qcById: Record<string, string>;
  onQC: (request: SupplyRequest, result: "PASS" | "FAIL") => void;
}
⋮----
<PageSection title=
```

## File: apps/factory-portal/components/supply-requests/SupplyRequestHeader.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo } from 'react';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import type { SupplyRequest } from './types';
⋮----
interface SupplyRequestHeaderProps {
  requests: SupplyRequest[];
}
⋮----
export function SupplyRequestHeader({ requests }: SupplyRequestHeaderProps)
⋮----
<KpiStatCard label=
```

## File: apps/factory-portal/components/supply-requests/types.ts
```typescript
export interface SupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  shipped_quantity?: number;
  received_quantity?: number;
  recommended_qty?: number;
  unit_volume_vu?: number;
}
⋮----
export interface SupplyRequest {
  request_id: string;
  warehouse_id: string;
  warehouse_name?: string;
  supplier_id: string;
  state: string;
  priority: string;
  requested_delivery_date: string;
  total_volume_vu: number;
  notes: string;
  item_count?: number;
  items?: SupplyRequestItem[];
  created_at: string;
  sla_due_at?: string;
  sla_status?: "ON_TIME" | "AT_RISK" | "BREACHED" | "MET" | "N/A" | string;
  sla_hours_remaining?: number;
}
```

## File: apps/factory-portal/components/transfers/TransferFilters.tsx
```typescript
import React from 'react';
⋮----
interface TransferFiltersProps {
  stateFilter: string;
  setStateFilter: (filter: string) => void;
  stateFilters: string[];
}
⋮----
export function TransferFilters({ stateFilter, setStateFilter, stateFilters }: TransferFiltersProps)
```

## File: apps/factory-portal/components/transfers/TransferList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
⋮----
export interface Transfer {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  created_at: string;
  updated_at: string;
}
⋮----
export function stateClass(state: string): string
⋮----
export function priorityTone(priority: string): { background: string; color: string }
⋮----
interface TransferListProps {
  transfers: Transfer[];
}
```

## File: apps/factory-portal/components/ClientPolicyBanner.tsx
```typescript
import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { desktopClientPolicyContext } from "@pegasusx/desktop-bridge";
import { getClientPolicy } from "@/lib/client-policy";
⋮----
type PolicyState = {
  forceUpdate: boolean;
  outdated: boolean;
  minimumVersion: string;
  updateUrl?: string;
  deferReason?: string;
};
⋮----
// Policy endpoint optional on local dev stacks.
```

## File: apps/factory-portal/components/EmptyState.tsx
```typescript
import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";
⋮----
type EmptyStateVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error";
⋮----
interface EmptyStateProps {
  icon?: ReactNode;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}
⋮----
function resolveVariant(headline: string, body?: string): EmptyStateVariant
```

## File: apps/factory-portal/components/FactoryPageState.tsx
```typescript
import type { ReactNode } from 'react';
⋮----
import EmptyState from './EmptyState';
import { PageSkeleton } from './Skeleton';
⋮----
export type FactoryPageStateKind =
  | 'loading'
  | 'empty'
  | 'no-results'
  | 'error'
  | 'offline'
  | 'restricted'
  | 'auth-failure';
⋮----
interface FactoryPageStateProps {
  kind: FactoryPageStateKind;
  title?: string;
  subtitle?: string;
  headline?: string;
  body?: string;
  actionLabel?: string;
  onAction?: () => void;
  imageUrl?: string;
  skeleton?: ReactNode;
  className?: string;
}
```

## File: apps/factory-portal/components/FactoryRuntimeBanner.tsx
```typescript
export type FactoryRuntimeTone = 'live' | 'refreshing' | 'warning' | 'offline' | 'error';
⋮----
interface FactoryRuntimeBannerProps {
  tone: FactoryRuntimeTone;
  message: string;
}
⋮----
export default function FactoryRuntimeBanner({ tone, message }: FactoryRuntimeBannerProps)
```

## File: apps/factory-portal/components/KpiStatCard.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  className?: string;
  align?: "start" | "center";
};
⋮----
export function KpiStatCard({ label, value, sub, className = "", align = "start" }: KpiStatCardProps)
⋮----
⋮----
export function KpiStatGrid({ children, columns = 3 }: { children: ReactNode; columns?: 2 | 3 | 4 })
```

## File: apps/factory-portal/components/LanguageSwitcher.tsx
```typescript
import {
  supportedLocales,
  syncDocumentLocale,
  type Locale,
} from "@pegasusx/i18n";
import { usePortalLocale, usePortalT } from "@/lib/i18n";
```

## File: apps/factory-portal/components/ListToolbar.tsx
```typescript
type ListToolbarProps = {
  page: number;
  pageCount: number;
  totalLabel: string;
  onPrev: () => void;
  onNext: () => void;
  onExport?: () => void;
  exportLabel?: string;
  exportDisabled?: boolean;
};
⋮----
export function ListToolbar({
  page,
  pageCount,
  totalLabel,
  onPrev,
  onNext,
  onExport,
  exportLabel = "Export CSV",
  exportDisabled = false,
}: ListToolbarProps)
```

## File: apps/factory-portal/components/LocaleBootstrap.tsx
```typescript
import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasusx/i18n/browser";
⋮----
export default function LocaleBootstrap(): null
```

## File: apps/factory-portal/components/NotificationPanel.tsx
```typescript
import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';
import { motion, AnimatePresence } from 'framer-motion';
import Icon from './Icon';
⋮----
interface NotificationPanelProps {
  open: boolean;
  onClose: () => void;
  items: Notification[];
  unreadCount: number;
  onMarkRead: (id: string) => void;
  onMarkAllRead: () => void;
}
⋮----
function timeAgo(iso: string): string
⋮----
const handler = (e: MouseEvent) =>
```

## File: apps/factory-portal/components/PageSection.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
};
⋮----
export function PageSection({ title, description, actions, children, className = "" }: PageSectionProps)
⋮----
```

## File: apps/factory-portal/components/PageTransition.tsx
```typescript
import { motion, useReducedMotion } from "framer-motion";
import { ReactNode } from "react";
import { usePathname } from "next/navigation";
⋮----
interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}
⋮----
export default function PageTransition({ children, className = "" }: PageTransitionProps)
```

## File: apps/factory-portal/components/Providers.tsx
```typescript
import type { ReactNode } from "react";
import { Suspense } from "react";
import AuthGuard from "./AuthGuard";
import FactoryShell from "./FactoryShell";
import { PageSkeleton } from "./Skeleton";
import { ToastProvider } from "./Toast";
import { ThemeProvider } from "./ThemeProvider";
import { PortalOfflineTray } from "@/lib/portal-offline-tray";
import { DesktopDeepLinkBootstrap } from "@/lib/desktop-deep-link";
import { EnterpriseDesktopUpdateBootstrap } from "@/lib/desktop-updater";
⋮----
export default function Providers({ children }: { children: ReactNode })
```

## File: apps/factory-portal/components/Skeleton.tsx
```typescript
import type { CSSProperties } from 'react';
⋮----
export function Skeleton({ className = '', style }: { className?: string; style?: CSSProperties })
```

## File: apps/factory-portal/components/ThemeProvider.tsx
```typescript
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';
⋮----
export type ThemeMode = 'system' | 'light' | 'dark';
⋮----
interface ThemeCtx {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (m: ThemeMode) => void;
  cycle: () => void;
}
⋮----
export const useTheme = () =>
⋮----
function getSystemPreference(): 'light' | 'dark'
⋮----
function getStoredMode(): ThemeMode
⋮----
function resolveMode(mode: ThemeMode): 'light' | 'dark'
⋮----
function applyTheme(resolved: 'light' | 'dark')
⋮----
export function ThemeProvider({ children }: { children: ReactNode })
⋮----
const handler = () =>
```

## File: apps/factory-portal/components/Toast.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
⋮----
type ToastType = 'info' | 'success' | 'error' | 'warning';
⋮----
interface ToastAction {
  label: string;
  onClick: () => void;
}
⋮----
interface Toast {
  id: number;
  message: string;
  type: ToastType;
  action?: ToastAction;
  durationMs: number;
}
⋮----
interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: ToastAction, durationMs?: number) => void;
}
⋮----
export function useToast()
⋮----
function clamp(value: number, min: number, max: number)
⋮----
const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const finishDrag = () =>
⋮----
const onPointerUp = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerCancel = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
onPointerDown=
⋮----
event.stopPropagation();
toast.action?.onClick();
dismiss(toast.id);
```

## File: apps/factory-portal/lib/bridge.ts
```typescript

```

## File: apps/factory-portal/lib/client-policy.ts
```typescript
import { factoryApiBaseUrl } from '@/lib/auth';
⋮----
export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response>
```

## File: apps/factory-portal/lib/csv.ts
```typescript

```

## File: apps/factory-portal/lib/desktop-deep-link.tsx
```typescript
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";
⋮----
export function DesktopDeepLinkBootstrap()
```

## File: apps/factory-portal/lib/desktop-updater.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import {
  checkDesktopUpdate,
  desktopStoreListingUrl,
  installPendingDesktopUpdate,
  isDesktopStoreBuild,
  isTauri,
  openDesktopStoreListing,
  type DesktopUpdateInfo,
} from "@pegasusx/desktop-bridge";
⋮----
export function EnterpriseDesktopUpdateBootstrap()
⋮----
export async function openConfiguredDesktopStore(): Promise<boolean>
```

## File: apps/factory-portal/lib/factory-reconnect.ts
```typescript
export function getFactoryReconnectEpoch(): number
⋮----
export function notifyFactorySessionReconciled(): void
```

## File: apps/factory-portal/lib/factory-scope.ts
```typescript
import { readTokenFromCookie } from '@/lib/auth';
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
export function factoryOperatorId(): string
```

## File: apps/factory-portal/lib/i18n.ts
```typescript
import { useSyncExternalStore } from "react";
import {
  createTranslator,
  LOCALE_CHANGE_EVENT,
  resolveActiveLocale,
  type Locale,
} from "@pegasusx/i18n";
⋮----
function subscribe(onStoreChange: () => void)
⋮----
function getSnapshot(): Locale
⋮----
function getServerSnapshot(): Locale
⋮----
export function usePortalLocale(): Locale
⋮----
export function usePortalT()
```

## File: apps/factory-portal/lib/manifest-lifecycle.ts
```typescript
import {
  factoryManifestCompleteKey,
  factoryManifestDispatchKey,
  factoryManifestSealKey,
  factoryManifestStartLoadingKey,
} from '@pegasusx/api-core';
⋮----
export type ManifestLifecycleAction = {
  path: 'start-loading' | 'seal' | 'dispatch' | 'complete';
  label: string;
};
⋮----
export function manifestTransitionIdempotencyKey(
  manifestId: string,
  action: ManifestLifecycleAction['path'],
  factoryId = '',
): string
⋮----
export function nextManifestLifecycleAction(state: string): ManifestLifecycleAction | null
```

## File: apps/factory-portal/lib/money.ts
```typescript
export interface ParseMoneyOptions {
  allowNegative?: boolean;
}
⋮----
export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}
⋮----
function toSafeNumberOrNull(value: bigint): number | null
⋮----
function toMinorBigInt(value: number | bigint | null | undefined): bigint
⋮----
function groupAsciiDigits(digits: string, separator: string): string
⋮----
function stripSpaceGrouping(raw: string): string | null
⋮----
export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null
⋮----
export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function majorIntToMinorBigInt(major: number | bigint | string): bigint
⋮----
export function majorIntToMinor(major: number | bigint | string): number
⋮----
export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeTrunc(minor: number | bigint | null | undefined): number
⋮----
export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeRound(minor: number | bigint | null | undefined): number
⋮----
export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint
⋮----
export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number
⋮----
export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint
⋮----
export function applyBpsFee(amount: number, bps: number): number
⋮----
export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string
⋮----
export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string
⋮----
export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string
⋮----
export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string
⋮----
export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number
⋮----
export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string
```

## File: apps/factory-portal/lib/nav.ts
```typescript
export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };
⋮----
export function primaryNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function allNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function navSectionIsActive(
  section: NavSection,
  pathname: string,
  isActive: (pathname: string, href: string) => boolean,
): boolean
```

## File: apps/factory-portal/lib/session-reconcile.ts
```typescript
import { reconcileSession } from '@pegasusx/api-core';
import { apiFetch, factoryApiBaseUrl, readTokenFromCookie } from './auth';
import { notifyFactorySessionReconciled } from './factory-reconnect';
⋮----
export async function reconcileFactorySession(): Promise<void>
⋮----
export async function runFactorySessionReconcile(): Promise<void>
```

## File: apps/factory-portal/lib/use-factory-fleet-live-map.ts
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { useCallback, useEffect, useState } from 'react';
import { applyDriverLocationPatch, parseDriverLocationPatch } from '@pegasusx/ws-refresh-contract';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
⋮----
export type FactoryFleetLiveRoute = {
  manifest_id: string;
  route_id?: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
  live_location_available: boolean;
  location_stale?: boolean;
  driver_location?: {
    lat: number;
    lng: number;
    latitude?: number;
    longitude?: number;
  };
};
⋮----
export function useFactoryFleetLiveMap(pollMs = 15_000)
```

## File: apps/factory-portal/lib/use-factory-session-reconcile.ts
```typescript
import { useEffect, useRef } from 'react';
import { FACTORY_SESSION_RECONCILED } from '@/lib/factory-reconnect';
⋮----
export function useFactorySessionReconcile(onReconcile: () => void)
⋮----
const onEvent = () =>
```

## File: apps/factory-portal/lib/useNotifications.ts
```typescript
import { useState, useEffect, useCallback, useRef } from 'react';
import { PULSE_REFRESH_EVENTS } from '@pegasusx/ws-refresh-contract';
import { apiFetch, readTokenFromCookie, subscribeFactoryWS } from './auth';
⋮----
interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
}
⋮----
export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  read_at: string | null;
  created_at: string;
}
⋮----
interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}
⋮----
function normalizeNotification(item: BackendNotification): Notification
⋮----
export function useNotifications(options?: { enabled?: boolean })
⋮----
const handleWake = () =>
```

## File: apps/factory-portal/src/components/production/ProductionKioskScreen.tsx
```typescript
import React, { useState } from 'react';
⋮----
export const ProductionKioskScreen: React.FC = () =>
⋮----
setCurrentStatus("IN_PRODUCTION");
setIsJammed(false);
⋮----
onClick={() =>
```

## File: apps/factory-portal/src-tauri/src/commands/mod.rs
```rust
pub mod security;
⋮----
pub fn get_app_info() -> serde_json::Value {
```

## File: apps/factory-portal/src-tauri/src/commands/security.rs
```rust
use keyring::Entry;
use serde::Serialize;
⋮----
pub struct TokenResult {
pub success: bool,
pub token: Option<String>,
pub error: Option<String>,
⋮----
pub fn store_token(token: String, refresh_token: Option<String>) -> TokenResult {
⋮----
pub fn get_token() -> TokenResult {
⋮----
pub fn clear_token() -> TokenResult {
⋮----
fn read_token(service: &str) -> Result<Option<String>, String> {
⋮----
fn delete_if_present(service: &str, key: &str) -> bool {
```

## File: apps/factory-portal/src-tauri/src/lib.rs
```rust
mod commands;
⋮----
pub fn run() {
```

## File: apps/factory-portal/src-tauri/src/main.rs
```rust
fn main() {
```

## File: apps/factory-portal/src-tauri/Cargo.toml
```toml
[package]
name = "pegasus-factory-desktop"
version = "0.1.1"
description = "Pegasus — Factory Portal Desktop Shell"
authors = ["Pegasus"]
license = "proprietary"
repository = ""
default-run = "pegasus-factory-desktop"
edition = "2021"
rust-version = "1.77"

[build-dependencies]
tauri-build = { version = "2", features = [] }

[dependencies]
tauri = { version = "2", features = ["macos-private-api"] }
tauri-plugin-shell = "2"
tauri-plugin-dialog = "2"
tauri-plugin-fs = "2"
tauri-plugin-os = "2"
tauri-plugin-process = "2"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
keyring = { version = "3", features = ["apple-native", "windows-native"] }
log = "0.4"
env_logger = "0.11"

[target.'cfg(not(any(target_os = "android", target_os = "ios")))'.dependencies]
tauri-plugin-single-instance = { version = "2", features = ["deep-link"] }
tauri-plugin-deep-link = "2"
tauri-plugin-updater = "2"

[features]
default = ["custom-protocol"]
custom-protocol = ["tauri/custom-protocol"]
```

## File: apps/factory-portal/src-tauri/tauri.conf.json
```json
{
  "$schema": "https://raw.githubusercontent.com/tauri-apps/tauri/dev/crates/tauri-config-schema/schema.json",
  "productName": "Pegasus Factory Desktop",
  "version": "0.1.1",
  "identifier": "com.pegasusx.factory",
  "build": {
    "frontendDist": "../out",
    "devUrl": "http://localhost:3003",
    "beforeDevCommand": "pnpm run dev",
    "beforeBuildCommand": "pnpm run build:static"
  },
  "app": {
    "macOSPrivateApi": true,
    "withGlobalTauri": true,
    "windows": [
      {
        "label": "main",
        "title": "Factory Portal \u2014 Pegasus",
        "width": 1440,
        "height": 900,
        "minWidth": 1024,
        "minHeight": 700,
        "center": true,
        "resizable": true,
        "fullscreen": false,
        "transparent": false,
        "decorations": true,
        "titleBarStyle": "Overlay"
      }
    ],
    "security": {
      "dangerousDisableAssetCspModification": false,
      "csp": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob:; connect-src 'self' http://localhost:8180 http://127.0.0.1:8180 http://localhost:9099 http://127.0.0.1:9099 https://factory.void.pegasus.uz wss://localhost:8180 ws://localhost:8180"
    }
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "createUpdaterArtifacts": true,
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ],
    "macOS": {
      "minimumSystemVersion": "12.0",
      "frameworks": []
    }
  },
  "plugins": {
    "shell": {
      "open": true
    },
    "deep-link": {
      "desktop": {
        "schemes": [
          "pegasusx-factory"
        ]
      }
    },
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDhDREMyMEEzRkY5NEYzQjMKUldTejg1VC9veURjakZ6R3NoSmZ6NGIyOGVBZVBZZGs5TkdMWk9XRTArQ2UyakJNZ01XYXZtS0UK",
      "endpoints": [
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/factory-desktop/{{target}}/{{arch}}/updater.json"
      ],
      "windows": {
        "installMode": "basic"
      }
    }
  }
}
```

## File: apps/factory-portal/middleware.ts
```typescript
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
function isTokenExpired(token: string): boolean
⋮----
export function middleware(request: NextRequest)
```

## File: apps/factory-portal/vitest.config.ts
```typescript
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";
```

## File: apps/factory-portal/vitest.setup.ts
```typescript
getItem(key: string)
setItem(key: string, value: string)
removeItem(key: string)
clear()
get length()
key(index: number)
```

## File: apps/retailer-app-desktop/app/(dashboard)/assist/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Loader2, HandHelping } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type Section = { section_id: string; name: string };
type Ticket = {
  ticket_id: string;
  section_id: string;
  note: string;
  status: string;
  created_at?: string;
  claimed_by_user_id?: string;
};
⋮----
const create = async () =>
⋮----
const act = async (id: string, action: "claim" | "complete" | "cancel") =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/catalog/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ShoppingCart,
  Search,
  SlidersHorizontal,
  Package,
  Star,
  Layers,
  TrendingUp,
  Building2,
  ChevronRight,
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import CartDrawer from "../../../components/CartDrawer";
import CheckoutModal from "../../../components/CheckoutModal";
import ProductDetailDrawer from "../../../components/ProductDetailDrawer";
import EmptyState from "../../../components/EmptyState";
import { CatalogFilters } from "../../../components/catalog/CatalogFilters";
import { ProductGrid } from "../../../components/catalog/ProductGrid";
import { PageSection } from "../../../components/PageSection";
import { Skeleton } from "../../../components/Skeleton";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { useCart } from "../../../lib/cart";
import { isCatalogBlocked } from "../../../lib/stock-policy";
import { useWebSocket } from "../../../lib/ws";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import type { Product, Category, Supplier } from "../../../lib/types";
import {
  productDisplayPrice,
  productListPrice,
  productSalePrice,
} from "../../../lib/types";
⋮----
onClick=
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/credit/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { getRetailerToken } from "@/lib/auth";
⋮----
type Relationship = {
  supplier_id: string;
  credit_enabled: boolean;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  available_credit_minor?: number;
  current_balance_minor?: number;
  profile_status?: string;
  on_hold?: boolean;
};
⋮----
type Invoice = {
  invoice_id: string;
  supplier_id: string;
  order_id: string;
  status: string;
  balance_minor: number;
  due_at: string;
  terms_days: number;
  aging_bucket?: string;
};
⋮----
```

## File: apps/retailer-app-desktop/app/(dashboard)/dashboard/page.tsx
```typescript
import { useCallback, useMemo } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import Link from "next/link";
import {
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { CreditProfileCard } from "@/components/CreditProfileCard";
import { motion } from "framer-motion";
import { CommandBoard } from "../../../components/dashboard/CommandBoard";
import { KpiGrid } from "../../../components/dashboard/KpiGrid";
import { QuickReorderSection } from "../../../components/dashboard/QuickReorderSection";
import { AiPredictionSection } from "../../../components/dashboard/AiPredictionSection";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Product, RetailerAIPredictionsResponse } from "../../../lib/types";
import type { RetailerControlTowerPulse } from "@pegasusx/types";
import { usePortalT } from "@/lib/i18n";
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
```

## File: apps/retailer-app-desktop/app/(dashboard)/insights/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useMemo, useCallback } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import {
  RefreshCw,
  Package,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import EmptyState from "../../../components/EmptyState";
import SpendAnalytics from "./SpendAnalytics";
import { InsightsSummary } from "../../../components/insights/InsightsSummary";
import { InsightsSidebar } from "../../../components/insights/InsightsSidebar";
import { SellThroughPanel } from "../../../components/insights/SellThroughPanel";
import { useLiveData } from "../../../lib/hooks";
import { confirmAiOrder, rejectAiOrder } from "../../../lib/api";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { RetailerAIPrediction, RetailerAIPredictionsResponse, RetailerAnalytics } from "../../../lib/types";
import {
  aiPredictionQty,
  aiPredictionTitle,
  formatMinorAmount,
} from "../../../lib/types";
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
type DetailedAnalytics = {
  series?: Array<{
    order_id: string;
    total_minor: number;
    currency?: string;
    created_at: string;
  }>;
  by_category?: unknown[];
  from?: string;
  to?: string;
};
```

## File: apps/retailer-app-desktop/app/(dashboard)/insights/SpendAnalytics.tsx
```typescript
type SpendAnalyticsProps = {
  className?: string;
  categorySpend?: { name: string; value: number }[];
  spendTrend?: { day: string; spend: number }[];
};
```

## File: apps/retailer-app-desktop/app/(dashboard)/reports/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Download, Loader2 } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type Summary = {
  sales_minor?: number;
  sale_count?: number;
  on_hand_sku_count?: number;
  low_stock_count?: number;
  open_variances?: number;
  top_skus?: { sku: string; sales_minor: number; units: number }[];
};
⋮----
type SalesItem = { key: string; sales_minor: number; sale_count: number; units: number };
⋮----
function formatMoney(minor: number)
⋮----
const downloadCSV = async () =>
⋮----
title=
⋮----
⋮----
{formatMoney(row.sales_minor)} ·
```

## File: apps/retailer-app-desktop/app/(dashboard)/settings/capabilities/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  Layers,
  AlertTriangle,
  CheckCircle2,
  Info,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type PackStatus = {
  id: string;
  name: string;
  description: string;
  hard_deps?: string[];
  soft_deps?: string[];
  always_on?: boolean;
  enabled: boolean;
  config?: Record<string, unknown>;
};
⋮----
type CapabilitiesResponse = {
  retailer_id: string;
  capabilities: string[];
  packs: PackStatus[];
};
⋮----
type EvalResult = {
  status: string;
  pack_id?: string;
  message?: string;
  missing_hard_deps?: string[];
  missing_soft_deps?: string[];
  enable_all?: string[];
};
⋮----
const enablePack = async (packId: string, opts?: { acceptSoft?: boolean; enableDeps?: boolean }) =>
⋮----
const disablePack = async (packId: string) =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/layout.tsx
```typescript
import { usePathname, useRouter } from "next/navigation";
import { WebSocketProvider } from "../../lib/ws";
import { SessionReconcileListener } from "../../lib/session-reconcile-listener";
import { PendingCheckoutFlusher } from "../../lib/pending-checkout-flusher";
import { PendingPosFlusher } from "../../lib/pending-pos-flusher";
import { DesktopCacheBootstrap } from "../../lib/desktop-cache-bootstrap";
import { DesktopDeepLinkBootstrap } from "../../lib/desktop-deep-link";
import { NotificationsProvider } from "../../lib/notifications";
import { clearStoredToken } from "../../lib/bridge";
import { CartProvider } from "../../lib/cart";
import RetailerShell from "../../components/RetailerShell";
import ClientPolicyBanner from "../../components/ClientPolicyBanner";
import PaymentModal from "../../components/PaymentModal";
import ShopClosedModal from "../../components/ShopClosedModal";
import { RetailerOfflineTray } from "../../lib/retailer-offline-tray";
⋮----
export default function DashboardLayout({ children }: { children: React.ReactNode })
```

## File: apps/retailer-app-desktop/app/auth/select-org/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Building2, Loader2 } from "lucide-react";
import type { RetailerMembershipDTO } from "@pegasusx/types";
import {
  clearStashedMemberships,
  listMemberships,
  loadStashedMemberships,
  readPendingOrgToken,
  selectOrg,
} from "@/lib/multi-org-auth";
⋮----
async function onSelect(retailerId: string)
⋮----
clearStashedMemberships();
```

## File: apps/retailer-app-desktop/app/auth/layout.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';
```

## File: apps/retailer-app-desktop/app/setup/address/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { retailerSetupKey } from "@pegasusx/api-core";
import { getRetailerId } from "@/lib/retailer-profile";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import { SETUP_TAX_KEY } from "@/components/setup/constants";
⋮----
function getCookie(name: string)
⋮----
async function handleSubmit()
```

## File: apps/retailer-app-desktop/app/setup/tax/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { PortalField, PortalInput, PortalActions, FormAlert } from "@/components/portal";
import { SETUP_TAX_KEY } from "@/components/setup/constants";
⋮----
function handleContinue()
```

## File: apps/retailer-app-desktop/app/setup/layout.tsx
```typescript
import SetupWizardShell from "@/components/setup/SetupWizardShell";
⋮----
export default function SetupLayout({ children }: { children: React.ReactNode })
```

## File: apps/retailer-app-desktop/app/setup/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";
⋮----
export default function SetupIndexPage()
```

## File: apps/retailer-app-desktop/app/error.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
```

## File: apps/retailer-app-desktop/app/layout.tsx
```typescript
import type { Metadata } from "next";
import { Plus_Jakarta_Sans, EB_Garamond } from "next/font/google";
import LocaleBootstrap from "../components/LocaleBootstrap";
import Providers from "../components/Providers";
```

## File: apps/retailer-app-desktop/components/auto-order/AutoOrderList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { PageSection } from "@/components/PageSection";
import type { Prediction } from "@/lib/types";
⋮----
<PageSection title=
```

## File: apps/retailer-app-desktop/components/catalog/ProductGrid.tsx
```typescript
import { motion, AnimatePresence } from "framer-motion";
import { Package } from "lucide-react";
import { Skeleton } from "../Skeleton";
import EmptyState from "../EmptyState";
import { isCatalogBlocked } from "../../lib/stock-policy";
import { packCurrency, readCachedAuthSession } from "@pegasusx/api-core";
import { productDisplayPrice, productListPrice, productSalePrice } from "../../lib/types";
import type { Product } from "../../lib/types";
⋮----
export interface EmptyStateConfig {
  headline: string;
  body: string;
  variant: "restricted" | "offline" | "error" | "no-products" | "no-results";
}
⋮----
export interface ProductGridProps {
  loadingProducts: boolean;
  filteredProducts: Product[];
  emptyStateConfig: EmptyStateConfig;
  loadIssue: "restricted" | "offline" | "error" | null;
  refreshAll: () => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  setSelectedProduct: (p: Product) => void;
  addToCart: (p: Product) => void;
}
⋮----
{productDisplayPrice(product).toLocaleString()}
⋮----
e.stopPropagation();
```

## File: apps/retailer-app-desktop/components/dashboard/AiPredictionSection.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState } from "react";
import Link from "next/link";
import { PageSection } from "../PageSection";
import EmptyState from "../EmptyState";
import { confirmAiOrder, rejectAiOrder } from "../../lib/api";
import type { RetailerAIPrediction } from "../../lib/types";
import {
  aiPredictionQty,
  aiPredictionTitle,
  formatMinorAmount,
} from "../../lib/types";
⋮----
interface AiPredictionSectionProps {
  predictionList: RetailerAIPrediction[];
  onRefresh: () => void;
}
⋮----
const runAction = async (orderId: string, action: "confirm" | "reject") =>
⋮----
title=
```

## File: apps/retailer-app-desktop/components/dashboard/KpiGrid.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import Link from "next/link";
import { ShoppingCart, PackageSearch, Brain, Truck, Layers3 } from "lucide-react";
import { BentoGrid, BentoCard } from "../BentoGrid";
import CountUp from "../CountUp";
⋮----
interface KpiGridProps {
  activeOrdersLength: number;
  predictionListLength: number;
  productListLength: number;
  cartQuantity: number;
  completedOrdersLength: number;
  blockedPredictionCount: number;
  uniqueSuppliersCount: number;
}
⋮----
<QuickAction href="/catalog" icon=
⋮----
label=
```

## File: apps/retailer-app-desktop/components/dashboard/QuickReorderSection.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import { motion } from "framer-motion";
import { Package, ArrowUpRight } from "lucide-react";
import { PageSection } from "../PageSection";
import EmptyState from "../EmptyState";
import type { Product } from "../../lib/types";
⋮----
interface QuickReorderSectionProps {
  reorderProducts: Product[];
  onRefresh: () => void;
  onAddToCart: (product: Product) => void;
}
⋮----
title=
⋮----
onClick=
```

## File: apps/retailer-app-desktop/components/insights/InsightsSidebar.tsx
```typescript
import { Zap, ArrowUpRight, ChevronRight } from "lucide-react";
import { moneyCurrency } from "../../lib/payment-catalog";
import type { TopProduct } from "../../lib/types";
⋮----
type DetailedSeriesRow = {
  order_id: string;
  total_minor?: number;
  currency?: string;
};
⋮----
interface InsightsSidebarProps {
  totalThisMonth: number;
  topProducts: TopProduct[];
  detailedError: unknown;
  detailedSeries: DetailedSeriesRow[];
}
```

## File: apps/retailer-app-desktop/components/insights/InsightsSummary.tsx
```typescript
import { BarChart3, Brain, TrendingUp, Zap } from "lucide-react";
import { BentoGrid, BentoCard } from "../../components/BentoGrid";
import CountUp from "../../components/CountUp";
import MiniSparkline from "../../components/MiniSparkline";
import type { RetailerAIPrediction, TopProduct } from "../../lib/types";
⋮----
interface InsightsSummaryProps {
  totalThisMonth: number;
  sparkRevenue: number[];
  predList: RetailerAIPrediction[];
  sparkOrders: number[];
  topProducts: TopProduct[];
}
```

## File: apps/retailer-app-desktop/components/orders/OrderFilters.tsx
```typescript
import { RefreshCw } from "lucide-react";
import { usePortalT } from "@/lib/i18n";
⋮----
interface OrderFiltersProps {
  activeTab: "ALL" | "ACTIVE" | "COMPLETED";
  setActiveTab: (tab: "ALL" | "ACTIVE" | "COMPLETED") => void;
  isOrdersRefreshing: boolean;
  refreshAll: () => void;
}
⋮----
aria-label=
```

## File: apps/retailer-app-desktop/components/portal/index.ts
```typescript

```

## File: apps/retailer-app-desktop/components/settings/OverrideComponents.tsx
```typescript
import { useState, useEffect } from "react";
import { Loader2, CheckCircle2, ChevronDown, ChevronRight } from "lucide-react";
import { Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
⋮----
function getBrowserStorage(): Storage | null
⋮----
onClick=
⋮----
label=
⋮----
hasHistory=
```

## File: apps/retailer-app-desktop/components/setup/constants.ts
```typescript
export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];
⋮----
export function setupStepIndex(pathname: string): number
```

## File: apps/retailer-app-desktop/components/setup/SetupWizardShell.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useTheme } from "@/components/ThemeProvider";
import Icon from "@/components/Icon";
import { SETUP_STEPS, setupStepIndex } from "./constants";
⋮----
function SetupThemeToggle()
```

## File: apps/retailer-app-desktop/components/tracking/TrackingStatus.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { Package, MapPin, Users, Truck } from "lucide-react";
import { BentoGrid, BentoCard } from "../BentoGrid";
import CountUp from "../CountUp";
import { PageSection } from "../PageSection";
import type { TrackingOrder } from "../../lib/types";
⋮----
function formatAmount(amount: number): string
⋮----
interface TrackingStatusProps {
  activeFulfillmentCount: number;
  approachingCount: number;
  suppliers: { id: string; name: string }[];
  avgItems: number;
  recentReceipts: TrackingOrder[];
  selectedSupplierIds: Set<string>;
  toggleSupplier: (id: string) => void;
}
⋮----
title=
```

## File: apps/retailer-app-desktop/components/BentoGrid.tsx
```typescript
import { type ReactNode } from "react";
import { motion, AnimatePresence } from "framer-motion";
⋮----
interface BentoGridProps {
  children: ReactNode;
  className?: string;
  staggerChildren?: number;
}
⋮----
interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}
⋮----
export function BentoSkeleton({
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoSkeletonProps)
```

## File: apps/retailer-app-desktop/components/ClientPolicyBanner.tsx
```typescript
import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { desktopClientPolicyContext } from "@pegasusx/desktop-bridge";
import { getClientPolicy } from "../lib/api";
⋮----
type PolicyState = {
  forceUpdate: boolean;
  outdated: boolean;
  minimumVersion: string;
  updateUrl?: string;
  deferReason?: string;
};
⋮----
// Policy endpoint optional on local dev stacks.
```

## File: apps/retailer-app-desktop/components/CountUp.tsx
```typescript
import { useEffect, useRef, useState } from 'react';
⋮----
interface CountUpProps {
  end: number;
  duration?: number;
  prefix?: string;
  suffix?: string;
  decimals?: number;
  className?: string;
}
⋮----
const step = (now: number) =>
```

## File: apps/retailer-app-desktop/components/CreditProfileCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { CreditCard, Loader2, AlertTriangle } from "lucide-react";
import { apiFetch } from "../lib/auth";
⋮----
type CreditProfile = {
  retailer_id: string;
  supplier_id: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  risk_tier?: string;
  status: string;
  delinquency_count?: number;
};
```

## File: apps/retailer-app-desktop/components/EmptyState.tsx
```typescript
import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";
⋮----
type EmptyStateVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "production"
  | "no-orders"
  | "no-products"
  | "no-predictions"
  | "no-suppliers";
⋮----
interface EmptyStateProps {
  icon?: ReactNode;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}
⋮----
function resolveVariant(headline: string, body?: string): EmptyStateVariant
```

## File: apps/retailer-app-desktop/components/LanguageSwitcher.tsx
```typescript
import {
  supportedLocales,
  syncDocumentLocale,
  type Locale,
} from "@pegasusx/i18n";
import { usePortalLocale, usePortalT } from "@/lib/i18n";
```

## File: apps/retailer-app-desktop/components/LocaleBootstrap.tsx
```typescript
import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasusx/i18n/browser";
⋮----
export default function LocaleBootstrap(): null
```

## File: apps/retailer-app-desktop/components/LoyaltyCard.tsx
```typescript
import { useLiveData } from "../lib/hooks";
import type { LoyaltyLedgerResponse, LoyaltyTierView } from "@pegasusx/types";
```

## File: apps/retailer-app-desktop/components/MiniSparkline.tsx
```typescript
import { useMemo } from 'react';
⋮----
interface MiniSparklineProps {
  data: number[];
  width?: number;
  height?: number;
  color?: string;
  className?: string;
}
```

## File: apps/retailer-app-desktop/components/OrderTimelinePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { apiFetch } from '../lib/auth';
⋮----
function formatWhen(iso: string): string
⋮----
return <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
⋮----
{formatWhen(entry.created_at)}
⋮----
Proposed delivery:
```

## File: apps/retailer-app-desktop/components/OrgSwitcher.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Building2, ChevronDown, Loader2 } from "lucide-react";
import type { RetailerMembershipDTO } from "@pegasusx/types";
import { listMemberships, readActiveOrgId, switchOrg } from "@/lib/multi-org-auth";
import { getRetailerId } from "@/lib/retailer-profile";
⋮----
async function onSwitch(retailerId: string)
⋮----
// Hard reload so POS/cart/assist remount with new JWT scope.
```

## File: apps/retailer-app-desktop/components/PageSection.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
};
⋮----
export function PageSection({ title, description, actions, children, className = "" }: PageSectionProps)
⋮----
```

## File: apps/retailer-app-desktop/components/Providers.tsx
```typescript
import type { ReactNode } from "react";
import PageTransition from "./PageTransition";
import { ThemeProvider } from "./ThemeProvider";
import { EnterpriseDesktopUpdateBootstrap } from "../lib/desktop-updater";
⋮----
export default function Providers({ children }: { children: ReactNode })
```

## File: apps/retailer-app-desktop/components/Skeleton.tsx
```typescript
import type { CSSProperties } from "react";
⋮----
export function Skeleton({ className = "", style }: { className?: string; style?: CSSProperties })
```

## File: apps/retailer-app-desktop/components/ThemeProvider.tsx
```typescript
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';
⋮----
export type ThemeMode = 'system' | 'light' | 'dark';
⋮----
interface ThemeContextValue {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (mode: ThemeMode) => void;
  cycle: () => void;
}
⋮----
export const useTheme = () =>
⋮----
function getSystemPreference(): 'light' | 'dark'
⋮----
function getStoredMode(): ThemeMode
⋮----
function resolveMode(mode: ThemeMode): 'light' | 'dark'
⋮----
function applyTheme(resolved: 'light' | 'dark')
⋮----
export function ThemeProvider({ children }: { children: ReactNode })
⋮----
const handleThemeChange = () =>
```

## File: apps/retailer-app-desktop/lib/api.ts
```typescript
import { apiFetch } from './auth';
import {
  retailerConfirmAIKey,
  retailerConfirmPreorderKey,
  retailerAcceptDeliveryProposalKey,
  retailerRejectDeliveryProposalKey,
  retailerRejectAIKey,
  retailerEditPreorderKey,
  retailerSetupKey,
  retailerProfileUpdateKey,
} from '@pegasusx/api-core';
⋮----
export async function confirmAiOrder(orderId: string): Promise<Response>
⋮----
export async function rejectAiOrder(orderId: string, reason: string): Promise<Response>
⋮----
export async function confirmPreorder(orderId: string): Promise<Response>
⋮----
export async function editPreorder(
  orderId: string,
  requestedDeliveryDate: string,
  lineItems: Array<{ sku: string; name: string; quantity: number; unit_price_minor: number }>,
): Promise<Response>
⋮----
export async function acceptDeliveryProposal(orderId: string): Promise<Response>
⋮----
export async function rejectDeliveryProposal(orderId: string, reason: string): Promise<Response>
⋮----
export async function setupRetailer(
  payload: Record<string, unknown>,
  retailerId: string,
): Promise<Response>
⋮----
export async function getPricingRules(): Promise<Response>
⋮----
export async function getDetailedAnalytics(from?: string, to?: string): Promise<Response>
⋮----
export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response>
⋮----
export async function deactivateCard(tokenId: string): Promise<Response>
⋮----
export async function setDefaultCard(tokenId: string): Promise<Response>
⋮----
export type FileClaimLine = {
  sku: string;
  quantity: number;
  reason: string;
};
⋮----
export type FileClaimEvidence = {
  evidence_type: string;
  uri: string;
  mime_type: string;
};
⋮----
export type RetailerClaim = {
  claim_id: string;
  order_id: string;
  claim_type: string;
  status: string;
  description?: string;
  amount_minor?: number;
  currency?: string;
  created_at?: string;
};
⋮----
export type ClaimEligibility = {
  eligible: boolean;
  ends_at: string | null;
  window_hours: number;
  hours_remaining: number;
  policy_source: string;
  photo_required_types?: string[];
  order_status?: string;
  reason?: string;
};
⋮----
export async function getClaimEligibility(
  orderId: string,
): Promise<ClaimEligibility>
⋮----
export async function listOrderClaims(orderId: string): Promise<Response>
⋮----
export async function fileOrderClaim(
  orderId: string,
  body: {
    claim_type: string;
    description: string;
    line_items: FileClaimLine[];
    evidences: FileClaimEvidence[];
  },
  idempotencyKey: string,
): Promise<Response>
⋮----
export async function getMediaUploadTicket(
  purpose: string,
  orderId?: string,
  ext = 'jpg',
): Promise<Response>
⋮----
export async function uploadClaimPhoto(
  file: File,
  orderId: string,
): Promise<string>
⋮----
export function claimTypeNeedsPhoto(claimType: string): boolean
⋮----
async function compressImageToJpeg(file: File, quality: number): Promise<Blob>
⋮----
export async function getRetailerTracking(): Promise<Response>
```

## File: apps/retailer-app-desktop/lib/auth.ts
```typescript
import { useState, useEffect } from 'react';
import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession } from '@pegasusx/api-core';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { createTranslator, detectBrowserLocale, translateProblemDetail } from '@pegasusx/i18n';
import type { ProblemDetail } from '@pegasusx/types';
import { isProblemDetail } from '@pegasusx/types';
⋮----
export function retailerApiBaseUrl(): string
const getTranslator = () =>
⋮----
export function readToken(): string
⋮----
/**
 * Hydration-safe token hook. Returns '' on the first (SSR-matching) render,
 * then reads the cookie after mount so server and client HTML always agree.
 */
export function useToken(): string
⋮----
/**
 * Get a valid retailer token. On desktop, checks OS keyring first.
 * Then reads from cookie / local storage.
 * Throws if no token is available.
 */
export async function getRetailerToken(): Promise<string>
⋮----
// Desktop: try OS keyring first
⋮----
} catch { /* fall through */ }
⋮----
function readRefreshFromCookie(): string
⋮----
async function tryRefreshToken(): Promise<string | null>
⋮----
export async function apiFetch(path: string, init?: RequestInit): Promise<Response>
```

## File: apps/retailer-app-desktop/lib/bridge.ts
```typescript

```

## File: apps/retailer-app-desktop/lib/clear-org-scoped-state.ts
```typescript
import { cacheClearAll } from "@pegasusx/desktop-cache";
import { clearParkedPosCart } from "./pending-pos-sales";
⋮----
export type ClearOrgScopedStateResult = {
  clearedLocalKeys: string[];
  clearedSessionKeys: string[];
  parkedCartCleared: boolean;
  kvCacheCleared: boolean;
};
⋮----
export async function clearOrgScopedState(): Promise<ClearOrgScopedStateResult>
```

## File: apps/retailer-app-desktop/lib/desktop-cache-bootstrap.tsx
```typescript
import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";
import { initRetailerProfile } from "./retailer-profile";
⋮----
export function DesktopCacheBootstrap()
```

## File: apps/retailer-app-desktop/lib/desktop-deep-link.tsx
```typescript
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";
⋮----
export function DesktopDeepLinkBootstrap()
```

## File: apps/retailer-app-desktop/lib/desktop-updater.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import {
  checkDesktopUpdate,
  desktopStoreListingUrl,
  installPendingDesktopUpdate,
  isDesktopStoreBuild,
  isTauri,
  openDesktopStoreListing,
  type DesktopUpdateInfo,
} from "@pegasusx/desktop-bridge";
⋮----
export function EnterpriseDesktopUpdateBootstrap()
⋮----
export async function openConfiguredDesktopStore(): Promise<boolean>
```

## File: apps/retailer-app-desktop/lib/dock-pending-patches.ts
```typescript
import type { TrackingOrder } from "./types";
⋮----
export type DockOrderPatch = {
  orderId: string;
  patch: Partial<TrackingOrder>;
  updatedAt: string;
};
⋮----
function readAll(): DockOrderPatch[]
⋮----
function writeAll(rows: DockOrderPatch[])
⋮----
export function queueDockOrderPatch(orderId: string, patch: Partial<TrackingOrder>)
⋮----
export function clearDockPendingPatches()
⋮----
export function applyDockPendingPatches(orders: TrackingOrder[]): TrackingOrder[]
```

## File: apps/retailer-app-desktop/lib/i18n.ts
```typescript
import { useSyncExternalStore } from "react";
import {
  createTranslator,
  LOCALE_CHANGE_EVENT,
  resolveActiveLocale,
  type Locale,
} from "@pegasusx/i18n";
⋮----
function subscribe(onStoreChange: () => void)
⋮----
function getSnapshot(): Locale
⋮----
function getServerSnapshot(): Locale
⋮----
export function usePortalLocale(): Locale
⋮----
export function usePortalT()
```

## File: apps/retailer-app-desktop/lib/money.ts
```typescript
export interface ParseMoneyOptions {
  allowNegative?: boolean;
}
⋮----
export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}
⋮----
function toSafeNumberOrNull(value: bigint): number | null
⋮----
function toMinorBigInt(value: number | bigint | null | undefined): bigint
⋮----
function groupAsciiDigits(digits: string, separator: string): string
⋮----
function stripSpaceGrouping(raw: string): string | null
⋮----
export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null
⋮----
export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function majorIntToMinorBigInt(major: number | bigint | string): bigint
⋮----
export function majorIntToMinor(major: number | bigint | string): number
⋮----
export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeTrunc(minor: number | bigint | null | undefined): number
⋮----
export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeRound(minor: number | bigint | null | undefined): number
⋮----
export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint
⋮----
export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number
⋮----
export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint
⋮----
export function applyBpsFee(amount: number, bps: number): number
⋮----
export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string
⋮----
export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string
⋮----
export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string
⋮----
export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string
⋮----
export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number
⋮----
export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string
```

## File: apps/retailer-app-desktop/lib/multi-org-auth.ts
```typescript
import { storeToken, isTauri } from "./bridge";
import { retailerApiBaseUrl } from "./auth";
import { clearOrgScopedState } from "./clear-org-scoped-state";
import { setRetailerProfile } from "./retailer-profile";
import type {
  RetailerLoginResponse,
  RetailerMembershipDTO,
  RetailerMembershipsResponse,
} from "@pegasusx/types";
⋮----
export function isPendingOrgSelectResponse(
  data: RetailerLoginResponse | null | undefined,
): boolean
⋮----
export function persistFullAuthTokens(data: {
  token: string;
  refresh_token?: string;
}): void
⋮----
export function persistPendingOrgToken(token: string, expiresInSec = 420): void
⋮----
export function readPendingOrgToken(): string
⋮----
export function stashPendingMemberships(memberships: RetailerMembershipDTO[]): void
⋮----
export function loadStashedMemberships(): RetailerMembershipDTO[]
⋮----
export function clearStashedMemberships(): void
⋮----
export async function applyFullAuthResponse(
  data: RetailerLoginResponse,
  opts?: { clearScoped?: boolean },
): Promise<void>
⋮----
// Stash org id separately for switcher highlight
⋮----
export function readActiveOrgId(): string
⋮----
export async function selectOrg(retailerId: string): Promise<RetailerLoginResponse>
⋮----
export async function switchOrg(retailerId: string): Promise<RetailerLoginResponse>
⋮----
export async function listMemberships(): Promise<RetailerMembershipDTO[]>
```

## File: apps/retailer-app-desktop/lib/nav.ts
```typescript
import type { ElementType } from "react";
import {
  Activity,
  BarChart3,
  Box,
  Building2,
  Clock,
  Container,
  FileBarChart,
  HandHelping,
  LayoutDashboard,
  LayoutGrid,
  MapPin,
  PackageSearch,
  Radar,
  RefreshCcw,
  Settings,
  ShoppingCart,
  Store,
} from "lucide-react";
⋮----
export type NavEntry = {
  href: string;
  icon: ElementType;
  labelKey: string;
  perm?: string;
  pack?: string;
};
export type NavSection = { labelKey?: string; items: NavEntry[] };
⋮----
export function primaryNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function allNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function navSectionIsActive(
  section: NavSection,
  pathname: string,
  isActive: (pathname: string, href: string) => boolean,
): boolean
```

## File: apps/retailer-app-desktop/lib/notifications-core.ts
```typescript
import type { WsMessage } from './ws';
⋮----
export type BackendNotificationItem = {
  id?: string;
  notification_id?: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at?: string | null;
  created_at: string;
  handoff_metadata?: import('@pegasusx/types').HandoffCardMetadata;
};
⋮----
export type RetailerNotificationItem = {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  readAt: string | null;
  createdAt: string;
  handoffMetadata?: import('@pegasusx/types').HandoffCardMetadata;
};
⋮----
export function normalizeNotification(item: BackendNotificationItem): RetailerNotificationItem
⋮----
export function shouldRefreshNotificationInbox(message: WsMessage | null): boolean
```

## File: apps/retailer-app-desktop/lib/notifications.tsx
```typescript
import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { apiFetch } from './auth';
import { useWebSocket } from './ws';
import {
  normalizeNotification,
  shouldRefreshNotificationInbox,
  type BackendNotificationItem,
  type RetailerNotificationItem,
} from './notifications-core';
⋮----
type BackendNotificationsResponse = {
  notifications?: BackendNotificationItem[];
  unread_count?: number;
  has_more?: boolean;
};
⋮----
async function fetchNotificationPage(offset: number): Promise<BackendNotificationsResponse>
⋮----
type NotificationsContextType = {
  items: RetailerNotificationItem[];
  unreadCount: number;
  loading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  loadMore: () => Promise<void>;
  markRead: (notificationId: string) => Promise<void>;
  markAllRead: () => Promise<void>;
};
⋮----
export function NotificationsProvider({ children }: { children: React.ReactNode })
⋮----
export function useRetailerNotifications()
```

## File: apps/retailer-app-desktop/lib/order-receipt.ts
```typescript
import { apiFetch } from "@/lib/auth";
⋮----
export async function openRetailerOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void>
⋮----
export async function openTrackingReceipt(order: {
  order_id: string;
  fiscal_qr?: string;
  latest_fiscal_receipt_id?: string;
}): Promise<void>
```

## File: apps/retailer-app-desktop/lib/payment-catalog.ts
```typescript
import { packCurrency, readCachedAuthSession, selectablePackPsps } from '@pegasusx/api-core';
import type { PSPListing } from "@pegasusx/types";
⋮----
export function catalogGatewayCodes(catalog: Array<{ code?: string }> | null | undefined): string[]
⋮----
export function catalogOmits(
  catalog: Array<{ code?: string }> | null | undefined,
  codes: string[],
): boolean
⋮----
/** Card rails the retailer may tap. Empty catalog never invents Adyen/Stripe. */
export function filterRetailerCardGateways(
  incoming: string[] | null | undefined,
  catalogCodes: string[] | null | undefined,
): string[]
⋮----
export function checkoutGatewayForMethod(
  method: string,
  catalogCodes: string[] | null | undefined,
): string
⋮----
export function displayPackCurrency(raw?: string | null, packCurrencyCode = ""): string
⋮----
export function sessionPackCurrency(): string
⋮----
export function moneyCurrency(raw?: string | null): string
⋮----
export function retailerCatalogGateways(catalog: PSPListing[] | null | undefined): string[]
```

## File: apps/retailer-app-desktop/lib/pending-checkout-flusher.tsx
```typescript
import { useEffect } from "react";
import { flushPendingCheckouts } from "./pending-checkout";
⋮----
export function PendingCheckoutFlusher()
⋮----
const onOnline = () =>
```

## File: apps/retailer-app-desktop/lib/pending-checkout.ts
```typescript
import { apiFetch } from "./auth";
import {
  enqueuePendingCheckout as cacheEnqueue,
  isRetryableCheckoutError,
  listPendingCheckouts as cacheList,
  pendingCheckoutQueuedMessage as cacheQueuedMessage,
  removePendingCheckout as cacheRemove,
  updatePendingCheckout,
  type PendingCheckout,
} from "@pegasusx/desktop-cache";
⋮----
export async function listPendingCheckouts(): Promise<PendingCheckout[]>
⋮----
export async function enqueuePendingCheckout(
  payload: unknown,
  idempotencyKey: string,
): Promise<PendingCheckout>
⋮----
export async function removePendingCheckout(id: string): Promise<void>
⋮----
export async function flushPendingCheckouts(): Promise<{ flushed: number; failed: number }>
⋮----
export function pendingCheckoutQueuedMessage(error: unknown): string
```

## File: apps/retailer-app-desktop/lib/receiving-window.ts
```typescript
export function normalizeReceivingWindow(value: string): string
⋮----
export function validateReceivingWindowField(value: string): string | undefined
```

## File: apps/retailer-app-desktop/lib/retailer-profile.ts
```typescript
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import type { RetailerProfile } from "./types";
⋮----
function readLegacyLocalProfile(): RetailerProfile | null
⋮----
export async function initRetailerProfile(): Promise<void>
⋮----
export function getRetailerProfile(): RetailerProfile | null
⋮----
export function getRetailerId(): string
⋮----
/** Persist profile to SQLite (and localStorage only on non-Tauri web dev). */
export async function setRetailerProfile(profile: RetailerProfile): Promise<void>
⋮----
export async function mergeRetailerProfile(
  patch: Partial<RetailerProfile> & { retailer_id?: string },
): Promise<void>
```

## File: apps/retailer-app-desktop/lib/session-reconcile-listener.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect } from "react";
import { useWebSocket } from "./ws";
import { reconcileRetailerSession } from "./session-reconcile";
⋮----
export function SessionReconcileListener()
```

## File: apps/retailer-app-desktop/lib/session-reconcile.ts
```typescript
import { apiFetch } from "./auth";
⋮----
export async function reconcileRetailerSession(): Promise<void>
```

## File: apps/retailer-app-desktop/lib/use-payment-catalog.ts
```typescript
import { packCurrency, readCachedAuthSession, selectablePackPsps } from '@pegasusx/api-core';
import type { PSPListing, RetailerPaymentCatalogResponse } from "@pegasusx/types";
import { useEffect, useState } from "react";
import { apiFetch } from "./auth";
import { retailerCatalogGateways } from "./payment-catalog";
⋮----
export function useRetailerPaymentCatalog()
```

## File: apps/retailer-app-desktop/lib/use-retailer-session-reconcile.ts
```typescript
import { useEffect, useRef } from "react";
⋮----
export function useRetailerSessionReconcile(onReconcile: () => void)
⋮----
const onEvent = () =>
```

## File: apps/retailer-app-desktop/src/components/dock/ShelfAlertModal.tsx
```typescript
import React from 'react';
import { AlertTriangle } from 'lucide-react';
⋮----
interface ShelfAlertModalProps {
  visible: boolean;
  productId: string;
  currentStock: number;
  onDismiss: () => void;
}
⋮----
export const ShelfAlertModal: React.FC<ShelfAlertModalProps> = ({
  visible,
  productId,
  currentStock,
  onDismiss
}) =>
```

## File: apps/retailer-app-desktop/src-tauri/src/commands/mod.rs
```rust
pub mod security;
⋮----
pub fn get_app_info() -> serde_json::Value {
```

## File: apps/retailer-app-desktop/src-tauri/src/commands/security.rs
```rust
use keyring::Entry;
use serde::Serialize;
⋮----
pub struct TokenResult {
pub success: bool,
pub token: Option<String>,
pub error: Option<String>,
⋮----
pub fn store_token(token: String, refresh_token: Option<String>) -> TokenResult {
⋮----
pub fn get_token() -> TokenResult {
⋮----
pub fn clear_token() -> TokenResult {
⋮----
fn read_token(service: &str) -> Result<Option<String>, String> {
⋮----
fn delete_if_present(service: &str, key: &str) -> bool {
```

## File: apps/retailer-app-desktop/src-tauri/src/lib.rs
```rust
mod commands;
⋮----
pub fn run() {
```

## File: apps/retailer-app-desktop/src-tauri/src/main.rs
```rust
fn main() {
```

## File: apps/retailer-app-desktop/src-tauri/Cargo.toml
```toml
[package]
name = "pegasus-retailer-desktop"
version = "0.1.1"
description = "Pegasus Retailer Desktop"
authors = ["Pegasus"]
license = ""
repository = ""
edition = "2021"
rust-version = "1.77.2"

# See more keys and their definitions at https://doc.rust-lang.org/cargo/reference/manifest.html

[lib]
name = "app_lib"
crate-type = ["staticlib", "cdylib", "rlib"]

[build-dependencies]
tauri-build = { version = "2.5.6", features = [] }

[dependencies]
serde_json = "1.0"
serde = { version = "1.0", features = ["derive"] }
keyring = { version = "3", features = ["apple-native", "windows-native"] }
log = "0.4"
tauri = { version = "2.11", features = [] }
tauri-plugin-log = "2"
tauri-plugin-process = "2"
tauri-plugin-sql = { version = "2", features = ["sqlite"] }

[target.'cfg(not(any(target_os = "android", target_os = "ios")))'.dependencies]
tauri-plugin-single-instance = { version = "2", features = ["deep-link"] }
tauri-plugin-deep-link = "2"
tauri-plugin-updater = "2"
```

## File: apps/retailer-app-desktop/clean.tsx
```typescript

```

## File: apps/retailer-app-desktop/middleware.ts
```typescript
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
function isTokenExpired(token: string): boolean
⋮----
export function middleware(request: NextRequest)
```

## File: apps/retailer-app-desktop/vitest.config.ts
```typescript
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";
```

## File: apps/retailer-app-desktop/vitest.setup.ts
```typescript
getItem(key: string)
setItem(key: string, value: string)
removeItem(key: string)
clear()
get length()
key(index: number)
```

## File: apps/supplier-portal/app/(dashboard)/activity/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw, Activity } from "lucide-react";
⋮----
type SupplierActivityEvent = {
  id: string;
  type: string;
  description: string;
  timestamp: string;
};
⋮----
type SupplierActivityResponse = {
  events: SupplierActivityEvent[];
};
⋮----
<PageSection title=
```

## File: apps/supplier-portal/app/(dashboard)/compliance/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import type { Route } from "next";
import { supplierFetch } from "@/lib/auth";
import { sessionSupplierId } from "@/lib/supplier-scope";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
⋮----
interface DashboardStats {
  fiscalizing: number;
  fiscalFailed: number;
  forceCompleted: number;
  buyerAcceptancePending: number;
  buyerAcceptanceRejected: number;
  claimMismatches: number;
  creditFrozen: number;
  openCreditNotes: number;
  openReverseLogisticsTasks: number;
  openCashDiscrepancies: number;
}
⋮----
interface ProblemOrder {
  orderId: string;
  status: string;
  fiscalStatus: string;
  ehfId?: string;
  buyerAcceptanceStatus?: string;
  forceCompletedAt?: string;
  forceReason?: string;
  claimId?: string;
  claimedAmountMinor?: number;
  createdAt: string;
}
⋮----
const handleExport = async () =>
⋮----
onClick=
⋮----
<MetricCard title=
⋮----
title=
```

## File: apps/supplier-portal/app/(dashboard)/ledger/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw, Receipt } from "lucide-react";
⋮----
type PaymentLedgerEntry = {
  id: string;
  entry_type: string;
  amount_minor: number;
  currency: string;
  order_id?: string;
  occurred_at: string;
};
⋮----
type PaymentLedgerResponse = {
  items: PaymentLedgerEntry[];
};
⋮----
<PageSection title=
```

## File: apps/supplier-portal/app/(dashboard)/settings/fx-rates/page.tsx
```typescript
import { useMemo } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw } from "lucide-react";
import type { FxRateRow, FxRatesListResponse } from "@pegasusx/types";
import { sessionPackCurrency } from "@pegasusx/api-core";
⋮----
function formatRate(row: FxRateRow): string
```

## File: apps/supplier-portal/app/(portal)/_components/PortalSurface.tsx
```typescript

```

## File: apps/supplier-portal/app/(portal)/analytics/demand/flywheel/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { FlywheelDemandItem } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/analytics/demand/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useEffect, useMemo, useState } from "react";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import { createSupplierApi } from "@/lib/api";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
import type {
  SupplierDemandAccuracyResponse,
  SupplierDemandHistoryResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';
⋮----
function pctLabel(ratio: number | null | undefined): string
⋮----
description=
⋮----
<KpiStatCard label=
⋮----
label=
⋮----
confidence=
⋮----
stale=
```

## File: apps/supplier-portal/app/(portal)/analytics/knowledge-graph/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierKnowledgeGraph } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/analytics/route-performance/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
⋮----
type RoutePerfRow = {
  route_id: string;
  driver_id?: string;
  planned_stops?: number;
  actual_stops?: number;
  planned_duration_sec?: number;
  actual_duration_sec?: number;
  replan_count?: number;
  computed_at: string;
};
⋮----
export default function RoutePerformancePage()
⋮----
<PageChrome title=
```

## File: apps/supplier-portal/app/(portal)/analytics/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useEffect, useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import Icon from "@/components/Icon";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import type {
  SupplierAnalyticsRevenueResponse,
  SupplierAnalyticsVelocityResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
function formatMoney(minor: number, currency: string)
⋮----
export default function AnalyticsPage()
⋮----
description=
⋮----
<KpiStatCard label=
⋮----
label=
```

## File: apps/supplier-portal/app/(portal)/analytics/RevenueHeatmap.tsx
```typescript
import { useMemo } from 'react';
import MapGL, { Source, Layer, NavigationControl } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
⋮----
import { cellToBoundary } from 'h3-js';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
type RevenueHeatmapProps = {
  className?: string;
  cells?: { h3: string; revenue: number }[];
};
```

## File: apps/supplier-portal/app/(portal)/catalog/[productId]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/app/(portal)/catalog/components/types.ts
```typescript
export type SaleUnit = "UNIT" | "CASE";
⋮----
export type CatalogProduct = {
  product_id: string;
  name: string;
  category_id: string;
  price_minor: number;
  currency: string;
  unit: string;
  unit_volume_vu: number;
  stackable?: boolean;
  max_stack_height?: number;
  units_per_case?: number;
  sale_unit?: SaleUnit;
  barcode?: string;
  image_url?: string;
  is_active: boolean;
  version: number;
};
⋮----
export type CatalogCategory = {
  category_id: string;
  name: string;
};
⋮----
export type CreateProductFormState = {
  name: string;
  category_id: string;
  description: string;
  price_minor: string;
  unit_volume_vu: string;
  units_per_case: string;
  sale_unit: SaleUnit;
  barcode: string;
};
```

## File: apps/supplier-portal/app/(portal)/catalog/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { supplierFetch } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { BulkImportWizard } from "@/components/BulkImportWizard";
import type { CatalogProduct, CatalogCategory, CreateProductFormState } from "./components/types";
import { ALLOWED_IMAGE_TYPES, MAX_IMAGE_SIZE } from "./components/types";
import { CreateProductForm } from "./components/CreateProductForm";
import { sessionPackCurrency } from "@pegasusx/api-core";
import { CatalogTable } from "./components/CatalogTable";
⋮----
export default function CatalogPage()
⋮----
async function uploadCatalogImage(file: File): Promise<string>
⋮----
const handleSaveProductImage = async (product: CatalogProduct, file: File) =>
⋮----
const handleSaveProductEdits = async (product: CatalogProduct, updates: Partial<CatalogProduct>) =>
⋮----
const handleCreateProduct = async (
    form: CreateProductFormState,
    normalizedBarcode: string | undefined,
    unitsPerCase: number | undefined,
    imageFile: File | null
  ) =>
⋮----
description=
⋮----
onClick=
⋮----
onImport={(rows) =>
```

## File: apps/supplier-portal/app/(portal)/crm/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { Fragment, useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-core";
import type { SupplierCRMRetailer, SupplierCRMRetailerDetail } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";
import Icon from "@/components/Icon";
⋮----
function fmtMinor(n: number): string
⋮----
export default function SupplierCRMPage()
⋮----
onClick={() =>
```

## File: apps/supplier-portal/app/(portal)/dashboard/use-dashboard-data.ts
```typescript
import { usePolling } from '@pegasusx/api-react';
import { useCallback, useEffect, useRef, useState } from "react";
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import type { OrderStatus } from "@pegasusx/types";
import type { ManifestState, SupplierDashboardResponse } from "@pegasusx/types";
import {
  canonicalizeOrderStatus,
  emptyManifestStateCounts,
  emptyOrderStatusCounts,
  incrementOrderStatusCount,
} from "@pegasusx/types";
import {
  DASHBOARD_ROLLUP_REFRESH_EVENTS,
  shouldRefetchDashboardRollup,
} from "@pegasusx/ws-refresh-contract";
import { createSupplierApi } from "@/lib/api";
import { orderStatusFromWsRaw } from "@/lib/dashboard-command";
import { supplierDashboardCacheKey } from "@/lib/supplier-cache-keys";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
⋮----
export interface DashboardMetrics {
  ordersByStatus: Partial<Record<OrderStatus, number>>;
  manifestsByState: Record<ManifestState, number>;
  revenueToday: number;
  completedToday: number;
  attemptedToday: number;
  activeDrivers: number;
  totalDrivers: number;
  deliveryCompletionRate: number;
  retailersOrderedToday: number;
  totalRetailers: number;
  fleetVuUsed: number;
  fleetVuTotal: number;
  fleetVuAvailable: boolean;
  asOf: string | null;
}
⋮----
export interface DispatchManifest {
  id: string;
  status: "DRAFT" | "LOADING" | "DISPATCHED";
  ordersCount: number;
  driverName: string;
}
⋮----
export interface WsEventLog {
  id: string;
  type: string;
  timestamp: string;
  description: string;
}
⋮----
export interface DashboardData {
  metrics: DashboardMetrics;
  recentManifests: DispatchManifest[];
  recentEvents: WsEventLog[];
}
⋮----
type DashboardCacheBundle = {
  data: DashboardData;
  isPaymentConfigured: boolean | null;
};
⋮----
function mapDashboard(resp: SupplierDashboardResponse): DashboardData
⋮----
export function useDashboardData()
```

## File: apps/supplier-portal/app/(portal)/dashboard/use-dashboard-history.ts
```typescript
import { useCallback, useEffect, useState } from "react";
import { ApiError } from '@pegasusx/api-core';
import type { DashboardHistoryRange, HistorySeries } from "@pegasusx/types";
import {
  historySeriesFromValues,
  sliceDatedSeries,
  yesterdayRevenueDeltaPct,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
⋮----
export type DashboardHistory = {
  range: DashboardHistoryRange;
  setRange: (range: DashboardHistoryRange) => void;
  velocity: HistorySeries;
  revenue: HistorySeries;
  revenueDeltaPct: number | null;
  loading: boolean;
};
⋮----
export function useDashboardHistory(): DashboardHistory
```

## File: apps/supplier-portal/app/(portal)/delivery-zones/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { DeliveryZonesList } from "@/components/delivery-zones";
⋮----
export default function DeliveryZonesPage()
⋮----
title=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/dispatch/use-dispatch-data.ts
```typescript
import { usePolling } from '@pegasusx/api-react';
import { useCallback, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { SUPPLIER_DISPATCH_REFRESH_EVENTS } from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
⋮----
export interface ManifestData {
  id: string;
  status: "DRAFT" | "LOADING" | "DISPATCHED";
  driverName: string;
  vehiclePlate: string;
  orderCount: number;
  totalVu: number;
  estimatedTime: string;
}
⋮----
function formatEta(status: string, updatedAt: string): string
⋮----
export function useDispatchData()
```

## File: apps/supplier-portal/app/(portal)/exceptions/claims/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import type {
  Claim,
  ClaimSettlementMode,
  ClaimSettlementResult,
} from "@pegasusx/types";
import { CLAIM_SETTLEMENT_MODES } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";
import { moneyCurrency } from "@pegasusx/api-core";
⋮----
async function approve(claimId: string)
⋮----
async function reject(claimId: string)
⋮----
title=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/exceptions/negotiations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { PageChrome } from "@/components/PageChrome";
⋮----
export default function NegotiationsExceptionsPage()
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/exceptions/shop-closed/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierShopClosedResolveKey } from "@pegasusx/api-core";
import type { ShopClosedAttemptRow } from "@pegasusx/types";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from '@/components/PageChrome';
⋮----
const resolve = async (attemptId: string, action: "WAIT" | "BYPASS" | "RETURN_TO_DEPOT") =>
⋮----
description=
⋮----
emptyMessage=
⋮----
onClick=
```

## File: apps/supplier-portal/app/(portal)/exceptions/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierExceptionRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { ExceptionsList } from "@/components/exceptions/ExceptionsList";
import { PlaybookRunsPanel } from "@/components/exceptions/PlaybookRunsPanel";
⋮----
title=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/factories/components/FactoryList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { SupplierTopologyFactory } from "@pegasusx/types";
⋮----
interface FactoryListProps {
  factories: SupplierTopologyFactory[];
  onAddFirst: () => void;
}
```

## File: apps/supplier-portal/app/(portal)/factories/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyFactory, SupplierTopologyUpdateRequest, SupplierTopologyWarehouse } from "@pegasusx/types";
import { factoryToTopologyInput, warehouseToTopologyInput } from "@/lib/topology";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { FactoryForm } from "./components/FactoryForm";
import { FactoryList } from "./components/FactoryList";
⋮----
const load = () =>
⋮----
const addFactory = async (name: string, location: LocationValue, extras: { country_code: string }) =>
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/fleet/orders/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierFleetOrderRow } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow } from "@/components/portal";
import { usePagination } from "@/lib/use-pagination";
⋮----
title=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/geo-report/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { GeoReportLanesList } from "@/components/geo-report";
⋮----
title=
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/inventory/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import { sessionPackCurrency, supplierInventoryAdjustKey } from "@pegasusx/api-core";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { InventoryTable, type InventoryRow } from "@/components/inventory";
⋮----
type InventoryApiItem = {
  product_id: string;
  warehouse_id: string;
  quantity_on_hand?: number;
  quantity_reserved?: number;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
};
⋮----
function mapInventoryRow(item: InventoryApiItem): InventoryRow
⋮----
export default function PortalInventoryPage()
⋮----
const loadInventory = () =>
⋮----
const adjustRow = async (row: InventoryRow) =>
⋮----
const updatePolicy = async (row: InventoryRow, policy: string) =>
⋮----
description=
⋮----
onApply={(row) =>
⋮----
onPolicyChange={(row, policy) =>
```

## File: apps/supplier-portal/app/(portal)/manifests/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/app/(portal)/manifests/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierManifestRow } from "@pegasusx/types";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { ManifestsTable } from "@/components/manifests";
⋮----
export default function ManifestsPage()
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/operations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { ApiError, supplierBroadcastKey, supplierPaymentBypassKey } from "@pegasusx/api-core";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { OperatorBroadcast, broadcastRoles, ReplenishmentAction, PaymentBypass } from "@/components/operations";
import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import ReplenishmentTraceabilityPanel from "@/components/ReplenishmentTraceabilityPanel";
import { errorToMessage } from "../../payments/_shared/finance";
⋮----
function supplierScopeId(): string
⋮----
const onBroadcast = async () =>
⋮----
const onReplenishment = async () =>
⋮----
const onPaymentBypass = async () =>
⋮----
description=
⋮----
<KpiStatCard label=
⋮----
```

## File: apps/supplier-portal/app/(portal)/orders/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/app/(portal)/orders/page.tsx
```typescript
import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ApiError } from '@pegasusx/api-core';
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from '@pegasusx/desktop-cache';
import { isTauri } from '@pegasusx/desktop-bridge';
import type { SupplierOrder } from '@pegasusx/types';
import { ORDER_STATUS_FUNNEL, canonicalizeOrderStatus } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import { supplierOrdersCacheKey } from '@/lib/supplier-cache-keys';
import { downloadCsv } from '@/lib/csv';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';
import { ListToolbar } from '@/components/ListToolbar';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import { usePortalT } from '@/lib/i18n';
import { OrdersList, type OrderFilter } from './components/OrdersList';
⋮----
export default function OrdersPage()
⋮----
const exportCsv = async () =>
⋮----
description=
⋮----
onPrev=
⋮----
onRefresh={async () =>
```

## File: apps/supplier-portal/app/(portal)/pricing/[productId]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/app/(portal)/pricing/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import type { CatalogProduct } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow } from "@/components/portal";
⋮----
function formatPrice(product: CatalogProduct)
⋮----
title=
```

## File: apps/supplier-portal/app/(portal)/profile/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierProfileUpdateKey } from "@pegasusx/api-core";
import type { SupplierProfile, SupplierProfileUpdateRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
import { SupplierIdentityCard, ContactDetailsForm, draftFromProfile, type ProfileDraft } from "@/components/profile";
⋮----
async function saveProfile()
⋮----
description=
⋮----
onReset={() =>
```

## File: apps/supplier-portal/app/(portal)/reconciliation/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { formatMinor, loadFinanceAuthoritySnapshot } from "@/app/payments/_shared/finance";
⋮----
export default function ReconciliationPage()
⋮----
description=
⋮----
label=
```

## File: apps/supplier-portal/app/(portal)/settings/planning/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierSeasonalOverrideCreateKey } from "@pegasusx/api-core/idempotency";
import type { SeasonalOverrideInput, SeasonalTemplatesResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import SignalIngestOpsPanel from "@/components/SignalIngestOpsPanel";
import FactoryPlanningOpsPanel from "@/components/settings/planning/FactoryPlanningOpsPanel";
import { CreateOverrideForm, SeasonalOverridesTable, ForecastAccuracyPanel, type OverrideForm } from "@/components/settings/planning";
⋮----
async function createOverride()
⋮----
description=
⋮----
<h2 className="bento-card-title">
```

## File: apps/supplier-portal/app/(portal)/settings/playbooks/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import type { ControlTowerPlaybook } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
⋮----
const deactivate = async (id: string) =>
⋮----
title=
```

## File: apps/supplier-portal/app/(portal)/settings/segmentation/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { PageChrome } from "@/components/PageChrome";
import { SegmentationPanel } from "@/components/settings/SegmentationPanel";
⋮----
export default function SegmentationSettingsPage()
⋮----
title=
```

## File: apps/supplier-portal/app/(portal)/supply-lanes/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useEffect, useState } from "react";
import Link from "next/link";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { SupplyLanesList } from "@/components/supply-lanes";
⋮----
export default function SupplyLanesPage()
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/topology/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { TopologyEditor } from "@/components/TopologyEditor";
import { PageChrome } from "@/components/PageChrome";
⋮----
export default function TopologyPage()
⋮----
const loadTopology = () =>
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/warehouses/components/WarehouseList.tsx
```typescript
import Link from "next/link";
import { usePortalT } from "@/lib/i18n";
import { coverageModeLabel } from "@/lib/coverage";
import type { SupplierTopologyWarehouse } from "@pegasusx/types";
⋮----
interface WarehouseListProps {
  warehouses: SupplierTopologyWarehouse[];
  modes?: Record<string, string>;
  onAddFirst: () => void;
}
⋮----
{(w.address || "Coordinates on file").toString()} · {coverageModeLabel(modes[w.warehouse_id])} ·
```

## File: apps/supplier-portal/app/(portal)/warehouses/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyCoverageCity, SupplierTopologyResponse, SupplierTopologyUpdateRequest } from "@pegasusx/types";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { WarehouseForm } from "./components/WarehouseForm";
import { WarehouseList } from "./components/WarehouseList";
import { factoryToTopologyInput, warehouseToTopologyInput } from "@/lib/topology";
⋮----
const load = () =>
⋮----
const addWarehouse = async (
    name: string,
    location: LocationValue,
    radius: number,
    extras: { country_code: string; coverage_cities: SupplierTopologyCoverageCity[]; primary_factory_id: string },
  ) =>
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/error.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
```

## File: apps/supplier-portal/app/(portal)/layout.tsx
```typescript
import AuthGuard from "@/components/AuthGuard";
import { ToastProvider } from "@/components/Toast";
⋮----
export default function PortalLayout({ children }: { children: React.ReactNode })
```

## File: apps/supplier-portal/app/(portal)/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";
⋮----
export default function PortalIndex()
```

## File: apps/supplier-portal/app/ai/recommendations/layout.tsx
```typescript

```

## File: apps/supplier-portal/app/ai/recommendations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierAIRecommendation, SupplierAIRecommendationDecision } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { FormAlert } from "@/components/portal";
import { useEffect, useMemo, useState } from "react";
⋮----
type StatusFilter = "ALL" | "PENDING" | "ACKNOWLEDGED" | "OVERRIDDEN" | "DISMISSED";
⋮----
function formatDateTime(value?: string)
⋮----
function formatPercent(value: number)
⋮----
function errorToMessage(error: unknown)
⋮----
function isRestricted(error: unknown)
⋮----
export default function SupplierAIRecommendationsPage()
⋮----
const updateOfflineState = () =>
⋮----
async function loadRecommendations()
⋮----
async function recordDecision(recommendation: SupplierAIRecommendation, decision: SupplierAIRecommendationDecision)
⋮----
onChange=
```

## File: apps/supplier-portal/app/auth/oidc/callback/page.tsx
```typescript
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { exchangeOIDC } from "@/lib/oidc";
⋮----
export default function OIDCCallbackPage()
```

## File: apps/supplier-portal/app/auth/register/wizard-state.ts
```typescript
export type StepId = "identity" | "verification" | "profile";
⋮----
export interface Country {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export interface IdentityStep {
  countryCode: string;
  phoneLocal: string;
}
⋮----
export interface VerificationStep {
  otpCode: string;
  idToken: string;
}
⋮----
export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
}
⋮----
export interface WizardState {
  step: StepId;
  identity: IdentityStep;
  verification: VerificationStep;
  profile: ProfileStep;
}
⋮----
export function validateIdentity(s: IdentityStep): Record<string, string>
⋮----
export function validateVerification(s: VerificationStep): Record<string, string>
⋮----
export function validateProfile(s: ProfileStep): Record<string, string>
```

## File: apps/supplier-portal/app/auth/layout.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';
```

## File: apps/supplier-portal/app/earnings/layout.tsx
```typescript

```

## File: apps/supplier-portal/app/org-fleet/components/DriverTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { describeHomeNode, describeVehicle, ReadyState } from "./utils";
```

## File: apps/supplier-portal/app/org-fleet/components/MetricsOverview.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
export function MetricCard(props: { label: string; value: number })
⋮----
export function MetricsOverview({
  warehouses,
  factories,
  orgMembers,
  fleetEntities,
}: {
  warehouses: number;
  factories: number;
  orgMembers: number;
  fleetEntities: number;
})
⋮----
<MetricCard label=
```

## File: apps/supplier-portal/app/org-fleet/components/OrgMemberTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { ReadyState, orgRoleOptions, formatRole, describeMemberNode, toErrorMessage, isErrorMessage, supplierScopeId } from "./utils";
import type { Role, HomeNodeType } from "@pegasusx/types";
import { ApiClient } from "@pegasusx/api-core";
import { supplierOrgMemberDeactivateKey, supplierOrgMemberUpdateKey } from "@pegasusx/api-core";
import { StatusText } from "./utils";
⋮----
async function deactivateOrgMember(userId: string)
⋮----
async function saveOrgMemberEdit(userId: string, role: Role, nodeType: HomeNodeType, nodeID: string)
⋮----
onClick=
```

## File: apps/supplier-portal/app/org-fleet/components/utils.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import type {
  HomeNodeType,
  Role,
  SupplierOrgMemberCreateRequest,
  SupplierTopologyResponse,
} from "@pegasusx/types";
⋮----
export function supplierScopeId(): string
⋮----
export type ReadyState = {
  status: "ready";
  topology: SupplierTopologyResponse;
  orgMembers: Awaited<ReturnType<ApiClient["getSupplierOrgMembers"]>>["items"];
  drivers: Awaited<ReturnType<ApiClient["getSupplierFleetDrivers"]>>["items"];
  vehicles: Awaited<ReturnType<ApiClient["getSupplierFleetVehicles"]>>["items"];
};
⋮----
export type OrgFormState = {
  name: string;
  email: string;
  phone: string;
  password: string;
  role: Role;
  nodeType: HomeNodeType;
  nodeID: string;
};
⋮----
export type DriverFormState = {
  name: string;
  phone: string;
  pin: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
  vehicleID: string;
};
⋮----
export type VehicleFormState = {
  label: string;
  licensePlate: string;
  homeNodeType: HomeNodeType;
  homeNodeID: string;
};
⋮----
export function StatusText(props: { message: string; isError: boolean })
⋮----
export function buildOrgMemberRequest(form: OrgFormState): SupplierOrgMemberCreateRequest
⋮----
export function orgEffectiveNodeType(form: OrgFormState): HomeNodeType
⋮----
export function nodeOptionsFor(nodeType: HomeNodeType, topology: SupplierTopologyResponse)
⋮----
export function describeMemberNode(
  member: ReadyState["orgMembers"][number],
  topology: SupplierTopologyResponse,
)
⋮----
export function describeHomeNode(nodeType: HomeNodeType, nodeID: string, topology: SupplierTopologyResponse)
⋮----
export function describeVehicle(vehicleID: string | undefined, vehicles: ReadyState["vehicles"])
⋮----
export function formatRole(role: Role)
⋮----
export function toErrorMessage(error: unknown)
⋮----
export function isErrorMessage(message: string)
⋮----
export function newIdempotencyKey()
```

## File: apps/supplier-portal/app/org-fleet/components/VehicleTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { describeHomeNode, ReadyState } from "./utils";
```

## File: apps/supplier-portal/app/org-fleet/layout.tsx
```typescript

```

## File: apps/supplier-portal/app/org-fleet/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReadyState, toErrorMessage } from "./components/utils";
import { MetricsOverview } from "./components/MetricsOverview";
import { OrgMemberForm } from "./components/OrgMemberForm";
import { OrgMemberTable } from "./components/OrgMemberTable";
import { DriverForm } from "./components/DriverForm";
import { DriverTable } from "./components/DriverTable";
import { VehicleForm } from "./components/VehicleForm";
import { VehicleTable } from "./components/VehicleTable";
⋮----
type LoadState =
  | { status: "loading" }
  | ReadyState
  | { status: "error"; message: string };
```

## File: apps/supplier-portal/app/payments/_shared/finance.ts
```typescript
import { ApiClient, ApiError } from '@pegasusx/api-core';
import { supplierApiBaseUrl, supplierFetch } from "@/lib/auth";
import type {
  PaymentLedgerResponse,
  ReconciliationMismatchRow,
  SettlementAuthorityResponse,
  SettlementAuthorityRow,
  SettlementCurrencyTotal,
} from "@pegasusx/types";
import { useEffect, useRef, useState } from "react";
⋮----
export type DataSource = "settlement_authority" | "ledger_fallback";
⋮----
export interface FinanceAuthoritySnapshot {
  source: DataSource;
  authority: SettlementAuthorityResponse;
  ledger: PaymentLedgerResponse;
  mismatches: ReconciliationMismatchRow[];
  refreshedAt: string;
}
⋮----
export interface SupplierFinanceLiveState {
  status: "connecting" | "live" | "degraded";
  message: string;
  attempts: number;
  lastEventType?: string;
  lastEventAt?: string;
}
⋮----
interface SupplierWebSocketSessionResponse {
  token: string;
  expires_at: string;
  websocket_url: string;
}
⋮----
export async function loadFinanceAuthoritySnapshot(api: ApiClient): Promise<FinanceAuthoritySnapshot>
⋮----
async function loadReconciliationMismatches(api: ApiClient): Promise<ReconciliationMismatchRow[]>
⋮----
export function useSupplierFinanceLiveRefresh(onSignal: (eventType: string) => void): SupplierFinanceLiveState
⋮----
const clearTimers = () =>
⋮----
const closeStream = () =>
⋮----
const handleEvent = (eventType: string | null) =>
⋮----
function extractSessionError(payload: SupplierWebSocketSessionResponse | { error?: string } | null): string
⋮----
function parseEventType(raw: unknown): string | null
⋮----
export function buildSummaryFromLedger(response: PaymentLedgerResponse): SettlementAuthorityResponse
⋮----
export function parseDate(value: string): number
⋮----
export function formatDateTime(value: string): string
⋮----
export function formatMinor(amountMinor: number, currency: string): string
⋮----
export function createIdempotencyKey(prefix: string): string
⋮----
export function errorToMessage(error: unknown): string
```

## File: apps/supplier-portal/app/payments/layout.tsx
```typescript

```

## File: apps/supplier-portal/app/payments/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ApiClient } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useEffect, useMemo, useState } from "react";
import type { FinanceAuthoritySnapshot } from "./_shared/finance";
import {
  errorToMessage,
  formatDateTime,
  formatMinor,
  loadFinanceAuthoritySnapshot,
  useSupplierFinanceLiveRefresh,
} from "./_shared/finance";
import { PageChrome } from "@/components/PageChrome";
import { FormAlert, PortalSection } from "@/components/portal";
⋮----
type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };
⋮----
async function load()
⋮----
description=
```

## File: apps/supplier-portal/app/setup/billing/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import {
  SelectionOption,
  SetupCallout,
  SetupField,
  SetupFooter,
  SetupInput,
  SetupPageHeader,
  SetupSection,
} from "@/components/setup/SetupPrimitives";
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";
import { gatewayLabel } from "@/lib/coverage";
⋮----
type GatewayId = string;
⋮----
type BillingState = {
  bankName: string;
  accountHolder: string;
  accountNumber: string;
  swiftBic: string;
  iban: string;
  selectedGateways: GatewayId[];
  paymentAcceptor: "SUPPLIER" | "WAREHOUSE";
};
⋮----
function validate(): Record<string, string>
⋮----
function toggleGateway(id: GatewayId)
⋮----
async function submit()
⋮----
function skip()
⋮----
function goBack()
⋮----
subtitle=
⋮----
description=
⋮----
title=
⋮----
toggleGateway(gateway.code);
⋮----
const bytes = crypto.getRandomValues(new Uint8Array(16));
```

## File: apps/supplier-portal/app/setup/business/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierBusinessSetupKey } from "@pegasusx/api-core";
import {
  SetupCallout,
  SetupField,
  SetupFooter,
  SetupInput,
  SetupPageHeader,
  SetupSection,
} from "@/components/setup/SetupPrimitives";
⋮----
interface BusinessState {
  taxId: string;
  registrationNumber: string;
  headquartersAddress: string;
  city: string;
  postalCode: string;
}
⋮----
function validate(): Record<string, string>
⋮----
async function submit()
⋮----
function skip()
⋮----
subtitle=
⋮----
description=
⋮----
onChange=
```

## File: apps/supplier-portal/app/setup/layout.tsx
```typescript
import SetupWizardShell from "@/components/setup/SetupWizardShell";
⋮----
export default function SetupLayout({ children }: { children: React.ReactNode })
```

## File: apps/supplier-portal/app/tracking/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { DispatchSidebarNav } from "../../components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../components/dispatch/CargoPhotoCarousel";
```

## File: apps/supplier-portal/app/error.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
```

## File: apps/supplier-portal/app/layout.tsx
```typescript
import type { Metadata } from "next";
import { Plus_Jakarta_Sans, EB_Garamond } from "next/font/google";
import Providers from "@/components/Providers";
import LocaleBootstrap from "@/components/LocaleBootstrap";
```

## File: apps/supplier-portal/app/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { redirect } from "next/navigation";
⋮----
export default function HomePage()
```

## File: apps/supplier-portal/components/compliance/MetricCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { type ReactNode } from "react";
import { Info } from "lucide-react";
⋮----
interface MetricCardProps {
  title: string;
  value: string | number;
  icon: ReactNode;
  trend?: {
    direction: "up" | "down" | "neutral";
    value: string;
  };
  tooltipText?: string;
  isAlert?: boolean;
}
⋮----
? "text-[var(--desk-danger)]" // Often in compliance, up is bad (more mismatches)
```

## File: apps/supplier-portal/components/control-tower/ScoredExceptionsPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import type { ScoredException } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
⋮----
return <p className="text-sm text-gray-400">
```

## File: apps/supplier-portal/components/delivery-zones/DeliveryZonesList.tsx
```typescript
import React from 'react';
import type { SupplierTopologyWarehouse } from '@pegasusx/types';
import { DataList, DataListRow } from "@/components/portal";
⋮----
interface DeliveryZonesListProps {
  warehouses: SupplierTopologyWarehouse[];
}
⋮----
export function DeliveryZonesList({ warehouses }: DeliveryZonesListProps)
```

## File: apps/supplier-portal/components/delivery-zones/index.ts
```typescript

```

## File: apps/supplier-portal/components/dispatch/CargoPhotoCarousel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import { PoDPhotoReport } from "@pegasusx/types";
⋮----
interface CargoPhotoCarouselProps {
  photos?: PoDPhotoReport[];
  onRequestPhoto?: () => void;
}
```

## File: apps/supplier-portal/components/dispatch/RouteTelemetryMap.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
⋮----
interface RouteTelemetryMapProps {
  vehicleCode?: string | null;
  etaSeconds?: number | null;
  distanceMilesLeft?: number | null;
  onChangeRoute?: () => void;
  hasLiveRoute?: boolean;
}
⋮----
const formatTime = (seconds: number) =>
```

## File: apps/supplier-portal/components/dispatch/ShipmentCardGrid.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import { VehicleShipmentCard } from "@pegasusx/types";
import { ShipmentCard } from "./ShipmentCard";
⋮----
interface ShipmentCardGridProps {
  shipments: VehicleShipmentCard[];
  selectedShipmentId?: string;
  onSelectShipment: (shipment: VehicleShipmentCard) => void;
}
⋮----
onSelect=
```

## File: apps/supplier-portal/components/dispatch/VehicleCapacityGauge.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
⋮----
interface VehicleCapacityGaugeProps {
  capacityPercentage?: number | null;
  vehicleCode?: string | null;
}
```

## File: apps/supplier-portal/components/exceptions/ExceptionsList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState } from "react";
import type { SupplierExceptionRow } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";
⋮----
interface ExceptionsListProps {
  exceptions: SupplierExceptionRow[];
  onResolved?: () => void;
}
⋮----
const resolve = async (row: SupplierExceptionRow) =>
⋮----
const canResolve = (kind: string) =>
```

## File: apps/supplier-portal/components/exceptions/PlaybookRunsPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import type { ControlTowerPlaybookRun } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
⋮----
const approve = async (runId: string) =>
⋮----
const skip = async (runId: string) =>
⋮----
onClick=
```

## File: apps/supplier-portal/components/geo-report/GeoReportLanesList.tsx
```typescript
import React from 'react';
import type { SupplierSupplyLaneRow } from '@pegasusx/types';
import { DataList, DataListRow } from "@/components/portal";
⋮----
interface GeoReportLanesListProps {
  lanes: SupplierSupplyLaneRow[];
}
⋮----
export function GeoReportLanesList({ lanes }: GeoReportLanesListProps)
```

## File: apps/supplier-portal/components/geo-report/index.ts
```typescript

```

## File: apps/supplier-portal/components/inventory/index.ts
```typescript

```

## File: apps/supplier-portal/components/manifests/index.ts
```typescript

```

## File: apps/supplier-portal/components/manifests/ManifestsTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import type { SupplierManifestRow } from '@pegasusx/types';
import StatusBadge from '@/components/StatusBadge';
⋮----
interface ManifestsTableProps {
  items: SupplierManifestRow[];
}
```

## File: apps/supplier-portal/components/operations/index.ts
```typescript

```

## File: apps/supplier-portal/components/operations/ReplenishmentAction.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import { PageSection } from '@/components/PageSection';
⋮----
interface ReplenishmentActionProps {
  replenishing: boolean;
  onReplenishment: () => void;
}
⋮----
export function ReplenishmentAction({ replenishing, onReplenishment }: ReplenishmentActionProps)
⋮----
title=
```

## File: apps/supplier-portal/components/orders/index.ts
```typescript

```

## File: apps/supplier-portal/components/orders/OrderActionDialog.tsx
```typescript
import { useEffect, useRef } from 'react';
⋮----
type OrderActionDialogProps = {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel: string;
  destructive?: boolean;
  reason: string;
  onReasonChange: (value: string) => void;
  reasonRequired?: boolean;
  submitting?: boolean;
  onConfirm: () => void;
  onClose: () => void;
};
⋮----
export function OrderActionDialog({
  open,
  title,
  description,
  confirmLabel,
  destructive = false,
  reason,
  onReasonChange,
  reasonRequired = false,
  submitting = false,
  onConfirm,
  onClose,
}: OrderActionDialogProps)
```

## File: apps/supplier-portal/components/orders/OrderKebabMenu.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useRef, useState } from 'react';
import Icon from '@/components/Icon';
⋮----
type OrderKebabMenuProps = {
  onViewDetails: () => void;
  onDelay?: () => void;
  onReject?: () => void;
  onReassign?: () => void;
  canDelay?: boolean;
  canReject?: boolean;
  canReassign?: boolean;
  delayLabel?: string;
  rejectLabel?: string;
  reassignLabel?: string;
  disabled?: boolean;
};
⋮----
const onDoc = (e: MouseEvent) =>
⋮----
e.stopPropagation();
e.preventDefault();
setOpen((v)
```

## File: apps/supplier-portal/components/orders/OrderLineItems.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { WarehouseOrderDetail } from '@pegasusx/types';
⋮----
interface OrderLineItemsProps {
  detail: WarehouseOrderDetail | null;
}
```

## File: apps/supplier-portal/components/orders/OrderStateChip.tsx
```typescript
import { ORDER_STATE_CHIP } from '@/lib/order-actions';
⋮----
export function OrderStateChip({ state }: { state: string })
```

## File: apps/supplier-portal/components/orders/ReDispatchDialog.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useEffect, useState, useCallback } from 'react';
import { ApiError } from '@pegasusx/api-core';
import { payloadApplyReassignKey, payloadRecommendReassignKey } from '@pegasusx/api-core/idempotency';
import type { ReassignmentCandidate } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
⋮----
export type ReDispatchDialogProps = {
  open: boolean;
  orderId: string;
  onClose: () => void;
  onSuccess: () => void;
};
⋮----
const handleReassign = async (newDriverId: string) =>
⋮----
body=
```

## File: apps/supplier-portal/components/portal/index.ts
```typescript

```

## File: apps/supplier-portal/components/profile/index.ts
```typescript

```

## File: apps/supplier-portal/components/profile/SupplierIdentityCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SupplierProfile } from '@pegasusx/types';
⋮----
export function ReadOnlyField({ label, value }: { label: string; value: string | undefined })
⋮----
interface SupplierIdentityCardProps {
  profile: SupplierProfile;
}
⋮----
export function SupplierIdentityCard({ profile }: SupplierIdentityCardProps)
⋮----
<ReadOnlyField label=
```

## File: apps/supplier-portal/components/settings/planning/index.ts
```typescript

```

## File: apps/supplier-portal/components/settings/planning/SeasonalOverridesTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SeasonalOverrideRow } from '@pegasusx/types';
⋮----
interface SeasonalOverridesTableProps {
  overrides: SeasonalOverrideRow[];
}
```

## File: apps/supplier-portal/components/settings/SegmentationPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import type { RetailerSegmentRow, SkuClassRow } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
⋮----
const bootstrap = async () =>
⋮----
const updateRetailer = async (retailerId: string, segment: string) =>
⋮----
const updateSku = async (sku: string, velocityClass: string) =>
```

## File: apps/supplier-portal/components/setup/constants.ts
```typescript
export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];
⋮----
export function setupStepIndex(pathname: string): number
```

## File: apps/supplier-portal/components/setup/SetupPrimitives.tsx
```typescript
import Icon from "@/components/Icon";
import { PortalField, PortalInput, FormAlert as PortalFormAlert } from "@/components/portal";
import type { InputHTMLAttributes, ReactNode } from "react";
⋮----
export function SetupPageHeader({
  icon,
  title,
  subtitle,
}: {
  icon: string;
  title: string;
  subtitle: string;
})
⋮----
export function SetupSection({
  icon,
  title,
  description,
  children,
}: {
  icon: string;
  title: string;
  description?: string;
  children: ReactNode;
})
⋮----
export function SetupField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
export function SetupCallout({
  variant = "info",
  children,
}: {
  variant?: "info" | "error";
  children: ReactNode;
})
```

## File: apps/supplier-portal/components/setup/SetupWizardShell.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useTheme } from "@/components/ThemeProvider";
import Icon from "@/components/Icon";
import { SETUP_STEPS, setupStepIndex } from "./constants";
⋮----
function SetupThemeToggle()
```

## File: apps/supplier-portal/components/supply-lanes/index.ts
```typescript

```

## File: apps/supplier-portal/components/supply-lanes/SupplyLanesList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import type { SupplierSupplyLaneRow } from '@pegasusx/types';
⋮----
interface SupplyLanesListProps {
  lanes: SupplierSupplyLaneRow[];
}
```

## File: apps/supplier-portal/components/AuthGuard.tsx
```typescript
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  clearSession,
  decodeJwtPayload,
  readIsConfigured,
  readTokenFromCookie,
  resolveSupplierToken,
} from "@/lib/auth";
```

## File: apps/supplier-portal/components/BentoGrid.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type BentoSize = "stat" | "anchor" | "list" | "control" | "wide" | "full";
⋮----
interface BentoGridProps {
  children: ReactNode;
  className?: string;
  theme?: "brutalist" | "apple";
}
⋮----
export function BentoGrid({
  children,
  className = "",
  theme = "apple",
}: BentoGridProps)
⋮----
interface BentoCardProps {
  children: ReactNode;
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}
⋮----
export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoCardProps)
⋮----
interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}
⋮----
export function BentoSkeleton({
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoSkeletonProps)
```

## File: apps/supplier-portal/components/ClientPolicyBanner.tsx
```typescript
import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { desktopClientPolicyContext } from "@pegasusx/desktop-bridge";
import { getClientPolicy } from "@/lib/client-policy";
⋮----
type PolicyState = {
  forceUpdate: boolean;
  outdated: boolean;
  minimumVersion: string;
  updateUrl?: string;
  deferReason?: string;
};
⋮----
// Policy endpoint optional on local/dev stacks.
```

## File: apps/supplier-portal/components/ControlTowerCommandPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useState } from "react";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import { supplierFetch } from "@/lib/auth";
import { sessionMapCenter } from "@pegasusx/api-core";
import { validateAndSimplifyPolygon } from "@pegasusx/validation";
⋮----
function packZonePolygon()
⋮----
```

## File: apps/supplier-portal/components/EmptyState.tsx
```typescript
import type { ReactNode } from "react";
import Icon from "./Icon";
⋮----
type EmptyStateProps = {
  icon?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  children?: ReactNode;
};
⋮----
export default function EmptyState({
  icon = "inventory",
  headline,
  body,
  action,
  onAction,
  children,
}: EmptyStateProps)
⋮----
```

## File: apps/supplier-portal/components/ForecastConfidenceCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ForecastConfidence } from "@pegasusx/types";
import { formatBaselineSourceLabel } from "@/lib/forecast-confidence";
⋮----
type Props = {
  confidence: ForecastConfidence;
  updatedAt?: string;
  stale?: boolean;
};
⋮----
function confidenceColor(pct?: number): string
```

## File: apps/supplier-portal/components/HandoffInboxCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { HandoffCardMetadata } from "@pegasusx/types";
import Link from "next/link";
```

## File: apps/supplier-portal/components/KpiStatCard.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  className?: string;
  align?: "start" | "center";
};
⋮----
export function KpiStatCard({ label, value, sub, className = "", align = "start" }: KpiStatCardProps)
⋮----
⋮----
export function KpiStatGrid({ children, columns = 3 }: { children: ReactNode; columns?: 2 | 3 | 4 })
```

## File: apps/supplier-portal/components/LanguageSwitcher.tsx
```typescript
import {
  supportedLocales,
  syncDocumentLocale,
  type Locale,
} from "@pegasusx/i18n";
import { usePortalLocale, usePortalT } from "@/lib/i18n";
```

## File: apps/supplier-portal/components/ListToolbar.tsx
```typescript
type ListToolbarProps = {
  page: number;
  pageCount: number;
  totalLabel: string;
  onPrev: () => void;
  onNext: () => void;
  onExport?: () => void;
  exportLabel?: string;
  exportDisabled?: boolean;
};
⋮----
export function ListToolbar({
  page,
  pageCount,
  totalLabel,
  onPrev,
  onNext,
  onExport,
  exportLabel = "Export CSV",
  exportDisabled = false,
}: ListToolbarProps)
```

## File: apps/supplier-portal/components/LocaleBootstrap.tsx
```typescript
import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasusx/i18n/browser";
⋮----
export default function LocaleBootstrap(): null
```

## File: apps/supplier-portal/components/MeioNetworkPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
⋮----
type MEIOSummary = {
  warehouses_scanned: number;
  skus_analyzed: number;
  insights_generated: number;
  transfer_recommendations: number;
  warehouse_balances: Array<{
    warehouse_id: string;
    critical_skus: number;
    warning_skus: number;
    avg_days_cover: number;
    target_stock?: number;
    on_hand_stock?: number;
  }>;
};
⋮----
<Stat label=
⋮----
updatedAt={formatForecastUpdatedAt(demandGeneratedAt)}
              stale=
```

## File: apps/supplier-portal/components/OrderTimelinePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
⋮----
function formatWhen(iso: string): string
⋮----
{formatWhen(entry.created_at)}
```

## File: apps/supplier-portal/components/PageSection.tsx
```typescript
import type { ReactNode } from "react";
import { PortalSection } from "@/components/portal";
⋮----
type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  icon?: string;
};
⋮----
export function PageSection({ title, description, actions, children, className = "", icon }: PageSectionProps)
```

## File: apps/supplier-portal/components/PlanningOutcomesPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
import type {
  SupplierDemandSummaryResponse,
  SupplierMEIONetworkSummary,
  SupplierReplenishmentPolicy,
} from "@pegasusx/types";
⋮----
type PipelineStage = {
  key: string;
  title: string;
  metric: string;
  detail: string;
  href: Route;
};
⋮----
updatedAt={formatForecastUpdatedAt(demand?.generated_at)}
              stale=
```

## File: apps/supplier-portal/components/Providers.tsx
```typescript
import type { ReactNode } from "react";
import SupplierShell from "./SupplierShell";
import { ThemeProvider } from "./ThemeProvider";
import { DesktopCacheBootstrap } from "@/lib/desktop-cache-bootstrap";
import { DesktopDeepLinkBootstrap } from "@/lib/desktop-deep-link";
import { EnterpriseDesktopUpdateBootstrap } from "@/lib/desktop-updater";
import { PortalOfflineTray } from "@/lib/portal-offline-tray";
⋮----
export default function Providers({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/components/ReplenishmentTraceabilityPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierReplenishmentTraceRow } from "@pegasusx/types";
```

## File: apps/supplier-portal/components/SignalIngestOpsPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import type { PlanningSignalIngestStatus } from "@pegasusx/types";
⋮----
function formatLag(seconds: number): string
⋮----
<KpiStatCard label=
⋮----
label=
```

## File: apps/supplier-portal/components/Skeleton.tsx
```typescript
import type { CSSProperties } from "react";
⋮----
export function Skeleton({ className = "", style }: { className?: string; style?: CSSProperties })
```

## File: apps/supplier-portal/components/StatusBadge.tsx
```typescript
import StatusChip from './StatusChip';
⋮----
export default function StatusBadge({ state, className = '' }: { state: string; className?: string })
```

## File: apps/supplier-portal/components/StatusChip.tsx
```typescript
interface StatusChipProps {
  status: string;
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}
⋮----
export default function StatusChip({ status, label, size = 'sm', className }: StatusChipProps)
```

## File: apps/supplier-portal/components/ThemeProvider.tsx
```typescript
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';
⋮----
export type ThemeMode = 'system' | 'light' | 'dark';
⋮----
type ThemeContextValue = {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (mode: ThemeMode) => void;
  cycle: () => void;
};
⋮----
function getSystemPreference(): 'light' | 'dark'
⋮----
function getStoredMode(): ThemeMode
⋮----
function resolveMode(mode: ThemeMode): 'light' | 'dark'
⋮----
function applyTheme(resolved: 'light' | 'dark')
⋮----
export function ThemeProvider({ children }: { children: ReactNode })
⋮----
const handler = () =>
⋮----
export function useTheme()
```

## File: apps/supplier-portal/components/Toast.tsx
```typescript
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
⋮----
type ToastTone = "info" | "success" | "error";
⋮----
type ToastMessage = {
  id: string;
  message: string;
  tone: ToastTone;
  action?: {
    label: string;
    onClick: () => void;
  };
};
⋮----
type ToastContextValue = {
  push: (message: string, tone?: ToastTone, action?: ToastMessage["action"]) => void;
  toast: (message: string, tone?: ToastTone, action?: ToastMessage["action"]) => void;
};
⋮----
export function ToastProvider({ children }: { children: ReactNode })
⋮----
toast.action!.onClick();
setMessages((current)
```

## File: apps/supplier-portal/lib/admin-scope.ts
```typescript
import { decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
⋮----
function readJwtRole(): string
⋮----
export function canAssignOrder(): boolean
⋮----
export function canPatchOrderStatus(): boolean
⋮----
export function canAdminOrderOps(): boolean
```

## File: apps/supplier-portal/lib/api.ts
```typescript
import { ApiClient } from '@pegasusx/api-core';
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";
⋮----
export function createSupplierApi(): ApiClient
```

## File: apps/supplier-portal/lib/bridge.ts
```typescript

```

## File: apps/supplier-portal/lib/client-policy.ts
```typescript
import { supplierApiBaseUrl } from '@/lib/auth';
⋮----
export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response>
```

## File: apps/supplier-portal/lib/coverage.ts
```typescript
import type { CoverageMode, PSPListing, ServicePin, ServicePinTargetType } from "@pegasusx/types";
import { selectablePackPsps } from '@pegasusx/api-core';
⋮----
export function coverageModeLabel(mode: string | undefined): string
⋮----
export function normalizeCoverageMode(mode: string | undefined): CoverageMode
⋮----
export function pinKey(pin: Pick<ServicePin, "target_type" | "target_id">): string
⋮----
export function gatewayLabel(code: string): string
⋮----
export function catalogGatewayCodes(catalog: PSPListing[] | null | undefined): string[]
⋮----
export function catalogOmits(catalog: PSPListing[] | null | undefined, codes: string[]): boolean
```

## File: apps/supplier-portal/lib/csv.ts
```typescript

```

## File: apps/supplier-portal/lib/dashboard-command.ts
```typescript
export function orderStatusFromWsRaw(raw: unknown): string
```

## File: apps/supplier-portal/lib/desktop-cache-bootstrap.tsx
```typescript
import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";
⋮----
export function DesktopCacheBootstrap()
```

## File: apps/supplier-portal/lib/desktop-deep-link.tsx
```typescript
import { useEffect } from "react";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";
⋮----
export function DesktopDeepLinkBootstrap()
```

## File: apps/supplier-portal/lib/desktop-updater.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import {
  checkDesktopUpdate,
  desktopStoreListingUrl,
  installPendingDesktopUpdate,
  isDesktopStoreBuild,
  isTauri,
  openDesktopStoreListing,
  type DesktopUpdateInfo,
} from "@pegasusx/desktop-bridge";
⋮----
export function EnterpriseDesktopUpdateBootstrap()
⋮----
export async function openConfiguredDesktopStore(): Promise<boolean>
```

## File: apps/supplier-portal/lib/forecast-confidence.ts
```typescript

```

## File: apps/supplier-portal/lib/i18n.ts
```typescript
import { useSyncExternalStore } from "react";
import {
  createTranslator,
  LOCALE_CHANGE_EVENT,
  resolveActiveLocale,
  type Locale,
} from "@pegasusx/i18n";
⋮----
function subscribe(onStoreChange: () => void)
⋮----
function getSnapshot(): Locale
⋮----
function getServerSnapshot(): Locale
⋮----
export function usePortalLocale(): Locale
⋮----
export function usePortalT()
```

## File: apps/supplier-portal/lib/money.ts
```typescript
export interface ParseMoneyOptions {
  allowNegative?: boolean;
}
⋮----
export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}
⋮----
function toSafeNumberOrNull(value: bigint): number | null
⋮----
function toMinorBigInt(value: number | bigint | null | undefined): bigint
⋮----
function groupAsciiDigits(digits: string, separator: string): string
⋮----
function stripSpaceGrouping(raw: string): string | null
⋮----
/**
 * Deterministically parses a human-entered major-unit decimal string (e.g. "19.99", "0.29", "19,99")
 * into minor units (tiyin / cents) as a BigInt by splitting whole and fractional digit characters.
 * Rejects malformed strings and non-zero sub-minor precision (> 2 fractional digits).
 */
export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null
⋮----
export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null
⋮----
/**
 * Converts integer major units (e.g. whole UZS) into minor units (tiyin) using BigInt.
 */
export function majorIntToMinorBigInt(major: number | bigint | string): bigint
⋮----
export function majorIntToMinor(major: number | bigint | string): number
⋮----
export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeTrunc(minor: number | bigint | null | undefined): number
⋮----
export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeRound(minor: number | bigint | null | undefined): number
⋮----
export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint
⋮----
export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number
⋮----
export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint
⋮----
export function applyBpsFee(amount: number, bps: number): number
⋮----
export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string
⋮----
export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string
⋮----
/**
 * Formats integer basis points (100 bps = 1%) into a percentage string without float division.
 */
export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string
⋮----
/**
 * Formats (numerator / denominator) as a percentage string with 1 decimal place (tenths of a percent)
 * using pure BigInt half-away-from-zero integer division.
 */
export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string
⋮----
/**
 * Computes round_half_up((numerator / denominator) * 100) as an integer percentage using BigInt.
 */
export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number
⋮----
/**
 * Formats a BigInt or integer major-unit value with thousands grouping.
 */
export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string
```

## File: apps/supplier-portal/lib/nav.ts
```typescript
export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };
⋮----
export function primaryNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function allNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function navSectionIsActive(
  section: NavSection,
  pathname: string,
  isActive: (pathname: string, href: string) => boolean,
): boolean
```

## File: apps/supplier-portal/lib/oidc.ts
```typescript
import { supplierApiBaseUrl, supplierFetch, persistSession } from "@/lib/auth";
⋮----
export type OIDCConfig = {
  supplier_id: string;
  issuer: string;
  client_id: string;
  audience?: string;
  authorization_endpoint?: string;
  redirect_uri?: string;
  enabled: boolean;
};
⋮----
export type OIDCDiscovery = {
  enabled: boolean;
  supplier_id: string;
  issuer: string;
  client_id: string;
  redirect_uri?: string;
  authorization_url: string;
};
⋮----
export function oidcCallbackUrl(): string
⋮----
export async function discoverOIDC(supplierId: string, nonce: string): Promise<OIDCDiscovery>
⋮----
export async function exchangeOIDC(supplierId: string, idToken: string, nonce: string): Promise<void>
⋮----
export async function getSupplierOIDC(): Promise<OIDCConfig | null>
⋮----
export async function putSupplierOIDC(body: {
  issuer: string;
  client_id: string;
  audience?: string;
  authorization_endpoint?: string;
  redirect_uri?: string;
}): Promise<OIDCConfig>
⋮----
export async function deleteSupplierOIDC(): Promise<void>
```

## File: apps/supplier-portal/lib/order-receipt.ts
```typescript
import { createSupplierApi } from "@/lib/api";
⋮----
export async function openSupplierOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void>
```

## File: apps/supplier-portal/lib/scenario-chart.ts
```typescript
import type { PlanningScenarioResult } from "@pegasusx/types";
⋮----
export type ScenarioChartRow = {
  name: string;
  baseline: number;
  upside: number;
};
⋮----
export function pickScenarioPair(scenarios: PlanningScenarioResult[]): {
  baseline: PlanningScenarioResult | null;
  upside: PlanningScenarioResult | null;
}
⋮----
export function buildScenarioChartRows(scenarios: PlanningScenarioResult[]): ScenarioChartRow[]
```

## File: apps/supplier-portal/lib/session-reconcile.ts
```typescript
import { reconcileSession } from '@pegasusx/api-core';
import { readTokenFromCookie, supplierApiBaseUrl, supplierSessionFetch } from "@/lib/auth";
import { getSupplierReconcileScope, notifySupplierSessionReconciled } from "@/lib/supplier-reconnect";
⋮----
export async function reconcileSupplierSession(
  query?: Record<string, string>,
): Promise<void>
⋮----
export async function runSupplierSessionReconcile(
  query?: Record<string, string>,
): Promise<void>
```

## File: apps/supplier-portal/lib/supplier-cache-keys.ts
```typescript
import { supplierScopeId } from "@/lib/supplier-scope";
⋮----
export function supplierDashboardCacheKey(): string
⋮----
export function supplierOrdersCacheKey(
  query: Record<string, string | number | undefined>,
): string
```

## File: apps/supplier-portal/lib/supplier-reconnect.ts
```typescript
export function getSupplierReconnectEpoch(): number
⋮----
export function setSupplierReconcileScope(query: Record<string, string>): void
⋮----
export function getSupplierReconcileScope(): Record<string, string>
⋮----
export function notifySupplierSessionReconciled(): void
```

## File: apps/supplier-portal/lib/supplier-scope.ts
```typescript
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
⋮----
export function supplierScopeId(): string
⋮----
export function sessionSupplierId(): string | null
```

## File: apps/supplier-portal/lib/supplier-warehouse-ops.ts
```typescript
import {
  warehouseOrderProposeDeliveryKey,
  warehouseOrderRejectKey,
} from '@pegasusx/api-core';
import { createSupplierApi } from '@/lib/api';
```

## File: apps/supplier-portal/lib/supplier-ws-events.ts
```typescript
export function parseSupplierWsEventType(raw: unknown): string | null
```

## File: apps/supplier-portal/lib/topology.ts
```typescript
import type {
  SupplierTopologyFactory,
  SupplierTopologyFactoryInput,
  SupplierTopologyWarehouse,
  SupplierTopologyWarehouseInput,
} from "@pegasusx/types";
⋮----
export function warehouseToTopologyInput(w: SupplierTopologyWarehouse): SupplierTopologyWarehouseInput
⋮----
export function factoryToTopologyInput(f: SupplierTopologyFactory): SupplierTopologyFactoryInput
```

## File: apps/supplier-portal/lib/use-fleet-live-map.ts
```typescript
import { usePolling } from '@pegasusx/api-react';
import { useCallback, useState } from "react";
import type { SupplierFleetLiveRoute } from "@pegasusx/types";
import { applyDriverLocationPatch, parseDriverLocationPatch } from "@pegasusx/ws-refresh-contract";
import { createSupplierApi } from "@/lib/api";
import {
  SUPPLIER_FLEET_LIVE_REFRESH_EVENTS,
  SUPPLIER_LOCATION_PATCH_EVENTS,
} from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
⋮----
export function useFleetLiveMap(pollMs = 15_000)
```

## File: apps/supplier-portal/lib/use-payment-catalog.ts
```typescript
import { useMarketPack } from '@pegasusx/api-react';
⋮----
import { packCurrency, selectablePackPsps } from '@pegasusx/api-core';
import type { PSPListing } from "@pegasusx/types";
import { useEffect, useState } from "react";
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
⋮----
export function useSupplierPaymentCatalog()
```

## File: apps/supplier-portal/lib/use-supplier-session-reconcile.ts
```typescript
import { useEffect, useRef } from "react";
import { SUPPLIER_SESSION_RECONCILED } from "@/lib/supplier-reconnect";
⋮----
export function useSupplierSessionReconcile(onReconcile: () => void)
⋮----
const onEvent = () =>
```

## File: apps/supplier-portal/lib/useNotifications.ts
```typescript
import { useState, useEffect, useCallback, useRef } from 'react';
import { reconnectDelayMs } from '@pegasusx/api-core';
import { readTokenFromCookie, resolveSupplierToken, supplierFetch } from './auth';
import { runSupplierSessionReconcile } from './session-reconcile';
import type { HandoffCardMetadata } from '@pegasusx/types';
⋮----
interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}
⋮----
interface RealtimeNotificationFrame {
  id?: string;
  type?: string;
  title?: string;
  body?: string;
  payload?: string;
  channel?: string;
  created_at?: string;
}
⋮----
function parseRealtimePayload(raw?: string): Record<string, unknown>
⋮----
function normalizeNotification(item: BackendNotification): Notification
⋮----
export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}
⋮----
interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}
⋮----
export function useNotifications()
⋮----
const reconnectIfNeeded = () =>
⋮----
const handleWake = () =>
⋮----
const handleVisible = () =>
```

## File: apps/supplier-portal/src-tauri/src/commands/mod.rs
```rust
pub mod security;
```

## File: apps/supplier-portal/src-tauri/src/commands/security.rs
```rust
use keyring::Entry;
use serde::Serialize;
⋮----
pub struct TokenResult {
pub success: bool,
pub token: Option<String>,
pub error: Option<String>,
⋮----
pub fn store_token(token: String) -> TokenResult {
⋮----
pub fn get_token() -> TokenResult {
⋮----
pub fn clear_token() -> TokenResult {
```

## File: apps/supplier-portal/src-tauri/src/lib.rs
```rust
mod commands;
⋮----
pub fn run() {
```

## File: apps/supplier-portal/src-tauri/src/main.rs
```rust
fn main() {
```

## File: apps/supplier-portal/src-tauri/Cargo.toml
```toml
[package]
name = "pegasusx-supplier-portal"
version = "0.1.1"
description = "PegasusX Supplier Portal shell"
authors = ["PegasusX"]
edition = "2021"
rust-version = "1.77"

[lib]
name = "app_lib"
crate-type = ["staticlib", "cdylib", "rlib"]

[build-dependencies]
tauri-build = { version = "2", features = [] }

[dependencies]
serde = { version = "1", features = ["derive"] }
serde_json = "1"
log = "0.4"
tauri = { version = "2", features = ["macos-private-api"] }
tauri-plugin-shell = "2"
tauri-plugin-dialog = "2"
tauri-plugin-fs = "2"
tauri-plugin-process = "2"
tauri-plugin-sql = { version = "2", features = ["sqlite"] }
keyring = { version = "3", features = ["apple-native", "windows-native"] }

[target.'cfg(not(any(target_os = "android", target_os = "ios")))'.dependencies]
tauri-plugin-single-instance = { version = "2", features = ["deep-link"] }
tauri-plugin-deep-link = "2"
tauri-plugin-updater = "2"

[features]
default = ["custom-protocol"]
custom-protocol = ["tauri/custom-protocol"]
```

## File: apps/supplier-portal/src-tauri/tauri.conf.json
```json
{
  "$schema": "https://raw.githubusercontent.com/tauri-apps/tauri/dev/crates/tauri-config-schema/schema.json",
  "productName": "PegasusX Supplier",
  "version": "0.1.1",
  "identifier": "com.pegasusx.supplier",
  "build": {
    "frontendDist": "../out",
    "devUrl": "http://localhost:3000",
    "beforeDevCommand": "pnpm run dev",
    "beforeBuildCommand": "pnpm run build:static"
  },
  "app": {
    "withGlobalTauri": true,
    "windows": [
      {
        "label": "main",
        "title": "PegasusX Supplier Portal",
        "width": 1440,
        "height": 900,
        "minWidth": 1024,
        "minHeight": 700,
        "center": true,
        "resizable": true,
        "fullscreen": false,
        "transparent": false,
        "decorations": true,
        "titleBarStyle": "Overlay"
      }
    ],
    "security": {
      "csp": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' http://localhost:8180 http://127.0.0.1:8180 http://localhost:9099 http://127.0.0.1:9099 http://10.0.2.2:8180 ws://localhost:8180 ws://127.0.0.1:8180 ws://10.0.2.2:8180 wss://localhost:8180"
    },
    "macOSPrivateApi": true
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "createUpdaterArtifacts": true,
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ],
    "android": {
      "minSdkVersion": 24,
      "versionCode": 1
    }
  },
  "plugins": {
    "shell": {
      "open": true
    },
    "deep-link": {
      "desktop": {
        "schemes": [
          "pegasusx-supplier"
        ]
      }
    },
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDhDREMyMEEzRkY5NEYzQjMKUldTejg1VC9veURjakZ6R3NoSmZ6NGIyOGVBZVBZZGs5TkdMWk9XRTArQ2UyakJNZ01XYXZtS0UK",
      "endpoints": [
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/supplier-desktop/{{target}}/{{arch}}/updater.json"
      ],
      "windows": {
        "installMode": "basic"
      }
    }
  }
}
```

## File: apps/supplier-portal/middleware.ts
```typescript
import { NextResponse, type NextRequest } from "next/server";
⋮----
export function middleware(req: NextRequest)
⋮----
function readIsRegistered(jwt: string): boolean
⋮----
function isTokenExpired(jwt: string): boolean
⋮----
function readIsConfigured(jwt: string): boolean
```

## File: apps/supplier-portal/vitest.setup.ts
```typescript
getItem(key: string)
setItem(key: string, value: string)
removeItem(key: string)
clear()
get length()
key(index: number)
```

## File: apps/warehouse-portal/app/analytics/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import AnalyticsChartGrid from '@/components/analytics/AnalyticsChartGrid';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
interface AnalyticsData {
  period: string;
  currency?: string;
  total_revenue: number;
  total_orders: number;
  avg_order_value: number;
  top_products: { product_name: string; total_sold?: number; total_qty?: number; revenue: number }[];
  daily: { date: string; revenue: number; orders: number }[];
  daily_breakdown?: { date: string; revenue: number; orders: number }[];
  fleet_utilization_pct: number;
  fleet_utilization?: { utilization_pct: number };
  import_freshness?: {
    applied_rows_30d: number;
    applied_skus_30d: number;
    quantity_delta_30d: number;
    last_session_id?: string;
    last_applied_at?: string;
  };
  import_anomaly_queue?: {
    open_rows_30d: number;
    affected_sessions_30d: number;
    last_session_id?: string;
    last_detected_at?: string;
    last_detail?: string;
  };
}
⋮----
const fmt = (n: number) =>
⋮----
const fmtCurrency = (n: number) =>
⋮----
description=
```

## File: apps/warehouse-portal/app/analytics/VelocityGauge.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import {
  RadialBarChart,
  RadialBar,
  ResponsiveContainer,
  PolarAngleAxis,
} from 'recharts';
⋮----
type VelocityGaugeProps = {
  className?: string;
  avgDispatchMinutes?: number | null;
};
```

## File: apps/warehouse-portal/app/auth/register/wizard-state.ts
```typescript
export type StepId = "identity" | "verification" | "profile";
⋮----
export interface Country {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export interface IdentityStep {
  countryCode: string;
  phoneLocal: string;
}
⋮----
export interface VerificationStep {
  otpCode: string;
  idToken: string;
}
⋮----
export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
}
⋮----
export interface WizardState {
  step: StepId;
  identity: IdentityStep;
  verification: VerificationStep;
  profile: ProfileStep;
}
⋮----
export function validateIdentity(s: IdentityStep): Record<string, string>
⋮----
export function validateVerification(s: VerificationStep): Record<string, string>
⋮----
export function validateProfile(s: ProfileStep): Record<string, string>
```

## File: apps/warehouse-portal/app/auth/layout.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useTheme } from '@/components/ThemeProvider';
```

## File: apps/warehouse-portal/app/claims/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
type ClaimLine = {
  sku: string;
  quantity: number;
  reason?: string;
  amount_minor?: number;
};
⋮----
type Claim = {
  claim_id: string;
  order_id: string;
  retailer_id: string;
  claim_type: string;
  status: string;
  amount_minor?: number;
  currency?: string;
  description?: string;
  line_items?: ClaimLine[];
  created_at?: string;
};
⋮----
title=
```

## File: apps/warehouse-portal/app/control-tower/use-control-tower-telemetry.ts
```typescript
import { useState, useEffect, useRef } from "react";
import { warehouseApiBaseUrl, apiFetch } from "@/lib/auth";
import { reconnectDelayMs, retryAfterSecondsFromResponse } from '@pegasusx/api-core';
import { runWarehouseSessionReconcile } from "@/lib/session-reconcile";
export interface NetworkNode { id: string; type: "warehouse" | "retailer" | "driver"; label: string; status: "active" | "idle" | "busy"; }
export interface NetworkLink { source: string; target: string; value: number; }
⋮----
interface H3Density {
  hex: string;
  count: number;
}
⋮----
interface ControlTowerData {
  networkNodes: NetworkNode[];
  networkLinks: NetworkLink[];
  h3Data: H3Density[];
}
⋮----
export function useControlTowerTelemetry(supplierId: string): ControlTowerData
⋮----
const clearTimers = () =>
⋮----
const closeSocket = () =>
⋮----
const scheduleReconnect = () =>
⋮----
const connect = async () =>
```

## File: apps/warehouse-portal/app/coverage/page.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import PageTransition from "@/components/PageTransition";
import { createWarehouseApi } from "@/lib/api";
import { coverageModeLabel, normalizeCoverageMode, pinKey } from "@/lib/coverage";
import type { ServicePin, SupplierTopologyCoverageCity, WarehouseOpsSupplyFactoryResponse } from "@pegasusx/types";
⋮----
Mode {coverageModeLabel(mode)}
⋮----
<li key=
```

## File: apps/warehouse-portal/app/credit/policy/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
⋮----
async function whFetch(path: string, init?: RequestInit): Promise<Response>
⋮----
type Program = {
  program_enabled: boolean;
  global_terms_days: number;
  global_default_limit_minor: number;
};
⋮----
type Relationship = {
  retailer_id: string;
  credit_enabled: boolean;
  terms_days: number;
  credit_limit_minor: number;
  profile_status?: string;
  current_balance_minor?: number;
};
```

## File: apps/warehouse-portal/app/crm/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
⋮----
interface Retailer {
  retailer_id: string;
  business_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date: string;
}
⋮----
const fmt = (n: number) =>
⋮----
description=
⋮----
<button type="button" onClick=
```

## File: apps/warehouse-portal/app/demand-forecast/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import type { ForecastConfidence, WarehouseReplenishmentInsight } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { ForecastChartPanel } from '@/components/forecast/ForecastChartPanel';
import { ForecastSkuTable } from '@/components/forecast/ForecastSkuTable';
⋮----
export interface ForecastProduct {
  product_id: string;
  product_name: string;
  current_stock: number;
  recommended_qty: number;
  days_until_stockout: number;
  priority: string;
  unit: string;
  sources: {
    incoming_orders: number;
    ai_prediction: number;
    pre_orders: number;
    burn_rate: number;
  };
  confidence?: ForecastConfidence;
}
⋮----
interface Forecast {
  warehouse_id: string;
  forecast_days: number;
  generated_at: string;
  products: ForecastProduct[];
}
⋮----
async function loadForecast()
⋮----
description=
⋮----
onChange=
```

## File: apps/warehouse-portal/app/dispatch-locks/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { ApiError } from '@pegasusx/api-core';
import { subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import type { WarehouseDispatchLock, WarehouseLiveEvent } from '@pegasusx/types';
⋮----
async function handleAcquire()
⋮----
async function handleRelease(lockId: string)
⋮----
description=
⋮----
emptyMessage=
⋮----
onClick=
```

## File: apps/warehouse-portal/app/dispatch-settings/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { ApiError, warehouseDispatchSettingsKey } from '@pegasusx/api-core';
import type { WarehouseDispatchPreview } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
⋮----
type DispatchPreviewState = Partial<WarehouseDispatchPreview> & { error?: string };
⋮----
const handleToggle = async (next: boolean) =>
⋮----
description=
⋮----
setLoading(true);
void load();
```

## File: apps/warehouse-portal/app/exceptions/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
⋮----
type ExceptionRow = {
  exception_id?: string;
  kind: string;
  order_id?: string;
  manifest_id?: string;
  reason?: string;
  status?: string;
  updated_at?: string;
  delivery_expectation?: { target_label?: string };
};
⋮----
<PageChrome title=
```

## File: apps/warehouse-portal/app/fleet-live-map/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
⋮----
description=
```

## File: apps/warehouse-portal/app/operations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import {
  warehouseBroadcastKey,
  warehouseBroadcastTemplateCreateKey,
  warehouseBroadcastTemplateDeleteKey,
} from '@pegasusx/api-core';
import type { BroadcastTemplate, RetailerOverridePreview } from '@pegasusx/types';
import { PageChrome } from '@/components/PageChrome';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId, warehouseScopeQuery } from '@/lib/warehouse-scope';
import { OperationsBroadcastForm } from '@/components/operations/OperationsBroadcastForm';
import { OperationsPricingPreview } from '@/components/operations/OperationsPricingPreview';
⋮----
function applyTemplate(
  template: BroadcastTemplate,
  templateDate: string,
  customReason: string,
): { title: string; body: string; role: string }
⋮----
export default function WarehouseOperationsPage()
⋮----
const onSelectTemplate = (template: BroadcastTemplate) =>
⋮----
const onBroadcast = async () =>
⋮----
const onDeleteTemplate = async (template: BroadcastTemplate) =>
⋮----
description=
```

## File: apps/warehouse-portal/app/orders/[id]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/warehouse-portal/app/orders/components/OrdersList.tsx
```typescript
import React from 'react';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
export type OrdersTab = 'active' | 'preorders';
⋮----
export interface OrderRow {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at: string;
}
⋮----
export interface OrdersListProps {
  tab: OrdersTab;
  loading: boolean;
  filter: string;
  activeItems: OrderRow[];
  preorderItems: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string, isPreorder: boolean, currentDate?: string) => void;
  onReject: (orderId: string, isPreorder: boolean) => void;
}
⋮----
function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean
⋮----
const fmt = (n: number) =>
⋮----
amountLabel=
```

## File: apps/warehouse-portal/app/payment-config/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { createWarehouseApi } from '@/lib/api';
import { catalogOmits, gatewayLabel } from '@/lib/coverage';
import { useWarehousePaymentCatalog } from '@/lib/use-payment-catalog';
import type { PSPListing } from '@pegasusx/types';
⋮----
description=
⋮----
<button type="button" onClick=
```

## File: apps/warehouse-portal/app/preorders/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { redirect } from 'next/navigation';
⋮----
export default function PreordersRedirectPage()
```

## File: apps/warehouse-portal/app/profile/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo } from 'react';
import Link from 'next/link';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { clearSession, readTokenFromCookie } from '@/lib/auth';
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
description=
⋮----
```

## File: apps/warehouse-portal/app/replenishment/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import type { WarehouseReplenishmentInsight } from '@pegasusx/types';
import { ApiError, warehouseReplenishmentInsightActionKey } from '@pegasusx/api-core';
import { warehouseApi } from '@/lib/warehouse-api';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { ReplenishmentList } from '@/components/replenishment/ReplenishmentList';
⋮----
description=
⋮----
setLoading(true);
void load();
⋮----
label=
⋮----
<KpiStatCard label=
⋮----
<PageSection title=
⋮----
onDismiss={(id) =>
```

## File: apps/warehouse-portal/app/settings/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState, type ChangeEvent } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { LocationPicker, resolveLocationValue, type LocationValue } from '@/components/LocationPicker';
import { hasValidCoordinates } from '@/lib/geocode';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
import type { DeliveryFeeRules, WarehouseOpsLocationResponse, WarehouseOpsSettings, WarehouseOpsSettingsPatchRequest } from '@pegasusx/types';
import { warehouseOpsLocationKey, warehouseOpsSettingsKey } from '@pegasusx/api-core';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { OpsSettingsForm } from '@/components/settings/OpsSettingsForm';
import { ReturnPolicySettingsSection } from '@/components/settings/ReturnPolicySettingsSection';
import { useWarehousePaymentCatalog } from '@/lib/use-payment-catalog';
⋮----
async function save()
⋮----
async function saveLocation()
⋮----
description=
⋮----
<PortalSection icon="warehouse" title=
```

## File: apps/warehouse-portal/app/setup/location/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import Icon from "@/components/Icon";
import { FormAlert, PortalField, PortalInput } from "@/components/portal";
import { LocationPicker, resolveLocationValue, type LocationValue } from "@/components/LocationPicker";
import { hasValidCoordinates } from "@/lib/geocode";
import {
  apiFetch,
  decodeJwtPayload,
  persistSession,
  readTokenFromCookie,
  refreshWarehouseSession,
} from "@/lib/auth";
import { useWarehousePaymentCatalog } from "@/lib/use-payment-catalog";
⋮----
type WarehouseLocation = {
  warehouse_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
  country_code?: string;
  pack_country_code?: string;
};
⋮----
async function submit()
⋮----
// submit handler will try forward geocode before failing
```

## File: apps/warehouse-portal/app/setup/layout.tsx
```typescript
import SetupWizardShell from "@/components/setup/SetupWizardShell";
⋮----
export default function SetupLayout({ children }: { children: React.ReactNode })
```

## File: apps/warehouse-portal/app/stock-commitments/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { warehouseApi } from '@/lib/warehouse-api';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
⋮----
interface StockRow {
  sku_id: string;
  name?: string;
  image_url?: string;
  on_hand: number;
  available_qty: number;
  reserved_asap: number;
  reserved_scheduled: number;
  deficit_qty: number;
}
⋮----
description=
⋮----
emptyMessage=
⋮----
Available
```

## File: apps/warehouse-portal/app/supply-requests/[id]/layout.tsx
```typescript
export function generateStaticParams()
⋮----
export default function SupplyRequestDetailLayout({
  children,
}: {
  children: React.ReactNode;
})
```

## File: apps/warehouse-portal/app/supply-requests/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { ApiError } from '@pegasusx/api-core';
import { subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
import type { WarehouseLiveEvent, WarehouseSupplyRequest } from '@pegasusx/types';
⋮----
function chipClass(state: string): string
⋮----
description=
⋮----
onClick=
```

## File: apps/warehouse-portal/app/tracking/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { createWarehouseApi } from "@/lib/api";
import { DispatchSidebarNav } from "../../../supplier-portal/components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../../supplier-portal/components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../../supplier-portal/components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../../supplier-portal/components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../../supplier-portal/components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../../supplier-portal/components/dispatch/CargoPhotoCarousel";
⋮----
hasLiveRoute=
```

## File: apps/warehouse-portal/app/vehicles/[vehicleId]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/warehouse-portal/app/vehicles/[vehicleId]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import type { WarehouseFleetVehicle } from '@pegasusx/types';
import { fetchWarehouseVehicle } from '@/lib/use-warehouse-vehicles-live';
import { formatUnavailableReason, vehicleStatusLabel } from '@/lib/vehicle-fleet';
import { subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from '@/lib/fleet-ws-events';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import VehicleAvailabilityPanel from '@/components/VehicleAvailabilityPanel';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
⋮----
<PageSection title=
```

## File: apps/warehouse-portal/app/error.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
```

## File: apps/warehouse-portal/app/layout.tsx
```typescript
import type { Metadata } from "next";
import { Plus_Jakarta_Sans, EB_Garamond, IBM_Plex_Mono } from "next/font/google";
import Providers from "@/components/Providers";
import LocaleBootstrap from "@/components/LocaleBootstrap";
```

## File: apps/warehouse-portal/components/analytics/AnalyticsChartGrid.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
⋮----
interface AnalyticsChartGridProps {
  dailySeries: { date: string; revenue: number; orders: number }[];
  fmtCurrency: (n: number) => string;
}
```

## File: apps/warehouse-portal/components/dispatch/DispatchDriverList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { WarehouseDispatchDriver, WarehouseUnavailableDispatchDriver } from '@pegasusx/types';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
⋮----
function formatVU(vu: number)
⋮----
function formatUnavailableReason(reason?: string, note?: string)
⋮----
interface DispatchDriverListProps {
  drivers: WarehouseDispatchDriver[];
  unavailableDrivers: WarehouseUnavailableDispatchDriver[];
}
⋮----
/**
 * Available and unavailable drivers section of the Dispatch screen.
 *
 * Renders driver cards with status badges, VU capacity, and
 * unavailability reasons in a split layout.
 */
```

## File: apps/warehouse-portal/components/drivers/DriversList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type {
  WarehouseFleetDriver,
  WarehouseFleetVehicle,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import Icon from '@/components/Icon';
⋮----
function formatUnavailableReason(reason?: string)
⋮----
export interface DriversListProps {
  drivers: WarehouseFleetDriver[];
  vehicles: WarehouseFleetVehicle[];
  loading: boolean;
  assigningDriverId: string;
  handleAssignVehicle: (driverId: string, vehicleId: string) => void;
}
⋮----
function assignedVehicleLabel(driver: WarehouseFleetDriver)
⋮----
function assignedVehicleReason(driver: WarehouseFleetDriver)
⋮----
function vehicleOptionLabel(vehicle: WarehouseFleetVehicle)
```

## File: apps/warehouse-portal/components/operations/OperationsPricingPreview.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { RetailerOverridePreview } from '@pegasusx/types';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
⋮----
interface OperationsPricingPreviewProps {
  productId: string;
  setProductId: (val: string) => void;
  retailerId: string;
  setRetailerId: (val: string) => void;
  proposedPrice: string;
  setProposedPrice: (val: string) => void;
  previewLoading: boolean;
  preview: RetailerOverridePreview | null;
}
⋮----
<PortalSection title=
```

## File: apps/warehouse-portal/components/orders/index.ts
```typescript

```

## File: apps/warehouse-portal/components/orders/OrderActionDialog.tsx
```typescript
import { useEffect, useRef } from 'react';
⋮----
type OrderActionDialogProps = {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel: string;
  destructive?: boolean;
  reason: string;
  onReasonChange: (value: string) => void;
  reasonRequired?: boolean;
  submitting?: boolean;
  onConfirm: () => void;
  onClose: () => void;
};
⋮----
export function OrderActionDialog({
  open,
  title,
  description,
  confirmLabel,
  destructive = false,
  reason,
  onReasonChange,
  reasonRequired = false,
  submitting = false,
  onConfirm,
  onClose,
}: OrderActionDialogProps)
```

## File: apps/warehouse-portal/components/orders/OrderKebabMenu.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useRef, useState } from 'react';
import Icon from '@/components/Icon';
⋮----
type OrderKebabMenuProps = {
  onViewDetails: () => void;
  onProposeDate?: () => void;
  onReject?: () => void;
  canProposeDate?: boolean;
  canReject?: boolean;
  proposeDateLabel?: string;
  rejectLabel?: string;
  disabled?: boolean;
};
⋮----
const onDoc = (e: MouseEvent) =>
⋮----
e.stopPropagation();
e.preventDefault();
setOpen((v)
⋮----
onClick={(e) =>
```

## File: apps/warehouse-portal/components/orders/OrderLineItems.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { WarehouseOrderDetail } from '@pegasusx/types';
⋮----
export interface OrderLineItemsProps {
  order: WarehouseOrderDetail;
}
⋮----
const fmt = (n: number) =>
```

## File: apps/warehouse-portal/components/orders/OrderStateChip.tsx
```typescript
import { ORDER_STATE_CHIP } from '@/lib/order-actions';
⋮----
export function OrderStateChip({ state }: { state: string })
```

## File: apps/warehouse-portal/components/portal/index.ts
```typescript

```

## File: apps/warehouse-portal/components/preorders/PreordersList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
export interface PreordersListProps {
  loading: boolean;
  items: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string, isPreorder: boolean, currentDate?: string) => void;
  onReject: (orderId: string, isPreorder: boolean) => void;
}
⋮----
function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean
⋮----
const fmt = (n: number) =>
⋮----
onProposeDate={() =>
onReject={() =>
```

## File: apps/warehouse-portal/components/replenishment/ReplenishmentList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { WarehouseReplenishmentInsight } from '@pegasusx/types';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
⋮----
interface ReplenishmentListProps {
  insights: WarehouseReplenishmentInsight[];
  actingId: string | null;
  onApprove: (id: string) => void;
  onDismiss: (id: string) => void;
}
⋮----
const urgencyClass = (urgency: string) =>
⋮----
onClick=
```

## File: apps/warehouse-portal/components/returns/ReverseLogisticsPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
⋮----
type ReverseTask = {
  TaskId?: string;
  task_id?: string;
  OrderId?: string;
  order_id?: string;
  Status?: string;
  status?: string;
  ExpectedQtyJson?: string;
  expected_qty_json?: string;
};
⋮----
const taskId = (t: ReverseTask) =>
const orderId = (t: ReverseTask) =>
⋮----
export function ReverseLogisticsPanel()
⋮----
async function receive(task: ReverseTask)
⋮----
<li key=
```

## File: apps/warehouse-portal/components/setup/constants.ts
```typescript
export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];
⋮----
export function setupStepIndex(pathname: string): number
```

## File: apps/warehouse-portal/components/setup/SetupWizardShell.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { usePathname } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { useTheme } from "@/components/ThemeProvider";
import Icon from "@/components/Icon";
import { SETUP_STEPS, setupStepIndex } from "./constants";
⋮----
function SetupThemeToggle()
```

## File: apps/warehouse-portal/components/staff/StaffList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Icon from '@/components/Icon';
import type { WarehouseStaffMember } from '@pegasusx/types';
⋮----
interface StaffListProps {
  staff: WarehouseStaffMember[];
  loading: boolean;
}
```

## File: apps/warehouse-portal/components/treasury/TreasuryTransactionList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import EmptyState from '@/components/EmptyState';
import { Invoice } from '@/app/treasury/page';
⋮----
interface TreasuryTransactionListProps {
  invoices: Invoice[];
  fmt: (n: number) => string;
  resolveAmount: (inv: Invoice) => number;
  resolveCurrency: (inv: Invoice) => string;
  formatPayoutOwner: (inv: Invoice) => string;
}
⋮----
Owner {formatPayoutOwner(inv)}
```

## File: apps/warehouse-portal/components/vehicles/VehiclesList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import type { WarehouseFleetVehicle } from '@pegasusx/types';
import { formatUnavailableReason, vehicleStatusLabel } from '@/lib/vehicle-fleet';
import Icon from '@/components/Icon';
⋮----
interface VehiclesListProps {
  vehicles: WarehouseFleetVehicle[];
  loading: boolean;
}
```

## File: apps/warehouse-portal/components/AuthGuard.tsx
```typescript
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { clearSession, decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { getStoredToken as getDesktopToken } from "@/lib/bridge";
⋮----
function isTokenExpired(token: string): boolean
⋮----
export default function AuthGuard({ children }: { children: React.ReactNode })
```

## File: apps/warehouse-portal/components/ClientPolicyBanner.tsx
```typescript
import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";
import { desktopClientPolicyContext } from "@pegasusx/desktop-bridge";
import { getClientPolicy } from "@/lib/client-policy";
⋮----
type PolicyState = {
  forceUpdate: boolean;
  outdated: boolean;
  minimumVersion: string;
  updateUrl?: string;
  deferReason?: string;
};
⋮----
// Policy endpoint optional on local dev stacks.
```

## File: apps/warehouse-portal/components/DispatchPreviewMap.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef } from 'react';
import type { WarehouseDispatchProposedRoute } from '@pegasusx/types';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
⋮----
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
type DispatchPreviewMapProps = {
  routes: WarehouseDispatchProposedRoute[];
  className?: string;
};
⋮----
type RouteFeatureCollection = GeoJSON.FeatureCollection<GeoJSON.LineString>;
⋮----
mapRef.current = ref?.getMap() ?? null;
⋮----
...mapInitialViewState(readCachedAuthSession()?.pack),
```

## File: apps/warehouse-portal/components/EmptyState.tsx
```typescript
import { motion } from "framer-motion";
import { ReactNode, useEffect, useState } from "react";
⋮----
type EmptyStateVariant =
  | "loading"
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "reconnecting"
  | "stale";
⋮----
interface EmptyStateProps {
  icon?: ReactNode;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}
⋮----
function resolveVariant(headline: string, body?: string): EmptyStateVariant
```

## File: apps/warehouse-portal/components/ForecastConfidenceView.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ForecastConfidence } from '@pegasusx/types';
import {
  formatSourceBadge,
  isForecastBlocked,
  isSeasonalTemplateActive,
} from '@/lib/forecast-confidence';
⋮----
type Props = {
  confidence: ForecastConfidence;
  compact?: boolean;
};
⋮----
function confidenceColor(pct?: number): string
```

## File: apps/warehouse-portal/components/KpiStatCard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { HistorySeries } from "@pegasusx/types";
import { guardHistorySeries } from "@pegasusx/types";
import Link from "next/link";
import type { ReactNode } from "react";
import Icon from "./Icon";
⋮----
type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  icon?: string;
  href?: string;
  flag?: "alert" | "ok";
  bay?: "ops" | "inventory" | "fleet" | "finance";
  className?: string;
  align?: "start" | "center";
  spark?: HistorySeries | null;
};
⋮----
function Sparkline({ data }: { data: number[] })
⋮----
```

## File: apps/warehouse-portal/components/LanguageSwitcher.tsx
```typescript
import {
  supportedLocales,
  syncDocumentLocale,
  type Locale,
} from "@pegasusx/i18n";
import { usePortalLocale, usePortalT } from "@/lib/i18n";
```

## File: apps/warehouse-portal/components/ListToolbar.tsx
```typescript
type ListToolbarProps = {
  page: number;
  pageCount: number;
  totalLabel: string;
  onPrev: () => void;
  onNext: () => void;
  onExport?: () => void;
  exportLabel?: string;
  exportDisabled?: boolean;
};
⋮----
export function ListToolbar({
  page,
  pageCount,
  totalLabel,
  onPrev,
  onNext,
  onExport,
  exportLabel = "Export CSV",
  exportDisabled = false,
}: ListToolbarProps)
```

## File: apps/warehouse-portal/components/LocaleBootstrap.tsx
```typescript
import { useEffect } from "react";
import { bootstrapBrowserLocale } from "@pegasusx/i18n/browser";
⋮----
export default function LocaleBootstrap(): null
```

## File: apps/warehouse-portal/components/OrderTimelinePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
⋮----
function formatWhen(iso: string): string
⋮----
return <p className="text-sm text-[var(--desk-text-tertiary)]">
⋮----
{formatWhen(entry.created_at)}
⋮----
Proposed delivery:
```

## File: apps/warehouse-portal/components/PageSection.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type PageSectionProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bay?: "ops" | "inventory" | "fleet" | "finance";
};
⋮----
export function PageSection({
  title,
  description,
  actions,
  children,
  className = "",
  bay = "ops",
}: PageSectionProps)
```

## File: apps/warehouse-portal/components/PageTransition.tsx
```typescript
import { motion, useReducedMotion } from "framer-motion";
import { ReactNode } from "react";
import { usePathname } from "next/navigation";
⋮----
interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}
⋮----
export default function PageTransition({ children, className = "" }: PageTransitionProps)
```

## File: apps/warehouse-portal/components/Providers.tsx
```typescript
import type { ReactNode } from "react";
import { Suspense } from "react";
import WarehouseShell from "./WarehouseShell";
import { ThemeProvider } from "./ThemeProvider";
import { ToastProvider } from "./Toast";
import { PageSkeleton } from "./Skeleton";
import { DesktopCacheBootstrap } from "@/lib/desktop-cache-bootstrap";
import { DesktopDeepLinkBootstrap } from "@/lib/desktop-deep-link";
import { EnterpriseDesktopUpdateBootstrap } from "@/lib/desktop-updater";
import { PortalOfflineTray } from "@/lib/portal-offline-tray";
⋮----
export default function Providers({ children }: { children: ReactNode })
```

## File: apps/warehouse-portal/components/Skeleton.tsx
```typescript
import type { CSSProperties } from 'react';
⋮----
export function Skeleton({ className = '', style }: { className?: string; style?: CSSProperties })
```

## File: apps/warehouse-portal/components/ThemeProvider.tsx
```typescript
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useState, type ReactNode } from 'react';
⋮----
export type ThemeMode = 'system' | 'light' | 'dark';
⋮----
interface ThemeCtx {
  mode: ThemeMode;
  resolved: 'light' | 'dark';
  setMode: (m: ThemeMode) => void;
  cycle: () => void;
}
⋮----
export const useTheme = () =>
⋮----
function getSystemPreference(): 'light' | 'dark'
⋮----
function getStoredMode(): ThemeMode
⋮----
function resolveMode(mode: ThemeMode): 'light' | 'dark'
⋮----
function applyTheme(resolved: 'light' | 'dark')
⋮----
export function ThemeProvider({ children }: { children: ReactNode })
⋮----
const handler = () =>
```

## File: apps/warehouse-portal/components/Toast.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
⋮----
type ToastType = 'info' | 'success' | 'error' | 'warning';
⋮----
interface ToastAction {
  label: string;
  onClick: () => void;
}
⋮----
interface Toast {
  id: number;
  message: string;
  type: ToastType;
  action?: ToastAction;
  durationMs: number;
}
⋮----
interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: ToastAction, durationMs?: number) => void;
}
⋮----
export function useToast()
⋮----
function clamp(value: number, min: number, max: number)
⋮----
const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const finishDrag = () =>
⋮----
const onPointerUp = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerCancel = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
onPointerDown=
⋮----
event.stopPropagation();
toast.action?.onClick();
dismiss(toast.id);
```

## File: apps/warehouse-portal/lib/api.ts
```typescript
import { ApiClient } from '@pegasusx/api-core';
import { readTokenFromCookie, warehouseApiBaseUrl } from "@/lib/auth";
⋮----
export function createWarehouseApi(): ApiClient
```

## File: apps/warehouse-portal/lib/bridge.ts
```typescript

```

## File: apps/warehouse-portal/lib/client-policy.ts
```typescript
import { warehouseApiBaseUrl } from '@/lib/auth';
⋮----
export async function getClientPolicy(
  platform: string,
  version: string,
  channel = 'production',
): Promise<Response>
```

## File: apps/warehouse-portal/lib/coverage.ts
```typescript
import type { CoverageMode, PSPListing, ServicePin } from "@pegasusx/types";
import { selectablePackPsps } from '@pegasusx/api-core';
⋮----
export function coverageModeLabel(mode: string | undefined): string
⋮----
export function normalizeCoverageMode(mode: string | undefined): CoverageMode
⋮----
export function pinKey(pin: Pick<ServicePin, "target_type" | "target_id">): string
⋮----
export function gatewayLabel(code: string): string
⋮----
export function catalogGatewayCodes(catalog: PSPListing[] | null | undefined): string[]
⋮----
export function catalogOmits(catalog: PSPListing[] | null | undefined, codes: string[]): boolean
```

## File: apps/warehouse-portal/lib/csv.ts
```typescript

```

## File: apps/warehouse-portal/lib/desktop-cache-bootstrap.tsx
```typescript
import { useEffect } from "react";
import { initDesktopCache } from "@pegasusx/desktop-cache";
⋮----
export function DesktopCacheBootstrap()
```

## File: apps/warehouse-portal/lib/desktop-deep-link.tsx
```typescript
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { subscribeDesktopDeepLinks } from "@pegasusx/desktop-bridge";
⋮----
export function DesktopDeepLinkBootstrap()
```

## File: apps/warehouse-portal/lib/desktop-updater.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import {
  checkDesktopUpdate,
  desktopStoreListingUrl,
  installPendingDesktopUpdate,
  isDesktopStoreBuild,
  isTauri,
  openDesktopStoreListing,
  type DesktopUpdateInfo,
} from "@pegasusx/desktop-bridge";
⋮----
export function EnterpriseDesktopUpdateBootstrap()
⋮----
export async function openConfiguredDesktopStore(): Promise<boolean>
```

## File: apps/warehouse-portal/lib/dispatch-preview-cache.ts
```typescript
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import type {
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseUnavailableDispatchDriver,
} from "@pegasusx/types";
⋮----
export type DispatchPreviewSnapshot = {
  orders: WarehouseDispatchOrder[];
  drivers: WarehouseDispatchDriver[];
  unavailableDrivers: WarehouseUnavailableDispatchDriver[];
  proposedRoutes: WarehouseDispatchProposedRoute[];
  optimizerSource: string | null;
  optimizerWarnings: string[];
  windowConstrainedCount: number;
  fleetEffectiveCapacityVU: number;
  planFingerprint: string | null;
  cachedAt: string;
};
⋮----
export function dispatchPreviewCacheKey(
  warehouseId: string,
  orderIds: string[] = [],
): string
⋮----
export async function getDispatchPreviewCache(
  key: string,
): Promise<DispatchPreviewSnapshot | null>
⋮----
export async function setDispatchPreviewCache(
  key: string,
  snapshot: DispatchPreviewSnapshot,
): Promise<void>
⋮----
export function snapshotFromPreviewResponse(data: {
  undispatched_orders?: WarehouseDispatchOrder[];
  orders?: WarehouseDispatchOrder[];
  available_drivers?: WarehouseDispatchDriver[];
  drivers?: WarehouseDispatchDriver[];
  unavailable_drivers?: WarehouseUnavailableDispatchDriver[];
  proposed_routes?: WarehouseDispatchProposedRoute[];
  optimizer_source?: string | null;
  optimizer_warnings?: string[];
  window_constrained_count?: number;
  fleet_effective_capacity_vu?: number;
  plan_fingerprint?: string | null;
}): DispatchPreviewSnapshot
```

## File: apps/warehouse-portal/lib/dispatch-runs-cache.ts
```typescript
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
⋮----
export type DispatchRunRow = {
  run_id: string;
  status: string;
  manifest_count: number;
  orders_assigned: number;
  created_at: string;
};
⋮----
export function dispatchRunsCacheKey(warehouseId: string): string
⋮----
export async function getDispatchRunsCache(
  key: string,
): Promise<DispatchRunRow[] | null>
⋮----
export async function setDispatchRunsCache(
  key: string,
  runs: DispatchRunRow[],
): Promise<void>
```

## File: apps/warehouse-portal/lib/fleet-ws-events.ts
```typescript
import { DISPATCH_REFRESH_EVENTS, ORDER_STATUS_REFRESH_EVENTS, parseWsEventType } from '@pegasusx/ws-refresh-contract';
⋮----
export function parseWarehouseWsEventType(raw: unknown): string | null
```

## File: apps/warehouse-portal/lib/forecast-confidence.ts
```typescript
import type { ForecastConfidence } from '@pegasusx/types';
⋮----
function asNumber(value: unknown): number | undefined
⋮----
function asString(value: unknown): string | undefined
⋮----
/** Parse planning-brain confidence fields from demand_breakdown JSON. */
export function parseForecastConfidence(
  breakdown?: Record<string, unknown> | null,
): ForecastConfidence | null
⋮----
export function isForecastBlocked(confidence?: ForecastConfidence | null): boolean
⋮----
export function isSeasonalTemplateActive(confidence?: ForecastConfidence | null): boolean
⋮----
export function formatSourceBadge(source?: string): string
```

## File: apps/warehouse-portal/lib/i18n.ts
```typescript
import { useSyncExternalStore } from "react";
import {
  createTranslator,
  LOCALE_CHANGE_EVENT,
  resolveActiveLocale,
  type Locale,
} from "@pegasusx/i18n";
⋮----
function subscribe(onStoreChange: () => void)
⋮----
function getSnapshot(): Locale
⋮----
function getServerSnapshot(): Locale
⋮----
export function usePortalLocale(): Locale
⋮----
export function usePortalT()
```

## File: apps/warehouse-portal/lib/money.ts
```typescript
export interface ParseMoneyOptions {
  allowNegative?: boolean;
}
⋮----
export interface FormatMinorOptions {
  currency?: string;
  currencyPosition?: "suffix" | "prefix";
  minimumFractionDigits?: 0 | 1 | 2;
  maximumFractionDigits?: 0 | 1 | 2;
  groupSeparator?: string;
  decimalSeparator?: string;
  signDisplay?: "auto" | "always";
}
⋮----
function toSafeNumberOrNull(value: bigint): number | null
⋮----
function toMinorBigInt(value: number | bigint | null | undefined): bigint
⋮----
function groupAsciiDigits(digits: string, separator: string): string
⋮----
function stripSpaceGrouping(raw: string): string | null
⋮----
export function parseMajorToMinorBigInt(
  input: string,
  options?: ParseMoneyOptions,
): bigint | null
⋮----
export function parseMajorToMinor(
  input: string,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function parseMinorUnitsInt(
  input: string | number | bigint,
  options?: ParseMoneyOptions,
): number | null
⋮----
export function majorIntToMinorBigInt(major: number | bigint | string): bigint
⋮----
export function majorIntToMinor(major: number | bigint | string): number
⋮----
export function minorToWholeTruncBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeTrunc(minor: number | bigint | null | undefined): number
⋮----
export function minorToWholeRoundBigInt(minor: number | bigint | null | undefined): bigint
⋮----
export function minorToWholeRound(minor: number | bigint | null | undefined): number
⋮----
export function applyDiscountBpsBigInt(
  amountMinor: number | bigint,
  discountBps: number | bigint,
): bigint
⋮----
export function applyDiscountBpsMinor(amountMinor: number, discountBps: number): number
⋮----
export function applyBpsFeeBigInt(
  amount: number | bigint,
  bps: number | bigint,
): bigint
⋮----
export function applyBpsFee(amount: number, bps: number): number
⋮----
export function formatMinorUnits(
  minor: number | bigint | null | undefined,
  options?: FormatMinorOptions,
): string
⋮----
export function formatMinorFixed(
  minor: number | bigint | null | undefined,
  options?: { signDisplay?: "auto" | "always"; currency?: string },
): string
⋮----
export function formatBpsPercent(
  bps: number | bigint | null | undefined,
  fractionDigits: 0 | 1 | 2 = 2,
): string
⋮----
export function formatRatioPercentTenths(
  numerator: number | bigint,
  denominator: number | bigint,
): string
⋮----
export function ratioPercentInt(
  numerator: number | bigint,
  denominator: number | bigint,
  capAtHundred = true,
): number
⋮----
export function formatGroupedInteger(
  value: number | bigint | null | undefined,
  groupSeparator = ",",
): string
```

## File: apps/warehouse-portal/lib/order-receipt.ts
```typescript
import { createWarehouseApi } from "@/lib/api";
⋮----
export async function openWarehouseOrderReceipt(
  orderId: string,
  format: "html" | "pdf" = "html",
): Promise<void>
```

## File: apps/warehouse-portal/lib/session-reconcile.ts
```typescript
import { reconcileSession } from '@pegasusx/api-core';
import { apiFetch, readTokenFromCookie, warehouseApiBaseUrl, warehouseSessionFetch } from './auth';
import { notifyWarehouseSessionReconciled } from './warehouse-reconnect';
⋮----
export async function reconcileWarehouseSession(
  query?: Record<string, string>,
): Promise<void>
⋮----
export async function runWarehouseSessionReconcile(
  query?: Record<string, string>,
): Promise<void>
```

## File: apps/warehouse-portal/lib/use-payment-catalog.ts
```typescript
import { useMarketPack } from '@pegasusx/api-react';
⋮----
import { packCurrency, selectablePackPsps } from '@pegasusx/api-core';
import type { PSPListing } from "@pegasusx/types";
import { useEffect, useState } from "react";
import { createWarehouseApi } from "@/lib/api";
import { readTokenFromCookie, warehouseApiBaseUrl } from "@/lib/auth";
⋮----
export function useWarehousePaymentCatalog()
```

## File: apps/warehouse-portal/lib/use-warehouse-fleet-live-map.ts
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { useCallback, useEffect, useState } from 'react';
import { createWarehouseApi } from '@/lib/api';
import { subscribeWarehouseWS } from '@/lib/auth';
import { applyDriverLocationPatch, parseDriverLocationPatch } from '@pegasusx/ws-refresh-contract';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS, WAREHOUSE_LOCATION_PATCH_EVENTS } from '@/lib/fleet-ws-events';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
⋮----
export function useWarehouseFleetLiveMap(pollMs = 15_000)
```

## File: apps/warehouse-portal/lib/use-warehouse-session-reconcile.ts
```typescript
import { useEffect, useRef } from 'react';
import { WAREHOUSE_SESSION_RECONCILED } from '@/lib/warehouse-reconnect';
⋮----
export function useWarehouseSessionReconcile(onReconcile: () => void)
⋮----
const onEvent = () =>
```

## File: apps/warehouse-portal/lib/use-warehouse-vehicles-live.ts
```typescript
import { useCallback, useEffect, useRef, useState } from 'react';
import type { WarehouseFleetVehicle, WarehouseFleetVehicleListResponse } from '@pegasusx/types';
import { apiFetch, subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from '@/lib/fleet-ws-events';
⋮----
export function useWarehouseVehiclesLive(options: { enabled?: boolean } = {})
⋮----
export async function fetchWarehouseVehicle(vehicleId: string): Promise<WarehouseFleetVehicle | null>
```

## File: apps/warehouse-portal/lib/use-warehouse-ws-refresh.ts
```typescript
import { useEffect, useRef } from 'react';
import { subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_ORDERS_REFRESH_EVENTS } from '@/lib/fleet-ws-events';
⋮----
type UseWarehouseWsRefreshOptions = {
  eventTypes?: ReadonlySet<string>;
  debounceMs?: number;
  enabled?: boolean;
};
⋮----
export function useWarehouseWsRefresh(
  onSignal: (eventType: string) => void,
  {
    eventTypes = WAREHOUSE_ORDERS_REFRESH_EVENTS,
    debounceMs = 500,
    enabled = true,
  }: UseWarehouseWsRefreshOptions = {},
)
```

## File: apps/warehouse-portal/lib/useNotifications.ts
```typescript
import { useState, useEffect, useCallback, useRef } from 'react';
import { reconnectDelayMs } from '@pegasusx/api-core';
import { readTokenFromCookie, apiFetch } from './auth';
import { runWarehouseSessionReconcile } from './session-reconcile';
import type { HandoffCardMetadata } from '@pegasusx/types';
⋮----
interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}
⋮----
interface RealtimeNotificationFrame {
  id?: string;
  type?: string;
  title?: string;
  body?: string;
  payload?: string;
  channel?: string;
  created_at?: string;
}
⋮----
function parseRealtimePayload(raw?: string): Record<string, unknown>
⋮----
function normalizeNotification(item: BackendNotification): Notification
⋮----
export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}
⋮----
interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}
⋮----
export type WarehouseWsState = 'connected' | 'connecting' | 'reconnecting' | 'offline';
⋮----
export function useNotifications(options?: { enabled?: boolean })
⋮----
const reconnectIfNeeded = () =>
⋮----
const handleWake = () =>
⋮----
const handleOffline = () =>
⋮----
const handleVisible = () =>
```

## File: apps/warehouse-portal/lib/useStableCallback.ts
```typescript
import { useCallback, useRef } from "react";
⋮----
export function useStableCallback<T extends (...args: never[]) => unknown>(fn: T): T
```

## File: apps/warehouse-portal/lib/vehicle-fleet.ts
```typescript
import type { WarehouseVehicleUnavailableReason } from '@pegasusx/types';
⋮----
export function formatUnavailableReason(reason?: string, note?: string)
⋮----
export function vehicleStatusLabel(isActive: boolean)
```

## File: apps/warehouse-portal/lib/warehouse-api.ts
```typescript
import { createWarehouseApi } from "@/lib/api";
```

## File: apps/warehouse-portal/lib/warehouse-reconnect.ts
```typescript
export function getWarehouseReconnectEpoch(): number
⋮----
export function notifyWarehouseSessionReconciled(): void
```

## File: apps/warehouse-portal/lib/warehouse-scope.ts
```typescript
import { decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
⋮----
export function warehouseHomeNodeId(): string
⋮----
export function warehouseScopeQuery(): { warehouse_id?: string }
```

## File: apps/warehouse-portal/src-tauri/src/commands/mod.rs
```rust
pub mod security;
⋮----
pub fn get_app_info() -> serde_json::Value {
```

## File: apps/warehouse-portal/src-tauri/src/commands/security.rs
```rust
use keyring::Entry;
use serde::Serialize;
⋮----
pub struct TokenResult {
pub success: bool,
pub token: Option<String>,
pub error: Option<String>,
⋮----
pub fn store_token(token: String, refresh_token: Option<String>) -> TokenResult {
⋮----
pub fn get_token() -> TokenResult {
⋮----
pub fn clear_token() -> TokenResult {
⋮----
fn read_token(service: &str) -> Result<Option<String>, String> {
⋮----
fn delete_if_present(service: &str, key: &str) -> bool {
```

## File: apps/warehouse-portal/src-tauri/src/lib.rs
```rust
mod commands;
⋮----
pub fn run() {
```

## File: apps/warehouse-portal/src-tauri/src/main.rs
```rust
fn main() {
```

## File: apps/warehouse-portal/src-tauri/Cargo.toml
```toml
[package]
name = "pegasus-warehouse-desktop"
version = "0.1.1"
description = "PegasusX — Warehouse Portal Desktop Shell"
authors = ["PegasusX"]
license = "proprietary"
repository = ""
default-run = "pegasus-warehouse-desktop"
edition = "2021"
rust-version = "1.77"

[build-dependencies]
tauri-build = { version = "2", features = [] }

[dependencies]
tauri = { version = "2", features = ["macos-private-api"] }
tauri-plugin-shell = "2"
tauri-plugin-dialog = "2"
tauri-plugin-fs = "2"
tauri-plugin-os = "2"
tauri-plugin-process = "2"
tauri-plugin-sql = { version = "2", features = ["sqlite"] }

[target.'cfg(not(any(target_os = "android", target_os = "ios")))'.dependencies]
tauri-plugin-single-instance = { version = "2", features = ["deep-link"] }
tauri-plugin-deep-link = "2"
tauri-plugin-updater = "2"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
keyring = { version = "3", features = ["apple-native", "windows-native"] }
log = "0.4"
env_logger = "0.11"

[features]
default = ["custom-protocol"]
custom-protocol = ["tauri/custom-protocol"]
```

## File: apps/warehouse-portal/src-tauri/tauri.conf.json
```json
{
  "$schema": "https://raw.githubusercontent.com/tauri-apps/tauri/dev/crates/tauri-config-schema/schema.json",
  "productName": "PegasusX Warehouse Desktop",
  "version": "0.1.1",
  "identifier": "com.pegasusx.warehouse",
  "build": {
    "frontendDist": "../out",
    "devUrl": "http://localhost:3002",
    "beforeDevCommand": "pnpm run dev",
    "beforeBuildCommand": "pnpm run build:static"
  },
  "app": {
    "macOSPrivateApi": true,
    "withGlobalTauri": true,
    "windows": [
      {
        "label": "main",
        "title": "PegasusX Warehouse",
        "width": 1440,
        "height": 900,
        "minWidth": 1024,
        "minHeight": 700,
        "center": true,
        "resizable": true,
        "fullscreen": false,
        "transparent": false,
        "decorations": true,
        "titleBarStyle": "Overlay"
      }
    ],
    "security": {
      "dangerousDisableAssetCspModification": false,
      "csp": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob:; connect-src 'self' http://localhost:8180 http://127.0.0.1:8180 http://localhost:9099 http://127.0.0.1:9099 ws://localhost:8180 wss://localhost:8180"
    }
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "createUpdaterArtifacts": true,
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ],
    "macOS": {
      "minimumSystemVersion": "12.0",
      "frameworks": []
    }
  },
  "plugins": {
    "shell": {
      "open": true
    },
    "deep-link": {
      "desktop": {
        "schemes": [
          "pegasusx-warehouse"
        ]
      }
    },
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDhDREMyMEEzRkY5NEYzQjMKUldTejg1VC9veURjakZ6R3NoSmZ6NGIyOGVBZVBZZGs5TkdMWk9XRTArQ2UyakJNZ01XYXZtS0UK",
      "endpoints": [
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/warehouse-desktop/{{target}}/{{arch}}/updater.json"
      ],
      "windows": {
        "installMode": "basic"
      }
    }
  }
}
```

## File: apps/warehouse-portal/middleware.ts
```typescript
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
function isTokenExpired(token: string): boolean
⋮----
export function middleware(request: NextRequest)
```

## File: apps/warehouse-portal/vitest.config.ts
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'
```

## File: apps/warehouse-portal/vitest.setup.ts
```typescript
getItem(key: string)
setItem(key: string, value: string)
removeItem(key: string)
clear()
```

## File: apps/admin-portal/app/api/auth/logout/route.ts
```typescript
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
⋮----
export async function POST()
```

## File: apps/admin-portal/app/api/auth/session/route.ts
```typescript
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
⋮----
export async function GET()
```

## File: apps/admin-portal/app/compliance/cashier/page.tsx
```typescript
import React, { useState } from "react";
import type { CashierSessionView, DepotVaultLedgerEntry } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
export default function CashierVaultPage()
⋮----
const handleStartSession = async (e: React.FormEvent) =>
⋮----
const handleCountChange = (denom: number, count: number) =>
⋮----
const handleSubmitBlindCount = async (e: React.FormEvent) =>
⋮----
const handleCITHandover = async (e: React.FormEvent) =>
⋮----
onChange=
```

## File: apps/admin-portal/app/compliance/fiscal/page.tsx
```typescript
import React, { useState } from "react";
import type { FiscalPreflightResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
interface LineItemRow {
  sku: string;
  ikpu_code: string;
  package_code: string;
}
⋮----
export default function FiscalPreflightPage()
⋮----
const addLine = () =>
⋮----
const removeLine = (idx: number) =>
⋮----
const updateLine = (idx: number, field: keyof LineItemRow, val: string) =>
⋮----
const handlePreflight = async (e: React.FormEvent) =>
⋮----
onChange={(e) =>
⋮----
onChange={(e) => updateLine(idx, "sku", e.target.value)}
```

## File: apps/admin-portal/app/governance/audit/page.tsx
```typescript
import React, { useEffect, useState, useMemo } from "react";
import { api, type AuditRow } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
export default function AuditLedgerPage()
⋮----
const fetchAudit = async () =>
⋮----
const exportCSV = () =>
⋮----
onChange=
```

## File: apps/admin-portal/app/governance/flags/page.tsx
```typescript
import React, { useEffect, useState } from "react";
import { api, type FlagEval, type FlagOverride } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
const fetchPending = async () =>
⋮----
// Non-fatal if offline
⋮----
const handleEvaluate = async () =>
⋮----
const handleSetFlag = async (enabled: boolean) =>
⋮----
const handleApprove = async (item: FlagOverride) =>
⋮----
onChange=
```

## File: apps/admin-portal/app/governance/tenants/page.tsx
```typescript
import React, { useEffect, useState, useCallback } from "react";
import { api, type Tenant, type MatchQueueItem } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
export default function TenantsCatalogGovernancePage()
⋮----
// Match Queue State
⋮----
const handleTransitionTenant = async (t: Tenant, toStatus: string) =>
⋮----
const handleResolveMatch = async (id: string, decision: "ACCEPT" | "REJECT") =>
⋮----
onClick={() =>
```

## File: apps/admin-portal/app/infra/page.tsx
```typescript
import React, { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
interface CellTelemetry {
  id: string;
  region: string;
  role: "PRIMARY_CORE" | "REGIONAL_EDGE";
  status: "HEALTHY" | "DEGRADED" | "OFFLINE";
  spannerNodes: number;
  p99LatencyMs: number;
  commitLatencyMs: number;
  outboxLagSec: number;
  activeTotes: number;
}
⋮----
export default function InfraCommandPage()
⋮----
const fetchInfraStatus = async () =>
```

## File: apps/admin-portal/app/logistics/gate/page.tsx
```typescript
import React, { useState } from "react";
import type { YardGateCheckEvent } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
export default function YardGatePage()
⋮----
const handleRecordGateEvent = async (e: React.FormEvent) =>
⋮----
onChange={(e) =>
⋮----
onClick={() =>
```

## File: apps/admin-portal/app/overview/page.tsx
```typescript
import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { api, type FlagOverride, type Tenant } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { useAdminWS } from "@/lib/ws-provider";
import { deadLetterHealth, deadLetterLabel } from "@/lib/deadLetterHealth";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
// DLQ Drawer & Replay State
⋮----
const handleReplay = async (eventId: string) =>
⋮----
<tr key=
⋮----
<CopyableId value=
⋮----
ID:
```

## File: apps/admin-portal/app/traceability/actions/page.tsx
```typescript
import React, { useState } from "react";
import type { ActionExecutionRecord } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
type ActionTypeKey =
  | "ACTION_RELEASE_PICK_WAVE"
  | "ACTION_SEAL_VEHICLE_MANIFEST"
  | "ACTION_START_DRIVER_ROUTE"
  | "ACTION_COMPLETE_DOORSTEP_EPOD"
  | "ACTION_BATCH_SUPPLIER_PAYOUT";
⋮----
interface KineticActionSchema {
  label: string;
  description: string;
  stepName: string;
  defaultParams: Record<string, any>;
}
⋮----
// Dynamic parameters state based on actionType
⋮----
const handleActionTypeChange = (newType: ActionTypeKey) =>
⋮----
const handleGenerateKey = () =>
⋮----
const updateParam = (key: string, val: any) =>
⋮----
const handleExecute = async (e: React.FormEvent) =>
⋮----
const getRubricRemediation = (ruleName: string, reason?: string) =>
⋮----
onChange=
```

## File: apps/admin-portal/app/traceability/recall/page.tsx
```typescript
import React, { useState } from "react";
import type { RecallBlastRadiusReport } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
type ViewMode = "funnel" | "canvas";
⋮----
const handleLotRecall = async (e: React.FormEvent) =>
⋮----
const openDrilldown = (category: string, items: string[] | undefined) =>
⋮----
const exportCSV = () =>
⋮----
const loadDemo = () =>
```

## File: apps/admin-portal/app/traceability/sandbox/page.tsx
```typescript
import React, { useState } from "react";
import type { OntologyObject } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
⋮----
const handleApplyPatch = async (e: React.FormEvent) =>
⋮----
const handleExecuteCommit = async () =>
⋮----
const loadDemo = () =>
⋮----
setConflictState(null);
void handleApplyPatch(
```

## File: apps/admin-portal/app/error.tsx
```typescript
import React, { useEffect } from "react";
import { mapToHumanError } from "@/lib/errors";
⋮----
export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
})
⋮----
onClick={() =>
```

## File: apps/admin-portal/app/global-error.tsx
```typescript
import React from "react";
⋮----
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
})
⋮----
onClick={() =>
```

## File: apps/admin-portal/app/layout.tsx
```typescript
import type { Metadata } from "next";
⋮----
import { AdminSessionProvider } from "@/lib/session";
import { AdminWebSocketProvider } from "@/lib/ws-provider";
import { AdminAppShell } from "@/components/layout/AdminAppShell";
⋮----
export default function RootLayout({ children }: { children: React.ReactNode })
```

## File: apps/admin-portal/app/loading.tsx
```typescript
import { RouteSkeleton } from "@/components/ui/RouteSkeleton";
⋮----
export default function Loading()
```

## File: apps/admin-portal/components/auth/AdminAuthGate.tsx
```typescript
import React, { useState, useEffect, useCallback, type ReactNode } from "react";
import { useAdminToken } from "@/lib/session";
import { api } from "@/lib/api";
⋮----
type MfaGate =
  | { kind: "ok" }
  | { kind: "loading" }
  | { kind: "need_enroll" }
  | { kind: "enroll"; secret: string; otpauth: string }
  | { kind: "verify" }
  | { kind: "error"; message: string };
⋮----
onClick={async () =>
```

## File: apps/admin-portal/components/ui/BentoGrid.tsx
```typescript
import { type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { LoadingGlyph } from './Skeleton';
⋮----
interface BentoGridProps {
  children: ReactNode;
  className?: string;
  theme?: 'brutalist' | 'apple';
  staggerChildren?: number;
}
⋮----
export function BentoGrid({
  children,
  className = '',
  theme = 'apple',
  staggerChildren = 0.05
}: BentoGridProps)
⋮----
// -- Bento Card --------------------------------------------------------------
⋮----
type BentoSize = 'stat' | 'anchor' | 'list' | 'control' | 'wide' | 'full';
⋮----
interface BentoCardProps {
  children: ReactNode;
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
  interactive?: boolean;
  delay?: number;
}
⋮----
// Build size class — semantic size takes priority over legacy span/rowSpan
⋮----
// -- Bento Skeleton (per-cell placeholder) -----------------------------------
```

## File: apps/admin-portal/components/ui/CopyableId.tsx
```typescript
import React, { useState } from "react";
⋮----
interface CopyableIdProps {
  value?: string;
  id?: string;
  label?: string;
  truncateLength?: number;
  className?: string;
}
⋮----
const handleCopy = async (e: React.MouseEvent) =>
```

## File: apps/admin-portal/components/ui/CountUp.tsx
```typescript
import { useEffect, useRef } from 'react';
import { useMotionValue, useSpring, useTransform, motion } from 'framer-motion';
⋮----
export default function CountUp({
  value,
  duration = 2,
  delay = 0,
  className = '',
}: {
  value: string;
  duration?: number;
  delay?: number;
  className?: string;
})
⋮----
// If it was an integer, keep it integer. If it had decimals, keep 2 decimals.
```

## File: apps/admin-portal/components/ui/DesktopKeyboardCheatsheet.tsx
```typescript
import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";
⋮----
export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};
⋮----
export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
```

## File: apps/admin-portal/components/ui/EmptyState.tsx
```typescript
import { motion, useReducedMotion } from "framer-motion";
import { ReactNode } from "react";
import Icon from "./Icon";
⋮----
type EmptyStateVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "production"
  | "no-orders"
  | "no-products"
  | "no-predictions"
  | "no-suppliers";
⋮----
interface EmptyStateProps {
  icon?: ReactNode | string;
  imageUrl?: string;
  headline: string;
  body?: string;
  action?: string;
  onAction?: () => void;
  variant?: EmptyStateVariant;
}
⋮----
function resolveVariant(headline: string, body?: string): EmptyStateVariant
⋮----
function resolveAccent(variant: EmptyStateVariant)
```

## File: apps/admin-portal/components/ui/motion.ts
```typescript
export type CubicBezierEasing = [number, number, number, number];
⋮----
export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};
⋮----
export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition
```

## File: apps/admin-portal/components/ui/PageTransition.tsx
```typescript
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { ReactNode } from "react";
import { usePathname } from "next/navigation";
⋮----
interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}
⋮----
export default function PageTransition({ children, className = "" }: PageTransitionProps)
```

## File: apps/admin-portal/components/ui/PortalPrimitives.tsx
```typescript
import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
⋮----
export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
{optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
```

## File: apps/admin-portal/components/ui/RouteSkeleton.tsx
```typescript
import React from "react";
```

## File: apps/admin-portal/components/ui/Skeleton.tsx
```typescript
import type { CSSProperties } from 'react';
import { motion, useReducedMotion } from 'framer-motion';
⋮----
/** Full-page loading skeleton matching common dashboard layouts */
⋮----
/**
 * M3 Page-level loading skeleton.
 * Used by loading.tsx files across all routes.
 */
```

## File: apps/admin-portal/components/ui/StatsCard.tsx
```typescript
import { motion } from 'framer-motion';
import CountUp from './CountUp';
⋮----
export default function StatsCard({
  label,
  value,
  sub,
  trend,
  trendValue,
  accent,
  delay = 0,
  sparkline,
  className = '',
}: {
  label: string;
  value: string;
  sub?: string;
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  accent?: string;
  delay?: number;
  sparkline?: number[];
  className?: string;
})
⋮----
// Generate simple SVG sparkline path
⋮----
{/* Sparkline */}
⋮----
{/* Sub text */}
⋮----
{/* Top accent bar */}
```

## File: apps/admin-portal/components/ui/Toast.tsx
```typescript
import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
⋮----
type ToastType = 'info' | 'success' | 'error' | 'warning';
⋮----
interface ToastAction {
  label: string;
  onClick: () => void;
}
⋮----
interface Toast {
  id: number;
  message: string;
  type: ToastType;
  action?: ToastAction;
  durationMs: number;
}
⋮----
interface ToastContextValue {
  toast: (message: string, type?: ToastType, action?: ToastAction, durationMs?: number) => void;
}
⋮----
export function useToast()
⋮----
function clamp(value: number, min: number, max: number)
⋮----
const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const finishDrag = () =>
⋮----
const onPointerUp = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
const onPointerCancel = (event: React.PointerEvent<HTMLDivElement>) =>
⋮----
onPointerDown=
⋮----
event.stopPropagation();
toast.action?.onClick();
dismiss(toast.id);
```

## File: apps/admin-portal/components/AccuracyPanel.tsx
```typescript
import { useState } from "react";
import { api, type AccuracyRow } from "@/lib/api";
⋮----
const load = async () =>
⋮----
<button onClick=
```

## File: apps/admin-portal/components/FlagsPanel.tsx
```typescript
import { useEffect, useState } from "react";
import { api, type FlagEval, type FlagOverride } from "@/lib/api";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "./ui/PortalPrimitives";
⋮----
const evaluate = async () =>
⋮----
const setFlag = async (enabled: boolean) =>
⋮----
const approve = async () =>
⋮----
<button onClick=
```

## File: apps/admin-portal/components/FoundryTriadPanel.tsx
```typescript
import React, { useState } from 'react';
import type { ActionExecutionRecord } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
import OntologyBlastRadiusGraph from './OntologyBlastRadiusGraph';
import ScenarioSandboxManager from './ScenarioSandboxManager';
⋮----
export default function FoundryTriadPanel({ token }: { token: string })
⋮----
const handleExecuteAction = async (e: React.FormEvent) =>
```

## File: apps/admin-portal/components/OntologyBlastRadiusGraph.tsx
```typescript
import React, { useState } from 'react';
import type { RecallBlastRadiusReport, OntologyObject } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
⋮----
export default function OntologyBlastRadiusGraph({ token }: { token: string })
⋮----
const handleLotRecall = async (e: React.FormEvent) =>
⋮----
const handleSearchAround = async () =>
⋮----
onChange={(e) =>
```

## File: apps/admin-portal/components/OrderStateMachineGraph.tsx
```typescript
import React, { useMemo } from "react";
import {
  type OrderStatus,
  ORDER_LIFECYCLE_NODES,
  CANONICAL_ORDER_GRAPH,
  type LifecycleNodeMeta,
} from "@pegasusx/types";
⋮----
interface OrderStateMachineGraphProps {
  currentStatus: OrderStatus;
  onSelectTargetStatus?: (status: OrderStatus) => void;
  className?: string;
}
```

## File: apps/admin-portal/components/PartnerPanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { api, type PartnerKey } from "@/lib/api";
⋮----
const runDunning = async () =>
⋮----
<button onClick=
```

## File: apps/admin-portal/components/ScenarioSandboxManager.tsx
```typescript
import React, { useState } from 'react';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
⋮----
const handleApplyPatch = async (e: React.FormEvent) =>
⋮----
const handleResolve = async () =>
⋮----
const handleCommit = async () =>
⋮----
onChange=
```

## File: apps/admin-portal/lib/api.ts
```typescript
import type {
  Tenant,
  FlagOverride,
  AccuracyRow,
  AuditRow,
  FlagEval,
  MatchQueueItem,
  PartnerKey,
  BillingInvoice,
  BillingFeeSchedule,
} from "@pegasusx/types";
⋮----
const base = () =>
⋮----
async function req<T>(token: string, method: string, path: string, body?: unknown): Promise<T>
⋮----
async function executeAdminReq<T>(token: string, method: string, url: string, body?: unknown): Promise<T>
```

## File: apps/admin-portal/lib/enterprise-swr.ts
```typescript
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
⋮----
export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";
⋮----
type Listener = () => void;
⋮----
class EnterpriseNetworkLifecycleEngine
⋮----
constructor()
⋮----
public get isOnline(): boolean
⋮----
public get state(): NetworkState
⋮----
public get syncEpoch(): number
⋮----
private notify()
⋮----
private recordTransition()
⋮----
public setOnline(online: boolean)
⋮----
public async triggerSyncEpoch()
⋮----
public reset()
⋮----
export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};
⋮----
export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};
⋮----
export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T>
```

## File: apps/admin-portal/lib/errors.ts
```typescript
export interface HumanError {
  title: string;
  message: string;
  actionGuidance?: string;
  isRetryable: boolean;
  retryAfterSeconds?: number;
  rawError?: string;
}
⋮----
export function mapToHumanError(err: unknown): HumanError
```

## File: apps/admin-portal/lib/session.tsx
```typescript
import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
⋮----
interface SessionContextType {
  token: string;
  setToken: (t: string) => void;
  isLoading: boolean;
  logout: () => Promise<void>;
  login: (subject?: string, password?: string, directToken?: string) => Promise<{ ok: boolean; error?: string }>;
}
⋮----
export function AdminSessionProvider({ children }: { children: ReactNode })
⋮----
// Hydrate session from HTTP-only cookie endpoint on mount
⋮----
async function checkSession()
⋮----
export function useAdminToken()
```

## File: apps/admin-portal/src-tauri/src/commands/diagnostics.rs
```rust
use serde::Serialize;
use std::fs;
⋮----
pub struct DiagnosticExportResult {
pub success: bool,
pub path: Option<String>,
pub bytes_written: usize,
pub error: Option<String>,
⋮----
pub fn export_diagnostic_bundle_native(file_path: String, payload: String) -> DiagnosticExportResult {
```

## File: apps/admin-portal/src-tauri/src/commands/mod.rs
```rust
pub mod diagnostics;
pub mod security;
pub mod telemetry;
```

## File: apps/admin-portal/src-tauri/src/commands/security.rs
```rust
use keyring::Entry;
use serde::Serialize;
⋮----
pub struct TokenResult {
pub success: bool,
pub token: Option<String>,
pub error: Option<String>,
⋮----
pub fn store_token(token: String) -> TokenResult {
⋮----
pub fn get_token() -> TokenResult {
⋮----
pub fn clear_token() -> TokenResult {
```

## File: apps/admin-portal/src-tauri/src/commands/telemetry.rs
```rust
use serde::Serialize;
use std::time::Instant;
⋮----
pub struct DesktopSystemInfo {
pub os_name: String,
pub arch: String,
pub app_version: String,
pub is_desktop: bool,
pub default_cell: String,
⋮----
pub struct GatewayPingResult {
pub target_cell: String,
pub reachable: bool,
pub latency_ms: u64,
pub timestamp_utc: String,
pub error: Option<String>,
⋮----
pub fn get_desktop_system_info() -> DesktopSystemInfo {
⋮----
pub async fn ping_cell_gateway(cell_id: String) -> GatewayPingResult {
⋮----
fn chrono_lite_iso() -> String {
use std::time::SystemTime;
```

## File: apps/admin-portal/src-tauri/src/lib.rs
```rust
mod commands;
⋮----
pub fn run() {
```

## File: apps/admin-portal/src-tauri/src/main.rs
```rust
fn main() {
```

## File: apps/admin-portal/src-tauri/Cargo.toml
```toml
[package]
name = "pegasusx-admin-portal"
version = "1.0.1"
description = "PegasusX Control Tower & Ecosystem Telemetry Desktop"
authors = ["PegasusX"]
edition = "2021"
rust-version = "1.77"

[lib]
name = "app_lib"
crate-type = ["staticlib", "cdylib", "rlib"]

[build-dependencies]
tauri-build = { version = "2", features = [] }

[dependencies]
serde = { version = "1", features = ["derive"] }
serde_json = "1"
log = "0.4"
tauri = { version = "2", features = ["macos-private-api"] }
tauri-plugin-shell = "2"
tauri-plugin-dialog = "2"
tauri-plugin-fs = "2"
tauri-plugin-process = "2"
tauri-plugin-sql = { version = "2", features = ["sqlite"] }
keyring = { version = "3", features = ["apple-native", "windows-native"] }

[target.'cfg(not(any(target_os = "android", target_os = "ios")))'.dependencies]
tauri-plugin-single-instance = { version = "2", features = ["deep-link"] }
tauri-plugin-deep-link = "2"
tauri-plugin-updater = "2"

[features]
default = ["custom-protocol"]
custom-protocol = ["tauri/custom-protocol"]
```

## File: apps/admin-portal/src-tauri/tauri.conf.json
```json
{
  "$schema": "https://raw.githubusercontent.com/tauri-apps/tauri/dev/crates/tauri-config-schema/schema.json",
  "productName": "PegasusX Control Tower",
  "version": "1.0.1",
  "identifier": "com.pegasusx.admin",
  "build": {
    "frontendDist": "../out",
    "devUrl": "http://localhost:3000",
    "beforeDevCommand": "pnpm run dev",
    "beforeBuildCommand": "pnpm run build:static"
  },
  "app": {
    "withGlobalTauri": true,
    "windows": [
      {
        "label": "main",
        "title": "PegasusX Control Tower & Ecosystem Telemetry",
        "width": 1440,
        "height": 900,
        "minWidth": 1024,
        "minHeight": 720,
        "center": true,
        "resizable": true,
        "fullscreen": false,
        "transparent": false,
        "decorations": true,
        "titleBarStyle": "Overlay"
      }
    ],
    "security": {
      "csp": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' http://localhost:8180 http://127.0.0.1:8180 http://localhost:9099 http://127.0.0.1:9099 ws://localhost:8180 ws://127.0.0.1:8180 wss://localhost:8180 https://*.pegasusx.app"
    },
    "macOSPrivateApi": true
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "createUpdaterArtifacts": true,
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ]
  },
  "plugins": {
    "deep-link": {
      "desktop": {
        "schemes": [
          "pegasusx-admin"
        ]
      }
    },
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDhDREMyMEEzRkY5NEYzQjMKUldTejg1VC9veURjakZ6R3NoSmZ6NGIyOGVBZVBZZGs5TkdMWk9XRTArQ2UyakJNZ01XYXZtS0UK",
      "endpoints": [
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/admin-desktop/{{target}}/{{arch}}/updater.json"
      ]
    }
  }
}
```

## File: apps/admin-portal/middleware.ts
```typescript
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
⋮----
export function middleware(request: NextRequest)
```

## File: apps/admin-portal/next.config.ts
```typescript
import type { NextConfig } from "next";
```

## File: apps/factory-portal/app/auth/login/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard, dialCodeForCountry } from "@/components/ui";
import { persistSession, factoryApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
⋮----
type LoginStep = "phone" | "otp";
⋮----
export default function FactoryLoginPage()
⋮----
async function handleSendOtp(e: React.FormEvent)
⋮----
async function handleVerifyOtp(e: React.FormEvent)
⋮----
title=
⋮----
registerLabel=
⋮----
resetPhoneOtpFlow();
setStep("phone");
```

## File: apps/factory-portal/app/auth/register/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
  dialCodeForCountry,
  FormAlert,
} from "@/components/ui";
import { persistSession, factoryApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import {
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";
⋮----
function validateCurrent(): Record<string, string>
⋮----
async function next()
⋮----
function back()
⋮----
async function submit()
⋮----
title=
```

## File: apps/factory-portal/app/manifest-exceptions/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { ExplainStatusBanner, explainFromApiError } from '@/components/ui/ExplainStatusBanner';
import type { StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';
⋮----
import { ManifestException, ManifestExceptionsList } from '../../components/manifest-exceptions/ManifestExceptionsList';
⋮----
function exceptionSignature(items: ManifestException[])
⋮----
function formatSyncTime(value: number | null)
⋮----
const onOnline = () =>
const onOffline = () =>
⋮----
description=
⋮----
onClick=
⋮----
<KpiStatCard label=
```

## File: apps/factory-portal/app/manifests/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { ExplainStatusBanner, explainFromApiError } from '@/components/ui/ExplainStatusBanner';
import type { StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { MANIFEST_STATE_ORDER, manifestTransitionIdempotencyKey, nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
⋮----
interface ManifestTransfer {
  transfer_id: string;
  order_id?: string;
  state: string;
  total_vu?: number;
}
⋮----
interface ManifestTransition {
  action: string;
  from_state?: string;
  to_state?: string;
  at?: string;
  reason?: string;
}
⋮----
interface ManifestException {
  exception_id: string;
  transfer_id: string;
  reason: string;
  attempt_count: number;
}
⋮----
interface ManifestDetailPayload {
  manifest: {
    manifest_id: string;
    state: string;
    transfer_count?: number;
    total_volume_vu?: number;
    max_volume_vu?: number;
    driver_id?: string;
    vehicle_id?: string;
    updated_at?: string;
  };
  transfers: ManifestTransfer[];
  transitions: ManifestTransition[];
  exceptions: ManifestException[];
  route_id?: string;
  stop_count?: number;
  order_count?: number;
}
⋮----
function stateClass(state: string): string
⋮----
const runLifecycle = async () =>
⋮----
<PageChrome icon="manifests" title=
```

## File: apps/factory-portal/app/staff/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
⋮----
interface StaffDetail {
  id: string;
  staff_id?: string;
  name: string;
  role: string;
  phone?: string;
  status?: string;
  joined_at?: string;
}
⋮----
<PageChrome icon="staff" title=
⋮----
onChange={(e) =>
```

## File: apps/factory-portal/app/transfers/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { ExplainStatusBanner, explainFromApiError } from '@/components/ui/ExplainStatusBanner';
import { FACTORY_TRANSFER_STATES, type StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { downloadCsv, exportCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { ListToolbar } from '@/components/ListToolbar';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { TransferFilters } from '@/components/transfers/TransferFilters';
import { TransferList, type Transfer } from '@/components/transfers/TransferList';
⋮----
const exportCsvHandler = async () =>
⋮----
description=
⋮----
label=
⋮----
<KpiStatCard label=
```

## File: apps/factory-portal/app/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { etagFromResponse } from '@pegasusx/api-core';
import {
  FACTORY_DRIVER_DUTY,
  FACTORY_QC_RESULTS,
  FACTORY_SLA_STATUSES,
  FACTORY_TRANSFER_STATES,
  FACTORY_VEHICLE_STATES,
  MANIFEST_STATES,
  canonicalizeFactoryTransfer,
  canonicalizeFactoryVehicle,
  emptyFactoryDriverDuty,
  emptyFactoryTransferCounts,
  emptyFactoryVehicleCounts,
  emptyManifestStateCounts,
  type FactoryDashboardException,
} from '@pegasusx/types';
import { shouldRefetchDashboardRollup } from '@pegasusx/ws-refresh-contract';
import { HealthStrip, SourceChip, StatusStack } from '@/components/ui';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import EmptyState from '@/components/EmptyState';
import { PageChrome } from '@/components/PageChrome';
import PageTransition from '@/components/PageTransition';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';
⋮----
import { FactoryStats } from '@/components/dashboard/types';
import { DashboardActionGrid } from '@/components/dashboard/DashboardActionGrid';
import { DashboardMetrics } from '@/components/dashboard/DashboardMetrics';
import { DashboardAlerts } from '@/components/dashboard/DashboardAlerts';
import { usePortalT } from '@/lib/i18n';
⋮----
function foldCounts<T extends string>(
  raw: unknown,
  empty: () => Record<T, number>,
  canonicalize: (key: string) => string,
): Record<T, number>
⋮----
function foldExceptions(raw: unknown): FactoryDashboardException[]
⋮----
type DashboardLoadIssue = 'offline' | 'restricted' | 'error';
⋮----
const queueLiveRefresh = () =>
⋮----
description=
⋮----
<PageChrome icon="dashboard" title=
⋮----
<p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">
```

## File: apps/factory-portal/components/loading-bay/FactoryYardGateSection.tsx
```typescript
import React, { useState } from 'react';
import { foundryFactoryApi } from '@/lib/foundry-api';
import type { YardGateCheckEvent, YardBaySlot } from '@pegasusx/types';
⋮----
interface FactoryYardGateSectionProps {
  depotId?: string;
  supplierId?: string;
}
⋮----
export default function FactoryYardGateSection({
  depotId = 'factory-tashkent-01',
  supplierId = 'supp-apex-central',
}: FactoryYardGateSectionProps)
⋮----
const handleRecordGateEvent = async (e: React.FormEvent) =>
⋮----
onClick={() =>
⋮----
onChange={(e) =>
```

## File: apps/factory-portal/components/payload-override/PayloadOverrideForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { PortalField, PortalSelect } from '@/components/portal';
import type { Manifest, Transfer } from '../../app/payload-override/page';
⋮----
const handleBackdropKeyDown = (e: React.KeyboardEvent) =>
const stopPropagation = (event: React.SyntheticEvent) =>
⋮----
<h2 className="text-lg font-semibold">{t("factory_portal.payload_override.payload_override_form.text.move_transfer")}</h2>
        <p className="text-sm text-[var(--muted)]">
          Moving <span className="font-mono">{rebalanceModal.transfer.transfer_id.slice(0, 8)}</span>
          {' '}({rebalanceModal.transfer.volume_vu} VU) to another manifest
        </p>
        <PortalField id="target-manifest" label=
```

## File: apps/factory-portal/components/portal/PortalPrimitives.tsx
```typescript
import Icon from "@/components/Icon";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import {
  PortalField as KitPortalField,
  PortalInput as KitPortalInput,
  PortalSelect as KitPortalSelect,
  PortalSection as KitPortalSection,
  PortalActions as KitPortalActions,
  SelectionCard as KitSelectionCard,
  DataList as KitDataList,
  DataListRow as KitDataListRow,
  HubCard as KitHubCard,
  FormAlert as KitFormAlert,
} from "@/components/ui/PortalPrimitives";
⋮----
export function PortalSection({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitPortalSection>, "icon"> & { icon?: string })
```

## File: apps/factory-portal/components/supply-requests/SupplyRequestList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { Fragment, useState } from 'react';
import { motion } from 'framer-motion';
import { PageSection } from '@/components/PageSection';
import type { SupplyRequest } from './types';
import { ACTIONS, PRIORITY_COLORS, STATE_COLORS } from './constants';
⋮----
interface SupplyRequestListProps {
  pageItems: SupplyRequest[];
  selectedIds: Set<string>;
  transitioning: string | null;
  handleToggleAll: () => void;
  handleToggleOne: (id: string, e: React.ChangeEvent<HTMLInputElement>) => void;
  handleTransition: (request: SupplyRequest, action: string) => void;
  qcById: Record<string, string>;
  onQC: (request: SupplyRequest, result: "PASS" | "FAIL") => void;
}
⋮----
<PageSection title=
⋮----
onClick=
⋮----
onClick={(e) =>
```

## File: apps/factory-portal/components/ui/AuthLoginCard.tsx
```typescript
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthLoginStep = "phone" | "otp";
```

## File: apps/factory-portal/components/ui/AuthRegisterStepper.tsx
```typescript
import type { ReactNode } from "react";
```

## File: apps/factory-portal/components/ui/AuthRegisterSteps.tsx
```typescript
import type { Dispatch, SetStateAction } from "react";
import { PortalField } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};
⋮----
export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};
⋮----
export type AuthProfileStep = {
  legalName: string;
  contactName: string;
  email: string;
};
⋮----
type Errors = Record<string, string>;
```

## File: apps/factory-portal/components/ui/countries.ts
```typescript
export interface AuthCountry {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export function dialCodeForCountry(code: string): string
```

## File: apps/factory-portal/components/ui/DesktopKeyboardCheatsheet.tsx
```typescript
import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";
⋮----
export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};
⋮----
export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
```

## File: apps/factory-portal/components/ui/DesktopOfflineTray.tsx
```typescript
import { useState } from "react";
import { ChevronDown, ChevronUp, Clock, RefreshCw, WifiOff } from "lucide-react";
⋮----
export type DesktopQueuedAction = {
  id: string;
  label: string;
  subtitle?: string;
};
⋮----
export type DesktopOfflineTrayProps = {
  isOffline: boolean;
  queuedActions?: DesktopQueuedAction[];
  onRetryAll?: () => void | Promise<void>;
  retrying?: boolean;
  offlineMessage?: string;
  className?: string;
};
```

## File: apps/factory-portal/components/ui/ExplainStatusBanner.tsx
```typescript
import type { StatusExplain } from '@pegasusx/types';
⋮----
export type ExplainStatusBannerProps = {
  errorCode?: string | null;
  explain?: StatusExplain | null;
  detail?: string | null;
  className?: string;
};
```

## File: apps/factory-portal/components/ui/firebase-emulator-phone.ts
```typescript
function emulatorIdentityToolkitBase(): string
⋮----
export async function sendPhoneOtpViaEmulator(
  phone: string,
  apiKey = DEFAULT_API_KEY,
): Promise<void>
⋮----
export async function verifyPhoneOtpViaEmulator(
  code: string,
  apiKey = DEFAULT_API_KEY,
): Promise<string>
⋮----
export function resetEmulatorPhoneOtpFlow(): void
```

## File: apps/factory-portal/components/ui/GenericFleetLiveMap.tsx
```typescript
import { useEffect, useMemo, useRef } from 'react';
⋮----
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import { useMapLibreTeardown } from './map-chrome';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from './useAnimatedDriverMarkers';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
export interface GenericRouteType {
  driver_location?: { lat?: number; lng?: number; latitude?: number; longitude?: number };
  live_location_available?: boolean;
  location_stale?: boolean;
  route_geometry?: any;
  manifest_id: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
}
⋮----
export type FleetLiveMapProps = {
  routes: GenericRouteType[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  enable3DView?: boolean;
};
⋮----
function routeLineFeature(
  route: GenericRouteType,
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null
⋮----
const t = (key: string) =>
⋮----
mapRef.current = ref?.getMap() ?? null;
⋮----
...mapInitialViewState(readCachedAuthSession()?.pack),
```

## File: apps/factory-portal/components/ui/HealthStrip.tsx
```typescript
export type HealthStripItem = {
  key: string;
  label: string;
  count: number;
  href?: string;
};
⋮----
export type HealthStripProps = {
  items: HealthStripItem[];
  onSelect?: (key: string) => void;
  className?: string;
};
⋮----
onClick={onSelect ? () => onSelect(item.key) : undefined}
          disabled=
```

## File: apps/factory-portal/components/ui/map-chrome.tsx
```typescript
import { useEffect, useRef, useState, type RefObject } from "react";
import type maplibregl from "maplibre-gl";
⋮----
export function useLazyMapMount(rootMargin = "120px")
⋮----
export function useMapLibreTeardown(mapRef: RefObject<maplibregl.Map | null>)
⋮----
type Map3DViewToggleProps = {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  className?: string;
};
⋮----
export function Map3DViewToggle({ enabled, onChange, className = "" }: Map3DViewToggleProps)
⋮----
onClick=
```

## File: apps/factory-portal/components/ui/motion.ts
```typescript
export type CubicBezierEasing = [number, number, number, number];
⋮----
export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};
⋮----
export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition
```

## File: apps/factory-portal/components/ui/PackChip.tsx
```typescript
type PackChipProps = {
  currency?: string | null;
  receipts?: string | null;
  title?: string;
};
⋮----
export function PackChip({ currency, receipts, title }: PackChipProps)
```

## File: apps/factory-portal/components/ui/PageChrome.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export type PageChromeProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  children: ReactNode;
  renderLoading?: () => ReactNode;
  renderError?: (message: string) => ReactNode;
  renderEmpty?: () => ReactNode;
};
```

## File: apps/factory-portal/components/ui/PortalPrimitives.tsx
```typescript
import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
⋮----
export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
{optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
⋮----
{icon ? (
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]"
              style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-secondary)" }}
              aria-hidden
            >
              {icon}
            </div>
          ) : null}
          <div className="min-w-0">
            <h2 className="bento-card-title">{title}</h2>
⋮----
export function DataListRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
})
⋮----
type LinkProps = { href: string; className?: string; children: ReactNode };
⋮----
export function HubCard({
  href,
  title,
  description,
  icon,
  onClick,
  LinkComponent,
}: {
  href?: string;
  title: string;
  description: string;
  icon?: ReactNode;
  onClick?: () => void;
  LinkComponent?: ComponentType<any>;
})
⋮----
export function FormAlert({
  variant = "info",
  icon,
  children,
}: {
  variant?: "info" | "error";
  icon?: ReactNode;
  children: ReactNode;
})
```

## File: apps/factory-portal/components/ui/PulseTimeline.tsx
```typescript
import type { PulseEvent } from '@pegasusx/types';
⋮----
export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  className?: string;
  onSelect?: (event: PulseEvent) => void;
};
⋮----
function formatWhen(iso: string): string
⋮----
export function PulseTimeline({
  events,
  loading = false,
  error = null,
  emptyMessage = 'No recent activity yet.',
  className = '',
  onSelect,
}: PulseTimelineProps)
⋮----
return <div className=
```

## File: apps/factory-portal/components/ui/SessionPackChip.tsx
```typescript
import { useMarketPack } from '@pegasusx/api-react';
import { PackChip } from './PackChip';
import { fiscalReceiptLabel } from '@pegasusx/api-core';
⋮----
receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title=
```

## File: apps/factory-portal/components/ui/SourceChip.tsx
```typescript
export type HonestySource =
  | "live"
  | "spanner"
  | "empty"
  | "unavailable"
  | "memory"
  | "env_default";
⋮----
export function sourceChipLabel(source: string): string
⋮----
export type SourceChipProps = {
  source: string;
  className?: string;
};
⋮----
export function SourceChip({ source, className }: SourceChipProps)
```

## File: apps/factory-portal/components/ui/StatusStack.tsx
```typescript
import { statusStackModel, type StatusStackMode } from "@pegasusx/types";
import { SourceChip } from "./SourceChip";
⋮----
export type StatusStackProps = {
  dictionary: readonly string[];
  counts?: Record<string, number> | null;
  available?: boolean;
  source?: string;
  onSelect?: (key: string) => void;
  labelFor?: (key: string) => string;
  className?: string;
};
⋮----
function defaultLabel(key: string): string
⋮----
function sourceForMode(mode: StatusStackMode, source?: string): string
⋮----
onClick={onSelect ? () => onSelect(row.key) : undefined}
                disabled=
```

## File: apps/factory-portal/components/ui/use-desktop-connectivity.ts
```typescript
import { useCallback, useSyncExternalStore } from "react";
import { NetworkLifecycleEngine } from "@/lib/enterprise-swr";
⋮----
export type DesktopConnectivity = {
  isOffline: boolean;
  onOnline: (listener: () => void) => () => void;
};
⋮----
export function useDesktopConnectivity(): DesktopConnectivity
```

## File: apps/factory-portal/components/ui/useAnimatedDriverMarkers.ts
```typescript
import { useEffect, useMemo, useRef, useState } from 'react';
import type { GenericRouteType } from './GenericFleetLiveMap';
⋮----
type DriverAnimState = {
  driverId: string;
  fromLng: number;
  fromLat: number;
  toLng: number;
  toLat: number;
  startMs: number;
  color: string;
  label: string;
  stale: boolean;
};
⋮----
function lerp(a: number, b: number, t: number): number
⋮----
function easeOut(t: number): number
⋮----
function driverTarget(route: GenericRouteType): { lat: number; lng: number } | null
⋮----
export function useAnimatedDriverMarkers(
  routes: GenericRouteType[],
  durationMs = 1200,
): GeoJSON.FeatureCollection<GeoJSON.Point>
⋮----
const tick = () =>
```

## File: apps/factory-portal/components/AuthGuard.tsx
```typescript
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { readTokenFromCookie } from "@/lib/auth";
import { isTauri, getStoredToken } from "@/lib/bridge";
⋮----
function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
function isTokenExpired(token: string): boolean
⋮----
export default function AuthGuard({ children }: { children: React.ReactNode })
⋮----
async function checkAuth()
```

## File: apps/factory-portal/components/HandoffTimelinePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import { PulseTimeline } from '@/components/ui/PulseTimeline';
import type { PulseEvent } from '@pegasusx/types';
import { apiFetch } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
⋮----
function isHandoffEvent(event: PulseEvent): boolean
⋮----
const handler = () =>
```

## File: apps/factory-portal/components/Icon.tsx
```typescript
import {
  LayoutDashboard, Package, Truck, Users, ClipboardCheck, Settings,
  TrendingUp, LogOut, Menu, ChevronRight, Sun, Moon, Monitor, Warehouse,
  ArrowLeft, AlertTriangle, RefreshCw, ChevronLeft, Bell, type LucideIcon,
  Boxes, ArrowDownUp, Lightbulb, Plus, CircleCheck, CircleAlert, Keyboard,
} from 'lucide-react';
```

## File: apps/factory-portal/components/LotRecallBanner.tsx
```typescript
import React, { useState } from 'react';
import { foundryFactoryApi } from '@/lib/foundry-api';
import type { RecallBlastRadiusReport } from '@pegasusx/types';
⋮----
interface LotRecallBannerProps {
  initialLotId?: string;
  cellId?: string;
  supplierId?: string;
  onQuarantineChange?: (isQuarantined: boolean, report: RecallBlastRadiusReport | null) => void;
  className?: string;
}
⋮----
const handleCheckRecall = async (targetLot: string) =>
⋮----
const handleQuarantineExecution = async () =>
⋮----
onChange=
```

## File: apps/factory-portal/components/NetworkPulsePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@/components/ui/PulseTimeline";
import type { PulseEvent } from "@pegasusx/types";
import { apiFetch } from "@/lib/auth";
⋮----
const handler = () =>
```

## File: apps/factory-portal/components/PageChrome.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";
import { PageChrome as KitPageChrome } from "@/components/ui/PageChrome";
⋮----
type PageChromeProps = {
  title: string;
  description?: string;
  icon?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyIcon?: string;
  children?: ReactNode;
};
```

## File: apps/factory-portal/components/SessionPackChip.tsx
```typescript
import { SessionPackChip as GenericSessionPackChip } from '@/components/ui/SessionPackChip';
import { readTokenFromCookie, factoryApiBaseUrl } from '@/lib/auth';
⋮----
export function SessionPackChip()
⋮----
return <GenericSessionPackChip baseUrl=
```

## File: apps/factory-portal/lib/api.ts
```typescript
import { ApiClient } from "@pegasusx/api-core";
import { readTokenFromCookie, factoryApiBaseUrl } from "@/lib/auth";
⋮----
export function createFactoryApi(): ApiClient
```

## File: apps/factory-portal/lib/enterprise-swr.ts
```typescript
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@/lib/bridge";
import { readTokenFromCookie } from "./auth";
⋮----
export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";
⋮----
type Listener = () => void;
⋮----
class EnterpriseNetworkLifecycleEngine
⋮----
constructor()
⋮----
public get isOnline(): boolean
⋮----
public get state(): NetworkState
⋮----
public get syncEpoch(): number
⋮----
private notify()
⋮----
private recordTransition()
⋮----
public setOnline(online: boolean)
⋮----
public async triggerSyncEpoch()
⋮----
public reset()
⋮----
export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  cache?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};
⋮----
export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};
⋮----
export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T>
⋮----
const bootstrap = async () =>
```

## File: apps/factory-portal/lib/foundry-api.ts
```typescript
import { createFactoryApi } from "./api";
import type {
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
  ActionExecutionRecord,
  YardGateCheckEvent,
  AMLCheckRequest,
  AMLCheckResponse,
} from "@pegasusx/types";
```

## File: apps/factory-portal/lib/portal-offline-tray.tsx
```typescript
import { DesktopOfflineTray } from "@/components/ui/DesktopOfflineTray";
import { useDesktopConnectivity } from "@/components/ui/use-desktop-connectivity";
import { isTauri } from "@pegasusx/desktop-bridge";
⋮----
export function PortalOfflineTray()
```

## File: apps/factory-portal/lib/use-pagination.ts
```typescript
import { useMemo, useState } from "react";
⋮----
export function usePagination<T>(items: T[], pageSize: number)
```

## File: apps/factory-portal/next.config.ts
```typescript
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";
⋮----
async rewrites()
```

## File: apps/retailer-app-desktop/app/(dashboard)/control-tower/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { apiFetch } from "@/lib/auth";
import { Loader2, Activity, Package, ShoppingCart, Clock, HandHelping } from "lucide-react";
⋮----
type Pulse = {
  retailer_id: string;
  generated_at: string;
  open_orders: number;
  active_fulfillments: number;
  dock_pending: number;
  pos_open_sessions: number;
  open_shifts: number;
  open_assist_tickets: number;
  low_stock_sku_bins: number;
  shift_variances_7d: number;
  sales_minor_7d: number;
  capabilities: string[];
  empty: boolean;
};
⋮----
type Tile = {
  label: string;
  value: string | number;
  href?: string;
  icon: React.ElementType;
};
⋮----
function formatMoney(minor: number)
```

## File: apps/retailer-app-desktop/app/(dashboard)/hq/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Building2, Download, Loader2, RefreshCw } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type HqSummary = {
  day?: string;
  location_count?: number;
  sku_count?: number;
  qty_sold?: number;
  qty_voided?: number;
  gross_minor?: number;
  net_minor?: number;
  currency?: string;
  honest_empty?: boolean;
};
⋮----
type LocItem = {
  location_id: string;
  qty_sold: number;
  qty_voided: number;
  gross_minor: number;
  net_minor: number;
  currency: string;
};
⋮----
type SkuItem = {
  sku_id: string;
  qty_sold: number;
  qty_voided: number;
  net_minor: number;
  is_local?: boolean;
};
⋮----
function formatMoney(minor: number)
⋮----
function todayUTC()
⋮----
const downloadCSV = async () =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/pos/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Loader2,
  ShoppingCart,
  Trash2,
  CreditCard,
  Banknote,
  AlertTriangle,
  WifiOff,
  CloudUpload,
  PauseCircle,
  PlayCircle,
  XCircle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
import {
  clearParkedPosCart,
  countPendingForSession,
  enqueueOfflinePosSale,
  flushPendingPosSales,
  listPendingPosSales,
  loadParkedPosCart,
  newClientSaleId,
  provisionalReceipt,
  saveParkedPosCart,
  type PendingPosSale,
} from "@/lib/pending-pos-sales";
import { retailerPosSaleKey } from "@pegasusx/api-core";
import { moneyCurrency, sessionPackCurrency } from "@/lib/payment-catalog";
⋮----
type Register = {
  register_id: string;
  label: string;
  location_id: string;
  status: string;
};
⋮----
type Session = {
  session_id: string;
  register_id: string;
  status: string;
  opening_float_minor: number;
  currency: string;
};
⋮----
type CartLine = {
  sku: string;
  name: string;
  qty: number;
  unit_price_minor: number;
};
⋮----
type Sale = {
  sale_id: string;
  receipt_number: string;
  total_minor: number;
  status: string;
  lines: CartLine[];
  origin?: string;
  client_sale_id?: string;
};
⋮----
type ServerHold = {
  hold_id: string;
  location_id: string;
  register_id?: string;
  status: string;
  cart: { lines?: CartLine[] } | CartLine[] | null;
  note?: string;
  expires_at?: string;
};
⋮----
function formatMoney(minor: number, currency?: string)
⋮----
const on = () =>
const off = () =>
⋮----
/* feature optional */
⋮----
const parkServerHold = async () =>
⋮----
const resumeServerHold = async (hold: ServerHold) =>
⋮----
const voidServerHold = async (holdId: string) =>
⋮----
const createRegister = async () =>
⋮----
const openSession = async () =>
⋮----
const closeSession = async () =>
⋮----
const addLine = () =>
⋮----
const completeSale = async () =>
⋮----
const syncNow = async () =>
⋮----
const voidLast = async () =>
⋮----
title=
⋮----
onChange=
⋮----
⋮----
onClick={() =>
⋮----
{/* Cart */}
⋮----
<label htmlFor="pos-cart-sku" className="sr-only">
```

## File: apps/retailer-app-desktop/app/(dashboard)/procurement/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useState, useCallback } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import {
  Building2,
  Plus,
  ChevronRight,
  Package,
  HandCoins,
  ArrowUpRight,
  TrendingDown,
  FileText,
  AlertTriangle,
  X,
  Search,
  Loader2,
  Trash2,
  RefreshCw,
} from "lucide-react";
import { Chip } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import EmptyState from "../../../components/EmptyState";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import {
  retailerSupplierAddKey,
  retailerSupplierRemoveKey,
} from "@pegasusx/api-core";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Supplier, RetailerAnalytics } from "../../../lib/types";
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
/* swallow */
⋮----
onClick=
```

## File: apps/retailer-app-desktop/app/(dashboard)/sections/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Loader2, LayoutGrid } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type Section = {
  section_id: string;
  name: string;
  location_id: string;
  aisle_tag?: string;
  shelf_tag?: string;
  status: string;
  sku_count?: number;
};
⋮----
const create = async () =>
⋮----
const loadDetail = async (id: string) =>
⋮----
const saveSkus = async () =>
⋮----
const saveStaff = async () =>
⋮----
title=
⋮----
```

## File: apps/retailer-app-desktop/app/(dashboard)/settings/family/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  ArrowRightLeft,
  Copy,
  Loader2,
  Plus,
  Trash2,
  Users,
} from "lucide-react";
import { motion } from "framer-motion";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "../../../../lib/auth";
⋮----
type FamilyMember = {
  member_id: string;
  name: string;
  phone?: string;
  created_at?: string;
};
⋮----
type MigrateItem = {
  member_id: string;
  user_id: string;
  phone: string;
  name: string;
  retailer_role: string;
  temp_password?: string;
};
⋮----
type MigrateSkipped = {
  member_id: string;
  phone?: string;
  reason: string;
};
⋮----
type MigrateResult = {
  retailer_id: string;
  migrated: MigrateItem[];
  skipped: MigrateSkipped[];
  family_remaining: number;
  family_writes: string;
};
⋮----
const addMember = async () =>
⋮----
const removeMember = async (memberId: string) =>
⋮----
const migrateToTeam = async () =>
⋮----
const copyPassword = async (userId: string, password: string) =>
⋮----
description=
⋮----
onClick=
```

## File: apps/retailer-app-desktop/app/(dashboard)/settings/locations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  MapPin,
  Star,
  Plus,
  AlertTriangle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type Location = {
  location_id: string;
  name: string;
  delivery_address?: string;
  lat?: number;
  lng?: number;
  is_primary: boolean;
  is_active: boolean;
};
⋮----
const create = async () =>
⋮----
const setPrimary = async (id: string) =>
⋮----
const switchActive = async (id: string) =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/settings/team/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft,
  Loader2,
  UserPlus,
  Users,
  Shield,
  AlertTriangle,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
⋮----
type Member = {
  user_id: string;
  name: string;
  phone: string;
  retailer_role: string;
  is_owner: boolean;
  is_active: boolean;
};
⋮----
const invite = async () =>
⋮----
const deactivate = async (userId: string) =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/shifts/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Loader2, Clock, AlertTriangle } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
import { moneyCurrency, sessionPackCurrency } from "@/lib/payment-catalog";
⋮----
type TimeEntry = {
  entry_id: string;
  user_id: string;
  location_id: string;
  status: string;
  clock_in_at?: string;
  clock_out_at?: string;
  auto_closed?: boolean;
};
⋮----
type Shift = {
  shift_id: string;
  location_id: string;
  register_id?: string;
  status: string;
  opening_float_minor: number;
  closing_cash_minor?: number;
  expected_cash_minor?: number;
  variance_minor?: number;
  currency: string;
  linked_pos_session_id?: string;
  opened_at?: string;
  closed_at?: string;
};
⋮----
function formatMoney(minor: number, currency?: string)
⋮----
const clockIn = async () =>
⋮----
const clockOut = async () =>
⋮----
const openShift = async () =>
⋮----
const closeShift = async (shiftId: string) =>
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/stock/local-skus/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Loader2, Plus, RefreshCw } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";
import { moneyCurrency, sessionPackCurrency } from "@/lib/payment-catalog";
⋮----
type LocalSKU = {
  local_sku_id: string;
  barcode?: string;
  name: string;
  unit?: string;
  default_price_minor: number;
  currency?: string;
  is_active: boolean;
};
⋮----
const create = async () =>
⋮----
const toggleActive = async (row: LocalSKU) =>
⋮----
description=
⋮----
onClick=
⋮----
onClick={() =>
```

## File: apps/retailer-app-desktop/app/(dashboard)/stock/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import {
  Loader2,
  Package,
  ArrowLeftRight,
  Plus,
  AlertTriangle,
  ClipboardList,
  RotateCcw,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { StockRequestReturnModal } from "@/components/StockRequestReturnModal";
import { apiFetch } from "@/lib/auth";
⋮----
type Balance = {
  location_id: string;
  stock_bin: string;
  sku: string;
  on_hand: number;
  reserved: number;
  available: number;
};
⋮----
type Location = {
  location_id: string;
  name: string;
  is_primary: boolean;
};
⋮----
const openRequestReturn = (sku?: string) =>
⋮----
/* ignore */
⋮----
const receiveOrder = async () =>
⋮----
const adjust = async () =>
⋮----
const transfer = async () =>
⋮----
const count = async () =>
⋮----
title=
⋮----
setClaimOpen(false);
setClaimSku(undefined);
```

## File: apps/retailer-app-desktop/app/(dashboard)/tracking/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { AlertTriangle, RefreshCw, WifiOff } from "lucide-react";
import { motion } from "framer-motion";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useLiveData } from "@/lib/hooks";
import type {
  ActiveFulfillmentsResponse,
  TrackingOrder,
  TrackingResponse,
} from "@/lib/types";
import { useOptionalWebSocket, useWsEvent, type WsMessage } from "../../../lib/ws";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "../../../components/PageSection";
import NetworkPulsePanel from "../../../components/NetworkPulsePanel";
import { TrackingMap } from "../../../components/tracking/TrackingMap";
import { TrackingStatus } from "../../../components/tracking/TrackingStatus";
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
function formatAmount(amount: number): string
⋮----
function useColorScheme(): "light" | "dark"
⋮----
const handler = (e: MediaQueryListEvent) =>
⋮----
const toggleSupplier = (id: string) =>
⋮----
<PageSection title=
```

## File: apps/retailer-app-desktop/app/auth/login/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard, dialCodeForCountry } from "@/components/ui";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
⋮----
type LoginStep = "phone" | "otp";
⋮----
export default function RetailerLoginPage()
⋮----
async function handleSendOtp(e: React.FormEvent)
⋮----
async function handleVerifyOtp(e: React.FormEvent)
⋮----
title=
⋮----
registerLabel=
⋮----
resetPhoneOtpFlow();
setStep("phone");
```

## File: apps/retailer-app-desktop/app/auth/register/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
  dialCodeForCountry,
  FormAlert,
} from "@/components/ui";
import { storeToken } from "@/lib/bridge";
import { sessionMapCenter } from "@pegasusx/api-core";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import {
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";
⋮----
function validateCurrent(): Record<string, string>
⋮----
async function next()
⋮----
function back()
⋮----
async function submit()
⋮----
title=
```

## File: apps/retailer-app-desktop/app/auth/register/wizard-state.ts
```typescript
export type StepId = "identity" | "verification" | "profile";
⋮----
export interface IdentityStep {
  countryCode: string;
  phoneLocal: string;
}
⋮----
export interface VerificationStep {
  otpCode: string;
  idToken: string;
}
⋮----
export interface ProfileStep {
  legalName: string;
  contactName: string;
  email: string;
}
⋮----
export interface WizardState {
  step: StepId;
  identity: IdentityStep;
  verification: VerificationStep;
  profile: ProfileStep;
}
⋮----
export function validateIdentity(s: IdentityStep): Record<string, string>
⋮----
export function validateVerification(s: VerificationStep): Record<string, string>
⋮----
export function validateProfile(s: ProfileStep): Record<string, string>
```

## File: apps/retailer-app-desktop/components/auto-order/AutoOrderRules.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { Building2, Layers, Package, Box, CheckCircle2 } from "lucide-react";
import { PageSection } from "@/components/PageSection";
import type { AutoOrderSettings } from "@/lib/types";
⋮----
onChange=
⋮----
<PageSection title=
```

## File: apps/retailer-app-desktop/components/catalog/CatalogFilters.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { Search, SlidersHorizontal } from "lucide-react";
import { Skeleton } from "../Skeleton";
import type { Supplier } from "../../lib/types";
⋮----
export interface CatalogFiltersProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  hasActiveFilters: boolean;
  clearFilters: () => void;
  categoryTabs: string[];
  activeCategory: string;
  setActiveCategory: (category: string) => void;
  categorySuppliersLoading: boolean;
  categorySuppliersError: string | null;
  categorySuppliers: Supplier[];
  activeSupplier: string;
  setActiveSupplier: (supplierId: string) => void;
  supplierList: Supplier[];
}
⋮----
onClick=
```

## File: apps/retailer-app-desktop/components/dashboard/CommandBoard.tsx
```typescript
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ORDER_STATUS_FUNNEL, type RetailerControlTowerPulse } from "@pegasusx/types";
import { SourceChip, StatusStack } from "@/components/ui";
import { LoyaltyCard } from "../LoyaltyCard";
import { moneyCurrency } from "../../lib/payment-catalog";
⋮----
type CommandBoardProps = {
  pulse: RetailerControlTowerPulse | null;
  loading: boolean;
  error?: string | null;
};
⋮----
function salesLabel(minor: number): string
```

## File: apps/retailer-app-desktop/components/insights/SellThroughPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { DemandSourceChips } from "@/components/ui/DemandSourceChips";
import { Loader2 } from "lucide-react";
import { apiFetch } from "@/lib/auth";
⋮----
type SellThroughItem = {
  retailer_id?: string;
  location_id?: string;
  sku_id: string;
  day: string;
  qty_sold: number;
  qty_voided?: number;
  net_sold: number;
  qty_on_hand_eod?: number;
  source?: string;
};
⋮----
type SellThroughResponse = {
  source?: string;
  items?: SellThroughItem[];
};
```

## File: apps/retailer-app-desktop/components/portal/PortalPrimitives.tsx
```typescript
import Icon from "@/components/Icon";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import {
  PortalField as KitPortalField,
  PortalInput as KitPortalInput,
  PortalSelect as KitPortalSelect,
  PortalSection as KitPortalSection,
  PortalActions as KitPortalActions,
  SelectionCard as KitSelectionCard,
  DataList as KitDataList,
  DataListRow as KitDataListRow,
  HubCard as KitHubCard,
  FormAlert as KitFormAlert,
} from "@/components/ui/PortalPrimitives";
⋮----
export function PortalSection({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitPortalSection>, "icon"> & { icon?: string })
```

## File: apps/retailer-app-desktop/components/settings/ProfileFields.tsx
```typescript
import { AlertTriangle } from "lucide-react";
import { normalizeReceivingWindow } from "../../lib/receiving-window";
⋮----
aria-invalid=
⋮----
onChange=
```

## File: apps/retailer-app-desktop/components/ui/AuthLoginCard.tsx
```typescript
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthLoginStep = "phone" | "otp";
```

## File: apps/retailer-app-desktop/components/ui/AuthRegisterStepper.tsx
```typescript
import type { ReactNode } from "react";
```

## File: apps/retailer-app-desktop/components/ui/AuthRegisterSteps.tsx
```typescript
import type { Dispatch, SetStateAction } from "react";
import { PortalField } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};
⋮----
export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};
⋮----
export type AuthProfileStep = {
  legalName: string;
  contactName: string;
  email: string;
};
⋮----
type Errors = Record<string, string>;
```

## File: apps/retailer-app-desktop/components/ui/countries.ts
```typescript
export interface AuthCountry {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export function dialCodeForCountry(code: string): string
```

## File: apps/retailer-app-desktop/components/ui/DemandSourceChips.tsx
```typescript
import type { CSSProperties } from "react";
⋮----
export type DemandSourceCode = "STORE_POS" | "WHOLESALE_HISTORY" | string;
⋮----
export function demandSourceLabel(code: string): string
⋮----
export function normalizeDemandSources(sources?: string[] | null): string[]
⋮----
export type DemandSourceChipsProps = {
  sources?: string[] | null;
  className?: string;
  size?: "sm" | "md";
};
```

## File: apps/retailer-app-desktop/components/ui/DesktopKeyboardCheatsheet.tsx
```typescript
import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";
⋮----
export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};
⋮----
export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
```

## File: apps/retailer-app-desktop/components/ui/DesktopOfflineTray.tsx
```typescript
import { useState } from "react";
import { ChevronDown, ChevronUp, Clock, RefreshCw, WifiOff } from "lucide-react";
⋮----
export type DesktopQueuedAction = {
  id: string;
  label: string;
  subtitle?: string;
};
⋮----
export type DesktopOfflineTrayProps = {
  isOffline: boolean;
  queuedActions?: DesktopQueuedAction[];
  onRetryAll?: () => void | Promise<void>;
  retrying?: boolean;
  offlineMessage?: string;
  className?: string;
};
```

## File: apps/retailer-app-desktop/components/ui/ExplainStatusBanner.tsx
```typescript
import type { StatusExplain } from '@pegasusx/types';
⋮----
export type ExplainStatusBannerProps = {
  errorCode?: string | null;
  explain?: StatusExplain | null;
  detail?: string | null;
  className?: string;
};
```

## File: apps/retailer-app-desktop/components/ui/firebase-emulator-phone.ts
```typescript
function emulatorIdentityToolkitBase(): string
⋮----
export async function sendPhoneOtpViaEmulator(
  phone: string,
  apiKey = DEFAULT_API_KEY,
): Promise<void>
⋮----
export async function verifyPhoneOtpViaEmulator(
  code: string,
  apiKey = DEFAULT_API_KEY,
): Promise<string>
⋮----
export function resetEmulatorPhoneOtpFlow(): void
```

## File: apps/retailer-app-desktop/components/ui/GenericFleetLiveMap.tsx
```typescript
import { useEffect, useMemo, useRef } from 'react';
⋮----
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import { useMapLibreTeardown } from './map-chrome';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from './useAnimatedDriverMarkers';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
export interface GenericRouteType {
  driver_location?: { lat?: number; lng?: number; latitude?: number; longitude?: number };
  live_location_available?: boolean;
  location_stale?: boolean;
  route_geometry?: any;
  manifest_id: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
}
⋮----
export type FleetLiveMapProps = {
  routes: GenericRouteType[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  enable3DView?: boolean;
};
⋮----
function routeLineFeature(
  route: GenericRouteType,
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null
⋮----
const t = (key: string) =>
⋮----
mapRef.current = ref?.getMap() ?? null;
⋮----
...mapInitialViewState(readCachedAuthSession()?.pack),
```

## File: apps/retailer-app-desktop/components/ui/HandoffCard.tsx
```typescript
import type { HandoffCardMetadata } from '@pegasusx/types';
⋮----
export type HandoffCardProps = {
  metadata: HandoffCardMetadata;
  className?: string;
  onAction?: (link: string) => void;
};
```

## File: apps/retailer-app-desktop/components/ui/HealthStrip.tsx
```typescript
export type HealthStripItem = {
  key: string;
  label: string;
  count: number;
  href?: string;
};
⋮----
export type HealthStripProps = {
  items: HealthStripItem[];
  onSelect?: (key: string) => void;
  className?: string;
};
⋮----
onClick={onSelect ? () => onSelect(item.key) : undefined}
          disabled=
```

## File: apps/retailer-app-desktop/components/ui/map-chrome.tsx
```typescript
import { useEffect, useRef, useState, type RefObject } from "react";
import type maplibregl from "maplibre-gl";
⋮----
export function useLazyMapMount(rootMargin = "120px")
⋮----
export function useMapLibreTeardown(mapRef: RefObject<maplibregl.Map | null>)
⋮----
type Map3DViewToggleProps = {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  className?: string;
};
⋮----
export function Map3DViewToggle({ enabled, onChange, className = "" }: Map3DViewToggleProps)
⋮----
onClick=
```

## File: apps/retailer-app-desktop/components/ui/motion.ts
```typescript
export type CubicBezierEasing = [number, number, number, number];
⋮----
export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};
⋮----
export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition
```

## File: apps/retailer-app-desktop/components/ui/PackChip.tsx
```typescript
type PackChipProps = {
  currency?: string | null;
  receipts?: string | null;
  title?: string;
};
⋮----
export function PackChip({ currency, receipts, title }: PackChipProps)
```

## File: apps/retailer-app-desktop/components/ui/PageChrome.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export type PageChromeProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  children: ReactNode;
  renderLoading?: () => ReactNode;
  renderError?: (message: string) => ReactNode;
  renderEmpty?: () => ReactNode;
};
```

## File: apps/retailer-app-desktop/components/ui/PageSkeletons.tsx
```typescript
import type { CSSProperties } from "react";
```

## File: apps/retailer-app-desktop/components/ui/PortalPrimitives.tsx
```typescript
import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
⋮----
export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
{optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
⋮----
{icon ? (
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]"
              style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-secondary)" }}
              aria-hidden
            >
              {icon}
            </div>
          ) : null}
          <div className="min-w-0">
            <h2 className="bento-card-title">{title}</h2>
⋮----
export function DataListRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
})
⋮----
type LinkProps = { href: string; className?: string; children: ReactNode };
⋮----
export function HubCard({
  href,
  title,
  description,
  icon,
  onClick,
  LinkComponent,
}: {
  href?: string;
  title: string;
  description: string;
  icon?: ReactNode;
  onClick?: () => void;
  LinkComponent?: ComponentType<any>;
})
⋮----
export function FormAlert({
  variant = "info",
  icon,
  children,
}: {
  variant?: "info" | "error";
  icon?: ReactNode;
  children: ReactNode;
})
```

## File: apps/retailer-app-desktop/components/ui/PulseTimeline.tsx
```typescript
import type { PulseEvent } from '@pegasusx/types';
⋮----
export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  className?: string;
  onSelect?: (event: PulseEvent) => void;
};
⋮----
function formatWhen(iso: string): string
⋮----
export function PulseTimeline({
  events,
  loading = false,
  error = null,
  emptyMessage = 'No recent activity yet.',
  className = '',
  onSelect,
}: PulseTimelineProps)
⋮----
return <div className=
```

## File: apps/retailer-app-desktop/components/ui/SessionPackChip.tsx
```typescript
import { useMarketPack } from '@pegasusx/api-react';
import { PackChip } from './PackChip';
import { fiscalReceiptLabel } from '@pegasusx/api-core';
⋮----
receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title=
```

## File: apps/retailer-app-desktop/components/ui/SourceChip.tsx
```typescript
export type HonestySource =
  | "live"
  | "spanner"
  | "empty"
  | "unavailable"
  | "memory"
  | "env_default";
⋮----
export function sourceChipLabel(source: string): string
⋮----
export type SourceChipProps = {
  source: string;
  className?: string;
};
⋮----
export function SourceChip({ source, className }: SourceChipProps)
```

## File: apps/retailer-app-desktop/components/ui/StatusStack.tsx
```typescript
import { statusStackModel, type StatusStackMode } from "@pegasusx/types";
import { SourceChip } from "./SourceChip";
⋮----
export type StatusStackProps = {
  dictionary: readonly string[];
  counts?: Record<string, number> | null;
  available?: boolean;
  source?: string;
  onSelect?: (key: string) => void;
  labelFor?: (key: string) => string;
  className?: string;
};
⋮----
function defaultLabel(key: string): string
⋮----
function sourceForMode(mode: StatusStackMode, source?: string): string
⋮----
onClick={onSelect ? () => onSelect(row.key) : undefined}
                disabled=
```

## File: apps/retailer-app-desktop/components/ui/use-desktop-connectivity.ts
```typescript
import { useCallback, useSyncExternalStore } from "react";
import { NetworkLifecycleEngine } from "@/lib/enterprise-swr";
⋮----
export type DesktopConnectivity = {
  isOffline: boolean;
  onOnline: (listener: () => void) => () => void;
};
⋮----
export function useDesktopConnectivity(): DesktopConnectivity
```

## File: apps/retailer-app-desktop/components/ui/useAnimatedDriverMarkers.ts
```typescript
import { useEffect, useMemo, useRef, useState } from 'react';
import type { GenericRouteType } from './GenericFleetLiveMap';
⋮----
type DriverAnimState = {
  driverId: string;
  fromLng: number;
  fromLat: number;
  toLng: number;
  toLat: number;
  startMs: number;
  color: string;
  label: string;
  stale: boolean;
};
⋮----
function lerp(a: number, b: number, t: number): number
⋮----
function easeOut(t: number): number
⋮----
function driverTarget(route: GenericRouteType): { lat: number; lng: number } | null
⋮----
export function useAnimatedDriverMarkers(
  routes: GenericRouteType[],
  durationMs = 1200,
): GeoJSON.FeatureCollection<GeoJSON.Point>
⋮----
const tick = () =>
```

## File: apps/retailer-app-desktop/components/ui/VirtualScrollList.tsx
```typescript
import type { CSSProperties, ReactNode } from "react";
⋮----
export type VirtualScrollListProps<T> = {
  items: readonly T[];
  height?: CSSProperties["height"];
  className?: string;
  itemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  virtualizationThreshold?: number;
};
⋮----
export function VirtualScrollList<T>({
  items,
  height = "28rem",
  className,
  itemKey,
  renderItem,
}: VirtualScrollListProps<T>)
⋮----
<div key=
```

## File: apps/retailer-app-desktop/components/CartDrawer.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import {
  X,
  ShoppingBag,
  Package,
  CreditCard,
  ChevronRight,
  Minus,
  Plus,
  Trash2,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { packCurrency, readCachedAuthSession } from "@pegasusx/api-core";
import { useCart } from "../lib/cart";
import { effectiveCartMaxQuantity } from "../lib/stock-policy";
⋮----
interface CartDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  onCheckout: () => void;
}
⋮----
updateQuantity(item.product_id, item.quantity - 1)
⋮----
updateQuantity(item.product_id, item.quantity + 1)
⋮----
onClick=
```

## File: apps/retailer-app-desktop/components/FileClaimPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Camera, Loader2 } from "lucide-react";
import { claimFileKey } from "@pegasusx/api-core";
import {
  claimTypeNeedsPhoto,
  fileOrderClaim,
  getClaimEligibility,
  listOrderClaims,
  uploadClaimPhoto,
  type ClaimEligibility,
  type RetailerClaim,
} from "../lib/api";
import { moneyCurrency } from "../lib/payment-catalog";
import type { Order } from "../lib/types";
⋮----
function formatEligibleUntil(endsAt: string | null | undefined): string | null
⋮----
type Props = {
  order: Order;
  onFiled?: (claim: RetailerClaim) => void;
  initialSku?: string;
};
⋮----
// Allow submit when eligibility unknown (fetch failed) — server still enforces window.
⋮----
const onPickFile = async (file: File | null) =>
⋮----
const submit = async () =>
⋮----
setQtyBySku((m) => (
⋮----
```

## File: apps/retailer-app-desktop/components/Icon.tsx
```typescript
import {
  LayoutDashboard,
  ShoppingCart,
  PackageSearch,
  MapPin,
  Container,
  Activity,
  BarChart3,
  Settings,
  Bell,
  LogOut,
  Menu,
  ChevronRight,
  Sun,
  Moon,
  Monitor,
  ArrowLeft,
  AlertTriangle,
  RefreshCw,
  ChevronLeft,
  Plus,
  CircleCheck,
  CircleAlert,
  Store,
  CreditCard,
  Users,
  Keyboard,
  type LucideIcon,
} from "lucide-react";
⋮----
export default function Icon({
  name,
  size = 24,
  className = "",
}: {
  name: string;
  size?: number;
  className?: string;
})
```

## File: apps/retailer-app-desktop/components/KpiStatCard.tsx
```typescript
import type { ReactNode } from "react";
⋮----
type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  className?: string;
  align?: "start" | "center";
  icon?: ReactNode;
};
```

## File: apps/retailer-app-desktop/components/NetworkPulsePanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@/components/ui/PulseTimeline";
import type { PulseEvent } from "@pegasusx/types";
import { apiFetch } from "../lib/auth";
⋮----
export default function NetworkPulsePanel({ className }: { className?: string })
```

## File: apps/retailer-app-desktop/components/PageChrome.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "@/components/ui/PageSkeletons";
import { PageChrome as KitPageChrome } from "@/components/ui/PageChrome";
⋮----
type EmptyVariant =
  | "no-data"
  | "no-results"
  | "offline"
  | "restricted"
  | "error"
  | "no-orders"
  | "no-products"
  | "no-predictions"
  | "no-suppliers";
⋮----
type PageChromeProps = {
  title: string;
  description?: string;
  icon?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form" | "catalog";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyVariant?: EmptyVariant;
  children?: ReactNode;
  className?: string;
};
```

## File: apps/retailer-app-desktop/components/PageTransition.tsx
```typescript
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { duration, easing, reducedMotionTransition } from "@/components/ui/motion";
import { ReactNode } from "react";
import { usePathname } from "next/navigation";
⋮----
interface PageTransitionProps {
  children: ReactNode;
  className?: string;
}
⋮----
export default function PageTransition({
  children,
  className = "",
}: PageTransitionProps)
⋮----
// Calm, high-fidelity transition: subtle fade + layout shift
```

## File: apps/retailer-app-desktop/components/ProductDetailDrawer.tsx
```typescript
import { useEffect } from "react";
import {
  X,
  Package,
  ShoppingCart,
  Building2,
  Layers,
  Info,
  ChevronRight,
} from "lucide-react";
import { Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { useCart } from "../lib/cart";
import { isCatalogBlocked } from "../lib/stock-policy";
import type { Product, Variant } from "../lib/types";
import { packCurrency, readCachedAuthSession } from "@pegasusx/api-core";
import {
  productDisplayPrice,
  productListPrice,
  productSalePrice,
} from "../lib/types";
⋮----
interface ProductDetailDrawerProps {
  product: Product | null;
  isOpen: boolean;
  onClose: () => void;
}
⋮----
const onKey = (e: KeyboardEvent) =>
⋮----
{productListPrice(product).toLocaleString()}
⋮----
{productDisplayPrice(product).toLocaleString()}
```

## File: apps/retailer-app-desktop/components/SessionPackChip.tsx
```typescript
import { SessionPackChip as GenericSessionPackChip } from '@/components/ui/SessionPackChip';
import { readToken, retailerApiBaseUrl } from '@/lib/auth';
⋮----
export function SessionPackChip()
⋮----
return <GenericSessionPackChip baseUrl=
```

## File: apps/retailer-app-desktop/components/ShopClosedModal.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import { AlertTriangle, Camera, Loader2 } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { apiFetch } from "../lib/auth";
import { uploadClaimPhoto } from "../lib/api";
import { retailerShopClosedResponseKey } from "@pegasusx/api-core";
import { useWebSocket, useWsEvent, type WsMessage } from "../lib/ws";
⋮----
type ShopClosedAlert = {
  order_id: string;
  driver_name: string;
  attempt_id?: string;
  options: string[];
};
⋮----
function optionLabel(option: string): string
⋮----
function messageToAlert(msg: WsMessage): ShopClosedAlert | null
⋮----
function formatShopClosedError(raw: string): string
⋮----
onClick=
```

## File: apps/retailer-app-desktop/components/StockRequestReturnModal.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, RotateCcw, Search, X } from "lucide-react";
import { apiFetch } from "@/lib/auth";
import { getRetailerProfile } from "@/lib/retailer-profile";
import type { LineItem, Order, TrackingOrder } from "@/lib/types";
import { FileClaimPanel } from "./FileClaimPanel";
⋮----
type ListOrderRow = {
  order_id: string;
  retailer_id?: string;
  supplier_id?: string;
  amount?: number;
  total_amount?: number;
  total_minor?: number;
  payment_gateway?: string;
  payment_status?: string;
  state?: string;
  status?: string;
  route_id?: string | null;
  order_source?: string | null;
  auto_confirm_at?: string | null;
  deliver_before?: string | null;
  delivery_token?: string | null;
  version?: number;
  created_at?: string;
  items?: Array<{
    line_item_id?: string;
    order_id?: string;
    sku_id?: string;
    sku_name?: string;
    product_id?: string;
    product_name?: string;
    quantity: number;
    unit_price: number;
    status?: string;
  }>;
};
⋮----
type Props = {
  open: boolean;
  onClose: () => void;
  preferredSku?: string;
};
⋮----
function orderState(row: ListOrderRow): string
⋮----
function mapItems(orderId: string, row: ListOrderRow): LineItem[]
⋮----
function toOrder(row: ListOrderRow): Order
⋮----
function trackingToOrder(t: TrackingOrder): Order
⋮----
const pickOrder = async (order: Order) =>
⋮----
aria-label=
⋮----
onChange=
```

## File: apps/retailer-app-desktop/lib/enterprise-swr.ts
```typescript
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import { getRetailerId } from "./retailer-profile";
⋮----
export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";
⋮----
type Listener = () => void;
⋮----
class EnterpriseNetworkLifecycleEngine
⋮----
constructor()
⋮----
public get isOnline(): boolean
⋮----
public get state(): NetworkState
⋮----
public get syncEpoch(): number
⋮----
private notify()
⋮----
private recordTransition()
⋮----
public setOnline(online: boolean)
⋮----
public async triggerSyncEpoch()
⋮----
public reset()
⋮----
export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  cache?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};
⋮----
export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};
⋮----
export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T>
⋮----
// Zero-Jitter Offline Gating: Do not attempt network queries if offline
⋮----
// NON-DESTRUCTIVE: Keep data intact! Never wipe data on background revalidation failure.
⋮----
// Initial Bootstrap & Hydration
⋮----
const bootstrap = async () =>
⋮----
// Coordinated Sync Epoch Subscription
⋮----
// Viewport awareness: Only revalidate if document is visible
```

## File: apps/retailer-app-desktop/lib/hooks.ts
```typescript
import { useCallback } from "react";
import { apiFetch } from "./auth";
import { useEnterpriseSWR } from "./enterprise-swr";
⋮----
type LiveDataError = Error & { status?: number };
⋮----
type MutateOptions = {
  silent?: boolean;
};
⋮----
type UseLiveDataOptions = {
  cache?: boolean;
};
⋮----
export function useLiveData<T>(
  url: string,
  interval = 0,
  options: UseLiveDataOptions = {},
)
```

## File: apps/retailer-app-desktop/lib/pending-pos-flusher.tsx
```typescript
import { useEffect } from "react";
import { flushPendingPosSales } from "./pending-pos-sales";
⋮----
export function PendingPosFlusher()
⋮----
const run = () =>
⋮----
const handleVisibility = () =>
```

## File: apps/retailer-app-desktop/lib/pending-pos-sales.ts
```typescript
import { apiFetch } from "./auth";
import {
  clearParkedPosCart,
  countActivePendingPosSales,
  countPendingForSession,
  enqueuePendingPosSale as cacheEnqueue,
  isRetryablePosSyncError,
  listPendingPosSales as cacheList,
  loadParkedPosCart,
  removePendingPosSale as cacheRemove,
  saveParkedPosCart,
  updatePendingPosSale as cacheUpdate,
  type ParkedPosCart,
  type PendingPosSale,
} from "@pegasusx/desktop-cache";
import { retailerPosSaleKey } from '@pegasusx/api-core';
⋮----
export async function listPendingPosSales(): Promise<PendingPosSale[]>
⋮----
export async function enqueueOfflinePosSale(input: {
  clientSaleId: string;
  clientReceipt: string;
  sessionId: string;
  payload: unknown;
}): Promise<PendingPosSale>
⋮----
export async function removePendingPosSale(id: string): Promise<void>
⋮----
export async function flushPendingPosSales(): Promise<{
  flushed: number;
  failed: number;
  remaining: number;
}>
⋮----
export function newClientSaleId(): string
⋮----
export function provisionalReceipt(seq: number): string
```

## File: apps/retailer-app-desktop/lib/retailer-offline-tray.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import {
  DesktopOfflineTray,
  useDesktopConnectivity,
  type DesktopQueuedAction,
} from "@/components/ui";
import { isTauri } from "@pegasusx/desktop-bridge";
import { flushPendingCheckouts, listPendingCheckouts } from "./pending-checkout";
⋮----
export function RetailerOfflineTray()
⋮----
const onOnline = () =>
⋮----
const onRetryAll = async () =>
```

## File: apps/retailer-app-desktop/lib/stock-policy.ts
```typescript
import type { Product } from './types';
⋮----
export interface StockAwareProduct {
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  available_stock?: number;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
}
⋮----
export function isCatalogBlocked(product: StockAwareProduct): boolean
⋮----
export function productMaxQuantity(product: StockAwareProduct): number | null
⋮----
export function effectiveCartMaxQuantity(
  item: StockAwareProduct & { product_id?: string },
  previewCaps?: Record<string, number>,
  previewStockPolicyReject = true,
): number | null
⋮----
export function orderableCapsFromPreview(preview: {
  orderable_quantities?: Record<string, number>;
  max_quantities?: Record<string, number>;
}): Record<string, number> | undefined
⋮----
export function clampCartQuantity(
  product: StockAwareProduct,
  quantity: number,
  previewCaps?: Record<string, number>,
  previewStockPolicyReject = true,
): number
⋮----
export function stockMetaFromProduct(product: Product): StockAwareProduct
```

## File: apps/retailer-app-desktop/lib/types.ts
```typescript
export interface LineItem {
  line_item_id: string;
  order_id: string;
  sku_id: string;
  sku_name?: string;
  quantity: number;
  unit_price: number;
  status: string;
}
⋮----
export interface Order {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  amount: number;
  payment_gateway: string;
  payment_status: string;
  state: string;
  route_id: string | null;
  order_source: string | null;
  confirmation_status?: string | null;
  preorder_badge?: string | null;
  proposed_delivery_date?: string | null;
  delivery_proposal_reason?: string | null;
  auto_confirm_at: string | null;
  deliver_before: string | null;
  delivery_token: string | null;
  customer_stage?: string | null;
  customer_progress_percent?: number | null;
  version: number;
  created_at: string;
  items?: LineItem[];
}
⋮----
export interface ProcurementOrderResponse {
  status: "PROCUREMENT_AUTHORIZED";
  order_id: string;
  retailer_id: string;
  supplier_id?: string;
  state: string;
  amount: number;
  total: number;
  currency: string;
  order_source: "PROCUREMENT";
  created_at: string;
}
⋮----
export interface Variant {
  id: string;
  size: string;
  pack: string;
  pack_count: number;
  weight_per_unit: string;
  price: number;
}
⋮----
export interface ProductOffer {
  product_id?: string;
  list_price_minor?: number;
  sale_price_minor?: number;
  discount_bps?: number;
  promotion_label?: string;
}
⋮----
export interface Product {
  id: string;
  product_id?: string;
  name: string;
  description: string;
  nutrition?: string;
  image_url: string;
  supplier_id: string;
  supplier_name: string;
  supplier_category: string;
  category_id: string;
  category_name: string;
  sell_by_block: boolean;
  units_per_block: number;
  price: number;
  variants?: Variant[];
  available_stock?: number;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
  offer?: ProductOffer;
}
⋮----
export function productSalePrice(product: Product): number | null
⋮----
export function productListPrice(product: Product): number
⋮----
export function productDisplayPrice(product: Product): number
⋮----
export interface Category {
  id: string;
  name: string;
  icon: string;
  product_count: number;
  supplier_count: number;
}
⋮----
export interface Supplier {
  id: string;
  name: string;
  logo_url: string;
  category: string;
  primary_category_id?: string;
  operating_category_ids?: string[];
  operating_category_names?: string[];
  order_count: number;
  is_active: boolean;
}
⋮----
export interface MonthlyExpense {
  month: string;
  total: number;
}
export interface TopSupplier {
  supplier_id: string;
  supplier_name: string;
  total: number;
  order_count: number;
}
export interface TopProduct {
  product_id: string;
  product_name: string;
  total: number;
  quantity: number;
}
export interface RetailerAnalytics {
  monthly_expenses: MonthlyExpense[];
  top_suppliers: TopSupplier[];
  top_products: TopProduct[];
  total_this_month: number;
  total_last_month: number;
}
⋮----
export interface Prediction {
  id: string;
  retailer_id?: string;
  retailerId: string;
  product_id?: string;
  product_name?: string;
  predicted_amount?: number;
  predicted_quantity?: number;
  trigger_date?: string;
  suggested_order_date?: string;
  predictedAmount: number;
  triggerDate: string;
  status: string;
  productName: string;
  predictedQuantity: number;
  confidence: number;
  reasoning: string;
  suggestedOrderDate: string;
  blocked?: boolean;
  blocked_reason?: string;
  label?: "insufficient_history" | "early_signal" | "standard" | string;
}
⋮----
export function isPredictionBlocked(
  item: Pick<Prediction, "blocked" | "blocked_reason" | "label">,
): boolean
⋮----
export interface RetailerAILineItem {
  sku: string;
  name?: string;
  quantity: number;
  unit_price_minor: number;
}
⋮----
export interface RetailerAIPrediction {
  order_id: string;
  order_source?: string;
  confirmation_status?: string;
  requested_delivery_date?: string;
  auto_confirm_at?: string;
  total_minor: number;
  currency: string;
  derived_from_order_id?: string;
  updated_at: string;
  line_items?: RetailerAILineItem[];
}
⋮----
export interface RetailerAIPredictionsResponse {
  items: RetailerAIPrediction[];
}
⋮----
export function aiPredictionTitle(item: RetailerAIPrediction): string
⋮----
export function aiPredictionQty(item: RetailerAIPrediction): number
⋮----
export function formatMinorAmount(minor: number, currency?: string): string
⋮----
/* ── Auto-Order Settings ── */
export interface SupplierOverride {
  supplier_id: string;
  enabled: boolean;
  has_history: boolean;
  analytics_start_date?: string;
}
export interface CategoryOverride {
  category_id: string;
  enabled: boolean;
  has_history: boolean;
  analytics_start_date?: string;
}
export interface ProductOverride {
  product_id: string;
  enabled: boolean;
}
export interface VariantOverride {
  variant_id: string;
  enabled: boolean;
}
export type AutoOrderExecutionMode = "off" | "shadow" | "draft" | "place";
⋮----
export interface AutoOrderShadowStats {
  proposal_count: number;
  matched_orders: number;
  wape: number;
  unmodified_accept_rate: number;
  window_days: number;
}
⋮----
export interface AutoOrderShadowProposal {
  proposal_id: string;
  retailer_id: string;
  sku: string;
  supplier_id?: string;
  proposed_qty: number;
  ip: number;
  reorder_point: number;
  order_up_to: number;
  confidence?: number;
  reason?: string;
  bucket_date: string;
  status: string;
  run_id?: string;
  created_at?: string;
}
⋮----
export interface AutoOrderSettings {
  global_enabled: boolean;
  execution_mode?: AutoOrderExecutionMode | string;
  has_any_history: boolean;
  analytics_start_date?: string;
  supplier_overrides: SupplierOverride[];
  category_overrides: CategoryOverride[];
  product_overrides: ProductOverride[];
  variant_overrides: VariantOverride[];
  shadow_stats?: AutoOrderShadowStats;
}
⋮----
export interface AutoOrderSkip {
  sku?: string;
  reason: string;
}
⋮----
export interface AutoOrderPlacedOrder {
  order_id: string;
  supplier_id?: string;
  line_count: number;
  total_minor: number;
  skus?: string[];
}
⋮----
export interface AutoOrderRun {
  run_id: string;
  retailer_id: string;
  started_at: string;
  finished_at?: string;
  mode: AutoOrderExecutionMode | string;
  draft_lines: number;
  placed_lines?: number;
  placed_orders?: AutoOrderPlacedOrder[];
  skipped?: AutoOrderSkip[];
  status: string;
  message?: string;
  suggestions_seen?: number;
  schedule_bucket?: string;
  candidate_source?: string;
}
⋮----
export interface AutoOrderRunsResponse {
  items: AutoOrderRun[];
}
⋮----
export interface FamilyMigrateItem {
  member_id: string;
  user_id: string;
  phone: string;
  name: string;
  retailer_role: string;
  temp_password?: string;
}
⋮----
export interface FamilyMigrateSkipped {
  member_id: string;
  phone?: string;
  reason: string;
}
⋮----
export interface FamilyMigrateResult {
  retailer_id: string;
  migrated: FamilyMigrateItem[];
  skipped: FamilyMigrateSkipped[];
  family_remaining: number;
  family_writes: string;
}
⋮----
export interface FamilyMembersListResponse {
  members: Array<{
    member_id: string;
    name: string;
    phone?: string;
    created_at?: string;
  }>;
  family_writes?: "open" | "gone" | string;
  migrate?: string;
}
⋮----
export interface SupplierOrderResult {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  total: number;
  item_count: number;
}
⋮----
export interface StockWarning {
  sku: string;
  requested: number;
  available: number;
  backorder_qty: number;
  accepts_backorder: boolean;
}
⋮----
export interface CheckoutPreviewResponse {
  ok: boolean;
  blocked?: boolean;
  code?: string;
  message?: string;
  rejected_skus?: string[];
  oos_items?: string[];
  shortfall?: Record<string, number>;
  stock_warnings?: StockWarning[];
  max_quantities?: Record<string, number>;
  orderable_quantities?: Record<string, number>;
  line_errors?: Record<string, string>;
  backordered_item_count?: number;
  show_stock_counts?: boolean;
  preorder_min_lead_days?: number;
  preorder_max_lead_days?: number;
  order_line_min_quantity?: number;
  order_line_max_quantity?: number;
  delivery_fee_minor?: number;
  delivery_distance_km?: number;
  default_out_of_stock_policy?: string;
  checkout_policy_token?: string;
  checkout_policy_expires_at?: string;
  order_acceptance_open?: boolean;
  order_acceptance_window_label?: string;
  next_order_acceptance_at?: string;
}
⋮----
export interface UnifiedCheckoutResponse {
  status: string;
  invoice_id: string;
  total: number;
  currency?: string;
  supplier_orders: SupplierOrderResult[];
  backordered_item_count?: number;
  backorder_order_id?: string;
  stock_warnings?: StockWarning[];
}
⋮----
export interface OrderCurrencyOptions {
  enabled: boolean;
  operating_currency: string;
  allowlist: string[];
}
⋮----
export interface CashCheckoutResponse {
  order_id: string;
  state: string;
  amount: number;
  driver_id?: string;
  retailer_id: string;
  message: string;
}
⋮----
export interface CardCheckoutResponse {
  order_id: string;
  state: string;
  amount: number;
  gateway: string;
  resolved_gateway?: string;
  policy_source?: string;
  allowed_gateways?: string[];
  policy_reason?: string;
  payment_url: string;
  invoice_id: string;
  session_id?: string;
  attempt_id?: string;
  attempt_no?: number;
  retailer_id: string;
  message: string;
}
⋮----
export interface ActiveFulfillmentItem {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  state: string;
  adjusted_amount: number;
  item_count: number;
}
⋮----
export interface ActiveFulfillmentsResponse {
  fulfillments: ActiveFulfillmentItem[];
  count: number;
}
⋮----
export interface PendingPaymentSession {
  session_id?: string;
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  gateway?: string;
  locked_amount: number;
  currency: string;
  status: string;
  current_attempt_no: number;
  invoice_id?: string;
  redirect_url?: string;
  expires_at?: string;
  created_at: string;
  updated_at?: string;
}
⋮----
export interface PendingPaymentsResponse {
  pending_payments: PendingPaymentSession[];
  count: number;
}
⋮----
export interface CancelOrderRequest {
  order_id: string;
  retailer_id: string;
  version: number;
}
⋮----
export interface RetailerProfile {
  id: string;
  name: string;
  company: string;
  email: string;
  avatar_url: string | null;
  receiving_window_open?: string | null;
  receiving_window_close?: string | null;
  country_code?: string;
}
⋮----
export interface LineItemHistory {
  skuId: string;
  quantity: number;
  unitPrice: number;
  orderDate: string;
  minimumOrderQty: number;
  stepSize: number;
}
⋮----
export interface TrackingOrderItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  line_total: number;
}
⋮----
export interface TrackingRouteGeometry {
  route_id?: string;
  encoded_polyline?: string;
  coordinates: Array<{ lat: number; lng: number }>;
  source: string;
  stop_count?: number;
}
⋮----
export interface TrackingOrder {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  driver_id: string;
  state: string;
  total_amount: number;
  order_source: string;
  driver_latitude: number | null;
  driver_longitude: number | null;
  is_approaching: boolean;
  delivery_token: string;
  created_at: string;
  fiscal_status?: string;
  fiscal_qr?: string;
  latest_fiscal_receipt_id?: string;
  items: TrackingOrderItem[];
  route_geometry?: TrackingRouteGeometry;
}
⋮----
export interface TrackingResponse {
  status?: "idle" | "active";
  orders: TrackingOrder[];
  recent_receipts?: TrackingOrder[];
  events?: Array<{
    event_type: string;
    order_id: string;
    status?: string;
    occurred_at: string;
  }>;
}
```

## File: apps/retailer-app-desktop/src-tauri/tauri.conf.json
```json
{
  "$schema": "../node_modules/@tauri-apps/cli/config.schema.json",
  "productName": "Pegasus Retailer Desktop",
  "version": "0.1.1",
  "identifier": "com.pegasus.retailer",
  "build": {
    "frontendDist": "../out",
    "devUrl": "http://localhost:3001",
    "beforeDevCommand": "npm run dev",
    "beforeBuildCommand": "pnpm run build:static"
  },
  "app": {
    "windows": [
      {
        "label": "main",
        "title": "Pegasus Retailer",
        "width": 800,
        "height": 600,
        "resizable": true,
        "fullscreen": false
      }
    ],
    "security": {
      "csp": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob: https:; connect-src 'self' http://localhost:* http://127.0.0.1:* https://*.pegasusx.app ws://localhost:* wss://localhost:*"
    }
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "createUpdaterArtifacts": true,
    "icon": [
      "icons/32x32.png",
      "icons/128x128.png",
      "icons/128x128@2x.png",
      "icons/icon.icns",
      "icons/icon.ico"
    ]
  },
  "plugins": {
    "deep-link": {
      "desktop": {
        "schemes": [
          "pegasusx-retailer"
        ]
      }
    },
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDhDREMyMEEzRkY5NEYzQjMKUldTejg1VC9veURjakZ6R3NoSmZ6NGIyOGVBZVBZZGs5TkdMWk9XRTArQ2UyakJNZ01XYXZtS0UK",
      "endpoints": [
        "https://storage.googleapis.com/pegasusx-ssmr-app-updates/retailer-desktop/{{target}}/{{arch}}/updater.json"
      ],
      "windows": {
        "installMode": "basic"
      }
    }
  }
}
```

## File: apps/retailer-app-desktop/next.config.ts
```typescript
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { NextConfig } from "next";
⋮----
async rewrites()
```

## File: apps/supplier-portal/app/(portal)/catalog/[productId]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { normalizeEanBarcode } from "@pegasusx/validation";
import { PageChrome } from '@/components/PageChrome';
⋮----
type CatalogProduct = {
  product_id: string;
  name: string;
  category_id: string;
  price_minor: number;
  currency: string;
  unit: string;
  unit_volume_vu: number;
  barcode?: string;
  is_active: boolean;
  version: number;
};
⋮----
type ProductDraft = {
  name: string;
  barcode: string;
  unit_volume_vu: string;
  price_minor: string;
};
⋮----
function draftFromProduct(product: CatalogProduct): ProductDraft
⋮----
async function saveProduct()
⋮----
<Field label=
⋮----
onChange=
⋮----
{saveError ? (
            <p className="md-typescale-body-small" style={{ color: "var(--color-md-error)" }}>
              {saveError}
            </p>
          ) : null}
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              className="md-btn md-btn-filled md-typescale-label-large px-6 py-2 disabled:opacity-50"
              disabled={!dirty || saving}
              onClick={() => void saveProduct()}
            >
              {saving ? "Saving…" : "Save changes"}
            </button>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2 disabled:opacity-50"
              disabled=
```

## File: apps/supplier-portal/app/(portal)/catalog/components/CatalogTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useRef } from "react";
import type { CatalogProduct, SaleUnit } from "./types";
import { ALLOWED_IMAGE_TYPES } from "./types";
import { normalizeEanBarcode } from "@pegasusx/validation";
import { ListToolbar } from "@/components/ListToolbar";
import { usePagination } from "@/lib/use-pagination";
⋮----
interface CatalogTableProps {
  products: CatalogProduct[];
  onSaveEdits: (product: CatalogProduct, updates: Partial<CatalogProduct>) => Promise<void>;
  onImageChange: (product: CatalogProduct, file: File) => Promise<void>;
}
⋮----
const handleSave = async (product: CatalogProduct) =>
⋮----
const handleImageChange = async (event: React.ChangeEvent<HTMLInputElement>) =>
⋮----
setDraftBarcode(prev => (
```

## File: apps/supplier-portal/app/(portal)/catalog/components/CreateProductForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useRef } from "react";
import type { CatalogCategory, CreateProductFormState, SaleUnit } from "./types";
import { ALLOWED_IMAGE_TYPES } from "./types";
import { normalizeEanBarcode } from "@pegasusx/validation";
⋮----
interface CreateProductFormProps {
  categories: CatalogCategory[];
  currency: string;
  initialCategory?: string;
  onCancel: () => void;
  onSave: (
    form: CreateProductFormState,
    normalizedBarcode: string | undefined,
    unitsPerCase: number | undefined,
    imageFile: File | null
  ) => Promise<void>;
}
⋮----
const handleSubmit = async () =>
```

## File: apps/supplier-portal/app/(portal)/chargebacks/claims/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import type { PaymentLedgerEntry } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { moneyCurrency } from "@pegasusx/api-core";
⋮----
function fmt(n?: number)
⋮----
function claimIdFromRef(ref?: string, source?: string): string
⋮----
title=
⋮----
emptyMessage=
⋮----
{claimIdFromRef(it.reference_id, it.source)}
                    <div className="text-[10px] text-[var(--muted)]">{it.reference_id}</div>
                  </td>
                  <td className="py-2.5 pr-3 font-mono text-xs">{it.retailer_id || "—"}</td>
                  <td className="py-2.5 pr-3 text-xs">{it.gateway || "—"}</td>
                  <td className="py-2.5 pr-3 text-right font-mono font-medium">
                    {fmt(it.amount_minor)}
```

## File: apps/supplier-portal/app/(portal)/control-tower/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useEffect, useState } from "react";
import {
  HexagonalControlTowerMap,
  LiveEKGNetworkGraph,
  type NetworkNode,
  type NetworkLink,
  GlassmorphismPanel,
} from "@/components/ui";
import { useControlTowerTelemetry } from "./use-control-tower-telemetry";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Legend,
} from "recharts";
import { ScoredExceptionsPanel } from "@/components/control-tower/ScoredExceptionsPanel";
import { PlaybookRunsPanel } from "@/components/exceptions/PlaybookRunsPanel";
import { sessionSupplierId } from "@/lib/supplier-scope";
import { createSupplierApi } from "@/lib/api";
import { buildScenarioChartRows, type ScenarioChartRow } from "@/lib/scenario-chart";
⋮----
onClick=
```

## File: apps/supplier-portal/app/(portal)/control-tower/use-control-tower-telemetry.ts
```typescript
import { useState, useEffect } from "react";
import { supplierApiBaseUrl } from "@/lib/auth";
import { runSupplierSessionReconcile } from "@/lib/session-reconcile";
import { SSE_SUPPLIER_ENDPOINT } from "@pegasusx/ws-refresh-contract";
⋮----
export interface NetworkNode { id: string; type: "warehouse" | "retailer" | "driver"; label: string; status: "active" | "idle" | "busy"; }
export interface NetworkLink { source: string; target: string; value: number; }
⋮----
interface H3Density {
  hex: string;
  count: number;
}
⋮----
interface ControlTowerData {
  networkNodes: NetworkNode[];
  networkLinks: NetworkLink[];
  h3Data: H3Density[];
}
⋮----
export function useControlTowerTelemetry(supplierId: string): ControlTowerData
⋮----
const closeStream = () =>
⋮----
const handlePayload = (raw: unknown) =>
```

## File: apps/supplier-portal/app/(portal)/credit/admin-disable/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
⋮----
async function submit()
⋮----
title=
⋮----
onClick={() =>
```

## File: apps/supplier-portal/app/(portal)/credit/collections/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";
⋮----
type CreditProfile = {
  retailer_id: string;
  supplier_id: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  delinquency_count?: number;
  status: string;
  utilization_bps?: number;
  needs_attention?: boolean;
  updated_at?: string;
};
⋮----
function formatMinor(n: number): string
⋮----
async function patchProfile(retailerId: string, body: Record<string, unknown>)
⋮----
title=
⋮----
onChange=
```

## File: apps/supplier-portal/app/(portal)/credit/policy/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
import { CreditEnableModal } from "@/components/CreditEnableModal";
⋮----
type Program = {
  supplier_id: string;
  program_enabled: boolean;
  global_terms_days: number;
  global_grace_days: number;
  global_default_limit_minor: number;
  timezone?: string;
};
⋮----
type Relationship = {
  retailer_id: string;
  credit_enabled: boolean;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  use_global_defaults: boolean;
  profile_status?: string;
  current_balance_minor?: number;
  available_credit_minor?: number;
  on_hold?: boolean;
};
⋮----
async function enableProgram(ackAt: string)
⋮----
async function enableRetailer(ackAt: string)
⋮----
async function patchDefaults()
⋮----
async function hold(rid: string, unhold: boolean)
⋮----
title=
```

## File: apps/supplier-portal/app/(portal)/dashboard/page.tsx
```typescript
import Link from "next/link";
import { useRouter } from "next/navigation";
import { BentoCard, BentoGrid } from "@/components/BentoGrid";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import PlanningOutcomesPanel from "@/components/PlanningOutcomesPanel";
import NetworkPulsePanel from "@/components/NetworkPulsePanel";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
import { FormAlert } from "@/components/portal";
import { usePortalT } from "@/lib/i18n";
import { useDashboardData } from "./use-dashboard-data";
import { useDashboardHistory } from "./use-dashboard-history";
import { formatPackMoney, readCachedAuthSession } from "@pegasusx/api-core";
import { MANIFEST_STATES, ORDER_STATUS_FUNNEL, guardHistorySeries } from "@pegasusx/types";
import { HealthStrip, KpiStat, RangeToggle, SourceChip, StatusStack } from "@/components/ui";
⋮----
const formatCurrency = (minor: number) =>
⋮----
const openStatus = (status: string) =>
⋮----
description=
⋮----
value={formatCurrency(metrics.revenueToday)}
              delta=
⋮----
value={velocitySpark ? String(velocitySpark.points[velocitySpark.points.length - 1]) : "—"}
                  spark=
⋮----
value={revenueSpark ? formatCurrency(revenueSpark.points[revenueSpark.points.length - 1]) : "—"}
                  spark=
```

## File: apps/supplier-portal/app/(portal)/dispatch/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { ApiError, supplierDispatchKey } from "@pegasusx/api-core";
import type { SupplierDispatchCapacityWarning, SupplierDispatchPreview, SupplierTopologyWarehouse } from "@pegasusx/types";
import { useDispatchData, type ManifestData } from "./use-dispatch-data";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { setSupplierReconcileScope } from "@/lib/supplier-reconnect";
import { PageChrome } from "@/components/PageChrome";
import DispatchPreviewMap from "@/components/DispatchPreviewMap";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import Icon from "@/components/Icon";
⋮----
function warehouseQuery(warehouseId: string | null): { warehouse_id?: string }
⋮----
function supplierScopeId(): string
⋮----
type DispatchMode = "auto" | "manual";
⋮----
const toggleOrderSelection = (orderId: string) =>
⋮----
description=
⋮----
onClick=
⋮----
placeholder=
⋮----
onChange=
⋮----
onClick={() =>
⋮----
<KpiStatCard label=
⋮----
{previewError ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>{previewError}</p>
      ) : null}
```

## File: apps/supplier-portal/app/(portal)/entity-resolution/page.tsx
```typescript
import { useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-core";
import type { EntityResolutionExplainResponse, EntityResolutionResolveResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
async function runResolve()
⋮----
async function runExplain(type: string, id: string)
⋮----
<button
                    type="button"
                    className="ml-2 underline"
                    onClick={() =>
```

## File: apps/supplier-portal/app/(portal)/exceptions/early-complete/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useState } from "react";
import { supplierApproveEarlyCompleteKey } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-core";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from '@/components/PageChrome';
⋮----
const approve = async () =>
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/finance/credit-notes/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
⋮----
type CreditNoteRow = {
  credit_note_id: string;
  order_id: string;
  type: string;
  status: string;
  reason_code: string;
  total_gross_minor: number;
  created_at: string;
};
⋮----
const issue = async (id: string) =>
⋮----
const createManual = async () =>
⋮----
const loadOrderLines = async () =>
⋮----
title=
⋮----
onChange={(e) =>
```

## File: apps/supplier-portal/app/(portal)/finance/payouts/page.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { ApiError } from "@pegasusx/api-core";
import { supplierPayoutGenerateKey } from "@pegasusx/api-core/idempotency";
import type { PayoutBatch, PayoutRailInfo, SupplierPayoutPolicy } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";
⋮----
function fmtMinor(n: number): string
⋮----
async function generate()
⋮----
async function exportCsv(batchId: string)
⋮----
async function markPaid(batchId: string)
⋮----
async function dispatchLive(batchId: string)
⋮----
async function savePolicy()
⋮----
⋮----
{fmtMinor(b.net_payout_minor)}
```

## File: apps/supplier-portal/app/(portal)/fleet/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import ControlTowerCommandPanel from "@/components/ControlTowerCommandPanel";
import Icon from "@/components/Icon";
import { PageChrome } from "@/components/PageChrome";
⋮----
export default function FleetPage()
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/inventory/import/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useCallback, useMemo, useState } from "react";
import {
  ApiError,
  supplierImportApplyKey,
  supplierImportApproveKey,
  supplierImportCreateKey,
  supplierImportIngestKey,
} from "@pegasusx/api-core";
import type {
  SupplierImportApplyResponse,
  SupplierImportMappingCandidate,
  SupplierImportSession,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";
import { PageChrome } from '@/components/PageChrome';
⋮----
function supplierScopeId(): string
⋮----
type WizardStep = "upload" | "mapping" | "review" | "done";
⋮----
function prettyStatus(value: string): string
⋮----
async function approveSession()
⋮----
async function applySession()
⋮----
function onFileSelect(event: React.ChangeEvent<HTMLInputElement>)
⋮----
description=
⋮----
Required columns: <code>
```

## File: apps/supplier-portal/app/(portal)/labor-capacity/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { LaborDriverScore, LaborZoneCapacity } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";
import { useToast } from "@/components/Toast";
⋮----
const loadScore = async () =>
⋮----
const setAvailability = async () =>
```

## File: apps/supplier-portal/app/(portal)/loyalty/page.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierLoyaltyProgramPatchKey } from "@pegasusx/api-core";
import type { LoyaltyProgram } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { supplierScopeId } from "@/lib/supplier-scope";
⋮----
async function save()
```

## File: apps/supplier-portal/app/(portal)/manifest-exceptions/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierManifestExceptionRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
function shortId(id: string): string
⋮----
description=
⋮----
emptyMessage=
⋮----
onChange=
⋮----
⋮----
<td className="p-4 md-typescale-body-medium font-mono">
```

## File: apps/supplier-portal/app/(portal)/manifests/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  supplierManifestInjectKey,
  supplierManifestSealKey,
  supplierManifestStartLoadingKey,
} from "@pegasusx/api-core";
import type { SupplierManifestDetail } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from '@/components/PageChrome';
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
⋮----
const runAction = async (label: string, action: () => Promise<void>) =>
⋮----
<Metric label=
⋮----
<div className="md-typescale-label-medium text-[var(--color-md-outline)]">
⋮----
onChange=
⋮----
onClick={() =>
```

## File: apps/supplier-portal/app/(portal)/operations/replenishment-policies/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierReplenishmentPolicy } from "@pegasusx/types";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
⋮----
async function saveSafetyStock()
⋮----
description=
⋮----
label=
⋮----
<KpiStatCard label=
⋮----
```

## File: apps/supplier-portal/app/(portal)/ops/map/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import type { TwinOpsRouteView } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import LiveOpsMap, { LiveOpsSidePanel } from "@/components/LiveOpsMap";
import { PageChrome } from "@/components/PageChrome";
⋮----
description=
⋮----
onChange=
⋮----
<input id="page-checkbox-2" aria-label="Select Page option" type="checkbox" checked=
⋮----
<input id="page-checkbox-1" aria-label="Select Page option" type="checkbox" checked=
```

## File: apps/supplier-portal/app/(portal)/orders/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import type { SupplierOrder, WarehouseOrderDetail } from '@pegasusx/types';
import { ApiError, moneyCurrency, supplierPaymentBypassKey } from '@pegasusx/api-core';
import Icon from '@/components/Icon';
import { OrderTimelinePanel } from '@/components/OrderTimelinePanel';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { OrderLineItems } from '@/components/orders/OrderLineItems';
import { OrderOpsActions } from '@/components/orders/OrderOpsActions';
import { useToast } from '@/components/Toast';
import { createSupplierApi } from '@/lib/api';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';
⋮----
function formatMoney(minor: number, currency: string)
⋮----
function isoDeliveryDate(dateInput: string): string
⋮----
async function runMutation(label: string, fn: () => Promise<{ status?: string }>, requiresReason = false)
⋮----
description=
```

## File: apps/supplier-portal/app/(portal)/orders/components/OrdersList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { VirtualScrollList } from '@/components/ui';
import type { SupplierOrder } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-core';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { OrderActionDialog, OrderOpsCard, ReDispatchDialog, ProposeDelayDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';
⋮----
export type OrderFilter = 'ACTIVE' | 'SCHEDULED' | 'COMPLETED' | 'CANCELLED';
⋮----
function formatMoney(order: SupplierOrder)
⋮----
function formatTimestamp(value: string)
⋮----
function isoDeliveryDate(dateInput: string): string
⋮----
function liveStatusLabel(order: SupplierOrder)
⋮----
export interface OrdersListProps {
  orders: SupplierOrder[];
  filter: OrderFilter;
  loading: boolean;
  error: string | null;
  onRefresh: () => Promise<void>;
}
⋮----
const canWarehouseOps = (order: SupplierOrder) =>
⋮----
const runWarehouseMutation = async () =>
⋮----
warehouseOpsEnabled
                    ? () =>
⋮----
onSuccess={() =>
```

## File: apps/supplier-portal/app/(portal)/planning/page.tsx
```typescript
import { Suspense, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { planBrainTabFromQuery, type PlanBrainTab } from "@pegasusx/types";
import { PlanBrainTabs } from "@/components/ui";
import { PageChrome } from "@/components/PageChrome";
import PlanningBrainPanel from "@/components/PlanningBrainPanel";
import DigitalBrainPanel from "@/components/DigitalBrainPanel";
import FactoryPlanningOpsPanel from "@/components/settings/planning/FactoryPlanningOpsPanel";
import MeioNetworkPanel from "@/components/MeioNetworkPanel";
⋮----
export default function PlanningPage()
```

## File: apps/supplier-portal/app/(portal)/pricing/[productId]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import type { CatalogProduct, SupplierPromotion } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierPromotionCreateKey,
  supplierPromotionDeactivateKey,
  supplierPromotionUpdateKey,
} from "@pegasusx/api-core";
import { PageChrome } from '@/components/PageChrome';
⋮----
const save = async () =>
⋮----
const commitSave = async () =>
⋮----
description=
⋮----
emptyMessage=
⋮----
<input id="page-checkbox-2" aria-label="Select Page option" type="checkbox" checked=
⋮----
const formatMinor = (m: number) =>
```

## File: apps/supplier-portal/app/(portal)/pricing/retailer-overrides/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierRetailerPriceOverrideCreateKey,
  supplierRetailerPriceOverrideDeleteKey,
} from "@pegasusx/api-core";
import { supplierFetch } from "@/lib/auth";
import type { CreateRetailerPriceOverrideRequest, RetailerOverridePreview, RetailerPriceOverride } from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';
⋮----
type CatalogProductOption = {
  product_id: string;
  name: string;
  price_minor: number;
};
⋮----
type OverrideForm = {
  retailer_id: string;
  product_id: string;
  price: string;
  notes: string;
};
⋮----
function formatPrice(value: number): string
⋮----
async function handleCreate(e: React.FormEvent)
⋮----
async function handleDelete(overrideId: string)
⋮----
description=
⋮----
emptyMessage=
⋮----
setShowCreate((value)
setFormError(null);
⋮----
<Metric label=
⋮----
label=
⋮----
<Field label=
⋮----
onChange=
⋮----
```

## File: apps/supplier-portal/app/(portal)/promotions/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import {
  supplierPromotionCreateKey,
  supplierPromotionDeactivateKey,
  supplierPromotionUpdateKey,
} from "@pegasusx/api-core";
import type {
  PromoSimulateResult,
  SupplierPromotion,
  SupplierPromotionScopeType,
  SupplierPromotionUpsertRequest,
} from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
const emptyForm = (): SupplierPromotionUpsertRequest =>
⋮----
function resetForm()
⋮----
async function simulatePromo(promo: SupplierPromotion)
⋮----
function startEdit(promo: SupplierPromotion)
⋮----
async function handleSubmit(e: React.FormEvent)
⋮----
async function deactivate(promotionId: string)
⋮----
description=
⋮----
⋮----
<Field label=
⋮----
onChange={(e) =>
```

## File: apps/supplier-portal/app/(portal)/returns/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@heroui/react";
import { supplierResolveReturnKey } from "@pegasusx/api-core";
import { PageChrome } from "@/components/PageChrome";
import { supplierFetch } from "@/lib/auth";
⋮----
type Resolution = "WRITE_OFF" | "RETURN_TO_STOCK";
⋮----
type SupplierReturnRow = {
  return_id: string;
  line_item_id: string;
  order_id: string;
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  status: string;
  physical_status: string;
  received_qty: number;
  manifest_id?: string;
  driver_id?: string;
  driver_name?: string;
  reason: string;
  driver_notes?: string;
  retailer_id: string;
  retailer_name: string;
  created_at: string;
};
⋮----
function formatAmount(amountMinor: number): string
⋮----
async function handleResolve(returnId: string)
⋮----
description=
⋮----
emptyMessage=
⋮----
onChange=
⋮----
onPress=
⋮----
<Button size="sm" variant="ghost" onPress=
```

## File: apps/supplier-portal/app/(portal)/settings/integrations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import StatusChip from "@/components/StatusChip";
import { OIDCAttachCard } from "@/components/settings/OIDCAttachCard";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type {
  PartnerApiKeyMeta,
  PartnerDeadLetterAttempt,
  PartnerEdiDocument,
  PartnerExportJob,
  PartnerSftpConfig,
  PartnerAs2Config,
  PartnerCoaMap,
  PartnerWebhookSubscription,
} from "@pegasusx/types";
⋮----
async function issueKey()
⋮----
async function revokeKey(keyId: string)
⋮----
async function createWebhook()
⋮----
async function pingWebhook(id: string)
⋮----
async function deactivateWebhook(id: string)
⋮----
async function replayDead(id: string)
⋮----
async function requestExport()
⋮----
async function saveSftp()
⋮----
async function saveAs2()
⋮----
async function saveCoa()
⋮----
async function replayEdi(id: string)
⋮----
title=
```

## File: apps/supplier-portal/app/(portal)/settings/notification-preferences/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";
⋮----
type Pref = {
  event_type: string;
  channel: string;
  enabled: boolean;
  quiet_from?: string;
  quiet_to?: string;
};
⋮----
const save = async () =>
⋮----
<PageChrome title=
⋮----
onChange={(e) =>
⋮----
```

## File: apps/supplier-portal/app/(portal)/settings/return-policy/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierReturnPolicyPutKey } from "@pegasusx/api-core/idempotency";
import type { SupplierReturnPolicy } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
⋮----
async function save()
⋮----
title=
⋮----
onChange=
```

## File: apps/supplier-portal/app/(portal)/treasury/cash-reconciliations/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { CashReconciliationRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
⋮----
function formatMinor(n: number): string
⋮----
const accept = async (id: string) =>
⋮----
const writeOff = async (id: string) =>
⋮----
title=
⋮----
emptyMessage=
⋮----
Expected {formatMinor(row.expected_cash_minor)} · Declared
```

## File: apps/supplier-portal/app/(portal)/treasury/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { desktopPrint } from "@pegasusx/desktop-bridge";
import { downloadCsv } from "@/lib/csv";
import type { FinanceAuthoritySnapshot } from "../../payments/_shared/finance";
import { errorToMessage, formatMinor, loadFinanceAuthoritySnapshot } from "../../payments/_shared/finance";
⋮----
const exportTreasuryCsv = () =>
⋮----
description=
⋮----
<button type="button" onClick=
⋮----
<KpiStatCard label=
⋮----
label=
```

## File: apps/supplier-portal/app/(portal)/warehouses/[warehouseId]/coverage/page.tsx
```typescript
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { CoverageCityChips } from "@/components/CoverageCityChips";
import { createSupplierApi } from "@/lib/api";
import {
  coverageModeLabel,
  normalizeCoverageMode,
  PIN_TARGET_TYPES,
  pinKey,
} from "@/lib/coverage";
import { usePortalT } from "@/lib/i18n";
import type {
  ServicePin,
  ServicePinTargetType,
  SupplierCRMRetailer,
  SupplierRegion,
  SupplierTopologyCoverageCity,
} from "@pegasusx/types";
⋮----
const load = () =>
⋮----
const addPin = () =>
⋮----
const saveCoverage = async () =>
⋮----
const saveRegions = async () =>
⋮----
<li key=
⋮----
<button
                    type="button"
                    className="md-btn md-btn-text px-2"
                    onClick={() =>
⋮----
<button
                  type="button"
                  className="md-btn md-btn-text px-2"
                  onClick={() =>
```

## File: apps/supplier-portal/app/(portal)/warehouses/[warehouseId]/layout.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export function generateStaticParams()
⋮----
export default function Layout({ children }: { children: ReactNode })
```

## File: apps/supplier-portal/app/(portal)/warehouses/components/WarehouseForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";
⋮----
import { CoverageCityChips } from "@/components/CoverageCityChips";
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";
import { packMapCenter } from "@pegasusx/api-core";
import type { SupplierTopologyCoverageCity } from "@pegasusx/types";
⋮----
interface WarehouseFormProps {
  onSave: (
    name: string,
    location: LocationValue,
    radius: number,
    extras: {
      country_code: string;
      coverage_cities: SupplierTopologyCoverageCity[];
      primary_factory_id: string;
    },
  ) => Promise<void>;
  onCancel: () => void;
  factoryOptions?: Array<{ id: string; name: string }>;
}
⋮----
function defaultWarehouseLocation(pack: { map_center_lat?: number; map_center_lng?: number; status?: string } | null | undefined): LocationValue
⋮----
const handleSave = async () =>
```

## File: apps/supplier-portal/app/api/api/[...path]/route.ts
```typescript
import { NextRequest } from "next/server";
⋮----
export function generateStaticParams()
⋮----
function backendBaseURL()
⋮----
function targetURL(pathname: string, search: string)
⋮----
type RouteParams = { path: string[] };
type RouteContext = { params: Promise<RouteParams> };
⋮----
async function proxy(req: NextRequest, context: RouteContext)
⋮----
export async function GET(req: NextRequest, context: RouteContext)
⋮----
export async function POST(req: NextRequest, context: RouteContext)
⋮----
export async function PUT(req: NextRequest, context: RouteContext)
⋮----
export async function PATCH(req: NextRequest, context: RouteContext)
⋮----
export async function DELETE(req: NextRequest, context: RouteContext)
```

## File: apps/supplier-portal/app/api/api/ws-session/route.ts
```typescript
import { NextRequest } from "next/server";
⋮----
export function generateStaticParams()
⋮----
function backendBaseURL()
```

## File: apps/supplier-portal/app/auth/register/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
  dialCodeForCountry,
  FormAlert,
} from "@/components/ui";
import { persistSession, supplierFetch } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import {
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";
⋮----
function validateCurrent(): Record<string, string>
⋮----
async function next()
⋮----
function back()
⋮----
async function submit()
⋮----
title=
```

## File: apps/supplier-portal/app/earnings/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ApiClient, supplierChargebackKey, supplierChargebackReversalKey } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { PaymentChargebackRequest, PaymentChargebackReversalRequest, PaymentLedgerEntry } from "@pegasusx/types";
import { useEffect, useMemo, useState } from "react";
import type { FinanceAuthoritySnapshot } from "../payments/_shared/finance";
import {
  createIdempotencyKey,
  errorToMessage,
  formatDateTime,
  formatMinor,
  loadFinanceAuthoritySnapshot,
  parseDate,
  useSupplierFinanceLiveRefresh,
} from "../payments/_shared/finance";
import { PageChrome } from "@/components/PageChrome";
import { PortalSection } from "@/components/portal";
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";
⋮----
type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };
⋮----
type ActionState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "success"; message: string }
  | { status: "error"; message: string };
⋮----
async function load()
⋮----
async function submitChargeback(event: React.FormEvent<HTMLFormElement>)
⋮----
async function submitReversal(event: React.FormEvent<HTMLFormElement>)
⋮----
description=
⋮----
<MetricCard label=
⋮----
setChargebackForm((current) => (
⋮----
return [...entries].sort((left, right)
```

## File: apps/supplier-portal/app/org-fleet/components/DriverForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import type { HomeNodeType, SupplierFleetDriverCreateRequest } from "@pegasusx/types";
import { supplierFleetDriverCreateKey } from "@pegasusx/api-core";
import type { ApiClient } from "@pegasusx/api-core";
import {
  DriverFormState,
  defaultDriverForm,
  StatusText,
  isErrorMessage,
  toErrorMessage,
  supplierScopeId,
  nodeOptionsFor,
  ReadyState,
} from "./utils";
⋮----
async function submitDriver(event: React.FormEvent<HTMLFormElement>)
⋮----
```

## File: apps/supplier-portal/app/org-fleet/components/OrgMemberForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import type { Role, HomeNodeType } from "@pegasusx/types";
import { supplierOrgMemberCreateKey } from "@pegasusx/api-core";
import type { ApiClient } from "@pegasusx/api-core";
import {
  OrgFormState,
  defaultOrgForm,
  orgRoleOptions,
  buildOrgMemberRequest,
  StatusText,
  isErrorMessage,
  toErrorMessage,
  supplierScopeId,
  nodeOptionsFor,
  orgEffectiveNodeType,
  ReadyState,
} from "./utils";
⋮----
async function submitOrgMember(event: React.FormEvent<HTMLFormElement>)
```

## File: apps/supplier-portal/app/org-fleet/components/VehicleForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import type { HomeNodeType, SupplierFleetVehicleCreateRequest } from "@pegasusx/types";
import { supplierFleetVehicleCreateKey } from "@pegasusx/api-core";
import type { ApiClient } from "@pegasusx/api-core";
import {
  VehicleFormState,
  defaultVehicleForm,
  StatusText,
  isErrorMessage,
  toErrorMessage,
  supplierScopeId,
  nodeOptionsFor,
  ReadyState,
} from "./utils";
⋮----
async function submitVehicle(event: React.FormEvent<HTMLFormElement>)
⋮----
```

## File: apps/supplier-portal/components/dispatch/DispatchSidebarNav.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import { ArrowRight, Truck, Package, Wrench, User, BarChart2 } from "lucide-react";
⋮----
interface DispatchSidebarNavProps {
  activeTab?: string;
  onTabSelect?: (tab: string) => void;
  onRequestClick?: () => void;
}
```

## File: apps/supplier-portal/components/dispatch/PartnerFilterBar.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import { PartnerFilterMetric } from "@pegasusx/types";
⋮----
interface PartnerFilterBarProps {
  partnerFilters: PartnerFilterMetric[];
  selectedPartnerId?: string;
  onPartnerSelect: (partnerId?: string) => void;
  statusFilter: "ALL" | "ACTIVE" | "INACTIVE";
  onStatusFilterChange: (status: "ALL" | "ACTIVE" | "INACTIVE") => void;
  totalCount: number;
  activeCount: number;
  inactiveCount: number;
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
}
⋮----
placeholder=
⋮----
onChange=
⋮----
onClick=
```

## File: apps/supplier-portal/components/dispatch/ShipmentCard.tsx
```typescript
import React from "react";
import { VehicleShipmentCard } from "@pegasusx/types";
⋮----
interface ShipmentCardProps {
  shipment: VehicleShipmentCard;
  isSelected?: boolean;
  onSelect?: () => void;
}
⋮----
const formatTime = (seconds: number) =>
```

## File: apps/supplier-portal/components/inventory/InventoryTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
⋮----
export type InventoryRow = {
  sku_id: string;
  warehouse_id: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  currency: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
};
⋮----
interface InventoryTableProps {
  items: InventoryRow[];
  deltas: Record<string, string>;
  adjustingSku: string | null;
  policyUpdatingKey: string | null;
  onDeltaChange: (skuId: string, value: string) => void;
  onApply: (row: InventoryRow) => void;
  onPolicyChange: (row: InventoryRow, policy: string) => void;
}
⋮----
function rowKey(row: InventoryRow)
```

## File: apps/supplier-portal/components/operations/OperatorBroadcast.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import { PageSection } from '@/components/PageSection';
import { SUPPLIER_BROADCAST_TEMPLATES } from '@pegasusx/types';
⋮----
interface OperatorBroadcastProps {
  title: string;
  body: string;
  broadcastRole: (typeof broadcastRoles)[number];
  templateDate: string;
  broadcasting: boolean;
  onTitleChange: (v: string) => void;
  onBodyChange: (v: string) => void;
  onBroadcastRoleChange: (v: (typeof broadcastRoles)[number]) => void;
  onTemplateDateChange: (v: string) => void;
  onBroadcast: () => void;
}
⋮----
<PageSection title=
```

## File: apps/supplier-portal/components/operations/PaymentBypass.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import { PageSection } from '@/components/PageSection';
⋮----
interface PaymentBypassProps {
  orderId: string;
  bypassReason: string;
  bypassToken: string | null;
  confirmBypass: boolean;
  bypassing: boolean;
  onOrderIdChange: (v: string) => void;
  onBypassReasonChange: (v: string) => void;
  onConfirmBypassChange: (v: boolean) => void;
  onPaymentBypass: () => void;
}
⋮----
<PageSection title=
⋮----
disabled=
```

## File: apps/supplier-portal/components/orders/OrderOpsActions.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
⋮----
interface OrderOpsActionsProps {
  canWarehouseMutate: boolean;
  flags: { canDelay: boolean; canReject: boolean };
  acting: boolean;
  proposedDate: string;
  setProposedDate: (date: string) => void;
  reason: string;
  setReason: (reason: string) => void;
  onDelay: () => void;
  onReject: () => void;
}
```

## File: apps/supplier-portal/components/orders/OrderOpsCard.tsx
```typescript
import { OrderKebabMenu } from './OrderKebabMenu';
import { OrderStateChip } from './OrderStateChip';
import { orderActionFlags } from '@/lib/order-actions';
⋮----
export type OrderOpsCardProps = {
  orderId: string;
  retailerName: string;
  state: string;
  amountLabel: string;
  meta?: string;
  badge?: string;
  index?: number;
  disabled?: boolean;
  onOpenDetail: () => void;
  onDelay?: () => void;
  onReject?: () => void;
  onReassign?: () => void;
  showOpsMenu?: boolean;
  delayLabel?: string;
  rejectLabel?: string;
  reassignLabel?: string;
  canDelayOverride?: boolean;
  canRejectOverride?: boolean;
  canReassignOverride?: boolean;
};
⋮----
const stopPropagation = (e: React.SyntheticEvent) =>
```

## File: apps/supplier-portal/components/orders/ProposeDelayDialog.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
⋮----
interface ProposeDelayDialogProps {
  open: boolean;
  proposedDate: string;
  reason: string;
  submitting: boolean;
  onProposedDateChange: (date: string) => void;
  onReasonChange: (reason: string) => void;
  onClose: () => void;
  onConfirm: () => void;
}
```

## File: apps/supplier-portal/components/portal/PortalPrimitives.tsx
```typescript
import Icon from "@/components/Icon";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import {
  PortalField as KitPortalField,
  PortalInput as KitPortalInput,
  PortalSelect as KitPortalSelect,
  PortalSection as KitPortalSection,
  PortalActions as KitPortalActions,
  SelectionCard as KitSelectionCard,
  DataList as KitDataList,
  DataListRow as KitDataListRow,
  HubCard as KitHubCard,
  FormAlert as KitFormAlert,
} from "@/components/ui/PortalPrimitives";
⋮----
export function PortalSection({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitPortalSection>, "icon"> & { icon?: string })
```

## File: apps/supplier-portal/components/profile/ContactDetailsForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { SupplierProfile } from '@pegasusx/types';
⋮----
export type ProfileDraft = {
  legal_name: string;
  contact_name: string;
  email: string;
  phone: string;
  gln: string;
  gs1_company_prefix: string;
};
⋮----
export function draftFromProfile(profile: SupplierProfile): ProfileDraft
⋮----
export function EditableField({
  label,
  value,
  onChange,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
})
⋮----
interface ContactDetailsFormProps {
  draft: ProfileDraft;
  setDraft: React.Dispatch<React.SetStateAction<ProfileDraft | null>>;
  saveError: string | null;
  saving: boolean;
  dirty: boolean | null;
  onSave: () => void;
  onReset: () => void;
}
⋮----
label=
⋮----
onChange=
⋮----
setDraft((prev) => (prev ?
```

## File: apps/supplier-portal/components/settings/planning/CreateOverrideForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React from "react";
import type { SeasonalTemplatesResponse } from "@pegasusx/types";
⋮----
export type OverrideForm = {
  template_id: string;
  name: string;
  start_date: string;
  end_date: string;
  multiplier: string;
};
⋮----
interface CreateOverrideFormProps {
  data: SeasonalTemplatesResponse | null;
  form: OverrideForm;
  formError: string | null;
  saving: boolean;
  onFormChange: (form: OverrideForm) => void;
  onSubmit: () => void;
}
⋮----
function onTemplateChange(templateId: string)
```

## File: apps/supplier-portal/components/settings/planning/FactoryPlanningOpsPanel.tsx
```typescript
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { canPatchOrderStatus } from "@/lib/admin-scope";
import { supplierScopeId } from "@/lib/supplier-scope";
import { ApiError } from "@pegasusx/api-core";
import {
  supplierNetworkModePutKey,
  supplierPlanningKillSwitchKey,
  supplierPlanningPullMatrixKey,
  supplierPlanningPredictivePushKey,
} from "@pegasusx/api-core/idempotency";
import type { NetworkModeResponse, PullMatrixResponse, KillSwitchResponse } from "@pegasusx/types";
import { factoryPlanningDisabledCode } from "@pegasusx/types";
⋮----
function planningDisabledMessage(err: unknown): string | null
⋮----
async function putMode(next: string)
⋮----
async function runPredictivePush()
⋮----
async function runPullMatrix()
⋮----
async function runKillSwitch()
⋮----
```

## File: apps/supplier-portal/components/settings/planning/ForecastAccuracyPanel.tsx
```typescript
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { ApiError } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { sessionSupplierId } from "@/lib/supplier-scope";
import type { ForecastAccuracyDailyRow } from "@pegasusx/types";
import { SourceChip } from "@/components/ui";
⋮----
interface AccuracyAggregate {
  productId: string;
  warehouseId: string;
  wape28: number;
  bias28: number;
  trackingSignal: number;
  alertTs: boolean;
  days: number;
}
⋮----
function fmtPct(v: number): string
⋮----
function fmtTs(v: number): string
⋮----
function aggregate(rows: ForecastAccuracyDailyRow[]): AccuracyAggregate[]
```

## File: apps/supplier-portal/components/settings/OIDCAttachCard.tsx
```typescript
import { useEffect, useState } from "react";
import { deleteSupplierOIDC, getSupplierOIDC, oidcCallbackUrl, putSupplierOIDC, type OIDCConfig } from "@/lib/oidc";
⋮----
async function save(e: React.FormEvent)
⋮----
async function detach()
⋮----
onChange=
```

## File: apps/supplier-portal/components/ui/AuthLoginCard.tsx
```typescript
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthLoginStep = "phone" | "otp";
```

## File: apps/supplier-portal/components/ui/AuthRegisterStepper.tsx
```typescript
import type { ReactNode } from "react";
```

## File: apps/supplier-portal/components/ui/AuthRegisterSteps.tsx
```typescript
import type { Dispatch, SetStateAction } from "react";
import { PortalField } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};
⋮----
export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};
⋮----
export type AuthProfileStep = {
  legalName: string;
  contactName: string;
  email: string;
};
⋮----
type Errors = Record<string, string>;
```

## File: apps/supplier-portal/components/ui/countries.ts
```typescript
export interface AuthCountry {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export function dialCodeForCountry(code: string): string
```

## File: apps/supplier-portal/components/ui/DemandSourceChips.tsx
```typescript
import type { CSSProperties } from "react";
⋮----
export type DemandSourceCode = "STORE_POS" | "WHOLESALE_HISTORY" | string;
⋮----
export function demandSourceLabel(code: string): string
⋮----
export function normalizeDemandSources(sources?: string[] | null): string[]
⋮----
export type DemandSourceChipsProps = {
  sources?: string[] | null;
  className?: string;
  size?: "sm" | "md";
};
```

## File: apps/supplier-portal/components/ui/DesktopKeyboardCheatsheet.tsx
```typescript
import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";
⋮----
export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};
⋮----
export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
```

## File: apps/supplier-portal/components/ui/DesktopOfflineTray.tsx
```typescript
import { useState } from "react";
import { ChevronDown, ChevronUp, Clock, RefreshCw, WifiOff } from "lucide-react";
⋮----
export type DesktopQueuedAction = {
  id: string;
  label: string;
  subtitle?: string;
};
⋮----
export type DesktopOfflineTrayProps = {
  isOffline: boolean;
  queuedActions?: DesktopQueuedAction[];
  onRetryAll?: () => void | Promise<void>;
  retrying?: boolean;
  offlineMessage?: string;
  className?: string;
};
```

## File: apps/supplier-portal/components/ui/ExplainStatusBanner.tsx
```typescript
import type { StatusExplain } from '@pegasusx/types';
⋮----
export type ExplainStatusBannerProps = {
  errorCode?: string | null;
  explain?: StatusExplain | null;
  detail?: string | null;
  className?: string;
};
```

## File: apps/supplier-portal/components/ui/firebase-emulator-phone.ts
```typescript
function emulatorIdentityToolkitBase(): string
⋮----
export async function sendPhoneOtpViaEmulator(
  phone: string,
  apiKey = DEFAULT_API_KEY,
): Promise<void>
⋮----
export async function verifyPhoneOtpViaEmulator(
  code: string,
  apiKey = DEFAULT_API_KEY,
): Promise<string>
⋮----
export function resetEmulatorPhoneOtpFlow(): void
```

## File: apps/supplier-portal/components/ui/GenericFleetLiveMap.tsx
```typescript
import { useEffect, useMemo, useRef } from 'react';
⋮----
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import { useMapLibreTeardown } from './map-chrome';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from './useAnimatedDriverMarkers';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
export interface GenericRouteType {
  driver_location?: { lat?: number; lng?: number; latitude?: number; longitude?: number };
  live_location_available?: boolean;
  location_stale?: boolean;
  route_geometry?: any;
  manifest_id: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
}
⋮----
export type FleetLiveMapProps = {
  routes: GenericRouteType[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  enable3DView?: boolean;
};
⋮----
function routeLineFeature(
  route: GenericRouteType,
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null
⋮----
const t = (key: string) =>
⋮----
mapRef.current = ref?.getMap() ?? null;
⋮----
...mapInitialViewState(readCachedAuthSession()?.pack),
```

## File: apps/supplier-portal/components/ui/GlassmorphismPanel.tsx
```typescript
import React from "react";
import { motion, HTMLMotionProps } from "framer-motion";
⋮----
export interface GlassmorphismPanelProps extends HTMLMotionProps<"div"> {
  children: React.ReactNode;
  className?: string;
}
⋮----
export function GlassmorphismPanel({
  children,
  className = "",
  ...props
}: GlassmorphismPanelProps)
```

## File: apps/supplier-portal/components/ui/HandoffCard.tsx
```typescript
import type { HandoffCardMetadata } from '@pegasusx/types';
⋮----
export type HandoffCardProps = {
  metadata: HandoffCardMetadata;
  className?: string;
  onAction?: (link: string) => void;
};
```

## File: apps/supplier-portal/components/ui/HealthStrip.tsx
```typescript
export type HealthStripItem = {
  key: string;
  label: string;
  count: number;
  href?: string;
};
⋮----
export type HealthStripProps = {
  items: HealthStripItem[];
  onSelect?: (key: string) => void;
  className?: string;
};
⋮----
onClick={onSelect ? () => onSelect(item.key) : undefined}
          disabled=
```

## File: apps/supplier-portal/components/ui/HexagonalControlTowerMap.tsx
```typescript
import React, { useMemo, useState } from "react";
import MapGL from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import DeckGL from "@deck.gl/react";
import { H3HexagonLayer } from "@deck.gl/geo-layers";
import { Map3DViewToggle } from "./map-chrome";
⋮----
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-core";
import type { MarketPack } from "@pegasusx/types";
⋮----
export interface HexagonalControlTowerMapProps {
  data: { hex: string; count: number }[];
  show3DViewToggle?: boolean;
  pack?: MarketPack | null;
}
```

## File: apps/supplier-portal/components/ui/KpiStat.tsx
```typescript
import type { HistorySeries } from "@pegasusx/types";
import { guardHistorySeries } from "@pegasusx/types";
import type { ReactNode } from "react";
import { SourceChip } from "./SourceChip";
⋮----
export type KpiStatProps = {
  label: string;
  value: ReactNode;
  delta?: ReactNode;
  spark?: HistorySeries | null;
  source: string;
  className?: string;
};
```

## File: apps/supplier-portal/components/ui/LiveEKGNetworkGraph.tsx
```typescript
import React, { useEffect, useRef } from "react";
⋮----
export interface NetworkNode extends d3.SimulationNodeDatum {
  id: string;
  type: "warehouse" | "retailer" | "driver";
  label: string;
  status: "active" | "idle" | "busy";
}
⋮----
export interface NetworkLink extends d3.SimulationLinkDatum<NetworkNode> {
  source: string | NetworkNode;
  target: string | NetworkNode;
  value: number;
}
⋮----
export interface LiveEKGNetworkGraphProps {
  nodes: NetworkNode[];
  links: NetworkLink[];
  width?: number;
  height?: number;
}
⋮----
export function LiveEKGNetworkGraph({
  nodes,
  links,
  width = 800,
  height = 600,
}: LiveEKGNetworkGraphProps)
⋮----
function pulse()
⋮----
const drag = (simulation: any) =>
⋮----
function dragstarted(event: any, d: any)
function dragged(event: any, d: any)
function dragended(event: any, d: any)
```

## File: apps/supplier-portal/components/ui/map-chrome.tsx
```typescript
import { useEffect, useRef, useState, type RefObject } from "react";
import type maplibregl from "maplibre-gl";
⋮----
export function useLazyMapMount(rootMargin = "120px")
⋮----
export function useMapLibreTeardown(mapRef: RefObject<maplibregl.Map | null>)
⋮----
type Map3DViewToggleProps = {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  className?: string;
};
⋮----
export function Map3DViewToggle({ enabled, onChange, className = "" }: Map3DViewToggleProps)
⋮----
onClick=
```

## File: apps/supplier-portal/components/ui/motion.ts
```typescript
export type CubicBezierEasing = [number, number, number, number];
⋮----
export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};
⋮----
export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition
```

## File: apps/supplier-portal/components/ui/PackChip.tsx
```typescript
type PackChipProps = {
  currency?: string | null;
  receipts?: string | null;
  title?: string;
};
⋮----
export function PackChip({ currency, receipts, title }: PackChipProps)
```

## File: apps/supplier-portal/components/ui/PageChrome.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export type PageChromeProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  children: ReactNode;
  renderLoading?: () => ReactNode;
  renderError?: (message: string) => ReactNode;
  renderEmpty?: () => ReactNode;
};
```

## File: apps/supplier-portal/components/ui/PageSkeletons.tsx
```typescript
import type { CSSProperties } from "react";
```

## File: apps/supplier-portal/components/ui/PlanBrainTabs.tsx
```typescript
import type { PlanBrainTab } from "@pegasusx/types";
⋮----
export type PlanBrainTabsProps = {
  value: PlanBrainTab;
  onChange: (tab: PlanBrainTab) => void;
  className?: string;
};
⋮----
export function PlanBrainTabs({ value, onChange, className }: PlanBrainTabsProps)
```

## File: apps/supplier-portal/components/ui/PortalPrimitives.tsx
```typescript
import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
⋮----
export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
{optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
⋮----
{icon ? (
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]"
              style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-secondary)" }}
              aria-hidden
            >
              {icon}
            </div>
          ) : null}
          <div className="min-w-0">
            <h2 className="bento-card-title">{title}</h2>
⋮----
export function DataListRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
})
⋮----
type LinkProps = { href: string; className?: string; children: ReactNode };
⋮----
export function HubCard({
  href,
  title,
  description,
  icon,
  onClick,
  LinkComponent,
}: {
  href?: string;
  title: string;
  description: string;
  icon?: ReactNode;
  onClick?: () => void;
  LinkComponent?: ComponentType<any>;
})
⋮----
export function FormAlert({
  variant = "info",
  icon,
  children,
}: {
  variant?: "info" | "error";
  icon?: ReactNode;
  children: ReactNode;
})
```

## File: apps/supplier-portal/components/ui/PulseTimeline.tsx
```typescript
import type { PulseEvent } from '@pegasusx/types';
⋮----
export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  className?: string;
  onSelect?: (event: PulseEvent) => void;
};
⋮----
function formatWhen(iso: string): string
⋮----
export function PulseTimeline({
  events,
  loading = false,
  error = null,
  emptyMessage = 'No recent activity yet.',
  className = '',
  onSelect,
}: PulseTimelineProps)
⋮----
return <div className=
```

## File: apps/supplier-portal/components/ui/RangeToggle.tsx
```typescript
import type { DashboardHistoryRange } from "@pegasusx/types";
⋮----
export type RangeToggleProps = {
  value: DashboardHistoryRange;
  onChange: (range: DashboardHistoryRange) => void;
  className?: string;
};
```

## File: apps/supplier-portal/components/ui/SessionPackChip.tsx
```typescript
import { useMarketPack } from '@pegasusx/api-react';
import { PackChip } from './PackChip';
import { fiscalReceiptLabel } from '@pegasusx/api-core';
⋮----
receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title=
```

## File: apps/supplier-portal/components/ui/SourceChip.tsx
```typescript
export type HonestySource =
  | "live"
  | "spanner"
  | "empty"
  | "unavailable"
  | "memory"
  | "env_default";
⋮----
export function sourceChipLabel(source: string): string
⋮----
export type SourceChipProps = {
  source: string;
  className?: string;
};
⋮----
export function SourceChip({ source, className }: SourceChipProps)
```

## File: apps/supplier-portal/components/ui/StatusStack.preview.tsx
```typescript
import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts } from "@pegasusx/types";
import { StatusStack } from "./StatusStack";
```

## File: apps/supplier-portal/components/ui/StatusStack.tsx
```typescript
import { statusStackModel, type StatusStackMode } from "@pegasusx/types";
import { SourceChip } from "./SourceChip";
⋮----
export type StatusStackProps = {
  dictionary: readonly string[];
  counts?: Record<string, number> | null;
  available?: boolean;
  source?: string;
  onSelect?: (key: string) => void;
  labelFor?: (key: string) => string;
  className?: string;
};
⋮----
function defaultLabel(key: string): string
⋮----
function sourceForMode(mode: StatusStackMode, source?: string): string
⋮----
onClick={onSelect ? () => onSelect(row.key) : undefined}
                disabled=
```

## File: apps/supplier-portal/components/ui/use-desktop-connectivity.ts
```typescript
import { useCallback, useSyncExternalStore } from "react";
import { NetworkLifecycleEngine } from "@/lib/enterprise-swr";
⋮----
export type DesktopConnectivity = {
  isOffline: boolean;
  onOnline: (listener: () => void) => () => void;
};
⋮----
export function useDesktopConnectivity(): DesktopConnectivity
```

## File: apps/supplier-portal/components/ui/useAnimatedDriverMarkers.ts
```typescript
import { useEffect, useMemo, useRef, useState } from 'react';
import type { GenericRouteType } from './GenericFleetLiveMap';
⋮----
type DriverAnimState = {
  driverId: string;
  fromLng: number;
  fromLat: number;
  toLng: number;
  toLat: number;
  startMs: number;
  color: string;
  label: string;
  stale: boolean;
};
⋮----
function lerp(a: number, b: number, t: number): number
⋮----
function easeOut(t: number): number
⋮----
function driverTarget(route: GenericRouteType): { lat: number; lng: number } | null
⋮----
export function useAnimatedDriverMarkers(
  routes: GenericRouteType[],
  durationMs = 1200,
): GeoJSON.FeatureCollection<GeoJSON.Point>
⋮----
const tick = () =>
```

## File: apps/supplier-portal/components/ui/VirtualScrollList.tsx
```typescript
import type { CSSProperties, ReactNode } from "react";
⋮----
export type VirtualScrollListProps<T> = {
  items: readonly T[];
  height?: CSSProperties["height"];
  className?: string;
  itemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  virtualizationThreshold?: number;
};
⋮----
export function VirtualScrollList<T>({
  items,
  height = "28rem",
  className,
  itemKey,
  renderItem,
}: VirtualScrollListProps<T>)
⋮----
<div key=
```

## File: apps/supplier-portal/components/AdminOrderOpsPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import {
  adminForceCompleteKey,
  adminOrderAssignKey,
  adminOrderStatusPatchKey,
  ApiError,
} from '@pegasusx/api-core';
import type { SupplierOrder } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import StatusChip from '@/components/StatusChip';
⋮----
interface AdminOrderOpsPanelProps {
  order: SupplierOrder;
  busy: boolean;
  canAssign: boolean;
  canPatchStatus: boolean;
  onBusyChange: (orderId: string | null) => void;
  onSuccess: () => void;
  onError: (message: string) => void;
}
⋮----
const assignDriver = async () =>
⋮----
const patchStatus = async () =>
⋮----
const forceComplete = async () =>
```

## File: apps/supplier-portal/components/CoverageCityChips.tsx
```typescript
import { useState } from "react";
import type { SupplierTopologyCoverageCity } from "@pegasusx/types";
import { forwardGeocode } from "@/lib/geocode";
⋮----
type Props = {
  cities: SupplierTopologyCoverageCity[];
  onChange: (cities: SupplierTopologyCoverageCity[]) => void;
};
⋮----
const add = async () =>
```

## File: apps/supplier-portal/components/CreditEnableModal.tsx
```typescript
import { useMemo, useState } from "react";
⋮----
type Locale = keyof typeof COPY;
⋮----
export function CreditEnableModal(props: {
  open: boolean;
  locale?: Locale;
  confirmToken?: string;
  onCancel: () => void;
  onConfirm: (ackAt: string) => void;
  busy?: boolean;
})
⋮----
onChange=
⋮----
onClick={() =>
```

## File: apps/supplier-portal/components/FleetLiveMapPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import { Map3DViewToggle, useLazyMapMount } from '@/components/ui';
import FleetLiveMap from '@/components/FleetLiveMap';
import { useFleetLiveMap } from '@/lib/use-fleet-live-map';
⋮----
type FleetLiveMapPanelProps = {
  className?: string;
  show3DViewToggle?: boolean;
};
```

## File: apps/supplier-portal/components/Icon.tsx
```typescript
import {
  LayoutDashboard,
  BookOpen,
  CheckCircle,
  Landmark,
  Truck,
  ShieldCheck,
  TrendingUp,
  DollarSign,
  Package,
  BarChart3,
  ClipboardList,
  ClipboardCheck,
  RotateCcw,
  Users,
  Settings,
  Warehouse,
  Globe,
  XCircle,
  Heart,
  LogOut,
  Menu,
  ChevronRight,
  MapPin,
  Search,
  Bell,
  X,
  Phone,
  Mail,
  ArrowLeft,
  AlertTriangle,
  Sun,
  Moon,
  Monitor,
  CreditCard,
  Calendar,
  ChevronLeft,
  BadgeCheck,
  Pencil,
  AlertCircle,
  RefreshCw,
  Building2,
  Factory,
  Plus,
  Hexagon,
  PanelLeftClose,
  PanelLeft,
  Image,
  ArrowRight,
  Eye,
  EyeOff,
  User,
  UserPlus,
  Sparkles,
  Network,
  Keyboard,
  type LucideIcon,
} from "lucide-react";
⋮----
export default function Icon({
  name,
  size = 24,
  className = "",
  style,
}: {
  name: string;
  size?: number;
  className?: string;
  style?: React.CSSProperties;
})
```

## File: apps/supplier-portal/components/NetworkPulsePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@/components/ui";
import type { PulseEvent } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";
```

## File: apps/supplier-portal/components/NotificationPanel.tsx
```typescript
import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';
import { motion, AnimatePresence } from 'framer-motion';
import Icon from './Icon';
import { HandoffInboxCard } from './HandoffInboxCard';
⋮----
interface NotificationPanelProps {
  open: boolean;
  onClose: () => void;
  items: Notification[];
  unreadCount: number;
  onMarkRead: (id: string) => void;
  onMarkAllRead: () => void;
}
⋮----
function timeAgo(iso: string): string
⋮----
const stopPropagation = (e: React.SyntheticEvent) =>
⋮----
const handler = (e: MouseEvent) =>
```

## File: apps/supplier-portal/components/PageChrome.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";
import { PageChrome as KitPageChrome } from "@/components/ui/PageChrome";
⋮----
type PageChromeProps = {
  title: string;
  description?: string;
  icon?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyIcon?: string;
  children: ReactNode;
};
```

## File: apps/supplier-portal/components/PlanningBrainPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierPlanningScenarioKey } from "@pegasusx/api-core/idempotency";
import type {
  PlanningSAndOPSnapshot,
  PlanningScenarioCompareResult,
  PlanningScenarioResult,
} from "@pegasusx/types";
⋮----
const toggleSelect = (id: string) =>
⋮----
const cloneSelected = async () =>
⋮----
const compareSelected = async () =>
⋮----
const publishSelected = async () =>
⋮----
label=
⋮----
<button type="button" className="portal-btn w-fit" disabled={busy || !selectedIds[0]} onClick={() =>
⋮----
onChange=
⋮----
<Stat label=
```

## File: apps/supplier-portal/components/SessionPackChip.tsx
```typescript
import { SessionPackChip as GenericSessionPackChip } from '@/components/ui';
import { readTokenFromCookie, supplierApiBaseUrl } from '@/lib/auth';
⋮----
export function SessionPackChip()
⋮----
return <GenericSessionPackChip baseUrl=
```

## File: apps/supplier-portal/lib/enterprise-swr.ts
```typescript
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@/lib/bridge";
import { getSupplierToken } from "./auth";
⋮----
export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";
⋮----
type Listener = () => void;
⋮----
class EnterpriseNetworkLifecycleEngine
⋮----
constructor()
⋮----
public get isOnline(): boolean
⋮----
public get state(): NetworkState
⋮----
public get syncEpoch(): number
⋮----
private notify()
⋮----
private recordTransition()
⋮----
public setOnline(online: boolean)
⋮----
public async triggerSyncEpoch()
⋮----
public reset()
⋮----
export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  cache?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};
⋮----
export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};
⋮----
export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T>
⋮----
const bootstrap = async () =>
```

## File: apps/supplier-portal/lib/foundry-api.ts
```typescript
import { createSupplierApi } from "./api";
import { ApiError } from "@pegasusx/api-core";
import type {
  OntologyObject,
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ScenarioPatch,
  ScenarioPatchRequest,
  ScenarioCommitResult,
  AMLCheckRequest,
  AMLCheckResponse,
  ClawbackSliceRequest,
  SettlementSliceRecord,
  FXCorridorRequest,
  FXCorridorResponse,
} from "@pegasusx/types";
```

## File: apps/supplier-portal/lib/order-actions.ts
```typescript
export type OrderActionFlags = {
  canDelay: boolean;
  canReject: boolean;
  canOverflow: boolean;
  canReassign: boolean;
};
⋮----
export function orderActionFlags(state: string): OrderActionFlags
```

## File: apps/supplier-portal/lib/portal-offline-tray.tsx
```typescript
import { DesktopOfflineTray, useDesktopConnectivity } from "@/components/ui";
import { isTauri } from "@pegasusx/desktop-bridge";
⋮----
export function PortalOfflineTray()
```

## File: apps/supplier-portal/lib/use-pagination.ts
```typescript
import { useMemo, useState } from "react";
⋮----
export function usePagination<T>(items: T[], pageSize: number)
```

## File: apps/supplier-portal/lib/use-supplier-ws-refresh.ts
```typescript
import { useEffect, useRef } from "react";
import { getSupplierToken, supplierApiBaseUrl } from "@/lib/auth";
import { runSupplierSessionReconcile } from "@/lib/session-reconcile";
import { parseSupplierWsEventType } from "@/lib/supplier-ws-events";
import { SSE_SUPPLIER_ENDPOINT } from "@pegasusx/ws-refresh-contract";
⋮----
type UseSupplierWsRefreshOptions = {
  eventTypes: ReadonlySet<string>;
  debounceMs?: number;
  enabled?: boolean;
};
⋮----
export function useSupplierWsRefresh(
  onSignal: (eventType: string, raw?: unknown) => void,
  { eventTypes, debounceMs = 500, enabled = true }: UseSupplierWsRefreshOptions,
)
⋮----
const clearTimers = () =>
⋮----
const closeStream = () =>
⋮----
const handleEvent = (eventType: string | null, rawData: unknown) =>
```

## File: apps/supplier-portal/vitest.config.ts
```typescript
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";
```

## File: apps/warehouse-portal/app/auth/login/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard, dialCodeForCountry } from "@/components/ui";
import { persistSession, warehouseApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import { COUNTRIES } from "../register/wizard-state";
⋮----
type LoginStep = "phone" | "otp";
⋮----
export default function WarehouseLoginPage()
⋮----
async function handleSendOtp(e: React.FormEvent)
⋮----
async function handleVerifyOtp(e: React.FormEvent)
⋮----
title=
⋮----
registerLabel=
⋮----
resetPhoneOtpFlow();
setStep("phone");
```

## File: apps/warehouse-portal/app/auth/register/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AuthRegisterShell,
  AuthRegisterIdentityStep,
  AuthRegisterVerificationStep,
  AuthRegisterProfileStep,
  dialCodeForCountry,
  FormAlert,
} from "@/components/ui";
import { persistSession, warehouseApiBaseUrl } from "@/lib/auth";
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from "@/lib/firebase";
import {
  INITIAL_STATE,
  STEP_LABELS,
  STEP_ORDER,
  type WizardState,
  validateIdentity,
  validateVerification,
  validateProfile,
} from "./wizard-state";
⋮----
function validateCurrent(): Record<string, string>
⋮----
async function next()
⋮----
function back()
⋮----
async function submit()
⋮----
title=
```

## File: apps/warehouse-portal/app/bins/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { WarehouseBinLocation, StockLot } from '@pegasusx/types';
⋮----
const createBin = async () =>
⋮----
<PageChrome title=
```

## File: apps/warehouse-portal/app/control-tower/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState, useEffect, useMemo } from "react";
import {
  HexagonalControlTowerMap,
  LiveEKGNetworkGraph,
  type NetworkNode,
  type NetworkLink,
  GlassmorphismPanel,
} from "@/components/ui";
import { useControlTowerTelemetry } from "./use-control-tower-telemetry";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Legend,
} from "recharts";
import { cellToBoundary, latLngToCell } from "h3-js";
import { decodeJwtPayload, useToken } from "@/lib/auth";
⋮----
const generateH3Data = () =>
⋮----
onClick=
```

## File: apps/warehouse-portal/app/cycle-counts/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { CycleCount, InventoryAdjustment } from '@pegasusx/types';
⋮----
const createCount = async () =>
⋮----
const submitCount = async (count: CycleCount) =>
⋮----
<PageChrome title=
⋮----
onChange=
⋮----
const res = await apiFetch(
                              `/v1/warehouse/ops/inventory-adjustments/${encodeURIComponent(a.adjustment_id)}/approve`,
                              { method: 'POST', headers: { 'Idempotency-Key': `adj-ap-${a.adjustment_id}` } },
                            );
```

## File: apps/warehouse-portal/app/dispatch/rescues/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import type { WarehouseDispatchDriver } from '@pegasusx/types';
⋮----
interface RescueOption {
  driver_id: string;
  name: string;
  license_plate?: string;
  effective_capacity_vu: number;
  is_capacity_exceeded?: boolean;
}
⋮----
async function handleSelectBrokenDriver(driver: WarehouseDispatchDriver)
⋮----
async function handleProposeRescue(rescueDriverId: string)
⋮----
title=
```

## File: apps/warehouse-portal/app/drivers/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseAssignVehicleResponse,
  WarehouseFleetDriver,
  WarehouseFleetDriverListResponse,
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { warehouseAssignDriverVehicleKey, warehouseCreateDriverKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { DriversList } from '@/components/drivers/DriversList';
⋮----
async function handleCreate(e: React.FormEvent)
⋮----
async function handleAssignVehicle(driverId: string, vehicleId: string)
⋮----
description=
⋮----
onClick=
```

## File: apps/warehouse-portal/app/foundry/page.tsx
```typescript
import React, { useState } from 'react';
import { OntologyBlastRadiusViewer } from '@/components/foundry/OntologyBlastRadiusViewer';
import { ScenarioSandboxWorkbench } from '@/components/foundry/ScenarioSandboxWorkbench';
```

## File: apps/warehouse-portal/app/inventory/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback, useRef } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseAdjustInventoryKey, warehouseInventoryPolicyKey } from '@pegasusx/api-core';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import InventoryStockList from '@/components/inventory/InventoryStockList';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';
import type { WarehouseLiveEvent } from '@pegasusx/types';
⋮----
interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
  sku_id?: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
}
⋮----
async function handlePolicyChange(productId: string, policy: string)
⋮----
function openAdjustConfirm(item: InventoryItem)
⋮----
function closeAdjustFlow()
⋮----
async function handleAdjustConfirm()
⋮----
description=
⋮----
<label htmlFor="inventory-search-products" className="sr-only">
⋮----
onChange=
⋮----
onCancelAdjust={() =>
onPolicyChange={(productId, policy) =>
```

## File: apps/warehouse-portal/app/labor-capacity/page.tsx
```typescript
import { usePortalT } from '@/lib/i18n';
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
⋮----
type ZoneCapacity = {
  zoneH3: string;
  date: string;
  totalCapacity: number;
  usedCapacity: number;
  computedAt?: string;
};
⋮----
type DriverScore = {
  driverId: string;
  score: number;
  onTimeRate: number;
  completionRate: number;
  damageRate: number;
  shopClosedRate: number;
  feedbackScore: number;
  stopsPerHour: number;
};
⋮----
const loadScore = async () =>
⋮----
const setAvailability = async () =>
```

## File: apps/warehouse-portal/app/manifests/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';
import { WAREHOUSE_DISPATCH_REFRESH_EVENTS } from '@/lib/fleet-ws-events';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
⋮----
interface Manifest {
  manifest_id: string;
  driver_name: string;
  vehicle_label: string;
  license_plate: string;
  stop_count: number;
  status: string;
  created_at: string;
}
⋮----
description=
⋮----
setDate(e.target.value);
setLoading(true);
⋮----
const res = await apiFetch(`/v1/warehouse/manifests/$
⋮----
const zpl = await res.text();
⋮----
const url = URL.createObjectURL(blob);
const a = document.createElement('a');
⋮----
a.download = `$
```

## File: apps/warehouse-portal/app/orders/page.tsx
```typescript
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-core';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { downloadCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { ListToolbar } from '@/components/ListToolbar';
import { PageChrome } from '@/components/PageChrome';
import { OrderActionDialog, OrderProposeDateDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';
import { OrdersList, type OrderRow, type OrdersTab } from './components/OrdersList';
import { PreordersList } from '@/components/preorders/PreordersList';
import { usePortalT } from '@/lib/i18n';
⋮----
function isoDeliveryDate(dateInput: string): string
⋮----
const setTab = (nextTab: OrdersTab) =>
⋮----
const openDetail = (orderId: string) =>
⋮----
const handleProposeDate = (orderId: string, isPreorder: boolean, currentDate?: string) =>
⋮----
const handleReject = (orderId: string, isPreorder: boolean) =>
⋮----
const closeDialog = () =>
⋮----
async function submitDialog()
⋮----
const exportCsv = () =>
⋮----
description=
⋮----
setLoading(true);
void load();
⋮----
<div className="wh-tab-bar mb-5" role="tablist" aria-label=
⋮----
onClick=
```

## File: apps/warehouse-portal/app/pick-waves/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { PickWave, PickTask } from '@pegasusx/types';
⋮----
const openWave = async (waveId: string) =>
⋮----
const createWave = async () =>
⋮----
const confirmTask = async (task: PickTask, qty?: number) =>
⋮----
<PageChrome title=
```

## File: apps/warehouse-portal/app/products/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
⋮----
interface Product {
  product_id: string;
  name: string;
  sku_id: string;
  category: string;
  price_uzs: number;
  is_active: boolean;
}
⋮----
} catch { /* handled */ }
⋮----
const fmt = (n: number) =>
⋮----
description=
⋮----
<label htmlFor="search-products-catalog" className="sr-only">
⋮----
onClick=
```

## File: apps/warehouse-portal/app/returns/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { warehouseInboundConfirmKey, warehouseInboundScanKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
⋮----
import { ReturnsList, type InboundRow, isClaimTicket } from '@/components/returns/ReturnsList';
import { ReverseLogisticsPanel } from '@/components/returns/ReverseLogisticsPanel';
⋮----
// OPEN = PENDING|ON_TRUCK|ARRIVED|RECEIVING — includes claim reverse-logistics tickets.
⋮----
async function ensureSession()
⋮----
async function handleScan()
⋮----
async function handleConfirm(disposition: 'RESTOCK' | 'WRITE_OFF')
⋮----
description=
⋮----
<button type="button" className=
```

## File: apps/warehouse-portal/app/staff/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { warehouseCreateStaffKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import StaffList from '@/components/staff/StaffList';
import type {
  CreateWarehouseStaffRequest,
  CreateWarehouseStaffResponse,
  WarehouseStaffListResponse,
  WarehouseStaffMember,
  WarehouseStaffRole,
} from '@pegasusx/types';
⋮----
async function loadStaff()
⋮----
async function handleRegister(e: React.FormEvent)
⋮----
description=
```

## File: apps/warehouse-portal/app/supply-requests/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import {
  warehouseReceiveTransferKey,
  warehouseSupplyRequestTransitionKey,
} from '@pegasusx/api-core';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import type { WarehouseLiveEvent, WarehouseSupplyRequestDetail } from '@pegasusx/types';
⋮----
function chipClass(state: string): string
⋮----
export default function SupplyRequestDetailPage()
⋮----
// Ignore unrelated frames.
⋮----
async function handleReceiveTransfer()
⋮----
async function handleAction(action: string)
```

## File: apps/warehouse-portal/app/supply-requests/new/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { createWarehouseApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { warehouseCreateSupplyRequestKey, warehouseSupplyRequestTransitionKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
⋮----
interface ForecastItem {
  product_id: string;
  product_name: string;
  current_stock: number;
  recommended_qty: number;
  days_until_stockout: number;
  priority: string;
  unit: string;
}
⋮----
// Manual items (when not using forecast)
⋮----
}, [useForecast]); // eslint-disable-line react-hooks/exhaustive-deps
⋮----
async function loadForecast()
⋮----
async function handleSubmit(e: React.FormEvent)
⋮----
description=
⋮----
{/* Delivery date */}
⋮----
onChange=
⋮----
{/* Manual items */}
⋮----
{/* Notes */}
```

## File: apps/warehouse-portal/app/tomorrow-board/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import { PageChrome } from '@/components/PageChrome';
import type { DeliveryExpectation } from '@pegasusx/types';
⋮----
type BoardOrder = {
  order_id: string;
  status: string;
  retailer_id?: string;
  total_minor: number;
  delivery_expectation?: DeliveryExpectation;
};
⋮----
type BoardResponse = {
  date: string;
  preorders: BoardOrder[];
  deliver_before: BoardOrder[];
  draft_manifests: Array<{ manifest_id: string; state: string; stop_count: number }>;
  loading_manifests: Array<{ manifest_id: string; state: string; stop_count: number }>;
};
⋮----
export default function TomorrowBoardPage()
⋮----
<PageChrome title=
⋮----
onChange={(e) =>
```

## File: apps/warehouse-portal/app/transfers/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import { useToast } from '@/components/Toast';
import { warehouseOps } from '@/lib/warehouse-ops';
⋮----
async function run(label: string, fn: () => Promise<{ state?: string; transfer_id?: string }>)
⋮----
description=
```

## File: apps/warehouse-portal/app/treasury/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { desktopPrint } from '@pegasusx/desktop-bridge';
import { downloadCsv } from '@/lib/csv';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { HubCard } from '@/components/portal';
import EmptyState from '@/components/EmptyState';
import { TreasuryTransactionList } from '@/components/treasury/TreasuryTransactionList';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
interface TreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
  currency?: string;
}
⋮----
export interface Invoice {
  invoice_id: string;
  retailer_name: string;
  amount?: number;
  amount_uzs?: number;
  currency?: string;
  status: string;
  fee_amount?: number;
  net_payout_amount?: number;
  payout_owner_type?: string;
  payout_owner_id?: string;
  fee_policy_version?: string;
  settlement_target?: string;
  due_date: string;
  created_at: string;
}
⋮----
const fmt = (n: number) =>
const resolveAmount = (inv: Invoice) =>
const resolveCurrency = (inv: Invoice) =>
⋮----
const formatPayoutOwner = (inv: Invoice) =>
⋮----
const exportCsv = () =>
⋮----
const exportPdf = () =>
⋮----
description=
⋮----
label=
```

## File: apps/warehouse-portal/app/vehicles/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import { warehouseCreateVehicleKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseVehiclesLive } from '@/lib/use-warehouse-vehicles-live';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { VehiclesList } from '../../components/vehicles/VehiclesList';
⋮----
async function handleCreate(e: React.FormEvent)
⋮----
description=
```

## File: apps/warehouse-portal/app/page.tsx
```typescript
import { usePolling } from '@pegasusx/api-react';
⋮----
import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ApiError, moneyCurrency } from '@pegasusx/api-core';
import {
  ORDER_STATUS_FUNNEL,
  TRUCK_DUTY_STATUSES,
  canonicalizeOrderStatus,
  canonicalizeTruckDuty,
  emptyOrderStatusCounts,
  emptyTruckDutyCounts,
  type OrderStatusFunnel,
  type TruckDutyFunnel,
  type WarehouseHoldReason,
} from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
import PageTransition from '@/components/PageTransition';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';
import { PageSection } from '@/components/PageSection';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { SourceChip, StatusStack } from '@/components/ui';
import { usePortalT } from '@/lib/i18n';
⋮----
interface DashboardData {
  active_orders: number;
  completed_today_available: boolean;
  pending_dispatch: number;
  total_drivers: number;
  total_vehicles: number;
  today_revenue_available: boolean;
  low_stock_count: number;
  total_staff: number;
  history_available: boolean;
  orders_by_status: Record<OrderStatusFunnel, number>;
  truck_duty: Record<TruckDutyFunnel, number>;
  hold_reasons: WarehouseHoldReason[];
  demand_source: string;
}
⋮----
function foldOrderCounts(raw: unknown): Record<OrderStatusFunnel, number>
⋮----
function foldTruckDuty(raw: unknown): Record<TruckDutyFunnel, number>
⋮----
function foldHoldReasons(raw: unknown): WarehouseHoldReason[]
⋮----
type DashboardLoadIssue = 'offline' | 'restricted' | 'error';
⋮----
<PageChrome icon="dashboard" title=
⋮----
description=
⋮----
onClick=
⋮----
aria-label=
⋮----
title=
```

## File: apps/warehouse-portal/components/dispatch/DispatchPanelsAndModals.tsx
```typescript
import type { Dispatch, SetStateAction } from 'react';
import type {
  WarehouseDispatchCapacityWarning,
  WarehouseFleetVehicle,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { usePortalT } from '@/lib/i18n';
import { PageSection } from '@/components/PageSection';
⋮----
export function formatUnavailableReason(reason?: string, note?: string)
⋮----
export function formatVU(value: number)
⋮----
onChange=
⋮----
<label htmlFor=
⋮----
aria-label=
⋮----
placeholder=
⋮----
setCapacityPrompt(null);
void onRunSmartDispatch(
```

## File: apps/warehouse-portal/components/forecast/ForecastChartPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ForecastProduct } from '@/app/demand-forecast/page';
```

## File: apps/warehouse-portal/components/forecast/ForecastSkuTable.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
import type { ForecastProduct } from '@/app/demand-forecast/page';
```

## File: apps/warehouse-portal/components/foundry/OntologyBlastRadiusViewer.tsx
```typescript
import React, { useState } from 'react';
import type { RecallBlastRadiusReport, OntologyObject } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
⋮----
interface OntologyBlastRadiusViewerProps {
  defaultCellId?: string;
  defaultSupplierId?: string;
}
⋮----
export function OntologyBlastRadiusViewer({
  defaultCellId = 'cell-uz',
  defaultSupplierId = 'sup_samarkand_dairy',
}: OntologyBlastRadiusViewerProps)
⋮----
const handleTriggerRecall = async (e: React.FormEvent) =>
⋮----
const handleSearchAround = async () =>
⋮----
onChange={(e) =>
```

## File: apps/warehouse-portal/components/foundry/ScenarioSandboxWorkbench.tsx
```typescript
import React, { useState } from 'react';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
⋮----
interface ScenarioSandboxWorkbenchProps {
  defaultScenarioId?: string;
  defaultCellId?: string;
  defaultSupplierId?: string;
}
⋮----
const handleApplyPatch = async (e: React.FormEvent) =>
⋮----
const handleResolvePreview = async () =>
⋮----
const handleCommit = async () =>
⋮----
onChange=
```

## File: apps/warehouse-portal/components/inventory/InventoryStockList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
⋮----
interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
  sku_id?: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
}
⋮----
interface InventoryStockListProps {
  items: InventoryItem[];
  loading: boolean;
  search: string;
  lowOnly: boolean;
  adjusting: string | null;
  adjustVal: string;
  pulsedProductIds: string[];
  onAdjustValChange: (val: string) => void;
  onStartAdjust: (item: InventoryItem) => void;
  onReviewAdjust: (item: InventoryItem) => void;
  onCancelAdjust: () => void;
  onPolicyChange: (productId: string, policy: string) => void;
}
⋮----
onChange=
⋮----
onClick=
```

## File: apps/warehouse-portal/components/operations/OperationsBroadcastForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { type ReactNode } from 'react';
import type { BroadcastTemplate } from '@pegasusx/types';
import { PageSection } from '@/components/PageSection';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
⋮----
interface OperationsBroadcastFormProps {
  templates: BroadcastTemplate[];
  loading: boolean;
  title: string;
  setTitle: (val: string) => void;
  body: string;
  setBody: (val: string) => void;
  broadcastRole: (typeof broadcastRoles)[number];
  setBroadcastRole: (val: (typeof broadcastRoles)[number]) => void;
  templateDate: string;
  setTemplateDate: (val: string) => void;
  customReason: string;
  setCustomReason: (val: string) => void;
  saveAsTemplate: boolean;
  setSaveAsTemplate: (val: boolean) => void;
  broadcasting: boolean;
  savingTemplate: boolean;
  onSelectTemplate: (t: BroadcastTemplate) => void;
  onDeleteTemplate: (t: BroadcastTemplate) => void;
  onBroadcast: () => void;
}
⋮----
<PageSection title=
⋮----
<PortalSection title=
⋮----
<input id="operationsbroadcastform-checkbox-1" aria-label="Select Operationsbroadcastform option" type="checkbox" checked=
```

## File: apps/warehouse-portal/components/orders/OrderOpsCard.tsx
```typescript
import { motion } from 'framer-motion';
import { OrderKebabMenu } from './OrderKebabMenu';
import { OrderStateChip } from './OrderStateChip';
import { orderActionFlags } from '@/lib/order-actions';
⋮----
export type OrderDetailOpenMode = 'single' | 'double';
⋮----
export type OrderOpsCardProps = {
  orderId: string;
  retailerName: string;
  state: string;
  amountLabel: string;
  meta?: string;
  badge?: string;
  index?: number;
  disabled?: boolean;
  detailOpenMode?: OrderDetailOpenMode;
  onOpenDetail: () => void;
  onProposeDate?: () => void;
  onReject?: () => void;
  showOpsMenu?: boolean;
  showQuickActions?: boolean;
  proposeDateLabel?: string;
  rejectLabel?: string;
  canProposeDateOverride?: boolean;
  canRejectOverride?: boolean;
};
⋮----
export function OrderOpsCard({
  orderId,
  retailerName,
  state,
  amountLabel,
  meta,
  badge,
  index = 0,
  disabled = false,
  detailOpenMode = 'single',
  onOpenDetail,
  onProposeDate,
  onReject,
  showOpsMenu = true,
  showQuickActions = true,
  proposeDateLabel = 'Propose date',
  rejectLabel = 'Cancel order',
  canProposeDateOverride,
  canRejectOverride,
}: OrderOpsCardProps)
⋮----
const stopPropagation = (e: React.SyntheticEvent) =>
⋮----
function handleCardClick()
⋮----
function handleCardDoubleClick(e: React.MouseEvent)
```

## File: apps/warehouse-portal/components/orders/OrderProposeDateDialog.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useRef } from 'react';
⋮----
type OrderProposeDateDialogProps = {
  open: boolean;
  proposedDate: string;
  onProposedDateChange: (value: string) => void;
  reason: string;
  onReasonChange: (value: string) => void;
  submitting?: boolean;
  onConfirm: () => void;
  onClose: () => void;
  title?: string;
};
```

## File: apps/warehouse-portal/components/portal/PortalPrimitives.tsx
```typescript
import Icon from "@/components/Icon";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import {
  PortalField as KitPortalField,
  PortalInput as KitPortalInput,
  PortalSelect as KitPortalSelect,
  PortalSection as KitPortalSection,
  PortalActions as KitPortalActions,
  SelectionCard as KitSelectionCard,
  DataList as KitDataList,
  DataListRow as KitDataListRow,
  HubCard as KitHubCard,
  FormAlert as KitFormAlert,
} from "@/components/ui/PortalPrimitives";
⋮----
export function PortalSection({
  icon,
  ...props
}: Omit<ComponentProps<typeof KitPortalSection>, "icon"> & { icon?: string })
```

## File: apps/warehouse-portal/components/returns/ReturnsList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Icon from '@/components/Icon';
import React from 'react';
⋮----
export type InboundRow = {
  return_id: string;
  order_id: string;
  product_name: string;
  expected_qty: number;
  received_qty: number;
  reason: string;
  physical_status: string;
  driver_name?: string;
  driver_notes?: string;
  suggested_disposition: string;
  barcode?: string;
};
⋮----
export function isClaimTicket(row: InboundRow): boolean
⋮----
export interface ReturnsListProps {
  tab: 'inbound' | 'history';
  loading: boolean;
  list: InboundRow[];
  selected: Set<string>;
  onToggleSelect: (id: string) => void;
}
⋮----
onChange=
```

## File: apps/warehouse-portal/components/settings/OpsSettingsForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { ChangeEvent } from 'react';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
⋮----
export interface OpsSettingsFormProps {
  preorderMinLeadDays: string;
  setPreorderMinLeadDays: (val: string) => void;
  preorderMaxLeadDays: string;
  setPreorderMaxLeadDays: (val: string) => void;
  orderLineMin: string;
  setOrderLineMin: (val: string) => void;
  orderLineMax: string;
  setOrderLineMax: (val: string) => void;
  feeBaseMinor: string;
  setFeeBaseMinor: (val: string) => void;
  feeCurrency: string;
  feeTierKm: string;
  setFeeTierKm: (val: string) => void;
  feeTierMinor: string;
  setFeeTierMinor: (val: string) => void;
  showStockCounts: boolean;
  setShowStockCounts: (val: boolean) => void;
  policy: 'REJECT' | 'ACCEPT_BACKORDER';
  setPolicy: (val: 'REJECT' | 'ACCEPT_BACKORDER') => void;
  enforceOrderAcceptance: boolean;
  setEnforceOrderAcceptance: (val: boolean) => void;
  scheduleIs24h: boolean;
  setScheduleIs24h: (val: boolean) => void;
  scheduleTimezone: string;
  setScheduleTimezone: (val: string) => void;
  weekdayOpen: string;
  setWeekdayOpen: (val: string) => void;
  weekdayClose: string;
  setWeekdayClose: (val: string) => void;
  scheduleJSON: string;
  setScheduleJSON: (val: string) => void;
}
⋮----
<PortalSection icon="orders" title=
⋮----
<PortalSection icon="inventory" title=
⋮----
<input id="opssettingsform-checkbox-5" aria-label="Select Opssettingsform option" type="checkbox" checked=
⋮----
<input id="opssettingsform-checkbox-2" aria-label="Select Opssettingsform option" type="checkbox" checked=
⋮----
<input id="opssettingsform-checkbox-1" aria-label="Select Opssettingsform option" type="checkbox" checked=
```

## File: apps/warehouse-portal/components/settings/ReturnPolicySettingsSection.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { warehouseReturnPolicyPutKey } from '@pegasusx/api-core';
import type { WarehouseReturnPolicy } from '@pegasusx/types';
⋮----
async function save()
⋮----
onChange=
```

## File: apps/warehouse-portal/components/ui/AuthLoginCard.tsx
```typescript
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { PortalField, PortalInput, PortalSelect, FormAlert } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthLoginStep = "phone" | "otp";
```

## File: apps/warehouse-portal/components/ui/AuthRegisterStepper.tsx
```typescript
import type { ReactNode } from "react";
```

## File: apps/warehouse-portal/components/ui/AuthRegisterSteps.tsx
```typescript
import type { Dispatch, SetStateAction } from "react";
import { PortalField } from "./PortalPrimitives";
import { AUTH_COUNTRIES, dialCodeForCountry } from "./countries";
⋮----
export type AuthIdentityStep = {
  countryCode: string;
  phoneLocal: string;
};
⋮----
export type AuthVerificationStep = {
  otpCode: string;
  idToken: string;
};
⋮----
export type AuthProfileStep = {
  legalName: string;
  contactName: string;
  email: string;
};
⋮----
type Errors = Record<string, string>;
```

## File: apps/warehouse-portal/components/ui/countries.ts
```typescript
export interface AuthCountry {
  code: string;
  name: string;
  dialCode: string;
  currency: string;
}
⋮----
export function dialCodeForCountry(code: string): string
```

## File: apps/warehouse-portal/components/ui/DemandSourceChips.tsx
```typescript
import type { CSSProperties } from "react";
⋮----
export type DemandSourceCode = "STORE_POS" | "WHOLESALE_HISTORY" | string;
⋮----
export function demandSourceLabel(code: string): string
⋮----
export function normalizeDemandSources(sources?: string[] | null): string[]
⋮----
export type DemandSourceChipsProps = {
  sources?: string[] | null;
  className?: string;
  size?: "sm" | "md";
};
```

## File: apps/warehouse-portal/components/ui/DesktopKeyboardCheatsheet.tsx
```typescript
import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";
⋮----
export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};
⋮----
export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
```

## File: apps/warehouse-portal/components/ui/DesktopOfflineTray.tsx
```typescript
import { useState } from "react";
import { ChevronDown, ChevronUp, Clock, RefreshCw, WifiOff } from "lucide-react";
⋮----
export type DesktopQueuedAction = {
  id: string;
  label: string;
  subtitle?: string;
};
⋮----
export type DesktopOfflineTrayProps = {
  isOffline: boolean;
  queuedActions?: DesktopQueuedAction[];
  onRetryAll?: () => void | Promise<void>;
  retrying?: boolean;
  offlineMessage?: string;
  className?: string;
};
```

## File: apps/warehouse-portal/components/ui/ExplainStatusBanner.tsx
```typescript
import type { StatusExplain } from '@pegasusx/types';
⋮----
export type ExplainStatusBannerProps = {
  errorCode?: string | null;
  explain?: StatusExplain | null;
  detail?: string | null;
  className?: string;
};
```

## File: apps/warehouse-portal/components/ui/firebase-emulator-phone.ts
```typescript
function emulatorIdentityToolkitBase(): string
⋮----
export async function sendPhoneOtpViaEmulator(
  phone: string,
  apiKey = DEFAULT_API_KEY,
): Promise<void>
⋮----
export async function verifyPhoneOtpViaEmulator(
  code: string,
  apiKey = DEFAULT_API_KEY,
): Promise<string>
⋮----
export function resetEmulatorPhoneOtpFlow(): void
```

## File: apps/warehouse-portal/components/ui/GenericFleetLiveMap.tsx
```typescript
import { useEffect, useMemo, useRef } from 'react';
⋮----
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import { useMapLibreTeardown } from './map-chrome';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from './useAnimatedDriverMarkers';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
export interface GenericRouteType {
  driver_location?: { lat?: number; lng?: number; latitude?: number; longitude?: number };
  live_location_available?: boolean;
  location_stale?: boolean;
  route_geometry?: any;
  manifest_id: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
}
⋮----
export type FleetLiveMapProps = {
  routes: GenericRouteType[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  enable3DView?: boolean;
};
⋮----
function routeLineFeature(
  route: GenericRouteType,
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null
⋮----
const t = (key: string) =>
⋮----
mapRef.current = ref?.getMap() ?? null;
⋮----
...mapInitialViewState(readCachedAuthSession()?.pack),
```

## File: apps/warehouse-portal/components/ui/GlassmorphismPanel.tsx
```typescript
import React from "react";
import { motion, HTMLMotionProps } from "framer-motion";
⋮----
export interface GlassmorphismPanelProps extends HTMLMotionProps<"div"> {
  children: React.ReactNode;
  className?: string;
}
⋮----
export function GlassmorphismPanel({
  children,
  className = "",
  ...props
}: GlassmorphismPanelProps)
```

## File: apps/warehouse-portal/components/ui/HandoffCard.tsx
```typescript
import type { HandoffCardMetadata } from '@pegasusx/types';
⋮----
export type HandoffCardProps = {
  metadata: HandoffCardMetadata;
  className?: string;
  onAction?: (link: string) => void;
};
```

## File: apps/warehouse-portal/components/ui/HealthStrip.tsx
```typescript
export type HealthStripItem = {
  key: string;
  label: string;
  count: number;
  href?: string;
};
⋮----
export type HealthStripProps = {
  items: HealthStripItem[];
  onSelect?: (key: string) => void;
  className?: string;
};
⋮----
onClick={onSelect ? () => onSelect(item.key) : undefined}
          disabled=
```

## File: apps/warehouse-portal/components/ui/HexagonalControlTowerMap.tsx
```typescript
import React, { useMemo, useState } from "react";
import MapGL from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import DeckGL from "@deck.gl/react";
import { H3HexagonLayer } from "@deck.gl/geo-layers";
import { Map3DViewToggle } from "./map-chrome";
⋮----
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-core";
import type { MarketPack } from "@pegasusx/types";
⋮----
export interface HexagonalControlTowerMapProps {
  data: { hex: string; count: number }[];
  show3DViewToggle?: boolean;
  pack?: MarketPack | null;
}
```

## File: apps/warehouse-portal/components/ui/KpiStat.tsx
```typescript
import type { HistorySeries } from "@pegasusx/types";
import { guardHistorySeries } from "@pegasusx/types";
import type { ReactNode } from "react";
import { SourceChip } from "./SourceChip";
⋮----
export type KpiStatProps = {
  label: string;
  value: ReactNode;
  delta?: ReactNode;
  spark?: HistorySeries | null;
  source: string;
  className?: string;
};
```

## File: apps/warehouse-portal/components/ui/LiveEKGNetworkGraph.tsx
```typescript
import React, { useEffect, useRef } from "react";
⋮----
export interface NetworkNode extends d3.SimulationNodeDatum {
  id: string;
  type: "warehouse" | "retailer" | "driver";
  label: string;
  status: "active" | "idle" | "busy";
}
⋮----
export interface NetworkLink extends d3.SimulationLinkDatum<NetworkNode> {
  source: string | NetworkNode;
  target: string | NetworkNode;
  value: number;
}
⋮----
export interface LiveEKGNetworkGraphProps {
  nodes: NetworkNode[];
  links: NetworkLink[];
  width?: number;
  height?: number;
}
⋮----
export function LiveEKGNetworkGraph({
  nodes,
  links,
  width = 800,
  height = 600,
}: LiveEKGNetworkGraphProps)
⋮----
function pulse()
⋮----
const drag = (simulation: any) =>
⋮----
function dragstarted(event: any, d: any)
function dragged(event: any, d: any)
function dragended(event: any, d: any)
```

## File: apps/warehouse-portal/components/ui/map-chrome.tsx
```typescript
import { useEffect, useRef, useState, type RefObject } from "react";
import type maplibregl from "maplibre-gl";
⋮----
export function useLazyMapMount(rootMargin = "120px")
⋮----
export function useMapLibreTeardown(mapRef: RefObject<maplibregl.Map | null>)
⋮----
type Map3DViewToggleProps = {
  enabled: boolean;
  onChange: (enabled: boolean) => void;
  className?: string;
};
⋮----
export function Map3DViewToggle({ enabled, onChange, className = "" }: Map3DViewToggleProps)
⋮----
onClick=
```

## File: apps/warehouse-portal/components/ui/motion.ts
```typescript
export type CubicBezierEasing = [number, number, number, number];
⋮----
export type MotionTransition = {
  duration?: number;
  ease?: CubicBezierEasing;
};
⋮----
export function reducedMotionTransition(
  prefersReduced: boolean,
  transition: MotionTransition,
): MotionTransition
```

## File: apps/warehouse-portal/components/ui/PackChip.tsx
```typescript
type PackChipProps = {
  currency?: string | null;
  receipts?: string | null;
  title?: string;
};
⋮----
export function PackChip({ currency, receipts, title }: PackChipProps)
```

## File: apps/warehouse-portal/components/ui/PageChrome.tsx
```typescript
import type { ReactNode } from "react";
⋮----
export type PageChromeProps = {
  title: string;
  description?: string;
  icon?: ReactNode;
  actions?: ReactNode;
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  children: ReactNode;
  renderLoading?: () => ReactNode;
  renderError?: (message: string) => ReactNode;
  renderEmpty?: () => ReactNode;
};
```

## File: apps/warehouse-portal/components/ui/PageSkeletons.tsx
```typescript
import type { CSSProperties } from "react";
```

## File: apps/warehouse-portal/components/ui/PlanBrainTabs.tsx
```typescript
import type { PlanBrainTab } from "@pegasusx/types";
⋮----
export type PlanBrainTabsProps = {
  value: PlanBrainTab;
  onChange: (tab: PlanBrainTab) => void;
  className?: string;
};
⋮----
export function PlanBrainTabs({ value, onChange, className }: PlanBrainTabsProps)
```

## File: apps/warehouse-portal/components/ui/PortalPrimitives.tsx
```typescript
import type { ComponentType, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
⋮----
export function PortalField({
  id,
  label,
  optional,
  error,
  hint,
  children,
}: {
  id: string;
  label: string;
  optional?: boolean;
  error?: string;
  hint?: string;
  children: ReactNode;
})
⋮----
{optional ? <span style={{ fontWeight: 500, color: "var(--desk-text-tertiary)" }}> (optional)</span> : null}
      </label>
      {children}
⋮----
{icon ? (
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px]"
              style={{ background: "var(--desk-surface-subtle)", color: "var(--desk-text-secondary)" }}
              aria-hidden
            >
              {icon}
            </div>
          ) : null}
          <div className="min-w-0">
            <h2 className="bento-card-title">{title}</h2>
⋮----
export function DataListRow({
  children,
  onClick,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  className?: string;
})
⋮----
type LinkProps = { href: string; className?: string; children: ReactNode };
⋮----
export function HubCard({
  href,
  title,
  description,
  icon,
  onClick,
  LinkComponent,
}: {
  href?: string;
  title: string;
  description: string;
  icon?: ReactNode;
  onClick?: () => void;
  LinkComponent?: ComponentType<any>;
})
⋮----
export function FormAlert({
  variant = "info",
  icon,
  children,
}: {
  variant?: "info" | "error";
  icon?: ReactNode;
  children: ReactNode;
})
```

## File: apps/warehouse-portal/components/ui/PulseTimeline.tsx
```typescript
import type { PulseEvent } from '@pegasusx/types';
⋮----
export type PulseTimelineProps = {
  events: PulseEvent[];
  loading?: boolean;
  error?: string | null;
  emptyMessage?: string;
  className?: string;
  onSelect?: (event: PulseEvent) => void;
};
⋮----
function formatWhen(iso: string): string
⋮----
export function PulseTimeline({
  events,
  loading = false,
  error = null,
  emptyMessage = 'No recent activity yet.',
  className = '',
  onSelect,
}: PulseTimelineProps)
⋮----
return <div className=
```

## File: apps/warehouse-portal/components/ui/RangeToggle.tsx
```typescript
import type { DashboardHistoryRange } from "@pegasusx/types";
⋮----
export type RangeToggleProps = {
  value: DashboardHistoryRange;
  onChange: (range: DashboardHistoryRange) => void;
  className?: string;
};
```

## File: apps/warehouse-portal/components/ui/SessionPackChip.tsx
```typescript
import { useMarketPack } from '@pegasusx/api-react';
import { PackChip } from './PackChip';
import { fiscalReceiptLabel } from '@pegasusx/api-core';
⋮----
receipts={fiscalReceiptLabel(pack?.fiscal_adapter)}
      title=
```

## File: apps/warehouse-portal/components/ui/SourceChip.tsx
```typescript
export type HonestySource =
  | "live"
  | "spanner"
  | "empty"
  | "unavailable"
  | "memory"
  | "env_default";
⋮----
export function sourceChipLabel(source: string): string
⋮----
export type SourceChipProps = {
  source: string;
  className?: string;
};
⋮----
export function SourceChip({ source, className }: SourceChipProps)
```

## File: apps/warehouse-portal/components/ui/StatusStack.preview.tsx
```typescript
import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts } from "@pegasusx/types";
import { StatusStack } from "./StatusStack";
```

## File: apps/warehouse-portal/components/ui/StatusStack.tsx
```typescript
import { statusStackModel, type StatusStackMode } from "@pegasusx/types";
import { SourceChip } from "./SourceChip";
⋮----
export type StatusStackProps = {
  dictionary: readonly string[];
  counts?: Record<string, number> | null;
  available?: boolean;
  source?: string;
  onSelect?: (key: string) => void;
  labelFor?: (key: string) => string;
  className?: string;
};
⋮----
function defaultLabel(key: string): string
⋮----
function sourceForMode(mode: StatusStackMode, source?: string): string
⋮----
onClick={onSelect ? () => onSelect(row.key) : undefined}
                disabled=
```

## File: apps/warehouse-portal/components/ui/use-desktop-connectivity.ts
```typescript
import { useCallback, useSyncExternalStore } from "react";
import { NetworkLifecycleEngine } from "@/lib/enterprise-swr";
⋮----
export type DesktopConnectivity = {
  isOffline: boolean;
  onOnline: (listener: () => void) => () => void;
};
⋮----
export function useDesktopConnectivity(): DesktopConnectivity
```

## File: apps/warehouse-portal/components/ui/useAnimatedDriverMarkers.ts
```typescript
import { useEffect, useMemo, useRef, useState } from 'react';
import type { GenericRouteType } from './GenericFleetLiveMap';
⋮----
type DriverAnimState = {
  driverId: string;
  fromLng: number;
  fromLat: number;
  toLng: number;
  toLat: number;
  startMs: number;
  color: string;
  label: string;
  stale: boolean;
};
⋮----
function lerp(a: number, b: number, t: number): number
⋮----
function easeOut(t: number): number
⋮----
function driverTarget(route: GenericRouteType): { lat: number; lng: number } | null
⋮----
export function useAnimatedDriverMarkers(
  routes: GenericRouteType[],
  durationMs = 1200,
): GeoJSON.FeatureCollection<GeoJSON.Point>
⋮----
const tick = () =>
```

## File: apps/warehouse-portal/components/ui/VirtualScrollList.tsx
```typescript
import type { CSSProperties, ReactNode } from "react";
⋮----
export type VirtualScrollListProps<T> = {
  items: readonly T[];
  height?: CSSProperties["height"];
  className?: string;
  itemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  virtualizationThreshold?: number;
};
⋮----
export function VirtualScrollList<T>({
  items,
  height = "28rem",
  className,
  itemKey,
  renderItem,
}: VirtualScrollListProps<T>)
⋮----
<div key=
```

## File: apps/warehouse-portal/components/FleetLiveMapPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from 'react';
import { Map3DViewToggle, useLazyMapMount } from '@/components/ui';
import FleetLiveMap from '@/components/FleetLiveMap';
import { useWarehouseFleetLiveMap } from '@/lib/use-warehouse-fleet-live-map';
⋮----
type FleetLiveMapPanelProps = {
  className?: string;
  show3DViewToggle?: boolean;
};
```

## File: apps/warehouse-portal/components/HandoffTimelinePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import { PulseTimeline } from '@/components/ui';
import type { PulseEvent } from '@pegasusx/types';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
⋮----
function isHandoffEvent(event: PulseEvent): boolean
⋮----
type HandoffSource = 'warehouse' | 'factory';
⋮----
const handler = () =>
```

## File: apps/warehouse-portal/components/Icon.tsx
```typescript
import {
  LayoutDashboard, Package, Truck, Users, ClipboardCheck,
  TrendingUp, LogOut, Sun, Moon, Monitor, Warehouse, Lock,
  AlertTriangle, RefreshCw, type LucideIcon,
  ArrowDownUp, Lightbulb, Plus, ChevronRight, ChevronLeft,
  FileText, Send, CheckCircle, XCircle, Clock, BarChart3,
  ShoppingCart, Boxes, PackageOpen, CreditCard, DollarSign,
  UserCheck, Route, Settings, Keyboard,
} from 'lucide-react';
```

## File: apps/warehouse-portal/components/NetworkPulsePanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@/components/ui";
import type { PulseEvent } from "@pegasusx/types";
import { warehouseApi } from "@/lib/warehouse-api";
```

## File: apps/warehouse-portal/components/PageChrome.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Icon from "./Icon";
import EmptyState from "./EmptyState";
import { PageSkeleton } from "./Skeleton";
import { PageChrome as KitPageChrome } from "@/components/ui/PageChrome";
⋮----
type PageChromeProps = {
  title: string;
  description?: string;
  icon?: string;
  actions?: ReactNode;
  loading?: boolean;
  skeletonVariant?: "dashboard" | "table" | "form";
  error?: string | null;
  empty?: boolean;
  emptyMessage?: string;
  emptyIcon?: string;
  children: ReactNode;
};
```

## File: apps/warehouse-portal/components/SessionPackChip.tsx
```typescript
import { SessionPackChip as GenericSessionPackChip } from '@/components/ui';
import { readTokenFromCookie, warehouseApiBaseUrl } from '@/lib/auth';
⋮----
export function SessionPackChip()
⋮----
return <GenericSessionPackChip baseUrl=
```

## File: apps/warehouse-portal/components/VehicleAvailabilityPanel.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import type { WarehouseFleetVehicle, WarehouseVehicleUnavailableReason } from '@pegasusx/types';
import { warehouseUpdateVehicleKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import {
  formatUnavailableReason,
  VEHICLE_UNAVAILABLE_REASONS,
  vehicleStatusLabel,
} from '@/lib/vehicle-fleet';
⋮----
type VehicleAvailabilityPanelProps = {
  vehicle: WarehouseFleetVehicle;
  onUpdated?: (vehicle: WarehouseFleetVehicle) => void;
  compact?: boolean;
};
⋮----
export default function VehicleAvailabilityPanel({
  vehicle,
  onUpdated,
  compact = false,
}: VehicleAvailabilityPanelProps)
⋮----
async function toggleAvailability(nextActive: boolean)
⋮----
placeholder=
⋮----
onChange=
⋮----
onClick={() =>
```

## File: apps/warehouse-portal/lib/enterprise-swr.ts
```typescript
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "@pegasusx/desktop-cache";
import { isTauri } from "@/lib/bridge";
import { readTokenFromCookie } from "./auth";
⋮----
export type NetworkState =
  | "OFFLINE"
  | "CONNECTING"
  | "CONNECTED"
  | "REVALIDATING"
  | "FLAPPING"
  | "CAPTIVE_BLOCKED";
⋮----
type Listener = () => void;
⋮----
class EnterpriseNetworkLifecycleEngine
⋮----
constructor()
⋮----
public get isOnline(): boolean
⋮----
public get state(): NetworkState
⋮----
public get syncEpoch(): number
⋮----
private notify()
⋮----
private recordTransition()
⋮----
public setOnline(online: boolean)
⋮----
public async triggerSyncEpoch()
⋮----
public reset()
⋮----
export type EnterpriseSWROptions<T> = {
  interval?: number;
  initialData?: T;
  cache?: boolean;
  onSuccess?: (data: T) => void;
  onError?: (err: Error) => void;
};
⋮----
export type EnterpriseSWRResult<T> = {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  isRevalidating: boolean;
  isOffline: boolean;
  networkState: NetworkState;
  revalidate: () => Promise<void>;
};
⋮----
export function useEnterpriseSWR<T>(
  key: string,
  fetcher: (url: string, signal?: AbortSignal) => Promise<T>,
  options: EnterpriseSWROptions<T> = {},
): EnterpriseSWRResult<T>
⋮----
const bootstrap = async () =>
```

## File: apps/warehouse-portal/lib/order-actions.ts
```typescript
export type OrderActionFlags = {
  canDelay: boolean;
  canReject: boolean;
  canOverflow: boolean;
};
⋮----
export function orderActionFlags(state: string): OrderActionFlags
```

## File: apps/warehouse-portal/lib/portal-offline-tray.tsx
```typescript
import { DesktopOfflineTray, useDesktopConnectivity } from "@/components/ui";
import { isTauri } from "@pegasusx/desktop-bridge";
⋮----
export function PortalOfflineTray()
```

## File: apps/warehouse-portal/lib/use-pagination.ts
```typescript
import { useMemo, useState } from "react";
⋮----
export function usePagination<T>(items: T[], pageSize: number)
```

## File: apps/warehouse-portal/lib/warehouse-ops.ts
```typescript
import {
  warehouseDispatchLockAcquireKey,
  warehouseDispatchLockReleaseKey,
  warehouseEmergencyTransferKey,
  warehouseForceReceiveKey,
  warehouseOrderDelayKey,
  warehouseOrderOverflowKey,
  warehouseOrderProposeDeliveryKey,
  warehouseOrderRejectKey,
  warehouseReceiveTransferKey,
} from '@pegasusx/api-core';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId, warehouseScopeQuery } from '@/lib/warehouse-scope';
import { apiFetch } from '@/lib/auth';
```

## File: apps/admin-portal/app/analytics/page.tsx
```typescript
import React, { useState } from "react";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
type RoleCategory = "ALL" | "SUPPLIER" | "WAREHOUSE" | "FLEET" | "RETAILER";
⋮----
interface ScorecardMetric {
  name: string;
  category: "SUPPLIER" | "WAREHOUSE" | "FLEET" | "RETAILER";
  currentValue: string;
  targetValue: string;
  trend: "UP" | "DOWN" | "NEUTRAL";
  trendValue: string;
  status: "OPTIMAL" | "ATTENTION" | "CRITICAL";
  benchmarkDesc: string;
}
```

## File: apps/admin-portal/app/api/auth/login/route.ts
```typescript
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
⋮----
const BACKEND_BASE = () =>
⋮----
export async function POST(req: Request)
```

## File: apps/admin-portal/app/compliance/aml/page.tsx
```typescript
import React, { useState } from "react";
import type { AMLCheckResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
const handleCheck = async (e: React.FormEvent) =>
⋮----
onChange={(e) => setTin(e.target.value)}
                required
```

## File: apps/admin-portal/app/logistics/dvir/page.tsx
```typescript
import React, { useState } from "react";
import type { FleetDVIRInspection, DispatchCheckResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
export default function DVIRFleetPage()
⋮----
const togglePoint = (key: keyof typeof checklist) =>
⋮----
const handleSubmitInspection = async (e: React.FormEvent) =>
⋮----
const handleRunDispatchCheck = async () =>
⋮----
onChange={(e) =>
```

## File: apps/admin-portal/app/telemetry/page.tsx
```typescript
import React, { useEffect, useState, useMemo } from "react";
import { api, type AuditRow } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
type SeverityLevel = "ALL" | "CRITICAL" | "ERROR" | "WARN" | "AUDIT" | "INFO";
type RoleScope = "ALL" | "PLATFORM_ADMIN" | "SUPPLIER" | "WAREHOUSE" | "FLEET_DRIVER" | "DEPOT_CASHIER" | "RETAILER" | "YARD_SECURITY";
⋮----
interface OperationalEvent {
  id: string;
  timestamp: string;
  severity: "CRITICAL" | "ERROR" | "WARN" | "AUDIT" | "INFO";
  role: string;
  entityType: "SUPPLIER" | "WAREHOUSE" | "RETAILER" | "DRIVER" | "VEHICLE" | "ORDER" | "SYSTEM";
  entityId: string;
  action: string;
  summary: string;
  traceId: string;
  rawPayload: Record<string, unknown>;
}
⋮----
// Ingest Spanner audit records into live operational stream
⋮----
const pullAudits = async () =>
⋮----
const exportDiagnostics = () =>
⋮----
onChange=
```

## File: apps/admin-portal/components/layout/AdminAppShell.tsx
```typescript
import React, { useState, useEffect, type ReactNode } from "react";
import { AdminAuthGate } from "@/components/auth/AdminAuthGate";
import { AdminSidebar } from "./AdminSidebar";
import { AdminHeader } from "./AdminHeader";
import { useAdminToken } from "@/lib/session";
import { useAdminWS } from "@/lib/ws-provider";
import { api } from "@/lib/api";
import { DesktopKeyboardCheatsheet } from "@/components/ui/DesktopKeyboardCheatsheet";
⋮----
export function AdminAppShell({ children }: { children: ReactNode })
⋮----
const handler = (e: KeyboardEvent) =>
⋮----
async function loadBadges()
⋮----
onOpenSidebar=
```

## File: apps/admin-portal/components/layout/AdminHeader.tsx
```typescript
import React, { useState } from "react";
import { useAdminWS } from "@/lib/ws-provider";
import { useAdminToken } from "@/lib/session";
import { Keyboard } from "lucide-react";
⋮----
const statusBadge = () =>
⋮----
reconnect();
setShowWsPopover(false);
```

## File: apps/admin-portal/components/layout/AdminSidebar.tsx
```typescript
import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
⋮----
interface NavItem {
  href: string;
  label: string;
  badge?: string | number;
}
⋮----
interface NavSection {
  title: string;
  items: NavItem[];
}
```

## File: apps/admin-portal/components/ui/Icon.tsx
```typescript
import {
  LayoutDashboard, BookOpen, CheckCircle, Landmark, Truck, ShieldCheck,
  TrendingUp, DollarSign, Package, BarChart3, ClipboardList, ClipboardCheck,
  RotateCcw, Users, Settings, Warehouse, Globe, XCircle, Heart, LogOut,
  Menu, ChevronRight, MapPin, Search, Bell, X, Phone, Mail, ArrowLeft,
  AlertTriangle, Sun, Moon, Monitor, CreditCard, Calendar, ChevronLeft,
  BadgeCheck, Pencil, AlertCircle, RefreshCw, Building2, Factory, Plus,
  Hexagon, PanelLeftClose, PanelLeftOpen, Image, ArrowRight, Eye, EyeOff,
  User, UserPlus, Keyboard,
  type LucideIcon,
} from 'lucide-react';
```

## File: apps/admin-portal/components/CashierVaultMonitor.tsx
```typescript
import React, { useState } from 'react';
import { TIYIN_PER_UZS, STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse, type CashierSessionView } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
⋮----
export default function CashierVaultMonitor({ token }: { token: string })
⋮----
const handleCheckAML = async (e: React.FormEvent) =>
⋮----
const handleStartSession = async () =>
⋮----
const handleSubmitCount = async () =>
⋮----
onChange={(e) => setTin(e.target.value)}
```

## File: apps/admin-portal/components/FleetYardMonitor.tsx
```typescript
import React, { useState } from 'react';
import type { FleetDVIRInspection, YardGateCheckEvent, DispatchCheckResponse } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
⋮----
export default function FleetYardMonitor({ token }: { token: string })
⋮----
const handleTestDVIR = async (failCheck: boolean) =>
⋮----
const handleGateEvent = async () =>
⋮----
const handleDispatchCheck = async () =>
⋮----
onChange=
```

## File: apps/admin-portal/lib/foundry-api.ts
```typescript
import type {
  OntologyObject,
  OntologyGraphEdge,
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
  ActionExecutionRecord,
  ScenarioPatch,
  ScenarioPatchRequest,
  ScenarioCommitResult,
  StartCashierSessionRequest,
  CashierSessionView,
  BlindCountRequest,
  RRNTenderReclassificationRequest,
  CITHandoverRequest,
  DepotVaultLedgerEntry,
  FleetDVIRInspection,
  YardGateCheckEvent,
  DispatchCheckResponse,
  MidRouteRescueRequest,
  MidRouteRescueResult,
  SplitTenderRejectionInput,
  SplitTenderReconciliationResult,
  AMLCheckRequest,
  AMLCheckResponse,
  OfflineLeaseIssueRequest,
  OfflineQuotaLease,
  OfflineLeaseValidateRequest,
  OfflineLeaseValidateResponse,
  FiscalPreflightRequest,
  FiscalPreflightResponse,
  SlicePayoutEligibilityRequest,
  SlicePayoutEligibilityResponse,
  ClawbackSliceRequest,
  SettlementSliceRecord,
  Load3DRTIRequest,
  ReturnableContainerBalance,
  FXCorridorRequest,
  FXCorridorResponse,
  PickToteState,
  ToteTakeoverRequest,
} from "@pegasusx/types";
⋮----
const base = () =>
⋮----
async function req<T>(token: string, method: string, path: string, body?: unknown): Promise<T>
```

## File: apps/admin-portal/lib/use-admin-ws-refresh.ts
```typescript
import { useEffect, useRef } from "react";
import { api } from "@/lib/api";
⋮----
const jitteredDelay = (attempt: number) =>
⋮----
const closeSocket = () =>
⋮----
const connect = async () =>
```

## File: apps/admin-portal/lib/ws-provider.tsx
```typescript
import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import { useAdminToken } from "@/lib/session";
import { api } from "@/lib/api";
⋮----
export type WSConnectionStatus = "connected" | "connecting" | "disconnected" | "reconnecting";
⋮----
export interface WSEventEnvelope {
  type: string;
  room?: string;
  event_id?: string;
  last_event_id?: string;
  seq?: number;
  timestamp: string;
  payload?: Record<string, unknown>;
}
⋮----
interface WSContextType {
  status: WSConnectionStatus;
  lastHeartbeat: Date | null;
  lastEventTime: Date | null;
  eventBuffer: WSEventEnvelope[];
  reconnect: () => void;
  subscribe: (handler: (event: WSEventEnvelope) => void) => () => void;
  sendMessage: (payload: Record<string, unknown>) => void;
  joinRoom: (room: string) => void;
  leaveRoom: (room: string) => void;
  ackEvent: (eventId: string) => void;
  refreshKey: number;
}
⋮----
export function AdminWebSocketProvider({ children }: { children: ReactNode })
⋮----
const onOnline = () =>
const onOffline = () =>
```

## File: apps/factory-portal/components/ui/index.ts
```typescript

```

## File: apps/factory-portal/components/FactoryShell.tsx
```typescript
import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { memo, useEffect, useMemo, useState, useCallback, useRef } from 'react';
import { PanelLeft, PanelLeftClose } from 'lucide-react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { apiFetch, decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import { SessionPackChip } from './SessionPackChip';
import ClientPolicyBanner from './ClientPolicyBanner';
import NotificationPanel from './NotificationPanel';
import LanguageSwitcher from './LanguageSwitcher';
import { useNotifications } from '@/lib/useNotifications';
import { usePortalT } from '@/lib/i18n';
import { ALL_NAV_ITEMS, NAV, navSectionIsActive } from '@/lib/nav';
import { motion, useReducedMotion } from 'framer-motion';
import { DesktopKeyboardCheatsheet } from './ui/DesktopKeyboardCheatsheet';
⋮----
function isActiveRoute(pathname: string, href: string): boolean
⋮----
aria-label=
⋮----
const wakeRefresh = () =>
⋮----
const handler = (e: KeyboardEvent) =>
```

## File: apps/factory-portal/components/FleetLiveMap.tsx
```typescript
import { GenericFleetLiveMap } from '@/components/ui/GenericFleetLiveMap';
import type { ComponentProps } from 'react';
⋮----
export default function FleetLiveMap(props: ComponentProps<typeof GenericFleetLiveMap>)
```

## File: apps/factory-portal/components/LocationPicker.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
  hasValidCoordinates,
  locationFromResolved,
  resolvePlace,
  reverseGeocode,
  type ResolvedLocation,
} from "@/lib/geocode";
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-core";
import MapGL, { Marker, NavigationControl } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
⋮----
export type LocationValue = {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
};
⋮----
type LocationPickerProps = {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  label?: string;
};
⋮----
const onInputChange = (text: string) =>
⋮----
const pickSuggestion = async (placeId: string, description: string) =>
⋮----
const commitTypedAddress = async () =>
⋮----
const useMyLocation = () =>
⋮----
const applyFallback = () =>
⋮----
const handleMapClick = async (e: maplibregl.MapMouseEvent) =>
⋮----
onChange=
⋮----
setTimeout(() =>
⋮----
longitude={pinned ? parseFloat(value.lng) : undefined}
          latitude=
⋮----
<Marker longitude=
```

## File: apps/factory-portal/lib/auth.ts
```typescript
import { useState, useEffect } from 'react';
import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession, reconnectDelayMs } from '@pegasusx/api-core';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { runFactorySessionReconcile } from './session-reconcile';
⋮----
/** GS-C5: pin to JWT home_cell. Login (no token) stays on bootstrap. */
export function factoryApiBaseUrl(): string
⋮----
type FactoryLiveEventType = (typeof FACTORY_LIVE_EVENT_TYPES)[number];
⋮----
export interface FactoryLiveEvent {
  type: string;
  [key: string]: unknown;
}
⋮----
export type FactorySocketStatus = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'offline';
⋮----
export function readTokenFromCookie(): string
⋮----
function readRefreshFromCookie(): string
⋮----
export function persistSession(token: string, refreshToken?: string)
⋮----
export function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
export function useToken(): string
⋮----
export async function refreshFactorySession(): Promise<{ ok: boolean; isConfigured?: boolean }>
⋮----
export async function getFactoryToken(): Promise<string>
⋮----
export function httpAuthorizationToken(sessionJwt: string, firebaseIdToken?: string): string
⋮----
async function tryRefreshToken(): Promise<string | null>
⋮----
export async function apiFetch(path: string, init?: RequestInit): Promise<Response>
⋮----
function toWSBaseUrl(baseUrl: string): string
⋮----
async function readFactorySocketToken(): Promise<string>
⋮----
function isFactoryEventType(value: string): boolean
⋮----
export function parseFactoryLiveEvent(rawPayload: string): FactoryLiveEvent | null
⋮----
export function subscribeFactoryWS(options: {
  onMessage: (payload: string) => void;
  onStatusChange?: (status: FactorySocketStatus) => void;
}): () => void
⋮----
const clearReconnect = () =>
⋮----
const openSocket = async (isReconnect: boolean) =>
⋮----
const handleOnline = () =>
⋮----
const handleOffline = () =>
```

## File: apps/factory-portal/lib/firebase.ts
```typescript
import {
  resetEmulatorPhoneOtpFlow,
  sendPhoneOtpViaEmulator,
  verifyPhoneOtpViaEmulator,
} from "@/components/ui/firebase-emulator-phone";
import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithPhoneNumber,
  type Auth,
  type ConfirmationResult,
} from "firebase/auth";
⋮----
function shouldUseAuthEmulator(): boolean
⋮----
function resolveAuthEmulatorHost(): string
⋮----
export async function firebaseSignOut(): Promise<void>
⋮----
class EmulatorRecaptchaVerifier
⋮----
async verify(): Promise<string>
⋮----
_reset(): void
⋮----
export async function sendPhoneOtp(phone: string): Promise<void>
⋮----
export async function verifyPhoneOtp(code: string): Promise<string>
⋮----
export function resetPhoneOtpFlow(): void
```

## File: apps/factory-portal/lib/geocode.ts
```typescript
import { factoryApiBaseUrl } from "@/lib/auth";
⋮----
export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
  formatted_address?: string;
};
⋮----
export type AutocompletePrediction = {
  place_id: string;
  description: string;
};
⋮----
export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]>
⋮----
export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null>
⋮----
export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null>
⋮----
export async function forwardGeocode(address: string): Promise<ResolvedLocation | null>
⋮----
export function hasValidCoordinates(lat: string, lng: string): boolean
⋮----
export function locationFromResolved(loc: ResolvedLocation, fallbackAddress = ""): {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
}
```

## File: apps/retailer-app-desktop/app/(dashboard)/auto-order/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  AlertTriangle,
  Play,
  RefreshCw,
  Info,
  Loader2,
  History,
  ShoppingCart,
  Check,
} from "lucide-react";
import { DemandSourceChips } from "@/components/ui/DemandSourceChips";
import { AutoOrderRules } from "@/components/auto-order/AutoOrderRules";
import { PageChrome } from "@/components/PageChrome";
import { useLiveData } from "@/lib/hooks";
import { apiFetch } from "@/lib/auth";
import { getRetailerId } from "@/lib/retailer-profile";
import type {
  AutoOrderExecutionMode,
  AutoOrderRun,
  AutoOrderSettings,
  AutoOrderShadowProposal,
} from "@/lib/types";
⋮----
type RetailerReorderSuggestion = {
  sku: string;
  suggested_qty: number;
  adjusted_demand_per_day?: number;
  current_stock?: number;
  safety_stock?: number;
  sources?: string[];
  sell_through_velocity?: number;
  status?: string;
};
⋮----
type SoakGate = {
  decision?: { allowed: boolean; reasons?: string[]; stats?: AutoOrderSettings["shadow_stats"] };
  thresholds?: { min_proposals?: number; max_wape?: number; min_unmodified?: number; gate_disabled?: boolean };
  place_flag_enabled?: boolean;
};
⋮----
const runAutoOrder = async (mode: "shadow" | "draft") =>
⋮----
const setExecutionMode = async (mode: AutoOrderExecutionMode) =>
⋮----
const toggleGlobal = async (enabled: boolean, useHistory?: boolean) =>
⋮----
const toggleScoped = async (
    level: "supplier" | "category" | "product" | "variant",
    id: string,
    enabled: boolean,
    useHistory?: boolean
  ) =>
⋮----
const handleToggle = (
    type: "global" | "supplier" | "category" | "product" | "variant",
    enabled: boolean,
    hasHistory: boolean,
    id?: string
  ) =>
⋮----
const confirmAction = (useHistory: boolean) =>
⋮----
onClick={() =>
⋮----
{po.line_count} line(s)
```

## File: apps/retailer-app-desktop/app/(dashboard)/my-suppliers/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useState } from "react";
import {
  Store,
  RefreshCw,
  Search,
  Plus,
  Trash2,
  AlertTriangle,
  Building2,
  CheckCircle2,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { retailerSupplierAddKey, retailerSupplierRemoveKey } from "@pegasusx/api-core";
import { apiFetch } from "@/lib/auth";
import type { Supplier } from "@/lib/types";
⋮----
const handleSearch = async (query: string) =>
⋮----
const addSupplier = async (supplierId: string) =>
⋮----
const removeSupplier = async (supplierId: string) =>
⋮----
<PageSection title=
⋮----
onClick=
```

## File: apps/retailer-app-desktop/app/(dashboard)/notifications/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { AlertTriangle, CheckCheck, RefreshCw, WifiOff } from "lucide-react";
import { HandoffCard } from "@/components/ui/HandoffCard";
import { useRouter } from "next/navigation";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "../../../components/EmptyState";
import { ListRowSkeleton } from "../../../components/Skeleton";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useRetailerNotifications } from "../../../lib/notifications";
import { useOptionalWebSocket } from "../../../lib/ws";
⋮----
const stopPropagation = (e: React.SyntheticEvent) =>
⋮----
onClick=
⋮----
aria-label=
```

## File: apps/retailer-app-desktop/components/orders/OrderList.tsx
```typescript
import { AnimatePresence } from "framer-motion";
import { PackageOpen, ArrowUpRight } from "lucide-react";
import { Chip } from "@heroui/react";
import { VirtualScrollList } from "@/components/ui/VirtualScrollList";
import EmptyState from "../EmptyState";
import { ListRowSkeleton } from "../Skeleton";
import type { Order } from "../../lib/types";
⋮----
interface OrderListProps {
  loading: boolean;
  filtered: Order[];
  listEmptyState: {
    headline: string;
    body: string;
    variant: "no-results" | "restricted" | "offline" | "error" | "no-orders";
    action: string;
    onAction: () => void;
  };
  selectedId: string | null;
  setSelectedId: (id: string) => void;
  chipCfg: Record<string, { color: "warning" | "success" | "default" | "danger"; label: string }>;
  list: Order[];
}
```

## File: apps/retailer-app-desktop/components/tracking/TrackingMap.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useMemo, useEffect, useRef } from "react";
import { Truck, X } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { Skeleton } from "../Skeleton";
import EmptyState from "../EmptyState";
import MapGL, { Layer, Marker, NavigationControl, Source } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
⋮----
import { PageSection } from "../PageSection";
import type { TrackingOrder } from "../../lib/types";
import { moneyCurrency } from "../../lib/payment-catalog";
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-core";
⋮----
function formatAmount(amount: number): string
⋮----
interface TrackingMapProps {
  loading: boolean;
  visibleOrders: TrackingOrder[];
  ordersLength: number;
  emptyStateTitle: string;
  emptyStateMessage: string;
  loadIssue: string | null;
  refreshAll: () => void;
  colorScheme: "light" | "dark";
  activeFulfillmentCount: number;
  selectedOrder: TrackingOrder | null;
  setSelectedOrder: (order: TrackingOrder | null) => void;
}
⋮----
title=
⋮----
initialViewState={mapInitialViewState(readCachedAuthSession()?.pack)}
          style=
⋮----
e.originalEvent.stopPropagation();
setSelectedOrder(order);
⋮----
onClick=
⋮----
{formatAmount(selectedOrder.total_amount)}
```

## File: apps/retailer-app-desktop/components/ui/index.ts
```typescript

```

## File: apps/retailer-app-desktop/components/LocationPicker.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
  resolvePlace,
  reverseGeocode,
  type ResolvedLocation,
} from "@/lib/geocode";
⋮----
export type LocationValue = {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
};
⋮----
type LocationPickerProps = {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  label?: string;
};
⋮----
const onInputChange = (text: string) =>
⋮----
const pickSuggestion = async (placeId: string, description: string) =>
⋮----
const useMyLocation = () =>
⋮----
const applyFallback = () =>
⋮----
onChange=
```

## File: apps/retailer-app-desktop/lib/cart.tsx
```typescript
import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { apiFetch, readToken } from './auth';
import { useOptionalWebSocket } from './ws';
import {
  clampCartQuantity,
  orderableCapsFromPreview,
  stockMetaFromProduct,
  type StockAwareProduct,
} from './stock-policy';
import { getRetailerId } from './retailer-profile';
import { packCurrency, readCachedAuthSession } from '@pegasusx/api-core';
import type { CheckoutPreviewResponse, Product } from './types';
⋮----
function cartCurrency(): string
⋮----
export interface CartItem {
  product_id: string;
  supplier_id: string;
  name: string;
  price: number;
  quantity: number;
  image_url?: string;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  available_stock?: number;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
}
⋮----
interface AddToCartProduct extends StockAwareProduct {
  id: string;
  supplier_id: string;
  name: string;
  price: number;
  image_url?: string;
}
⋮----
type CartContextType = {
  items: CartItem[];
  addToCart: (product: AddToCartProduct, quantity?: number) => void;
  removeFromCart: (product_id: string) => void;
  updateQuantity: (product_id: string, quantity: number) => void;
  clearCart: () => void;
  subtotal: number;
  discount: number;
  total: number;
  previewOrderableQuantities: Record<string, number>;
  previewShowStockCounts: boolean;
  previewStockPolicyReject: boolean;
  checkoutPolicyToken: string | null;
  applyPreviewOrderableCaps: (preview: CheckoutPreviewResponse) => void;
};
⋮----
interface ServerCartItem {
  cart_id?: string;
  sku_id: string;
  supplier_id: string;
  quantity: number;
  unit_price: number;
  currency: string;
}
⋮----
interface ServerCartResponse {
  items: ServerCartItem[];
  total: number;
}
⋮----
function buildCartSignature(items: CartItem[]): string
⋮----
function toServerCartItems(items: CartItem[]): ServerCartItem[]
⋮----
interface CheckoutQuoteResponse {
  supplier_id: string;
  subtotal_minor: number;
  discount_minor: number;
  total_minor: number;
  currency: string;
}
⋮----
async function fetchPromotionTotals(cartItems: CartItem[]): Promise<{
  subtotal: number;
  discount: number;
  total: number;
}>
⋮----
async function fetchCheckoutPreviewCaps(
  cartItems: CartItem[],
): Promise<CheckoutPreviewResponse | null>
⋮----
export function CartProvider({ children }: { children: React.ReactNode })
⋮----
const onOrgSwitched = () =>
⋮----
/* ignore */
⋮----
const addToCart = (product: AddToCartProduct | Product, quantity = 1) =>
⋮----
const removeFromCart = (product_id: string) =>
⋮----
const updateQuantity = (product_id: string, quantity: number) =>
⋮----
const clearCart = () =>
```

## File: apps/retailer-app-desktop/lib/geocode.ts
```typescript
export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
  formatted_address?: string;
};
⋮----
export type AutocompletePrediction = {
  place_id: string;
  description: string;
};
⋮----
export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]>
⋮----
export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null>
⋮----
export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null>
⋮----
export async function forwardGeocode(address: string): Promise<ResolvedLocation | null>
```

## File: apps/retailer-app-desktop/lib/ws.tsx
```typescript
import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import type { WsEvent } from '@pegasusx/types';
import { readToken } from './auth';
⋮----
function reconnectDelayMs(attempt: number, baseMs = 3_000, maxMs = 60_000): number
⋮----
export type WsEventType = WsEvent["type"];
⋮----
export type WsMessage = (Record<string, any> & { type?: string });
⋮----
type WsEventPayload<T extends string> = T extends WsEventType
  ? Extract<WsEvent, { type: T }>
  : WsMessage;
⋮----
type WsEventHandler<T extends string = string> = (msg: WsEventPayload<T>) => void;
⋮----
type WebSocketContextType = {
  isConnected: boolean;
  reconnectEpoch: number;
  lastMessage: WsMessage | null;
  sendMessage: (msg: WsMessage) => void;
  joinRoom: (room: string) => void;
  leaveRoom: (room: string) => void;
  ackEvent: (eventId: string) => void;
  subscribe: <T extends string>(type: T, handler: WsEventHandler<T>) => () => void;
};
⋮----
export function WebSocketProvider({ children }: { children: React.ReactNode })
⋮----
const clearHeartbeat = () =>
⋮----
function connect()
⋮----
const onOnline = () =>
⋮----
const onOffline = () =>
⋮----
export function useWebSocket()
⋮----
export function useOptionalWebSocket()
⋮----
export function useWsEvent<T extends string>(type: T, handler: WsEventHandler<T>)
```

## File: apps/supplier-portal/app/(dashboard)/chargebacks/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { supplierFetch } from "@/lib/auth";
import { RefreshCw, AlertTriangle, CheckCircle2 } from "lucide-react";
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";
⋮----
// Reversal state
⋮----
const submitChargeback = async (e: React.FormEvent) =>
⋮----
const submitReversal = async (e: React.FormEvent) =>
⋮----
<PageSection title=
```

## File: apps/supplier-portal/app/(dashboard)/settings/tax-regimes/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { supplierFetch } from "@/lib/auth";
import { sessionPackCurrency } from "@pegasusx/api-core";
import {
  RefreshCw,
  Plus,
  Calendar,
  Percent,
  CheckCircle,
  FileJson
} from "lucide-react";
⋮----
type TaxRegimeVersion = {
  id: string;
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rate_bps: number;
  simplified: boolean;
  rules_json?: Record<string, unknown>;
  created_at: string;
};
⋮----
type TaxRegimeResponse = {
  regimes: TaxRegimeVersion[];
};
⋮----
export default function TaxRegimesPage()
⋮----
const handleCreate = async (e: React.FormEvent) =>
⋮----
const formatDate = (isoStr: string | undefined) =>
⋮----
title=
⋮----
onClick=
⋮----
<PageSection title=
⋮----
onChange=
```

## File: apps/supplier-portal/app/(portal)/analytics/demand/signals/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { DemandSignal, CreateSignalRequest } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
⋮----
const load = () =>
⋮----
const addSignal = async (e: React.FormEvent) =>
⋮----
description=
⋮----
emptyMessage=
```

## File: apps/supplier-portal/app/(portal)/factories/components/FactoryForm.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { AUTH_COUNTRIES } from "@/components/ui";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";
import { sessionMapCenter } from "@pegasusx/api-core";
⋮----
interface FactoryFormProps {
  onSave: (name: string, location: LocationValue, extras: { country_code: string }) => Promise<void>;
  onCancel: () => void;
}
⋮----
function emptyFactoryLocation(): LocationValue
⋮----
const handleSave = async () =>
```

## File: apps/supplier-portal/app/(portal)/finance/aml-exposure/page.tsx
```typescript
import React, { useState } from 'react';
import { PageChrome } from '@/components/PageChrome';
import { TIYIN_PER_UZS, STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse } from '@pegasusx/types';
import { foundrySupplierApi } from '@/lib/foundry-api';
⋮----
export default function SupplierAMLExposurePage()
⋮----
const handleCheck = async (e: React.FormEvent) =>
⋮----
onChange={(e) => setTin(e.target.value)}
⋮----
Statutory Ceiling:
⋮----
Cumulative Cash Today:
```

## File: apps/supplier-portal/app/(portal)/foundry/page.tsx
```typescript
import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { PageChrome } from '@/components/PageChrome';
import { LotRecallBlastRadiusView } from '@/components/foundry/LotRecallBlastRadiusView';
import { foundrySupplierApi } from '@/lib/foundry-api';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import {
  Layers,
  AlertTriangle,
  GitBranch,
  Sparkles,
  CheckCircle2,
  XCircle,
  RotateCw,
  Search,
  Trash2,
  Eye,
  FileCode,
  ArrowRight,
  ShieldCheck,
} from 'lucide-react';
⋮----
type FoundryTab = 'recall' | 'sandbox' | 'searcharound';
⋮----
const loadTemplate = (name: 'promo' | 'pack_snap' | 'discontinue') =>
⋮----
const handleApplyPatch = async (e?: React.FormEvent) =>
⋮----
const handleResolvePreview = async () =>
⋮----
const handleCommitScenario = async () =>
⋮----
const handleDiscardPatch = (indexToRemove: number) =>
⋮----
const handleRebaseConflict = (conflictObjectId: string) =>
⋮----
const handleExecuteSearchAround = async () =>
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
⋮----
onChange=
⋮----
onClick=
```

## File: apps/supplier-portal/app/(portal)/replenishment/suggestions/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { ReorderSuggestionRow } from "@pegasusx/types";
import { DemandSourceChips } from "@/components/ui";
import { PageChrome } from "@/components/PageChrome";
⋮----
const rowKey = (row: ReorderSuggestionRow) =>
⋮----
const toggleAll = () =>
⋮----
const toggleRow = (key: string) =>
⋮----
const dismiss = async (row: ReorderSuggestionRow) =>
⋮----
const createDraft = async (row: ReorderSuggestionRow) =>
⋮----
const bulkCreateDrafts = async () =>
⋮----
description=
```

## File: apps/supplier-portal/app/auth/login/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { AuthLoginCard, dialCodeForCountry } from "@/components/ui";
import { createSupplierApi } from "@/lib/api";
import { persistSession } from "@/lib/auth";
import { discoverOIDC } from "@/lib/oidc";
⋮----
type LoginStep = "phone" | "otp";
⋮----
export default function SupplierLoginPage()
⋮----
async function handleSendOtp(e: React.FormEvent)
⋮----
async function handleVerifyOtp(e: React.FormEvent)
⋮----
async function handleIdP(e: React.FormEvent)
⋮----
title=
⋮----
registerLabel=
⋮----
onBack=
```

## File: apps/supplier-portal/app/demand/payday-calendar/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState, useEffect } from "react";
⋮----
interface DemandSignal {
  signalId: string;
  type: string;
  scope: string;
  startAt: string;
  endAt: string;
  multiplier: number;
  meta?: {
    title?: string;
    description?: string;
  };
}
⋮----
multiplier: 1.5, // Default for payday
⋮----
const fetchPaydays = async () =>
⋮----
const handleSave = async () =>
⋮----
const handleDeactivate = async (id: string) =>
⋮----
onClick=
⋮----
<button onClick=
```

## File: apps/supplier-portal/app/demand/signals/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import React, { useState, useEffect } from "react";
import { Info } from "lucide-react";
⋮----
type SignalType = "PROMO" | "EVENT" | "PAYDAY" | "EVENT_DENSITY" | "COMPETITOR_PRESSURE";
⋮----
interface DemandSignal {
  signalId: string;
  type: SignalType;
  scope: string;
  sku?: string | null;
  startAt: string;
  endAt: string;
  multiplier: number;
  meta?: {
    title?: string;
    description?: string;
    campaignId?: string;
  };
}
⋮----
// Modal state
⋮----
// Form state
⋮----
const fetchSignals = async () =>
⋮----
const handleOpenCreate = () =>
⋮----
const handleOpenEdit = (sig: DemandSignal) =>
⋮----
startAt: sig.startAt.slice(0, 16), // datetime-local format
⋮----
const handleSave = async () =>
⋮----
const handleDeactivate = async (id: string) =>
⋮----
onChange=
⋮----
<button onClick=
⋮----
onClick=
```

## File: apps/supplier-portal/components/foundry/LotRecallBlastRadiusView.tsx
```typescript
import React, { useState } from 'react';
import type { RecallBlastRadiusReport } from '@pegasusx/types';
import { foundrySupplierApi } from '@/lib/foundry-api';
import {
  AlertTriangle,
  RotateCw,
  Package,
  Factory,
  Truck,
  FileText,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  HelpCircle,
  Radio,
} from 'lucide-react';
⋮----
interface LotRecallBlastRadiusViewProps {
  supplierId?: string;
}
⋮----
export function LotRecallBlastRadiusView({
  supplierId = 'sup_samarkand_dairy',
}: LotRecallBlastRadiusViewProps)
⋮----
const handleTriggerRecall = async (e?: React.FormEvent) =>
⋮----
onChange={(e) =>
```

## File: apps/supplier-portal/components/ui/index.ts
```typescript

```

## File: apps/supplier-portal/components/BulkImportWizard.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useRef } from 'react';
⋮----
export interface BulkImportProductRow {
  name: string;
  price: string;
  barcode: string;
  category: string;
  vu: string;
}
⋮----
const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) =>
```

## File: apps/supplier-portal/components/DigitalBrainPanel.tsx
```typescript
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ApiError } from "@pegasusx/api-core";
import type {
  ForecastConfidence,
  SparsityGateResult,
  SupplierKnowledgeGraph,
  TwinOpsRouteView,
} from "@pegasusx/types";
import { brainForecastLine, forecastConfidenceFromDemand, isForecastBlocked } from "@pegasusx/types";
import { SourceChip } from "@/components/ui";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import SignalIngestOpsPanel from "@/components/SignalIngestOpsPanel";
import { ForecastAccuracyPanel } from "@/components/settings/planning";
import { createSupplierApi } from "@/lib/api";
import { formatForecastUpdatedAt, isForecastStale } from "@/lib/forecast-confidence";
⋮----
type TwinPlane = "last_mile" | "factory";
⋮----
const invokeAgent = async (action: "approve_insight" | "open_supply_request" | "broadcast_template") =>
⋮----
updatedAt={formatForecastUpdatedAt(generatedAt)}
            stale=
⋮----
onChange=
```

## File: apps/supplier-portal/components/DispatchPreviewMap.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef } from 'react';
import type { DispatchProposedRoute, RouteGeometryWire } from '@pegasusx/types';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
⋮----
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-core';
⋮----
type DispatchPreviewMapProps = {
  routes: Array<DispatchProposedRoute | { route_geometry?: RouteGeometryWire; driver_name?: string; driver_id?: string }>;
  className?: string;
};
⋮----
type RouteFeatureCollection = GeoJSON.FeatureCollection<GeoJSON.LineString>;
⋮----
function toLineFeature(
  route: DispatchPreviewMapProps['routes'][number],
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null
```

## File: apps/supplier-portal/components/FleetLiveMap.tsx
```typescript
import React from 'react';
import { GenericFleetLiveMap } from '@/components/ui';
⋮----
export default function FleetLiveMap(props: React.ComponentProps<typeof GenericFleetLiveMap>)
```

## File: apps/supplier-portal/components/LocationPicker.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
  resolvePlace,
  reverseGeocode,
  type ResolvedLocation,
} from "@/lib/geocode";
⋮----
export type LocationValue = {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
};
⋮----
type LocationPickerProps = {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  label?: string;
};
⋮----
const onInputChange = (text: string) =>
⋮----
const pickSuggestion = async (placeId: string, description: string) =>
⋮----
const useMyLocation = () =>
⋮----
const applyFallback = () =>
⋮----
onChange=
```

## File: apps/supplier-portal/components/SupplierShell.tsx
```typescript
import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useRef, useMemo, useCallback, memo } from 'react';
import { Button } from '@heroui/react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import NotificationPanel from './NotificationPanel';
import ClientPolicyBanner from './ClientPolicyBanner';
import LanguageSwitcher from './LanguageSwitcher';
import { useNotifications } from '@/lib/useNotifications';
import { clearSession, readTokenFromCookie } from '@/lib/auth';
import { SessionPackChip } from './SessionPackChip';
import { usePortalT } from '@/lib/i18n';
import { ALL_NAV_ITEMS, NAV, navSectionIsActive } from '@/lib/nav';
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import { DesktopKeyboardCheatsheet } from './ui/DesktopKeyboardCheatsheet';
⋮----
function isActiveRoute(pathname: string, href: string): boolean
⋮----
function buildBreadcrumbs(
  pathname: string,
  t: (key: string) => string,
): { label: string; href: string }[]
⋮----
const handler = (e: MouseEvent | TouchEvent) =>
⋮----
aria-label=
⋮----
onClick=
```

## File: apps/supplier-portal/components/TopologyEditor.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import type {
  OutOfStockPolicy,
  SupplierTopologyCoverageCity,
  SupplierTopologyFactory,
  SupplierTopologyInventorySeed,
  SupplierTopologyResponse,
  SupplierTopologyUpdateRequest,
  SupplierTopologyWarehouse,
} from "@pegasusx/types";
import { AUTH_COUNTRIES } from "@/components/ui";
import { sessionMapCenter } from "@pegasusx/api-core";
import { CoverageCityChips } from "@/components/CoverageCityChips";
import { createSupplierApi } from "@/lib/api";
import { LocationPicker } from "@/components/LocationPicker";
import { useToast } from "@/components/Toast";
⋮----
type InventorySeedDraft = {
  key: string;
  product_id: string;
  quantity: string;
};
⋮----
type WarehouseDraft = {
  key: string;
  warehouse_id?: string;
  name: string;
  address: string;
  place_id?: string;
  lat: string;
  lng: string;
  coverage_radius_km: string;
  is_active: boolean;
  is_on_shift: boolean;
  transfer_mode: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id: string;
  primary_factory_id: string;
  secondary_factory_id: string;
  assigned_factory_ids: string[];
  country_code: string;
  coverage_cities: SupplierTopologyCoverageCity[];
  default_out_of_stock_policy: OutOfStockPolicy;
  initial_inventory: InventorySeedDraft[];
};
⋮----
type FactoryDraft = {
  key: string;
  factory_id?: string;
  name: string;
  address: string;
  place_id?: string;
  lat: string;
  lng: string;
  country_code: string;
  is_active: boolean;
};
⋮----
function warehouseDraftFromNode(node: SupplierTopologyWarehouse, key: string): WarehouseDraft
⋮----
function inventorySeedsFromDrafts(rows: InventorySeedDraft[]): SupplierTopologyInventorySeed[]
⋮----
function factoryDraftFromNode(node: SupplierTopologyFactory, key: string): FactoryDraft
⋮----
function draftsFromTopology(topology: SupplierTopologyResponse): {
  warehouses: WarehouseDraft[];
  factories: FactoryDraft[];
}
⋮----
function parseCoordinate(value: string, field: string): number
⋮----
function buildUpdateRequest(warehouses: WarehouseDraft[], factories: FactoryDraft[]): SupplierTopologyUpdateRequest
⋮----
type TopologyEditorProps = {
  initial: SupplierTopologyResponse;
  onSaved: (topology: SupplierTopologyResponse) => void;
};
⋮----
const reset = () =>
⋮----
const save = async () =>
⋮----
<h2 className="md-typescale-title-medium">
⋮----
label=
⋮----
setWarehouses((prev)
⋮----
setFactories((prev)
⋮----
⋮----
<input id="topologyeditor-checkbox-1" aria-label="Select Topologyeditor option" type="checkbox" checked=
```

## File: apps/supplier-portal/lib/auth.ts
```typescript
import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession } from '@pegasusx/api-core';
import { clearStoredToken, getStoredToken, isTauri, storeToken } from "@/lib/bridge";
⋮----
function tauriBackendBaseUrl(): string
⋮----
// 10.0.2.2 is the Android emulator host alias; desktop/macOS uses localhost.
⋮----
function usesApiProxy(base: string): boolean
⋮----
function supplierBootstrapUrl(): string
⋮----
export function supplierApiBaseUrl(): string
⋮----
export function readTokenFromCookie(): string
⋮----
function readRefreshFromCookie(): string
⋮----
export function persistSession(token: string, refreshToken?: string)
⋮----
// ignore quota errors on constrained devices
⋮----
export function clearSession()
⋮----
// ignore
⋮----
export function getSupplierToken(): string
⋮----
export async function resolveSupplierToken(): Promise<string>
⋮----
export function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
export function readIsConfigured(token: string): boolean
⋮----
async function tryRefreshToken(): Promise<string | null>
⋮----
export async function supplierFetch(path: string, init?: RequestInit): Promise<Response>
⋮----
function toApiPath(input: RequestInfo | URL): string
⋮----
export const supplierSessionFetch: typeof fetch = (input, init) =>
```

## File: apps/supplier-portal/lib/firebase.ts
```typescript
import {
  resetEmulatorPhoneOtpFlow,
  sendPhoneOtpViaEmulator,
  verifyPhoneOtpViaEmulator,
} from "@/components/ui";
import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithPhoneNumber,
  type Auth,
  type ConfirmationResult,
} from "firebase/auth";
⋮----
function shouldUseAuthEmulator(): boolean
⋮----
function resolveAuthEmulatorHost(): string
⋮----
export async function firebaseSignOut(): Promise<void>
⋮----
class EmulatorRecaptchaVerifier
⋮----
async verify(): Promise<string>
⋮----
_reset(): void
⋮----
export async function sendPhoneOtp(phone: string): Promise<void>
⋮----
export async function verifyPhoneOtp(code: string): Promise<string>
⋮----
export function resetPhoneOtpFlow(): void
```

## File: apps/supplier-portal/lib/geocode.ts
```typescript
export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
  formatted_address?: string;
};
⋮----
export type AutocompletePrediction = {
  place_id: string;
  description: string;
};
⋮----
export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]>
⋮----
export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null>
⋮----
export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null>
⋮----
export async function forwardGeocode(address: string): Promise<ResolvedLocation | null>
```

## File: apps/supplier-portal/lib/hooks.ts
```typescript
import { useCallback } from "react";
import { supplierFetch } from "./auth";
import { useEnterpriseSWR } from "./enterprise-swr";
⋮----
export function useLiveData<T>(path: string, interval = 0)
```

## File: apps/warehouse-portal/app/fleet/page.tsx
```typescript
import React, { useState } from 'react';
import { FleetDVIRScreen } from '@/components/fleet/FleetDVIRScreen';
import { YardGateCheckScreen } from '@/components/fleet/YardGateCheckScreen';
import { MidRouteRescueModal } from '@/components/fleet/MidRouteRescueModal';
import { Truck, ClipboardCheck, ArrowRightLeft, ShieldAlert } from 'lucide-react';
⋮----
export default function FleetPage()
```

## File: apps/warehouse-portal/app/orders/[id]/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import type { WarehouseOrderDetail } from '@pegasusx/types';
import { adminForceCompleteKey, ApiError } from '@pegasusx/api-core';
import Icon from '@/components/Icon';
import { OrderTimelinePanel } from '@/components/OrderTimelinePanel';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { OrderLineItems } from '@/components/orders/OrderLineItems';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { orderActionFlags } from '@/lib/order-actions';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';
⋮----
function isoDeliveryDate(dateInput: string): string
⋮----
const fmt = (n: number) =>
⋮----
async function runMutation(
    label: string,
    fn: () => Promise<unknown>,
    requiresReason = false,
  )
⋮----
description=
⋮----
runMutation(
```

## File: apps/warehouse-portal/components/cashier/BlindCountDenominationGrid.tsx
```typescript
import React, { useState, useMemo, useEffect } from 'react';
import { UZS_BANKNOTE_DENOMINATIONS, TIYIN_PER_UZS, type UZSBanknoteDenomination } from '@pegasusx/types';
import { Lock, RotateCcw, CheckCircle2, AlertCircle, ShieldAlert } from 'lucide-react';
⋮----
interface BlindCountDenominationGridProps {
  disabled?: boolean;
  attemptNumber: number;
  onSubmitCount: (denominations: Record<number, number>, totalMinor: number, toleranceMinor: number) => Promise<void>;
}
⋮----
const handleCountChange = (denom: UZSBanknoteDenomination, valStr: string) =>
⋮----
const handleIncrement = (denom: UZSBanknoteDenomination, delta: number) =>
⋮----
const handleSubmit = async (e?: React.FormEvent) =>
⋮----
const handleKeyDown = (e: KeyboardEvent) =>
⋮----
const handleReset = () =>
⋮----
{/* Denomination Cards Grid */}
⋮----
onChange=
⋮----
{/* Cashier strap & bundle quick increments */}
⋮----
{/* Footer controls: Tolerance threshold + Total cash tally + Submit action */}
```

## File: apps/warehouse-portal/components/cashier/CashierSessionView.tsx
```typescript
import React from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { TIYIN_PER_UZS } from '@pegasusx/types';
import { Lock, ShieldAlert, CheckCircle2, AlertTriangle, ShieldCheck, RefreshCw, FileText } from 'lucide-react';
⋮----
interface CashierSessionViewProps {
  session: CashierSessionViewType;
  onOpenReclassifyModal?: () => void;
  onRequestSupervisorOverride?: () => void;
}
```

## File: apps/warehouse-portal/components/cashier/CITHandoverManifest.tsx
```typescript
import React, { useState } from 'react';
import { TIYIN_PER_UZS, type DepotVaultLedgerEntry } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { Shield, CheckCircle2, Barcode, Printer, AlertCircle } from 'lucide-react';
⋮----
interface CITHandoverManifestProps {
  depotId?: string;
  supplierId?: string;
  onHandoverComplete?: (entry: DepotVaultLedgerEntry) => void;
}
⋮----
const handleSubmit = async (e: React.FormEvent) =>
⋮----
onChange=
```

## File: apps/warehouse-portal/components/cashier/RRNReclassificationModal.tsx
```typescript
import React, { useState, useMemo } from 'react';
import { TIYIN_PER_UZS, type CashierSessionView } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { X, ShieldAlert, CheckCircle2, Clock, CreditCard, AlertCircle } from 'lucide-react';
⋮----
interface RRNReclassificationModalProps {
  sessionId: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (updatedSession: CashierSessionView) => void;
}
⋮----
const handleSubmit = async (e: React.FormEvent) =>
```

## File: apps/warehouse-portal/components/dispatch/DispatchOrderList.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useRouter } from 'next/navigation';
import type { WarehouseDispatchOrder } from '@pegasusx/types';
import { VirtualScrollList } from '@/components/ui';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';
import { moneyCurrency } from '@pegasusx/api-core';
⋮----
function fmt(n: number)
⋮----
function formatVU(vu: number)
⋮----
interface DispatchOrderListProps {
  orders: WarehouseDispatchOrder[];
  selectedOrderIds: Set<string>;
  allSelected: boolean;
  opsActingId: string | null;
  toggleOrder: (orderId: string) => void;
  toggleSelectAll: () => void;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string) => void;
  onReject: (orderId: string) => void;
}
⋮----
onChange=
⋮----
onProposeDate={() =>
onReject={() =>
```

## File: apps/warehouse-portal/components/fleet/FleetDVIRScreen.tsx
```typescript
import React, { useState } from 'react';
import type { FleetDVIRInspection, DVIRChecklist8Point } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { ClipboardCheck, Truck, Wrench, AlertOctagon, CheckCircle2, ShieldAlert, AlertTriangle } from 'lucide-react';
⋮----
interface FleetDVIRScreenProps {
  onInspectionComplete?: (inspection: FleetDVIRInspection) => void;
}
⋮----
const toggleCheck = (key: keyof DVIRChecklist8Point) =>
⋮----
const setAllPass = () =>
⋮----
const setSimulateDefect = () =>
⋮----
const handleSubmit = async (e: React.FormEvent) =>
```

## File: apps/warehouse-portal/components/fleet/MidRouteRescueModal.tsx
```typescript
import React, { useState } from 'react';
import { TIYIN_PER_UZS, type RescueToteItem, type MidRouteRescueResult } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { X, ShieldAlert, AlertTriangle, Snowflake, CheckCircle2, ArrowRightLeft, AlertCircle } from 'lucide-react';
⋮----
interface MidRouteRescueModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (result: MidRouteRescueResult) => void;
}
⋮----
const handleSubmit = async (e: React.FormEvent) =>
```

## File: apps/warehouse-portal/components/fleet/YardGateCheckScreen.tsx
```typescript
import React, { useState } from 'react';
import type { YardGateCheckEvent, YardBaySlot } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { Truck, Shield, Barcode, CheckCircle2, AlertOctagon, ArrowDownRight, ArrowUpRight, AlertCircle } from 'lucide-react';
⋮----
interface YardGateCheckScreenProps {
  depotId?: string;
  supplierId?: string;
}
⋮----
export function YardGateCheckScreen({
  depotId = 'depot_tashkent_central',
  supplierId = 'sup_samarkand_dairy',
}: YardGateCheckScreenProps)
⋮----
const handleGateEvent = async (e: React.FormEvent) =>
⋮----
const getSlotBadge = (slot?: YardBaySlot | string) =>
⋮----
onChange={(e) =>
```

## File: apps/warehouse-portal/components/ui/index.ts
```typescript

```

## File: apps/warehouse-portal/components/FleetLiveMap.tsx
```typescript
import { GenericFleetLiveMap } from '@/components/ui';
import type { ComponentProps } from 'react';
⋮----
export default function FleetLiveMap(props: ComponentProps<typeof GenericFleetLiveMap>)
```

## File: apps/warehouse-portal/components/LocationPicker.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
  hasValidCoordinates,
  locationFromResolved,
  resolvePlace,
  reverseGeocode,
  type ResolvedLocation,
} from "@/lib/geocode";
⋮----
export type LocationValue = {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
};
⋮----
type LocationPickerProps = {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  label?: string;
};
⋮----
const onInputChange = (text: string) =>
⋮----
const pickSuggestion = async (placeId: string, description: string) =>
⋮----
const commitTypedAddress = async () =>
⋮----
const useMyLocation = () =>
⋮----
const applyFallback = () =>
⋮----
onChange=
⋮----
setTimeout(() =>
```

## File: apps/warehouse-portal/components/NotificationPanel.tsx
```typescript
import { useRef, useEffect } from 'react';
import { Notification } from '@/lib/useNotifications';
import { motion, AnimatePresence } from 'framer-motion';
import { HandoffCard } from '@/components/ui';
import Icon from './Icon';
⋮----
interface NotificationPanelProps {
  open: boolean;
  onClose: () => void;
  items: Notification[];
  unreadCount: number;
  onMarkRead: (id: string) => void;
  onMarkAllRead: () => void;
}
⋮----
function timeAgo(iso: string): string
⋮----
const stopPropagation = (e: React.SyntheticEvent) =>
⋮----
const handler = (e: MouseEvent) =>
```

## File: apps/warehouse-portal/components/WarehouseShell.tsx
```typescript
import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState, useRef, useMemo, useCallback, memo } from 'react';
import Icon from './Icon';
import { useTheme, type ThemeMode } from './ThemeProvider';
import { PanelLeftClose, PanelLeft } from 'lucide-react';
import NotificationPanel from './NotificationPanel';
import ClientPolicyBanner from './ClientPolicyBanner';
import LanguageSwitcher from './LanguageSwitcher';
import { useNotifications, type WarehouseWsState } from '@/lib/useNotifications';
import { clearSession, decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';
import { SessionPackChip } from './SessionPackChip';
import { usePortalT } from '@/lib/i18n';
import { ALL_NAV_ITEMS, NAV, navSectionIsActive } from '@/lib/nav';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { DesktopKeyboardCheatsheet } from './ui/DesktopKeyboardCheatsheet';
⋮----
function isActiveRoute(pathname: string, href: string): boolean
⋮----
function buildBreadcrumbs(
  pathname: string,
  t: (key: string) => string,
): { label: string; href: string }[]
⋮----
function liveIndicatorCopy(state: WarehouseWsState): { label: string; stale: boolean }
⋮----
function ThemeToggle()
⋮----
aria-label=
⋮----
const handler = (e: MouseEvent | TouchEvent) =>
⋮----
onClick=
```

## File: apps/warehouse-portal/lib/firebase.ts
```typescript
import {
  resetEmulatorPhoneOtpFlow,
  sendPhoneOtpViaEmulator,
  verifyPhoneOtpViaEmulator,
} from "@/components/ui";
import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithPhoneNumber,
  type Auth,
  type ConfirmationResult,
} from "firebase/auth";
⋮----
function shouldUseAuthEmulator(): boolean
⋮----
function resolveAuthEmulatorHost(): string
⋮----
export async function firebaseSignOut(): Promise<void>
⋮----
class EmulatorRecaptchaVerifier
⋮----
async verify(): Promise<string>
⋮----
_reset(): void
⋮----
export async function sendPhoneOtp(phone: string): Promise<void>
⋮----
export async function verifyPhoneOtp(code: string): Promise<string>
⋮----
export function resetPhoneOtpFlow(): void
```

## File: apps/warehouse-portal/lib/foundry-api.ts
```typescript
import { warehouseApi } from "./warehouse-api";
import { ApiError } from "@pegasusx/api-core";
import type {
  OntologyObject,
  OntologyGraphEdge,
  SearchAroundRequest,
  SearchAroundResponse,
  LotRecallRequest,
  RecallBlastRadiusReport,
  ActionExecutionRequest,
  ActionExecutionRecord,
  ScenarioPatch,
  ScenarioPatchRequest,
  ScenarioCommitResult,
  StartCashierSessionRequest,
  CashierSessionView,
  BlindCountRequest,
  RRNTenderReclassificationRequest,
  CITHandoverRequest,
  DepotVaultLedgerEntry,
  FleetDVIRInspection,
  YardGateCheckEvent,
  DispatchCheckResponse,
  MidRouteRescueRequest,
  MidRouteRescueResult,
  SplitTenderRejectionInput,
  SplitTenderReconciliationResult,
  AMLCheckRequest,
  AMLCheckResponse,
  OfflineLeaseIssueRequest,
  OfflineQuotaLease,
  OfflineLeaseValidateRequest,
  OfflineLeaseValidateResponse,
  FiscalPreflightRequest,
  FiscalPreflightResponse,
  SlicePayoutEligibilityRequest,
  SlicePayoutEligibilityResponse,
  ClawbackSliceRequest,
  SettlementSliceRecord,
  Load3DRTIRequest,
  ReturnableContainerBalance,
  FXCorridorRequest,
  FXCorridorResponse,
  PickToteState,
  ToteTakeoverRequest,
} from "@pegasusx/types";
```

## File: apps/warehouse-portal/lib/geocode.ts
```typescript
export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
  formatted_address?: string;
};
⋮----
export type AutocompletePrediction = {
  place_id: string;
  description: string;
};
⋮----
export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]>
⋮----
export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null>
⋮----
export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null>
⋮----
export async function forwardGeocode(address: string): Promise<ResolvedLocation | null>
⋮----
export function hasValidCoordinates(lat: string, lng: string): boolean
⋮----
export function locationFromResolved(loc: ResolvedLocation, fallbackAddress = ""): {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
}
```

## File: apps/warehouse-portal/lib/nav.ts
```typescript
export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };
⋮----
export function primaryNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function allNavHrefs(nav: NavSection[] = NAV): string[]
⋮----
export function navSectionIsActive(
  section: NavSection,
  pathname: string,
  isActive: (pathname: string, href: string) => boolean,
): boolean
```

## File: apps/admin-portal/app/finops/page.tsx
```typescript
import React, { useEffect, useState, useMemo } from "react";
import { api, type BillingInvoice, type BillingFeeSchedule } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
⋮----
type PerspectiveMode = "OVERALL" | "SUPPLIER" | "WAREHOUSE" | "RETAILER";
⋮----
const fetchBillingData = async () =>
⋮----
const handleRunMonthly = async () =>
```

## File: apps/admin-portal/app/page.tsx
```typescript
import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
⋮----
function RootRedirector()
⋮----
export default function HomePage()
```

## File: apps/factory-portal/app/loading-bay/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ExplainStatusBanner, explainFromApiError } from '@/components/ui/ExplainStatusBanner';
import type { StatusExplain } from '@pegasusx/types';
import { factoryBatchDispatchKey } from '@pegasusx/api-core';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import HandoffTimelinePanel from '@/components/HandoffTimelinePanel';
import LoadingBayControls from '@/components/loading-bay/LoadingBayControls';
import LoadingBayGrid from '@/components/loading-bay/LoadingBayGrid';
import LotRecallBanner from '@/components/LotRecallBanner';
import FactoryYardGateSection from '@/components/loading-bay/FactoryYardGateSection';
type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';
⋮----
interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}
⋮----
async function handleDispatch()
⋮----
description=
⋮----
onRefresh=
⋮----
onQuarantineChange=
⋮----
onChange=
⋮----
<KpiStatCard label=
⋮----
label=
⋮----
title=
```

## File: apps/retailer-app-desktop/app/(dashboard)/dock/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { Chip } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import {
  Truck,
  Package,
  QrCode,
  Clock,
  MapPin,
  ChevronDown,
  ChevronRight,
  Inbox,
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { QRCodeSVG } from "qrcode.react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useWsEvent, useOptionalWebSocket, type WsMessage } from "../../../lib/ws";
import {
  applyDockPendingPatches,
  clearDockPendingPatches,
  queueDockOrderPatch,
} from "../../../lib/dock-pending-patches";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import type { TrackingResponse, TrackingOrder } from "../../../lib/types";
import { moneyCurrency } from "../../../lib/payment-catalog";
⋮----
function getBrowserStorage(): Storage | null
⋮----
function formatAmount(amount: number): string
⋮----
function buildDeliveryQrPayload(order: TrackingOrder): string
⋮----
interface SupplierGroup {
  supplierId: string;
  supplierName: string;
  orders: TrackingOrder[];
  totalAmount: number;
  hasApproaching: boolean;
  hasArrived: boolean;
}
⋮----
const toggleSupplier = (id: string) =>
⋮----
const toggleToken = (orderId: string) =>
⋮----
value=
⋮----
onClick=
⋮----
{formatAmount(order.total_amount)}
```

## File: apps/retailer-app-desktop/app/(dashboard)/orders/page.tsx
```typescript
import { useState, useMemo, useCallback, useEffect, Suspense } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { useWsEvent } from "../../../lib/ws";
import {
  Copy,
  Truck,
  CheckCircle2,
  PackageOpen,
  MoreVertical,
  Clock,
  Layers3,
  ArrowUpRight,
  RefreshCw,
  AlertTriangle,
  WifiOff,
  XCircle,
  Loader2,
} from "lucide-react";
import { Chip } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import { OrderFilters } from "../../../components/orders/OrderFilters";
import { OrderList } from "../../../components/orders/OrderList";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import EmptyState from "../../../components/EmptyState";
import { PageSection } from "../../../components/PageSection";
import { VirtualScrollList } from "@/components/ui/VirtualScrollList";
import { ListRowSkeleton } from "../../../components/Skeleton";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { sessionPackCurrency } from "../../../lib/payment-catalog";
import { OrderTimelinePanel } from "../../../components/OrderTimelinePanel";
import { FileClaimPanel } from "../../../components/FileClaimPanel";
import {
  confirmAiOrder,
  rejectAiOrder,
  confirmPreorder,
  editPreorder,
  acceptDeliveryProposal,
  rejectDeliveryProposal,
  getClaimEligibility,
  type ClaimEligibility,
} from "../../../lib/api";
import {
  retailerCancelKey,
  retailerRequestCancelKey,
} from "@pegasusx/api-core";
import { toOrderCustomerStage } from "@pegasusx/types";
import { useOptionalWebSocket } from "../../../lib/ws";
import { getRetailerProfile } from "@/lib/retailer-profile";
import type { Order, TrackingResponse } from "../../../lib/types";
import { usePortalT } from "@/lib/i18n";
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
title=
⋮----
aria-label=
⋮----
{aiActionPending ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      "Confirm Suggestion"
                    )}
                  </button>
                </div>
              )}
```

## File: apps/retailer-app-desktop/app/(dashboard)/settings/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useCallback, useEffect, useMemo } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useRouter } from "next/navigation";
import {
  User,
  Mail,
  MapPin,
  Globe,
  Bell,
  CreditCard,
  Settings,
  Brain,
  AlertTriangle,
  Loader2,
  Building2,
  Layers,
  Package,
  Boxes,
  ChevronDown,
  ChevronRight,
  Info,
  CheckCircle2,
  RefreshCw,
  WifiOff,
  Clock,
  Users,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { CreditProfileCard } from "@/components/CreditProfileCard";
import { LoyaltyCard } from "@/components/LoyaltyCard";
import { motion, AnimatePresence } from "framer-motion";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { getPricingRules } from "../../../lib/api";
import { retailerProfileUpdateKey } from "@pegasusx/api-core";
import { useOptionalWebSocket } from "../../../lib/ws";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import { getRetailerProfile, getRetailerId, mergeRetailerProfile } from "@/lib/retailer-profile";
import type { AutoOrderSettings } from "../../../lib/types";
import {
  normalizeReceivingWindow,
  validateReceivingWindowField,
} from "../../../lib/receiving-window";
import { Toggle, OverrideRow, OverrideSection } from "../../../components/settings/OverrideComponents";
import { ProfileField, ProfileTimeField } from "../../../components/settings/ProfileFields";
⋮----
function getBrowserStorage(): Storage | null
⋮----
type ProfileFieldErrors = {
  name?: string;
  company?: string;
  receivingWindowOpen?: string;
  receivingWindowClose?: string;
};
⋮----
type SaveBanner = {
  kind: "success" | "error";
  message: string;
};
⋮----
type LoadIssue = "restricted" | "offline" | "error";
⋮----
function normalizeErrorMessage(error: unknown, fallback: string): string
⋮----
function validateProfileFields(
  name: string,
  company: string,
  receivingWindowOpen: string,
  receivingWindowClose: string,
): ProfileFieldErrors
⋮----
onClick=
⋮----
label=
⋮----
setProfileName(value);
clearProfileFieldError("name");
⋮----
setProfileReceivingWindowClose(value);
clearProfileFieldError("receivingWindowClose");
⋮----
setNotifOn(next);
⋮----
String(next),
⋮----
title=
⋮----
getLabel={(c) =>
getHasHistory={(c) =>
getAnalyticsDate={(c) =>
```

## File: apps/retailer-app-desktop/components/PaymentModal.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useCallback, useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  CreditCard,
  Banknote,
  X,
  Loader2,
  ExternalLink,
  ShieldCheck,
  CheckCircle2,
  AlertTriangle,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { PaymentRequired } from "@pegasusx/types";
import { retailerConfirmCashKey, retailerCardCheckoutKey } from "@pegasusx/api-core";
import { useWsEvent, useWebSocket, type WsMessage } from "../lib/ws";
import { packCurrency, readCachedAuthSession } from "@pegasusx/api-core";
import { apiFetch } from "../lib/auth";
import { displayPackCurrency, filterRetailerCardGateways } from "../lib/payment-catalog";
import { useRetailerPaymentCatalog } from "../lib/use-payment-catalog";
import type { CardCheckoutResponse, PendingPaymentSession, PendingPaymentsResponse } from "../lib/types";
⋮----
type PaymentEvent = Omit<PaymentRequired, "available_card_gateways" | "amount"> & {
  available_card_gateways?: string[];
  amount: number;
  original_amount?: number;
  type: string;
  session_id?: string;
  invoice_id?: string | null;
  message?: string;
  gateway?: string;
};
⋮----
type PaymentState =
  | "idle"
  | "choosing"
  | "confirm-cash"
  | "processing"
  | "cash-pending"
  | "fiscalizing"
  | "success"
  | "error";
⋮----
function formatAmount(amount: number): string
⋮----
function toFiniteNumber(value: unknown, fallback: number): number
⋮----
function sessionToPaymentEvent(session: PendingPaymentSession): PaymentEvent
⋮----
function wsMessageToPaymentEvent(msg: WsMessage): PaymentEvent
⋮----
/* -- Component -- */
⋮----
async function loadPendingPayments()
⋮----
async function resumeDeliveryPayment()
⋮----
async function reconcilePendingPayment()
⋮----
{formatAmount(event.amount)}
⋮----
The driver will collect {formatAmount(event.amount)}
⋮----
onClick=
```

## File: apps/retailer-app-desktop/lib/firebase.ts
```typescript
import {
  resetEmulatorPhoneOtpFlow,
  sendPhoneOtpViaEmulator,
  verifyPhoneOtpViaEmulator,
} from "@/components/ui/firebase-emulator-phone";
import { initializeApp, getApps, getApp } from "firebase/app";
import {
  getAuth,
  connectAuthEmulator,
  signInWithPhoneNumber,
} from "firebase/auth";
⋮----
function shouldUseAuthEmulator(): boolean
⋮----
function resolveAuthEmulatorHost(): string
⋮----
interface EmulatorAwareAuth {
  _emulatorConnected?: boolean;
}
⋮----
export async function firebaseSignOut(): Promise<void>
⋮----
type PhoneConfirmation = Awaited<ReturnType<typeof signInWithPhoneNumber>>;
⋮----
class EmulatorRecaptchaVerifier
⋮----
async verify(): Promise<string>
⋮----
_reset(): void
⋮----
export async function sendPhoneOtp(phone: string): Promise<void>
⋮----
export async function verifyPhoneOtp(code: string): Promise<string>
⋮----
export function resetPhoneOtpFlow(): void
```

## File: apps/supplier-portal/components/LiveOpsMap.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef, useState } from "react";
import type { TwinOpsRouteView, TwinVehicleInventoryRow } from "@pegasusx/types";
import MapGL, { Layer, NavigationControl, Source } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import { useMapLibreTeardown } from "@/components/ui";
import { createSupplierApi } from "@/lib/api";
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-core";
⋮----
type LiveOpsMapProps = {
  routes: TwinOpsRouteView[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  selectedRouteId?: string | null;
  onSelectRoute?: (route: TwinOpsRouteView | null) => void;
};
⋮----
ETA window:
⋮----
<div>Predicted:
```

## File: apps/warehouse-portal/app/dispatch/page.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import type {
  WarehouseDispatchCapacityWarning,
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseUnavailableDispatchDriver,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { ApiError, warehouseDispatchKey, warehouseUpdateVehicleKey } from '@pegasusx/api-core';
import { ExplainStatusBanner, explainFromApiError, VirtualScrollList } from '@/components/ui';
import type { StatusExplain } from '@pegasusx/types';
import { isTauri } from '@pegasusx/desktop-bridge';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import {
  dispatchPreviewCacheKey,
  getDispatchPreviewCache,
  setDispatchPreviewCache,
  snapshotFromPreviewResponse,
  type DispatchPreviewSnapshot,
} from '@/lib/dispatch-preview-cache';
import {
  dispatchRunsCacheKey,
  getDispatchRunsCache,
  setDispatchRunsCache,
  type DispatchRunRow,
} from '@/lib/dispatch-runs-cache';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { WAREHOUSE_DISPATCH_REFRESH_EVENTS, parseWarehouseWsEventType } from '@/lib/fleet-ws-events';
import { subscribeWarehouseWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import DispatchPreviewMap from '@/components/DispatchPreviewMap';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import HandoffTimelinePanel from '@/components/HandoffTimelinePanel';
import EmptyState from '@/components/EmptyState';
import { OrderActionDialog, OrderOpsCard, OrderProposeDateDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import DispatchOrderList from '@/components/dispatch/DispatchOrderList';
import DispatchDriverList from '@/components/dispatch/DispatchDriverList';
import {
  DispatchConfirmAndCapacityModals,
  DispatchDriverRescueSection,
  DispatchFleetTrucksSection,
  formatUnavailableReason,
  formatVU,
} from '@/components/dispatch/DispatchPanelsAndModals';
import { warehouseOps } from '@/lib/warehouse-ops';
⋮----
type CapacityPromptMode = 'manual' | 'auto';
type DispatchMode = 'manual' | 'smart';
⋮----
function routeVolumeVU(route: WarehouseDispatchProposedRoute)
⋮----
function isoDeliveryDate(dateInput: string): string
⋮----
// Early Complete
⋮----
const closeOpsDialog = () =>
⋮----
async function submitOpsDialog()
⋮----
async function handleLookupRescue()
⋮----
async function handleApproveRescue()
⋮----
const toggleOrder = (orderId: string) =>
⋮----
const toggleSelectAll = () =>
⋮----
const applyCapacitySuggestion = () =>
⋮----
async function handleToggleVehicleAvailability(vehicle: WarehouseFleetVehicle, nextActive: boolean)
⋮----
const fmt = (n: number) =>
⋮----
description=
⋮----
onClick=
⋮----
<KpiStatCard label=
⋮----
label=
⋮----
title=
⋮----
<PageSection title=
⋮----
onReject={(orderId) =>
```

## File: apps/warehouse-portal/components/cashier/AMLExposureCheck.tsx
```typescript
import React, { useState } from 'react';
import { TIYIN_PER_UZS, STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { Building2, ShieldCheck, AlertCircle, CheckCircle2 } from 'lucide-react';
⋮----
interface AMLExposureCheckProps {
  initialTin?: string;
  supplierId?: string;
}
⋮----
export function AMLExposureCheck({
  initialTin = '301234567',
  supplierId = 'sup_samarkand_dairy',
}: AMLExposureCheckProps)
⋮----
const handleCheck = async (e: React.FormEvent) =>
⋮----
Statutory Benchmark:
```

## File: apps/warehouse-portal/lib/auth.ts
```typescript
import { useEffect, useState } from 'react';
import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession, reconcileSession, reconnectDelayMs } from '@pegasusx/api-core';
import { clearStoredToken, getStoredToken, isTauri, storeToken } from '@/lib/bridge';
import { notifyWarehouseSessionReconciled } from '@/lib/warehouse-reconnect';
⋮----
/** GS-C5: pin to JWT home_cell. Login (no token) stays on bootstrap. */
export function warehouseApiBaseUrl(): string
⋮----
export function readTokenFromCookie(): string
⋮----
function readRefreshFromCookie(): string
⋮----
export function persistSession(token: string, refreshToken?: string)
⋮----
export function clearSession()
⋮----
export function useToken(): string
⋮----
export function decodeJwtPayload(token: string): Record<string, unknown> | null
⋮----
export async function getWarehouseToken(): Promise<string>
⋮----
async function tryRefreshToken(): Promise<string | null>
⋮----
export async function refreshWarehouseSession(): Promise<{ ok: boolean; isConfigured?: boolean }>
⋮----
export async function apiFetch(path: string, init?: RequestInit): Promise<Response>
⋮----
function toApiPath(input: RequestInfo | URL): string
⋮----
export const warehouseSessionFetch: typeof fetch = (input, init) =>
⋮----
export async function connectWarehouseWS(lastEventId?: string | null): Promise<WebSocket>
⋮----
export type WarehouseSocketStatus = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'offline';
⋮----
export function subscribeWarehouseWS(options: {
  onMessage: (payload: string) => void;
  onStatusChange?: (status: WarehouseSocketStatus) => void;
}): () => void
⋮----
const clearReconnect = () =>
⋮----
const clearHeartbeat = () =>
⋮----
const triggerWarehouseReconcile = () =>
⋮----
const scheduleReconnect = () =>
⋮----
const openSocket = (isReconnect: boolean) =>
⋮----
const handleOnline = () =>
⋮----
const handleOffline = () =>
```

## File: apps/retailer-app-desktop/components/CheckoutModal.tsx
```typescript
import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useRef } from "react";
import {
  X,
  Ticket,
  CreditCard,
  Loader2,
  ShieldCheck,
  Truck,
  ChevronRight,
  AlertTriangle,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useCart } from "../lib/cart";
import { orderableCapsFromPreview } from "../lib/stock-policy";
import { apiFetch } from "../lib/auth";
import { packCurrency, readCachedAuthSession } from "@pegasusx/api-core";
import { useWebSocket } from "../lib/ws";
import { useRouter } from "next/navigation";
import { getRetailerProfile } from "@/lib/retailer-profile";
import { checkoutGatewayForMethod } from "@/lib/payment-catalog";
import { useRetailerPaymentCatalog } from "@/lib/use-payment-catalog";
import type {
  ActiveFulfillmentsResponse,
  CheckoutPreviewResponse,
  PendingPaymentsResponse,
  UnifiedCheckoutResponse,
  RetailerProfile,
  StockWarning,
} from "../lib/types";
import type { PaymentGatewayDegradedPayload } from "@pegasusx/types";
import { retailerUnifiedCheckoutKey } from "@pegasusx/api-core";
import {
  enqueuePendingCheckout,
  pendingCheckoutQueuedMessage,
} from "../lib/pending-checkout";
⋮----
function getProfile(): RetailerProfile | null
⋮----
interface CheckoutModalProps {
  isOpen: boolean;
  onClose: () => void;
  total: number;
}
⋮----
async function reconcileCheckout()
⋮----
const handleInitiateCard = async () =>
⋮----
const handleConfirmCard = async () =>
⋮----
const handleCheckout = async () =>
⋮----
onChange=
⋮----
onClick=
⋮----
setPendingCardToken(null);
setCardOtpCode("");
```

## File: apps/retailer-app-desktop/components/RetailerShell.tsx
```typescript
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { useEffect, useState, useMemo, useCallback, memo } from "react";
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";
import {
  PanelLeftClose,
  PanelLeft,
  Menu,
  Bell,
  Search,
  LogOut,
  Store,
  X,
} from "lucide-react";
import { getRetailerProfile } from "@/lib/retailer-profile";
import { useWebSocket } from "../lib/ws";
import { useRetailerNotifications } from "../lib/notifications";
import { clearStoredToken } from "../lib/bridge";
import { useTheme, type ThemeMode } from "./ThemeProvider";
import Icon from "./Icon";
import { apiFetch } from "@/lib/auth";
import { SessionPackChip } from "./SessionPackChip";
import { OrgSwitcher } from "./OrgSwitcher";
import LanguageSwitcher from "./LanguageSwitcher";
import { usePortalT } from "@/lib/i18n";
import { NAV, navSectionIsActive, type NavSection } from "@/lib/nav";
import { DesktopKeyboardCheatsheet } from "./ui/DesktopKeyboardCheatsheet";
⋮----
type RetailerIdentity = { name: string; company: string; initials: string };
⋮----
function filterNavByPerms(
  sections: NavSection[],
  perms: string[] | null,
  packs: string[] | null,
): NavSection[]
⋮----
function isActiveRoute(pathname: string, href: string): boolean
⋮----
function buildBreadcrumbs(
  pathname: string,
  t: (key: string) => string,
): { label: string; href: string }[]
⋮----
aria-label=
⋮----
const handler = (e: KeyboardEvent) =>
```

## File: apps/warehouse-portal/app/cashier/page.tsx
```typescript
import React, { useState } from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { BlindCountDenominationGrid } from '@/components/cashier/BlindCountDenominationGrid';
import { CashierSessionView } from '@/components/cashier/CashierSessionView';
import { RRNReclassificationModal } from '@/components/cashier/RRNReclassificationModal';
import { CITHandoverManifest } from '@/components/cashier/CITHandoverManifest';
import { AMLExposureCheck } from '@/components/cashier/AMLExposureCheck';
import { Coins, Shield, Building2, Barcode, PlusCircle } from 'lucide-react';
⋮----
// RRN Modal State
⋮----
// Quick preset bag seals for high-velocity cashier operations
const fillSampleSeal = (barcode: string, drv: string, route: string) =>
⋮----
// Start new bag blind count session
const handleStartSession = async (e: React.FormEvent) =>
⋮----
const handleSubmitBlindCount = async (
    denominations: Record<number, number>,
    _totalMinor: number,
    toleranceMinor: number,
  ) =>
⋮----
onChange=
⋮----
onSuccess={(updated) =>
```
