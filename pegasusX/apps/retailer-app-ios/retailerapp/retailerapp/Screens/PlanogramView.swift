//
//  PlanogramView.swift
//  retailerapp
//
//  Retail OS Pack 8 Planogram & Shelf Vision Screen.
//  Provides shelf slotting schema visualization, aisle walk checklist, and Vision AI compliance auditing.
//

import SwiftUI

struct PlanogramSlotIOS: Identifiable {
    let id: String
    let shelfIndex: Int
    let colIndex: Int
    let expectedSkuName: String
    let facings: Int
    let minFacings: Int
}

struct PlanogramFindingIOS: Identifiable {
    let id: String
    let type: String // GAP, WRONG_SKU, EMPTY, OK
    let shelfIndex: Int
    let colIndex: Int
    let expectedSku: String
    let detectedSku: String?
    let confidence: Double
    var status: String // PENDING_REVIEW, ACCEPTED, DISMISSED
}

struct PlanogramView: View {
    @State private var selectedTab = 0
    @State private var banner: String?
    @State private var isBusy = false
    @State private var isRunningVision = false
    @State private var findings: [PlanogramFindingIOS] = [
        PlanogramFindingIOS(
            id: "f-1",
            type: "GAP",
            shelfIndex: 2,
            colIndex: 1,
            expectedSku: "Whole Milk 3.2% (1L)",
            detectedSku: nil,
            confidence: 0.95,
            status: "PENDING_REVIEW"
        ),
        PlanogramFindingIOS(
            id: "f-2",
            type: "WRONG_SKU",
            shelfIndex: 2,
            colIndex: 2,
            expectedSku: "Kefir 1% (500g)",
            detectedSku: "Soda Can 0.33L",
            confidence: 0.89,
            status: "PENDING_REVIEW"
        ),
        PlanogramFindingIOS(
            id: "f-3",
            type: "EMPTY",
            shelfIndex: 3,
            colIndex: 4,
            expectedSku: "Organic Butter 82% (200g)",
            detectedSku: nil,
            confidence: 0.92,
            status: "PENDING_REVIEW"
        )
    ]

