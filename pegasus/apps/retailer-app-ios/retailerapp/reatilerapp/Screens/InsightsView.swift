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
    @State private var analytics: RetailerAnalytics?
    @State private var detailed: RetailerDetailedAnalytics?
    @State private var isLoading = false
    @State private var selectedRange: DateRange = .month

    private let api = APIClient.shared

    private var delta: Int {
        guard let a = analytics else { return 0 }
        guard a.totalLastMonth > 0 else { return 0 }
        return Int(Double(a.totalThisMonth - a.totalLastMonth) / Double(a.totalLastMonth) * 100)
    }

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
                            withAnimation(AnimationConstants.fluid) { selectedRange = range_ }
                            Task { await loadDetailedAnalytics() }
                        } label: {
                            Text(range_.rawValue)
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 7)
                                .background(selectedRange == range_ ? AppTheme.accent : AppTheme.surfaceElevated)
                                .foregroundStyle(selectedRange == range_ ? .white : AppTheme.textSecondary)
                                .clipShape(.capsule)
                        }
                    }
                    Spacer()
                }
                .padding(.horizontal, AppTheme.spacingLG)

                // Loading spinner while data is not yet available
                if isLoading && analytics == nil {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                        .tint(AppTheme.accent)
                }

                // KPI Cards
                if let a = analytics {
                    HStack(spacing: 12) {
                        KPICard(
                            title: "This Month",
                            value: formatAmount(a.totalThisMonth),
                            subtitle: "Amount"
                        )
                        KPICard(
                            title: "vs Last Month",
                            value: delta >= 0 ? "+\(delta)%" : "\(delta)%",
                            subtitle: delta >= 0 ? "increase" : "decrease",
                            isPositive: delta < 0 // Lower spend is good
                        )
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }

                // Monthly Trend Chart
                if let expenses = analytics?.monthlyExpenses, !expenses.isEmpty {
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
                if let suppliers = analytics?.topSuppliers, !suppliers.isEmpty {
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
                if let products = analytics?.topProducts, !products.isEmpty {
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
                if analytics == nil && !isLoading {
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
                if let d = detailed {
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
            await loadAnalytics()
            await loadDetailedAnalytics()
        }
        .refreshable {
            await loadAnalytics()
            await loadDetailedAnalytics()
        }
    }

    // MARK: - API

    private func loadAnalytics() async {
        isLoading = true
        do {
            let result: RetailerAnalytics = try await api.get(path: "/v1/retailer/analytics/expenses")
            withAnimation(AnimationConstants.fluid) {
                analytics = result
            }
        } catch {
            // Use sample data as fallback
            withAnimation(AnimationConstants.fluid) {
                analytics = RetailerAnalytics(
                    monthlyExpenses: [
                        MonthlyExpense(month: "2025-10", total: 8_500_000),
                        MonthlyExpense(month: "2025-11", total: 12_300_000),
                        MonthlyExpense(month: "2025-12", total: 9_800_000),
                        MonthlyExpense(month: "2026-01", total: 15_200_000),
                        MonthlyExpense(month: "2026-02", total: 11_700_000),
                        MonthlyExpense(month: "2026-03", total: 14_100_000),
                    ],
                    topSuppliers: [
                        TopSupplierExpense(supplierID: "sup-001", supplierName: "Coca-Cola", total: 18_500_000, orderCount: 14),
                        TopSupplierExpense(supplierID: "sup-005", supplierName: "Local Farms", total: 12_200_000, orderCount: 11),
                        TopSupplierExpense(supplierID: "sup-002", supplierName: "Nestlé", total: 9_800_000, orderCount: 9),
                        TopSupplierExpense(supplierID: "sup-003", supplierName: "PepsiCo", total: 7_400_000, orderCount: 7),
                        TopSupplierExpense(supplierID: "sup-004", supplierName: "Unilever", total: 5_100_000, orderCount: 5),
                    ],
                    topProducts: [
                        TopProductExpense(productID: "prod-001", productName: "Organic Whole Milk", total: 8_200_000, quantity: 156),
                        TopProductExpense(productID: "prod-003", productName: "Free-Range Eggs", total: 6_500_000, quantity: 120),
                        TopProductExpense(productID: "prod-002", productName: "Sourdough Bread", total: 4_900_000, quantity: 89),
                        TopProductExpense(productID: "prod-005", productName: "Sparkling Water", total: 3_200_000, quantity: 210),
                        TopProductExpense(productID: "prod-004", productName: "Greek Yogurt", total: 2_800_000, quantity: 65),
                    ],
                    totalThisMonth: 14_100_000,
                    totalLastMonth: 11_700_000
                )
            }
        }
        isLoading = false
    }

    private func loadDetailedAnalytics() async {
        let to = Date()
        let from = Calendar.current.date(byAdding: .day, value: -selectedRange.days, to: to)!
        let fmt = ISO8601DateFormatter()
        fmt.formatOptions = [.withFullDate]
        let fromStr = fmt.string(from: from)
        let toStr = fmt.string(from: to)
        do {
            let result: RetailerDetailedAnalytics = try await api.get(
                path: "/v1/retailer/analytics/detailed?from=\(fromStr)&to=\(toStr)"
            )
            withAnimation(AnimationConstants.fluid) {
                detailed = result
            }
        } catch {
            // Keep existing data on failure
        }
    }

    // MARK: - Formatters

    private func formatAmount(_ value: Int) -> String {
        value.formatted(.number.grouping(.automatic)) + ""
    }

    private func abbreviateAmount(_ value: Int) -> String {
        if value >= 1_000_000 { return "\(value / 1_000_000)M" }
        if value >= 1_000 { return "\(value / 1_000)K" }
        return "\(value)"
    }
}

// MARK: - KPI Card

private struct KPICard: View {
    let title: String
    let value: String
    let subtitle: String
    var isPositive: Bool = true

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(.caption, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textTertiary)
            Text(value)
                .font(.system(.title3, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.7)
            Text(subtitle)
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }
}

// MARK: - Daily Spending Chart

private struct DailySpendingChartView: View {
    let data: [RetailerDayExpense]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Daily Spending")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                LineMark(
                    x: .value("Date", item.shortDate),
                    y: .value("Amount", item.total)
                )
                .foregroundStyle(AppTheme.accent)
                .interpolationMethod(.catmullRom)

                AreaMark(
                    x: .value("Date", item.shortDate),
                    y: .value("Amount", item.total)
                )
                .foregroundStyle(
                    .linearGradient(
                        colors: [AppTheme.accent.opacity(0.2), .clear],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .interpolationMethod(.catmullRom)
            }
            .frame(height: 200)
            .chartXAxis {
                AxisMarks(values: .automatic(desiredCount: 6)) { _ in
                    AxisValueLabel()
                        .font(.system(size: 9, design: .rounded))
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }
}

// MARK: - Orders by State

private let stateColors: [String: Color] = [
    "COMPLETED": .green,
    "ARRIVED": .blue,
    "IN_TRANSIT": .orange,
    "PENDING": .yellow,
    "LOADED": .purple,
    "CANCELLED": .pink,
    "CANCELLED_BY_ADMIN": .red,
]

private struct OrdersByStateView: View {
    let data: [OrderStateCount]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Orders by Status")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                SectorMark(
                    angle: .value("Count", item.count),
                    innerRadius: .ratio(0.6),
                    angularInset: 1.5
                )
                .foregroundStyle(stateColors[item.state] ?? .gray)
                .cornerRadius(3)
            }
            .frame(height: 200)

            // Legend
            FlowLayout(spacing: 8) {
                ForEach(data) { item in
                    HStack(spacing: 4) {
                        Circle()
                            .fill(stateColors[item.state] ?? .gray)
                            .frame(width: 8, height: 8)
                        Text("\(item.state.replacingOccurrences(of: "_", with: " ")) (\(item.count))")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }
}

// Simple flow layout for legend
private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = layout(proposal: proposal, subviews: subviews)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = layout(proposal: proposal, subviews: subviews)
        for (index, offset) in result.offsets.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + offset.x, y: bounds.minY + offset.y), proposal: .unspecified)
        }
    }

    private func layout(proposal: ProposedViewSize, subviews: Subviews) -> (offsets: [CGPoint], size: CGSize) {
        let maxWidth = proposal.width ?? .infinity
        var offsets: [CGPoint] = []
        var x: CGFloat = 0
        var y: CGFloat = 0
        var lineHeight: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > maxWidth && x > 0 {
                x = 0
                y += lineHeight + spacing
                lineHeight = 0
            }
            offsets.append(CGPoint(x: x, y: y))
            lineHeight = max(lineHeight, size.height)
            x += size.width + spacing
        }

        return (offsets, CGSize(width: maxWidth, height: y + lineHeight))
    }
}

