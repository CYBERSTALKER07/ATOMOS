//
//  InsightsView.swift
//  reatilerapp
//
//  Expense Analytics Dashboard — Swift Charts
//

import SwiftUI
import Charts

// MARK: - Data Models

struct MonthlyExpense: Codable, Identifiable {
    let month: String
    let total: Int

    var id: String { month }

    enum CodingKeys: String, CodingKey {
        case month
        case total = "total"
    }

    var shortMonth: String {
        // "2026-03" → "Mar"
        let parts = month.split(separator: "-")
        guard parts.count == 2, let m = Int(parts[1]) else { return month }
        return Calendar.current.shortMonthSymbols[m - 1]
    }
}

struct TopSupplierExpense: Codable, Identifiable {
    let supplierID: String
    let supplierName: String
    let total: Int
    let orderCount: Int

    var id: String { supplierID }

    enum CodingKeys: String, CodingKey {
        case supplierID = "supplier_id"
        case supplierName = "supplier_name"
        case total = "total"
        case orderCount = "order_count"
    }
}

struct TopProductExpense: Codable, Identifiable {
    let productID: String
    let productName: String
    let total: Int
    let quantity: Int

    var id: String { productID }

    enum CodingKeys: String, CodingKey {
        case productID = "product_id"
        case productName = "product_name"
        case total = "total"
        case quantity
    }
}

struct RetailerAnalytics: Codable {
    let monthlyExpenses: [MonthlyExpense]
    let topSuppliers: [TopSupplierExpense]
    let topProducts: [TopProductExpense]
    let totalThisMonth: Int
    let totalLastMonth: Int

    enum CodingKeys: String, CodingKey {
        case monthlyExpenses = "monthly_expenses"
        case topSuppliers = "top_suppliers"
        case topProducts = "top_products"
        case totalThisMonth = "total_this_month"
        case totalLastMonth = "total_last_month"
    }
}

// MARK: - Detailed Analytics Models

struct RetailerDayExpense: Codable, Identifiable {
    let date: String
    let total: Int
    let count: Int
    var id: String { date }

    var shortDate: String {
        String(date.suffix(5)) // "MM-DD"
    }
}

struct OrderStateCount: Codable, Identifiable {
    let state: String
    let count: Int
    var id: String { state }
}

struct CategorySpend: Codable, Identifiable {
    let category: String
    let total: Int
    let count: Int
    var id: String { category }
}

struct DayOfWeekPattern: Codable, Identifiable {
    let weekday: String
    let avg: Int
    let count: Int
    var id: String { weekday }
}

struct RetailerDetailedAnalytics: Codable {
    let dailySpending: [RetailerDayExpense]
    let ordersByState: [OrderStateCount]
    let categoryBreakdown: [CategorySpend]
    let weekdayPattern: [DayOfWeekPattern]
    let totalSpent: Int
    let totalOrders: Int
    let avgOrderValue: Int

    enum CodingKeys: String, CodingKey {
        case dailySpending = "daily_spending"
        case ordersByState = "orders_by_state"
        case categoryBreakdown = "category_breakdown"
        case weekdayPattern = "weekday_pattern"
        case totalSpent = "total_spent"
        case totalOrders = "total_orders"
        case avgOrderValue = "avg_order_value"
    }
}

// MARK: - Date Range

enum DateRange: String, CaseIterable {
    case week = "7D"
    case month = "1M"
    case quarter = "Q1"
    case halfYear = "6M"

    var days: Int {
        switch self {
        case .week: return 7
        case .month: return 30
        case .quarter: return 90
        case .halfYear: return 180
        }
    }
}

// MARK: - Insights View

