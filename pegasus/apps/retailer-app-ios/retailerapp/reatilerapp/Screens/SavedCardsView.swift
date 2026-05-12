import SwiftUI

struct SavedCardsView: View {
    @StateObject private var apiClient = APIClient.shared
    @State private var cards: [RetailerCardToken] = []
    @State private var isLoading = true
    @State private var errorMessage: String? = nil
    
    // Webview / OTP states for GlobalPay would theoretically go here, but for simplicity we mock the initiate/confirm flow UI
    @State private var showingAddCard = false
    @State private var isAddingCard = false
    @State private var optCode = ""
    @State private var pendingCardToken: String? = nil
    
    var body: some View {
        List {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, alignment: .center)
            } else if let error = errorMessage {
                Text(error)
                    .foregroundColor(.red)
            } else if cards.isEmpty {
                Text("No saved cards. Add one to checkout faster.")
                    .foregroundColor(.gray)
            } else {
                ForEach(cards) { card in
                    cardRow(card)
                }
            }
        }
        .navigationTitle("Saved Cards")
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
                            TextField("OTP Code", text: $optCode)
                                .keyboardType(.numberPad)
                            
                            Button("Confirm") {
                                Task {
                                    await confirmCard(token: token)
                                }
                            }
                            .disabled(optCode.isEmpty || isAddingCard)
                        }
                    } else {
                        Section("Add New Card") {
                            Text("This will initiate tokenization securely via GlobalPay.")
                                .font(.caption)
                                .foregroundColor(.gray)
                            
                            Button("Start Tokenization") {
                                Task {
                                    await initiateSave()
                                }
                            }
                            .disabled(isAddingCard)
                        }
                    }
                }
                .navigationTitle("Add Card")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") {
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
                Text("\(card.cardType) •••• \(card.cardLast4)")
                    .font(.headline)
                if card.isDefault {
                    Text("Default")
                        .font(.caption)
                        .foregroundColor(.blue)
                }
            }
            Spacer()
            
            Menu {
                if !card.isDefault {
                    Button("Set Default") {
                        Task { await setDefault(card.tokenId) }
                    }
                }
                Button("Remove", role: .destructive) {
                    Task { await deactivate(card.tokenId) }
                }
            } label: {
                Image(systemName: "ellipsis.circle")
            }
        }
    }
    
    private func loadCards() async {
        isLoading = true
        do {
            let fetched = try await APIClient.shared.getCards()
            cards = fetched.filter { $0.isActive }
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }
    
    private func initiateSave() async {
        isAddingCard = true
        do {
            let response = try await APIClient.shared.initiateCardSave()
            pendingCardToken = response.cardToken
        } catch {
            // Handle error logic
        }
        isAddingCard = false
    }
    
    private func confirmCard(token: String) async {
        isAddingCard = true
        do {
            let _ = try await APIClient.shared.confirmCardSave(cardToken: token, otpCode: optCode)
            showingAddCard = false
            pendingCardToken = nil
            optCode = ""
            await loadCards()
        } catch {
            // Error logic
        }
        isAddingCard = false
    }
    
    private func setDefault(_ tokenId: String) async {
        do {
            try await APIClient.shared.setDefaultCard(tokenId: tokenId)
            await loadCards()
        } catch {}
    }
    
    private func deactivate(_ tokenId: String) async {
        do {
            try await APIClient.shared.deactivateCard(tokenId: tokenId)
            await loadCards()
        } catch {}
    }
}
