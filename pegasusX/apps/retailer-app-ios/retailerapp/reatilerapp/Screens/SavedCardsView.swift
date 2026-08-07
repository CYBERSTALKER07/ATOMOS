import SwiftUI

struct SavedCardsView: View {
    var returnTo: String? = nil
    var orderId: String? = nil
    var sessionId: String? = nil
    var onReturnToPayment: (() -> Void)? = nil

    private var apiClient = APIClient.shared
    @State private var cards: [RetailerCardToken] = []
    @State private var isLoading = true
    @State private var errorMessage: String? = nil
    
    // Webview / OTP states for GlobalPay would theoretically go here, but for simplicity we mock the initiate/confirm flow UI
    @State private var showingAddCard = false
    @State private var isAddingCard = false
    @State private var optCode = ""
    @State private var pendingCardToken: String? = nil

    init(
        returnTo: String? = nil,
        orderId: String? = nil,
        sessionId: String? = nil,
        onReturnToPayment: (() -> Void)? = nil
    ) {
        self.returnTo = returnTo
        self.orderId = orderId
        self.sessionId = sessionId
        self.onReturnToPayment = onReturnToPayment
    }
    
    var body: some View {
        ResponsiveGridContentWrapper {
            if returnTo == "delivery_payment" {
                Section {
                    HStack {
                        Text("mobile_retailer.ui.add_a_card_then_return_to_complete_delivery_payment")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        Spacer()
                        Button("mobile_retailer.ui.return") {
                            onReturnToPayment?()
                        }
                        .font(.footnote.weight(.semibold))
                    }
                }
            }

            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, alignment: .center)
            } else if let error = errorMessage {
                Text(error)
                    .foregroundColor(.red)
            } else if cards.isEmpty {
                Text("mobile_retailer.ui.no_saved_cards_add_one_to_checkout_faster")
                    .foregroundColor(.gray)
            } else {
                ForEach(cards) { card in
                    cardRow(card)
                }
            }
        }
        .navigationTitle("retailer_desktop.settings.cards.text.saved_cards")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { showingAddCard = true }) {
                    Image(systemName: "plus")
                }
            }
        }
        .task {
            await loadCards()
        }
        .sheet(isPresented: $showingAddCard) {
            NavigationView {
                Form {
                    if let token = pendingCardToken {
                        Section("Confirm OTP") {
                            TextField("mobile_retailer.ui.otp_code", text: $optCode)
                                .keyboardType(.numberPad)
                            
                            Button("mobile_retailer.ui.confirm") {
                                Task {
                                    await confirmCard(token: token)
                                }
                            }
                            .disabled(optCode.isEmpty || isAddingCard)
                        }
                    } else {
                        Section("Add New Card") {
                            Text("mobile_retailer.ui.this_will_initiate_tokenization_securely_via_globalpay")
                                .font(.caption)
                                .foregroundColor(.gray)
                            
                            Button("mobile_retailer.ui.start_tokenization") {
                                Task {
                                    await initiateSave()
                                }
                            }
                            .disabled(isAddingCard)
                        }
                    }
                }
                .navigationTitle("mobile_retailer.ui.add_card")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") {
                            showingAddCard = false
                            pendingCardToken = nil
                            optCode = ""
                        }
                    }
                }
                .overlay {
                    if isAddingCard {
                        ProgressView()
                    }
                }
            }
        }
    }
    
    private func cardRow(_ card: RetailerCardToken) -> some View {
        HStack {
            Image(systemName: "creditcard.fill")
                .foregroundColor(card.isDefault ? .blue : .primary)
            
            VStack(alignment: .leading) {
                Text(L10n.format("mobile_retailer.ui.cardtype_cardlast4", "\(card.cardType)", "\(card.cardLast4)"))
                    .font(.headline)
                if card.isDefault {
                    Text("mobile_retailer.ui.default")
                        .font(.caption)
                        .foregroundColor(.blue)
                }
            }
            Spacer()
            
            Menu {
                if !card.isDefault {
                    Button("mobile_retailer.ui.set_default") {
                        Task { await setDefault(card.tokenId) }
                    }
                }
                Button("supplier_portal.demand.payday_calendar.text.remove", role: .destructive) {
                    Task { await deactivate(card.tokenId) }
                }
            } label: {
                Image(systemName: "ellipsis.circle")
            }
        }
    }
    
    private func loadCards() async {
        isLoading = true
        errorMessage = nil
        do {
            let fetched = try await APIClient.shared.getCards()
            cards = fetched.filter { $0.isActive }
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Saved cards access is restricted for this account.",
                offline: "Offline mode active. Showing latest saved cards.",
                fallback: "Saved cards could not be loaded. Please try again.",
            )
        }
        isLoading = false
    }
    
    private func initiateSave() async {
        isAddingCard = true
        errorMessage = nil
        do {
            let response = try await APIClient.shared.initiateCardSave()
            pendingCardToken = response.cardToken
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Card setup access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry card setup.",
                fallback: "Card setup could not be started. Please try again.",
            )
        }
        isAddingCard = false
    }
    
    private func confirmCard(token: String) async {
        isAddingCard = true
        errorMessage = nil
        do {
            let _ = try await APIClient.shared.confirmCardSave(cardToken: token, otpCode: optCode)
            showingAddCard = false
            pendingCardToken = nil
            optCode = ""
            await loadCards()
            if returnTo == "delivery_payment" {
                onReturnToPayment?()
            }
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Card confirmation access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry card confirmation.",
                fallback: "Card confirmation failed. Please verify OTP and try again.",
            )
        }
        isAddingCard = false
    }
    
    private func setDefault(_ tokenId: String) async {
        do {
            try await APIClient.shared.setDefaultCard(tokenId: tokenId)
            errorMessage = nil
            await loadCards()
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Default card update is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry default card update.",
                fallback: "Default card could not be updated. Please try again.",
            )
        }
    }
    
    private func deactivate(_ tokenId: String) async {
        do {
            try await APIClient.shared.deactivateCard(tokenId: tokenId)
            errorMessage = nil
            await loadCards()
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Card removal is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry card removal.",
                fallback: "Card could not be removed. Please try again.",
            )
        }
    }
}
