import SwiftUI

struct SetupView: View {
    @Environment(AuthManager.self) private var auth
    @State private var vm = OnboardingViewModel()

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: AppTheme.spacingXL) {
                    header
                    progressBar

                    if vm.step == .tax {
                        taxStep
                    } else {
                        addressStep
                    }

                    if let error = vm.errorMessage {
                        Text(error)
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.destructive)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal)
                    }
                }
                .padding(AppTheme.spacingXL)
            }
            .background(AppTheme.background)
            .navigationTitle("Company Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Sign Out") { auth.logout() }
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
            }
        }
    }

    private var header: some View {
        VStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                    .fill(AppTheme.accent)
                    .frame(width: 64, height: 64)
                Image(systemName: "building.2.fill")
                    .font(.system(size: 28, weight: .semibold))
                    .foregroundStyle(.white)
            }
            Text(vm.step == .tax ? "Provide your business identity" : "Where should we deliver?")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .multilineTextAlignment(.center)
        }
    }

    private var progressBar: some View {
        HStack(spacing: 8) {
            Capsule().fill(AppTheme.accent).frame(height: 4)
            Capsule().fill(vm.step == .address ? AppTheme.accent : AppTheme.separator.opacity(0.4)).frame(height: 4)
        }
    }

    private var taxStep: some View {
        VStack(spacing: AppTheme.spacingLG) {
            setupField(title: "Tax ID / VAT", text: $vm.taxId, icon: "doc.text")
            Button {
                _ = vm.advanceFromTax()
            } label: {
                Text("Continue")
                    .font(.system(.headline, design: .rounded, weight: .bold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(AppTheme.accent)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
            }
        }
    }

    private var addressStep: some View {
        VStack(spacing: AppTheme.spacingLG) {
            setupField(title: "Billing Address", text: $vm.billingAddress, icon: "building")
            setupField(title: "Shipping Address", text: $vm.shippingAddress, icon: "shippingbox")
            setupField(title: "City", text: $vm.city, icon: "mappin.and.ellipse")
            setupField(title: "Postal Code", text: $vm.postalCode, icon: "number")

            HStack(spacing: 12) {
                Button {
                    vm.step = .tax
                } label: {
                    Text("Back")
                        .font(.system(.headline, design: .rounded, weight: .semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(AppTheme.surfaceElevated)
                        .foregroundStyle(AppTheme.textPrimary)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                }

                Button {
                    Task {
                        if await vm.submit() {
                            Haptics.success()
                        } else {
                            Haptics.error()
                        }
                    }
                } label: {
                    Group {
                        if vm.isSubmitting {
                            ProgressView().tint(.white)
                        } else {
                            Text("Complete Setup")
                                .font(.system(.headline, design: .rounded, weight: .bold))
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(AppTheme.accent)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                }
                .disabled(vm.isSubmitting)
            }
        }
    }

    private func setupField(title: String, text: Binding<String>, icon: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title.uppercased())
                .font(.system(.caption2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
            HStack(spacing: 10) {
                Image(systemName: icon)
                    .foregroundStyle(AppTheme.textTertiary)
                TextField(title, text: text)
                    .font(.system(.body, design: .rounded, weight: .semibold))
            }
            .padding(14)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .stroke(AppTheme.separator.opacity(0.3), lineWidth: 1)
            )
        }
    }
}

#Preview {
    SetupView()
        .environment(AuthManager.shared)
}