// MARK: - Category Breakdown

private struct CategoryBreakdownView: View {
    let data: [CategorySpend]

    private var maxTotal: Int { data.map(\.total).max() ?? 1 }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Spending by Category")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                BarMark(
                    x: .value("Amount", item.total),
                    y: .value("Category", item.category)
                )
                .foregroundStyle(AppTheme.accent.opacity(0.8))
                .clipShape(.rect(cornerRadius: 4))
            }
            .chartXAxis {
                AxisMarks { value in
                    AxisValueLabel {
                        if let v = value.as(Int.self) {
                            Text(abbreviate(v))
                                .font(.system(size: 10, design: .rounded))
                        }
                    }
                }
            }
            .frame(height: CGFloat(data.count * 36))
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }

    private func abbreviate(_ value: Int) -> String {
        if value >= 1_000_000 { return "\(value / 1_000_000)M" }
        if value >= 1_000 { return "\(value / 1_000)K" }
        return "\(value)"
    }
}

// MARK: - Weekday Pattern

private struct WeekdayPatternView: View {
    let data: [DayOfWeekPattern]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Ordering Pattern by Day")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)

            Chart(data) { item in
                BarMark(
                    x: .value("Day", String(item.weekday.prefix(3))),
                    y: .value("Orders", item.count)
                )
                .foregroundStyle(AppTheme.accent)
                .clipShape(.rect(cornerRadius: 4))
            }
            .frame(height: 160)

            // Avg spend per day
            HStack {
                ForEach(data) { d in
                    VStack(spacing: 2) {
                        Text(String(d.weekday.prefix(3)))
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text(abbreviate(d.avg))
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
    }

    private func abbreviate(_ value: Int) -> String {
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
