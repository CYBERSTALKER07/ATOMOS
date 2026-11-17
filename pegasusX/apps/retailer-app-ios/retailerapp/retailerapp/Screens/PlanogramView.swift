//
//  PlanogramView.swift
//  retailerapp
//
//  Retail OS Pack 8 Planogram & Shelf Vision Screen.
//  Provides shelf slotting schema visualization, aisle walk checklist, and Vision AI compliance auditing.
//

import SwiftUI

struct PlanogramView: View {
    @State private var vm = PlanogramViewModel()

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            Picker("Mode", selection: $vm.selectedTab) {
                Text("Slotting").tag(0)
                Text("Checklist").tag(1)
                Text("Camera Vision").tag(2)
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingMD)
            .onChange(of: vm.selectedTab) { _, _ in Haptics.selection() }

            if let banner = vm.banner {
                Text(banner)
                    .font(.caption)
                    .foregroundStyle(AppTheme.accent)
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.bottom, AppTheme.spacingSM)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel(banner)
            }

            if vm.selectedTab == 0 {
                slottingGridTab
            } else if vm.selectedTab == 1 {
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
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
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
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Aisle 1 Dairy and Beverage Bay A, published, 4 shelves, 16 active facings")
                }

                ForEach(1...4, id: \.self) { shelfRow in
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("Shelf \(shelfRow) (\(vm.shelfDescription(for: shelfRow)))")
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textSecondary)
                            Spacer()
                            Text("Row \(shelfRow)")
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundStyle(AppTheme.textTertiary)
                        }

                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 10) {
                                ForEach(1...4, id: \.self) { slotCol in
                                    let skuLabel = vm.slotSkuLabel(row: shelfRow, col: slotCol)
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text("Slot \(slotCol)")
                                            .font(.caption2.weight(.bold))
                                            .foregroundStyle(AppTheme.accent)
                                        Text(skuLabel)
                                            .font(.system(.caption, design: .rounded, weight: .semibold))
                                            .lineLimit(2)
                                        Spacer()
                                        HStack {
                                            Text("2 facings")
                                                .font(.caption2)
                                                .foregroundStyle(AppTheme.textTertiary)
                                            Spacer()
                                            Circle()
                                                .fill(Color.green)
                                                .frame(width: 5, height: 5)
                                        }
                                    }
                                    .padding(8)
                                    .frame(width: 112, height: 88)
                                    .background(AppTheme.cardBackground)
                                    .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous))
                                    .overlay(
                                        RoundedRectangle(cornerRadius: AppTheme.radiusSM, style: .continuous)
                                            .stroke(AppTheme.separator, lineWidth: 0.5)
                                    )
                                    .accessibilityElement(children: .combine)
                                    .accessibilityLabel("Shelf \(shelfRow) Slot \(slotCol), \(skuLabel), 2 facings")
                                }
                            }
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }
            }
            .padding(.vertical, AppTheme.spacingMD)
        }
        .scrollDismissesKeyboard(.interactively)
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
        .scrollDismissesKeyboard(.interactively)
    }

    // MARK: - Tab 3: Camera Vision Compliance

    private var cameraVisionTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                LabCard {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Image(systemName: "camera.viewfinder")
                                .font(.title2)
                                .foregroundStyle(AppTheme.accent)
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Shelf Vision AI Auditor")
                                    .font(.headline)
                                Text("Capture aisle photo to run YOLO dense detector & embedding matching against published planogram.")
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                        }
                        .accessibilityElement(children: .combine)

                        Button {
                            Task { await vm.runVisionAudit() }
                        } label: {
                            HStack {
                                if vm.isRunningVision {
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
                            .background(AppTheme.accent, in: RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous))
                            .foregroundStyle(.white)
                        }
                        .pegasusMinimumHitTarget()
                        .disabled(vm.isRunningVision)
                        .accessibilityLabel("Capture shelf photo and audit planogram")
                    }
                    .padding(AppTheme.spacingLG)
                }

                Text("Vision Audit Findings (Review Required)")
                    .font(.system(.headline, design: .rounded, weight: .bold))
                    .padding(.horizontal, AppTheme.spacingLG)

                ForEach(vm.findings) { finding in
                    LabCard {
                        VStack(alignment: .leading, spacing: 10) {
                            VStack(alignment: .leading, spacing: 6) {
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
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(finding.type) on shelf \(finding.shelfIndex) slot \(finding.colIndex), expected \(finding.expectedSku), status \(finding.status)")

                            HStack(spacing: 12) {
                                Button("Accept") {
                                    vm.acceptFinding(id: finding.id)
                                }
                                .buttonStyle(.borderedProminent)
                                .pegasusMinimumHitTarget()
                                .disabled(finding.status != "PENDING_REVIEW")
                                .accessibilityLabel("Accept finding for \(finding.expectedSku)")

                                Button("Dismiss") {
                                    vm.dismissFinding(id: finding.id)
                                }
                                .buttonStyle(.bordered)
                                .pegasusMinimumHitTarget()
                                .disabled(finding.status != "PENDING_REVIEW")
                                .accessibilityLabel("Dismiss finding for \(finding.expectedSku)")

                                Spacer()

                                Text(finding.status)
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(finding.status == "ACCEPTED" ? Color.green : (finding.status == "DISMISSED" ? Color.gray : Color.orange))
                            }

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
                                    .pegasusMinimumHitTarget()
                                }
                                .accessibilityLabel("Open Store Stock Count Task")
                            }
                        }
                        .padding(AppTheme.spacingMD)
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                }
            }
            .padding(.vertical, AppTheme.spacingMD)
        }
        .scrollDismissesKeyboard(.interactively)
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
        .pegasusMinimumHitTarget()
        .onChange(of: isChecked) { _, _ in Haptics.selection() }
        .accessibilityLabel("\(title), \(isChecked ? "compliant" : "gap flagged")")
    }
}
