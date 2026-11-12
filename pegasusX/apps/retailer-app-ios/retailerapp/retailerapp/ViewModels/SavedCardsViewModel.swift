import Foundation
import Observation

@Observable
@MainActor
final class SavedCardsViewModel {
    var cards: [SavedCardItem] = [
        SavedCardItem(
            id: "card-1",
            panMasked: "8600 •••• •••• 0001",
            scheme: "UZCARD",
            holder: "ALISHER NAVOIY",
            expiry: "12/28",
            bankName: "O'zsanoatqurilishbank",
            isDefault: true
        ),
        SavedCardItem(
            id: "card-2",
            panMasked: "9860 •••• •••• 0002",
            scheme: "HUMO",
            holder: "ZULFIYA ISROILOVA",
            expiry: "08/27",
            bankName: "Ipak Yo'li Bank",
            isDefault: false
        )
    ]
    var showingAddCard = false
    var panInput = ""
    var expInput = ""
    var otpInput = ""
    var bindingStep = 0 // 0: input, 1: otp
    var statusMessage: String?

    func deleteCard(_ id: String) {
        cards.removeAll { $0.id == id }
        Haptics.selection()
    }

    func setDefaultCard(_ id: String) {
        for i in cards.indices {
            cards[i].isDefault = (cards[i].id == id)
        }
        Haptics.selection()
    }

    func requestOTP() {
        bindingStep = 1
        statusMessage = "SMS kod yuborildi: +998 90 *** ** 01"
        Haptics.light()
    }

    func autofillOTP() {
        otpInput = "123456"
        Haptics.selection()
    }

    func completeCardBinding() {
        let clean = panInput.replacingOccurrences(of: " ", with: "")
        let last4 = clean.count >= 4 ? String(clean.suffix(4)) : "0000"
        let scheme = clean.hasPrefix("9860") ? "HUMO" : "UZCARD"
        let newCard = SavedCardItem(
            id: "card-\(UUID().uuidString.prefix(6))",
            panMasked: "\(clean.prefix(4)) •••• •••• \(last4)",
            scheme: scheme,
            holder: "YANGI KARTA",
            expiry: expInput.isEmpty ? "12/28" : expInput,
            bankName: "O'zbekiston Banki",
            isDefault: cards.isEmpty
        )
        cards.append(newCard)
        resetAddCardState()
        showingAddCard = false
        Haptics.success()
    }

    func resetAddCardState() {
        panInput = ""
        expInput = ""
        otpInput = ""
        bindingStep = 0
        statusMessage = nil
    }
}
