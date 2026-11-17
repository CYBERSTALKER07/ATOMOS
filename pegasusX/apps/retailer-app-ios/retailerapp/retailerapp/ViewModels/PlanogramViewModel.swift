import Foundation
import Observation

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

@Observable
@MainActor
final class PlanogramViewModel {
    var selectedTab = 0
    var banner: String?
    var isBusy = false
    var isRunningVision = false
    var findings: [PlanogramFindingIOS] = [
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

    func shelfDescription(for row: Int) -> String {
        switch row {
        case 1: return "Top / Beverages"
        case 2: return "Eye Level / Milks"
        case 3: return "Yogurt & Butter"
        case 4: return "Bulk / Floor"
        default: return "Row \(row)"
        }
    }

    func slotSkuLabel(row: Int, col: Int) -> String {
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

    func runVisionAudit() async {
        guard !isRunningVision else { return }
        Haptics.light()
        isRunningVision = true
        banner = nil
        try? await Task.sleep(nanoseconds: 1_200_000_000)
        guard !Task.isCancelled else {
            isRunningVision = false
            return
        }
        isRunningVision = false
        banner = "Vision inference completed. \(findings.count) findings mapped to Bay A."
        Haptics.success()
    }

    func acceptFinding(id: String) {
        guard let idx = findings.firstIndex(where: { $0.id == id }) else { return }
        findings[idx].status = "ACCEPTED"
        Haptics.success()
    }

    func dismissFinding(id: String) {
        guard let idx = findings.firstIndex(where: { $0.id == id }) else { return }
        findings[idx].status = "DISMISSED"
        Haptics.selection()
    }
}