struct InsightsView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = InsightsViewModel()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: AppTheme.spacingXL) {
                // Header
                VStack(alignment: .leading, spacing: 4) {
                    Text("Expense Insights")
                        .font(.system(.title2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("Track your procurement spending")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, 8)

                // Date Range Picker
                HStack(spacing: 8) {
                    ForEach(DateRange.allCases, id: \.self) { range_ in
                        Button {
                            withAnimation(AnimationConstants.fluid) { vm.selectedRange = range_ }
                            Task { await vm.loadDetailedAnalytics() }
                        } label: {
                            Text(range_.rawValue)
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 7)
                                .background(vm.selectedRange == range_ ? AppTheme.accent : AppTheme.surfaceElevated)
                                .foregroundStyle(vm.selectedRange == range_ ? .white : AppTheme.textSecondary)
                                .clipShape(.capsule)
                        }
                    }
                    Spacer()
                }
                .padding(.horizontal, AppTheme.spacingLG)

                // Loading spinner while data is not yet available
                if vm.isLoading && vm.analytics == nil {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                        .tint(AppTheme.accent)
                }

                // KPI Cards
                if let a = vm.analytics {
                    HStack(spacing: 12) {
                        KPICard(
                            title: "This Month",
                            value: formatAmount(a.totalThisMonth),
                            subtitle: "Amount"
                        )
                        KPICard(
                            title: "vs Last Month",
                            value: vm.delta >= 0 ? "+\(vm.delta)%" : "\(vm.delta)%",
                            subtitle: vm.delta >= 0 ? "increase" : "decrease",
                            isPositive: vm.delta < 0 // Lower spend is good
                        )
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                if !vm.predictions.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("AI Demand Signals")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                            .padding(.horizontal, AppTheme.spacingLG)

                        ForEach(vm.predictions) { forecast in
                            HStack(spacing: AppTheme.spacingMD) {
                                VStack(alignment: .leading, spacing: 4) {
                                    HStack(spacing: 6) {
                                        Text(forecast.productName)
                                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                            .foregroundStyle(AppTheme.textPrimary)
                                        if forecast.isBlocked {
                                            Text("Insufficient history")
                                                .font(.system(size: 10, weight: .bold, design: .rounded))
                                                .foregroundStyle(AppTheme.warning)
                                                .padding(.horizontal, 8)
                                                .padding(.vertical, 3)
                                                .background(AppTheme.warning.opacity(0.12))
                                                .clipShape(Capsule())
                                        }
                                    }
                                    Text("\(forecast.predictedQuantity) units · \(forecast.confidencePercent)")
                                        .font(.system(.caption, design: .rounded))
                                        .foregroundStyle(AppTheme.textTertiary)
                                }
                                Spacer()
                                Button {
                                    Task { await vm.dismissPrediction(forecast) }
                                } label: {
                                    Text(vm.correctingId == forecast.id ? "Updating…" : "Dismiss")
                                        .font(.system(.caption, design: .rounded, weight: .bold))
                                        .foregroundStyle(AppTheme.destructive)
                                }
                                .disabled(vm.correctingId == forecast.id)
                            }
                            .padding(AppTheme.spacingMD)
                            .background(AppTheme.cardBackground)
                            .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                            .padding(.horizontal, AppTheme.spacingLG)
                        }
                    }
                }

                // Monthly Trend Chart
                if let expenses = vm.analytics?.monthlyExpenses, !expenses.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Monthly Trend")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)

                        Chart(expenses) { expense in
                            LineMark(
                                x: .value("Month", expense.shortMonth),
                                y: .value("Amount", expense.total)
                            )
                            .foregroundStyle(AppTheme.accent)
                            .interpolationMethod(.catmullRom)

                            AreaMark(
                                x: .value("Month", expense.shortMonth),
                                y: .value("Amount", expense.total)
                            )
                            .foregroundStyle(
                                .linearGradient(
                                    colors: [AppTheme.accent.opacity(0.15), .clear],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )
                            .interpolationMethod(.catmullRom)

                            PointMark(
                                x: .value("Month", expense.shortMonth),
                                y: .value("Amount", expense.total)
                            )
                            .foregroundStyle(AppTheme.accent)
                            .symbolSize(30)
                        }
                        .chartYAxis {
                            AxisMarks(position: .leading) { value in
                                AxisValueLabel {
                                    if let v = value.as(Int.self) {
                                        Text(abbreviateAmount(v))
                                            .font(.system(size: 10, design: .rounded))
                                            .foregroundStyle(AppTheme.textTertiary)
                                    }
                                }
                            }
                        }
                        .chartXAxis {
                            AxisMarks { value in
                                AxisValueLabel()
                                    .font(.system(size: 10, design: .rounded))
                            }
                        }
                        .frame(height: 200)
                    }
                    .padding(AppTheme.spacingLG)
                    .background(AppTheme.cardBackground)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                // Top Suppliers
                if let suppliers = vm.analytics?.topSuppliers, !suppliers.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Top Suppliers")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)

                        Chart(suppliers) { s in
                            BarMark(
                                x: .value("Amount", s.total),
                                y: .value("Supplier", s.supplierName)
                            )
                            .foregroundStyle(AppTheme.accent.opacity(0.8))
                            .clipShape(.rect(cornerRadius: 4))
                        }
                        .chartXAxis {
                            AxisMarks { value in
                                AxisValueLabel {
                                    if let v = value.as(Int.self) {
                                        Text(abbreviateAmount(v))
                                            .font(.system(size: 10, design: .rounded))
                                            .foregroundStyle(AppTheme.textTertiary)
                                    }
                                }
                            }
                        }
                        .frame(height: Double(suppliers.count * 44))
                    }
                    .padding(AppTheme.spacingLG)
                    .background(AppTheme.cardBackground)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                // Top Products
                if let products = vm.analytics?.topProducts, !products.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Top Products")
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)

                        ForEach(products) { product in
                            HStack(spacing: 12) {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(product.productName)
                                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                                        .foregroundStyle(AppTheme.textPrimary)
                                        .lineLimit(1)
                                    Text("\(product.quantity) units")
                                        .font(.system(.caption2, design: .rounded))
                                        .foregroundStyle(AppTheme.textTertiary)
                                }
                                Spacer()
                                Text(formatAmount(product.total))
                                    .font(.system(.subheadline, design: .monospaced, weight: .medium))
                                    .foregroundStyle(AppTheme.textPrimary)
                            }
                            .padding(.vertical, 6)
                            if product.id != products.last?.id {
                                Divider()
                            }
                        }
                    }
                    .padding(AppTheme.spacingLG)
                    .background(AppTheme.cardBackground)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                // Empty State
                if vm.analytics == nil && !vm.isLoading {
                    VStack(spacing: 16) {
                        Image(systemName: "chart.line.uptrend.xyaxis")
                            .font(.system(size: 40))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text("No Analytics Data")
                            .font(.system(.headline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Complete a few orders and your expense insights will appear here")
                            .font(.system(.subheadline, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 60)
                }

                // ── Advanced Insights Section ──
                if let d = vm.detailed {
                    // Summary KPIs
                    HStack(spacing: 12) {
                        KPICard(
                            title: "Total Spent",
                            value: formatAmount(d.totalSpent),
                            subtitle: "\(d.totalOrders) orders"
                        )
                        KPICard(
                            title: "Avg Order",
                            value: formatAmount(d.avgOrderValue),
                            subtitle: "per order"
                        )
                    }
                    .padding(.horizontal, AppTheme.spacingLG)

                    // Daily Spending Line
                    if !d.dailySpending.isEmpty {
                        DailySpendingChartView(data: d.dailySpending)
                            .padding(.horizontal, AppTheme.spacingLG)
                    }

                    // Orders by State
                    if !d.ordersByState.isEmpty {
                        OrdersByStateView(data: d.ordersByState)
                            .padding(.horizontal, AppTheme.spacingLG)
                    }

                    // Category Breakdown
                    if !d.categoryBreakdown.isEmpty {
                        CategoryBreakdownView(data: d.categoryBreakdown)
                            .padding(.horizontal, AppTheme.spacingLG)
                    }

                    // Weekday Pattern
                    if !d.weekdayPattern.isEmpty {
                        WeekdayPatternView(data: d.weekdayPattern)
                            .padding(.horizontal, AppTheme.spacingLG)
                    }
                }

                Spacer(minLength: 40)
            }
        }
        .background(AppTheme.background)
        .task {
            await vm.loadAnalytics()
            await vm.loadDetailedAnalytics()
        }
        .task(id: refreshCenter.refreshToken) {
            await vm.loadAnalytics()
            await vm.loadDetailedAnalytics()
        }
        .refreshable {
            await vm.loadAnalytics()
            await vm.loadDetailedAnalytics()
        }
    }

    private func formatAmount(_ value: Int) -> String {
        value.formatted(.number.grouping(.automatic)) + ""
    }

    private func abbreviateAmount(_ value: Int) -> String {
        if value >= 1_000_000 { return "\(value / 1_000_000)M" }
        if value >= 1_000 { return "\(value / 1_000)K" }
        return "\(value)"
    }
}



#Preview {
    NavigationStack {
        InsightsView()
    }
}