    var body: some View {
        VStack(spacing: 0) {
            Picker("Mode", selection: $selectedTab) {
                Text("Slotting").tag(0)
                Text("Checklist").tag(1)
                Text("Camera Vision").tag(2)
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)

            if let banner {
                Text(banner)
                    .font(.caption)
                    .foregroundStyle(AppTheme.accent)
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingSM)
            }

            if selectedTab == 0 {
                slottingGridTab
            } else if selectedTab == 1 {
                walkChecklistTab
            } else {
                cameraVisionTab
            }
        }
        .navigationTitle("Planograms & Vision")
        .navigationBarTitleDisplayMode(.inline)
    }

    // MARK: - Tab 1: Shelf Slotting Layout Grid

    private var slottingGridTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                LabCard {
                    VStack(alignment: .leading, spacing: 6) {
                        HStack {
                            Text("Aisle 1 — Dairy & Beverage Bay A")
                                .font(.system(.headline, design: .rounded, weight: .bold))
                            Spacer()
                            Text("PUBLISHED")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.green.opacity(0.15), in: Capsule())
                                .foregroundStyle(Color.green)
                        }
                        Text("4 Shelves · 16 Active Facings · Top-to-Bottom, Left-to-Right layout")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .padding(AppTheme.spacingMD)
                }

                ForEach(1...4, id: \.self) { shelfRow in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Shelf \(shelfRow) (\(shelfDescription(for: shelfRow)))")
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textSecondary)
                            Spacer()
                            Text("Row \(shelfRow)")
                                .font(.system(size: 10, design: .monospaced))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 10) {
                                ForEach(1...4, id: \.self) { slotCol in
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text("Slot \(slotCol)")
                                            .font(.system(size: 10, weight: .bold))
                                            .foregroundStyle(AppTheme.accent)
                                        Text(slotSkuLabel(row: shelfRow, col: slotCol))
                                            .font(.system(size: 11, weight: .semibold, design: .rounded))
                                            .lineLimit(2)
                                        Spacer()
                                        HStack {
                                            Text("2 facings")
                                                .font(.system(size: 9))
                                                .foregroundStyle(AppTheme.textTertiary)
                                            Spacer()
                                            Circle()
                                                .fill(Color.green)
                                                .frame(width: 5, height: 5)
                                        }
                                    }
                                    .padding(8)
                                    .frame(width: 105, height: 85)
                                    .background(AppTheme.cardBackground)
                                    .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusSM))
                                    .overlay(
                                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                                            .stroke(AppTheme.separator, lineWidth: 0.5)
                                    )
                                }
                            }
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }
            }
            .padding(.vertical, AppTheme.spacingMD)
        }
    }

    // MARK: - Tab 2: Aisle Walk Human Checklist

    private var walkChecklistTab: some View {
        List {
            Section("Aisle Walk Guidance") {
                Text("Inspect shelves top-to-bottom, left-to-right. Toggle to flag compliance or shelf gaps.")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
            }

            Section("Shelf 1 (Top / Beverages)") {
                ChecklistRow(title: "Slot 1 — Mineral Water 1.5L (3 facings)", isChecked: true)
                ChecklistRow(title: "Slot 2 — Premium Sparkling Water 1L (2 facings)", isChecked: true)
                ChecklistRow(title: "Slot 3 — Cola Classic 1.5L (2 facings)", isChecked: true)
                ChecklistRow(title: "Slot 4 — Lemon Tea 1L (2 facings)", isChecked: true)
            }

            Section("Shelf 2 (Eye Level / Dairy)") {
                ChecklistRow(title: "Slot 1 — Whole Milk 3.2% (4 facings)", isChecked: false)
                ChecklistRow(title: "Slot 2 — Kefir 1% (2 facings)", isChecked: false)
                ChecklistRow(title: "Slot 3 — Greek Yogurt 5% (3 facings)", isChecked: true)
                ChecklistRow(title: "Slot 4 — Sour Cream 15% (2 facings)", isChecked: true)
            }

            Section("Shelf 3 (Yogurt & Butter)") {
                ChecklistRow(title: "Slot 1 — Fruit Yogurt Berry (2 facings)", isChecked: true)
                ChecklistRow(title: "Slot 2 — Cottage Cheese 5% (2 facings)", isChecked: true)
                ChecklistRow(title: "Slot 3 — Sliced Cheddar (2 facings)", isChecked: true)
                ChecklistRow(title: "Slot 4 — Organic Butter 82% (2 facings)", isChecked: false)
            }

            Section("Shelf 4 (Floor / Bulk)") {
                ChecklistRow(title: "Slot 1 — UHT Milk 3.2% 12-Pack (4 facings)", isChecked: true)
                ChecklistRow(title: "Slot 2 — Mineral Water 6-Pack (4 facings)", isChecked: true)
            }
        }
    }

    // MARK: - Tab 3: Camera Vision Compliance

    private var cameraVisionTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                LabCard {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Image(systemName: "camera.viewfinder")
                                .font(.system(size: 26))
                                .foregroundStyle(AppTheme.accent)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Shelf Vision AI Auditor")
                                    .font(.headline)
                                Text("Capture aisle photo to run YOLO dense detector & embedding matching against published planogram.")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                        }

                        Button(action: { runVisionAudit() }) {
                            HStack {
                                if isRunningVision {
                                    ProgressView()
                                        .tint(.white)
                                        .padding(.trailing, 4)
                                    Text("Analyzing Shelf Image...")
                                } else {
                                    Image(systemName: "camera.fill")
                                    Text("Capture Shelf Photo & Audit")
                                }
                            }
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 12)
                            .background(AppTheme.accent, in: RoundedRectangle(cornerRadius: AppTheme.radiusMD))
                            .foregroundStyle(.white)
                        }
                        .disabled(isRunningVision)
                    }
                    .padding(AppTheme.spacingLG)
                }

                Text("Vision Audit Findings (Review Required)")
                    .font(.system(.headline, design: .rounded, weight: .bold))
                    .padding(.horizontal, AppTheme.spacingLG)

                ForEach($findings) { $finding in
                    LabCard {
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                Image(systemName: finding.type == "GAP" ? "exclamationmark.triangle.fill" : (finding.type == "WRONG_SKU" ? "arrow.triangle.2.circlepath" : "questionmark.circle.fill"))
                                    .foregroundStyle(finding.type == "GAP" ? Color.red : Color.orange)
                                Text("\(finding.type) — Shelf \(finding.shelfIndex), Slot \(finding.colIndex)")
                                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                                Spacer()
                                Text("\(Int(finding.confidence * 100))% conf")
                                    .font(.system(.caption2, design: .monospaced))
                                    .foregroundStyle(AppTheme.textSecondary)
                            }

                            Text("Expected: \(finding.expectedSku)")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textPrimary)

                            if let detected = finding.detectedSku {
                                Text("Detected: \(detected)")
                                    .font(.caption)
                                    .foregroundStyle(Color.red)
                            }

                            HStack(spacing: 12) {
                                Button("Accept") {
                                    finding.status = "ACCEPTED"
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(finding.status != "PENDING_REVIEW")

                                Button("Dismiss") {
                                    finding.status = "DISMISSED"
                                }
                                .buttonStyle(.bordered)
                                .disabled(finding.status != "PENDING_REVIEW")

                                Spacer()

                                Text(finding.status)
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundStyle(finding.status == "ACCEPTED" ? Color.green : (finding.status == "DISMISSED" ? Color.gray : Color.orange))
                            }

                            // Advisory link to Store Stock count task
                            if finding.status == "ACCEPTED" && (finding.type == "GAP" || finding.type == "EMPTY") {
                                Divider()
                                NavigationLink(destination: StoreStockView()) {
                                    HStack {
                                        Text("Open Store Stock Count Task")
                                            .font(.system(.caption, design: .rounded, weight: .semibold))
                                        Image(systemName: "arrow.right")
                                            .font(.caption2)
                                    }
                                    .foregroundStyle(AppTheme.accent)
                                }
                            }
                        }
                        .padding(AppTheme.spacingMD)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }
            }
            .padding(.vertical, AppTheme.spacingMD)
        }
    }

    private func shelfDescription(for row: Int) -> String {
        switch row {
        case 1: return "Top / Beverages"
        case 2: return "Eye Level / Milks"
        case 3: return "Yogurt & Butter"
        case 4: return "Bulk / Floor"
        default: return "Row \(row)"
        }
    }

    private func slotSkuLabel(row: Int, col: Int) -> String {
        switch (row, col) {
        case (1, 1): return "Mineral Water 1.5L"
        case (1, 2): return "Sparkling Water 1L"
        case (1, 3): return "Cola Classic 1.5L"
        case (1, 4): return "Lemon Tea 1L"
        case (2, 1): return "Whole Milk 3.2%"
        case (2, 2): return "Kefir 1% 500g"
        case (2, 3): return "Greek Yogurt 5%"
        case (2, 4): return "Sour Cream 15%"
        case (3, 1): return "Berry Yogurt"
        case (3, 2): return "Cottage Cheese"
        case (3, 3): return "Sliced Cheddar"
        case (3, 4): return "Butter 82%"
        case (4, 1): return "UHT Milk 12pk"
        case (4, 2): return "Water 6pk"
        case (4, 3): return "Juice 6pk"
        case (4, 4): return "Soda 12pk"
        default: return "SKU-\(row)\(col)"
        }
    }

    private func runVisionAudit() {
        isRunningVision = true
        banner = nil
        Task {
            try? await Task.sleep(nanoseconds: 1_200_000_000)
            isRunningVision = false
            banner = "Vision inference completed. 3 findings mapped to Bay A."
        }
    }
}

private struct ChecklistRow: View {
    let title: String
    @State var isChecked: Bool

    var body: some View {
        Toggle(isOn: $isChecked) {
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(.subheadline, design: .rounded))
                Text(isChecked ? "Compliant" : "Gap Flagged")
                    .font(.caption2)
                    .foregroundStyle(isChecked ? Color.green : Color.red)
            }
        }
    }
}
