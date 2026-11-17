import SwiftUI
import CoreLocation
import MapKit

struct LoginView: View {
    @Environment(AuthManager.self) private var auth
    @State private var vm = LoginViewModel()
    @FocusState private var focusedField: Field?

    enum Field: Hashable {
        case phone, password, storeName, ownerName, address, taxId, windowOpen, windowClose, ceilingHeight
    }

    var body: some View {
        @Bindable var vm = vm
        GeometryReader { geo in
            ZStack {
                AppTheme.meshBackground.ignoresSafeArea()

                Circle()
                    .fill(AppTheme.accent.opacity(0.06))
                    .frame(width: 300, height: 300)
                    .offset(x: -100, y: -geo.size.height * 0.35)
                Circle()
                    .fill(AppTheme.accentSoft.opacity(0.12))
                    .frame(width: 200, height: 200)
                    .offset(x: 120, y: geo.size.height * 0.3)

                ScrollView {
                    VStack(spacing: 0) {
                        Spacer(minLength: geo.size.height * 0.08)

                        VStack(spacing: AppTheme.spacingMD) {
                            ZStack {
                                Circle()
                                    .fill(AppTheme.accentGradient)
                                    .frame(width: 80, height: 80)
                                    .shadow(color: AppTheme.accent.opacity(0.3), radius: 16, y: 8)

                                Image(systemName: "storefront.fill")
                                    .font(.title.weight(.semibold))
                                    .foregroundStyle(.white)
                            }
                            .scaleEffect(vm.logoScale)

                            Text("auth.login.title")
                                .font(.system(.largeTitle, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)

                            Text("mobile_retailer.ui.retailer_portal")
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        .accessibilityElement(children: .combine)
                        .padding(.bottom, AppTheme.spacingHuge)

                        VStack(spacing: AppTheme.spacingXL) {
                            formField(
                                label: "Phone Number",
                                icon: "phone",
                                text: $vm.phoneInput,
                                placeholder: "+998 XX XXX XX XX",
                                field: .phone,
                                isSecure: false,
                                keyboard: .phonePad
                            )

                            formField(
                                label: "Password",
                                icon: "lock",
                                text: $vm.password,
                                placeholder: "Enter your password",
                                field: .password,
                                isSecure: true,
                                keyboard: .default
                            )

                            if !vm.isLoginMode {
                                formField(
                                    label: "Store Name",
                                    icon: "building.2",
                                    text: $vm.storeName,
                                    placeholder: "Your store name",
                                    field: .storeName,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                formField(
                                    label: "Owner Name",
                                    icon: "person",
                                    text: $vm.ownerName,
                                    placeholder: "Full name of owner",
                                    field: .ownerName,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                formField(
                                    label: "Store Address",
                                    icon: "mappin.and.ellipse",
                                    text: $vm.addressText,
                                    placeholder: "Street address, city",
                                    field: .address,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                HStack(spacing: AppTheme.spacingSM) {
                                    Button {
                                        Haptics.light()
                                        vm.showLocationPicker = true
                                    } label: {
                                        HStack(spacing: 6) {
                                            Image(systemName: "map")
                                                .font(.subheadline.weight(.medium))
                                            Text("mobile_retailer.ui.open_map")
                                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                        }
                                        .foregroundStyle(AppTheme.textPrimary)
                                        .frame(maxWidth: .infinity)
                                        .padding(.vertical, 12)
                                        .background(AppTheme.cardBackground)
                                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                                        .overlay {
                                            RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                                                .strokeBorder(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                        }
                                    }
                                    .pegasusMinimumHitTarget()
                                    .accessibilityLabel("Open map location picker")

                                    Button {
                                        Task { await vm.shareCurrentLocation() }
                                    } label: {
                                        HStack(spacing: 6) {
                                            if vm.locating {
                                                ProgressView()
                                                    .scaleEffect(0.7)
                                                    .tint(AppTheme.textPrimary)
                                            } else {
                                                Image(systemName: "location.fill")
                                                    .font(.subheadline.weight(.medium))
                                            }
                                            Text(vm.locating ? "Locating..." : "Share Location")
                                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                        }
                                        .foregroundStyle(AppTheme.textPrimary)
                                        .frame(maxWidth: .infinity)
                                        .padding(.vertical, 12)
                                        .background(AppTheme.cardBackground)
                                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                                        .overlay {
                                            RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                                                .strokeBorder(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                        }
                                    }
                                    .pegasusMinimumHitTarget()
                                    .disabled(vm.locating)
                                    .accessibilityLabel("Share current GPS location")
                                }

                                if !vm.locationLabel.isEmpty {
                                    Text(L10n.format("mobile_retailer.ui.location_locationlabel_2", "\(vm.locationLabel)"))
                                        .font(.system(.caption2, design: .rounded))
                                        .foregroundStyle(AppTheme.textTertiary)
                                        .accessibilityElement(children: .combine)
                                        .accessibilityLabel("Selected store location: \(vm.locationLabel)")
                                }

                                formField(
                                    label: "Tax ID (optional)",
                                    icon: "doc.text",
                                    text: $vm.taxId,
                                    placeholder: "INN / Tax identification",
                                    field: .taxId,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                VStack(alignment: .leading, spacing: 8) {
                                    Text("mobile_retailer.ui.receiving_window_2")
                                        .font(.system(.caption, design: .rounded, weight: .medium))
                                        .foregroundStyle(AppTheme.textTertiary)
                                    HStack(spacing: 8) {
                                        formField(
                                            label: "Opens (HH:MM)",
                                            icon: "clock",
                                            text: $vm.receivingWindowOpen,
                                            placeholder: "09:00",
                                            field: .windowOpen,
                                            isSecure: false,
                                            keyboard: .numbersAndPunctuation
                                        )
                                        formField(
                                            label: "Closes (HH:MM)",
                                            icon: "clock.badge.xmark",
                                            text: $vm.receivingWindowClose,
                                            placeholder: "18:00",
                                            field: .windowClose,
                                            isSecure: false,
                                            keyboard: .numbersAndPunctuation
                                        )
                                    }
                                }

                                VStack(alignment: .leading, spacing: 8) {
                                    Text("mobile_retailer.ui.loading_access_type")
                                        .font(.system(.caption, design: .rounded, weight: .medium))
                                        .foregroundStyle(AppTheme.textTertiary)
                                    HStack(spacing: 6) {
                                        ForEach([("STREET_PARKING", "Street"), ("ALLEYWAY", "Alley"), ("LOADING_DOCK", "Dock")], id: \.0) { value, label in
                                            Button {
                                                vm.toggleAccessType(value)
                                            } label: {
                                                Text(label)
                                                    .font(.system(.caption, design: .rounded, weight: .semibold))
                                                    .padding(.horizontal, 12)
                                                    .padding(.vertical, 10)
                                                    .background(vm.accessType == value ? AppTheme.accent : AppTheme.cardBackground)
                                                    .foregroundStyle(vm.accessType == value ? Color.white : AppTheme.textPrimary)
                                                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                                                    .overlay {
                                                        RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                                                            .strokeBorder(vm.accessType == value ? AppTheme.accent : AppTheme.separator.opacity(0.5), lineWidth: 1)
                                                    }
                                            }
                                            .pegasusMinimumHitTarget()
                                            .accessibilityLabel("Loading access \(label)\(vm.accessType == value ? ", selected" : "")")
                                        }
                                    }
                                }

                                formField(
                                    label: "Storage Ceiling Height cm (optional)",
                                    icon: "arrow.up.to.line",
                                    text: $vm.storageCeilingHeightText,
                                    placeholder: "e.g. 300",
                                    field: .ceilingHeight,
                                    isSecure: false,
                                    keyboard: .decimalPad
                                )
                            }
                        }
                        .padding(.horizontal, AppTheme.spacingXL)
                        .opacity(vm.formOpacity)

                        if let error = auth.errorMessage {
                            HStack(spacing: AppTheme.spacingSM) {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .font(.caption)
                                Text(error)
                                    .font(.system(.caption, design: .rounded))
                            }
                            .foregroundStyle(AppTheme.destructive)
                            .padding(.top, AppTheme.spacingMD)
                            .transition(.move(edge: .top).combined(with: .opacity))
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Error: \(error)")
                        }

                        Button {
                            focusedField = nil
                            Task { await vm.submit(auth: auth) }
                        } label: {
                            HStack(spacing: AppTheme.spacingSM) {
                                if auth.isLoading {
                                    ProgressView()
                                        .tint(.white)
                                        .scaleEffect(0.8)
                                } else {
                                    Text(vm.isLoginMode ? "Sign In" : "Create Account")
                                    Image(systemName: "arrow.right")
                                        .font(.subheadline.weight(.semibold))
                                }
                            }
                            .font(.system(.headline, design: .rounded))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 16)
                            .background(AppTheme.accentGradient)
                            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                            .shadow(color: AppTheme.accent.opacity(vm.isFormValid ? 0.35 : 0), radius: 12, y: 6)
                        }
                        .pegasusMinimumHitTarget()
                        .disabled(!vm.isFormValid || auth.isLoading)
                        .opacity(vm.isFormValid ? 1 : 0.5)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.top, AppTheme.spacingXXL)
                        .opacity(vm.formOpacity)
                        .accessibilityLabel(vm.isLoginMode ? "Sign In" : "Create Account")

                        Button {
                            Haptics.selection()
                            withAnimation(.easeInOut(duration: 0.3)) {
                                vm.isLoginMode.toggle()
                            }
                            auth.errorMessage = nil
                            focusedField = .phone
                        } label: {
                            Text(vm.isLoginMode ? "Don't have an account? Sign Up" : "Already have an account? Sign In")
                                .font(.system(.subheadline, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                                .pegasusMinimumHitTarget()
                        }
                        .padding(.top, AppTheme.spacingMD)
                        .accessibilityLabel(vm.isLoginMode ? "Switch to Sign Up" : "Switch to Sign In")

                        Spacer(minLength: AppTheme.spacingHuge)
                    }
                    .frame(minHeight: geo.size.height)
                }
                .scrollDismissesKeyboard(.interactively)
                .scrollIndicators(.hidden)
            }
        }
        .defaultFocus($focusedField, .phone)
        .onAppear {
            withAnimation(AnimationConstants.hero) {
                vm.logoScale = 1.0
            }
            withAnimation(.easeOut(duration: 0.6).delay(0.3)) {
                vm.formOpacity = 1.0
            }
            focusedField = .phone
        }
        .onSubmit {
            advanceFocusOrSubmit()
        }
        .sheet(isPresented: $vm.showLocationPicker) {
            LocationPickerView(
                initialLatitude: vm.latitude != 0 ? vm.latitude : packMapCoordinate().lat,
                initialLongitude: vm.longitude != 0 ? vm.longitude : packMapCoordinate().lng
            ) { lat, lng, display in
                vm.setLocation(lat: lat, lng: lng, label: display)
                focusedField = .taxId
            }
        }
    }

    private func advanceFocusOrSubmit() {
        switch focusedField {
        case .phone:
            focusedField = .password
        case .password:
            if vm.isLoginMode {
                if vm.isFormValid && !auth.isLoading {
                    focusedField = nil
                    Task { await vm.submit(auth: auth) }
                }
            } else {
                focusedField = .storeName
            }
        case .storeName:
            focusedField = .ownerName
        case .ownerName:
            focusedField = .address
        case .address:
            focusedField = .taxId
        case .taxId:
            focusedField = .windowOpen
        case .windowOpen:
            focusedField = .windowClose
        case .windowClose:
            focusedField = .ceilingHeight
        case .ceilingHeight, nil:
            focusedField = nil
        }
    }

    private func formField(
        label: String,
        icon: String,
        text: Binding<String>,
        placeholder: String,
        field: Field,
        isSecure: Bool,
        keyboard: UIKeyboardType
    ) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            Text(label)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
                .textCase(.uppercase)

            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: icon)
                    .font(.body.weight(.medium))
                    .foregroundStyle(focusedField == field ? AppTheme.accent : AppTheme.textTertiary)
                    .frame(width: 20)

                if isSecure {
                    SecureField(placeholder, text: text)
                        .font(.system(.body, design: .rounded))
                        .focused($focusedField, equals: field)
                        .accessibilityLabel(label)
                } else {
                    TextField(placeholder, text: text)
                        .font(.system(.body, design: .rounded))
                        .focused($focusedField, equals: field)
                        .keyboardType(keyboard)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .accessibilityLabel(label)
                }
            }
            .padding(AppTheme.spacingMD)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            .overlay {
                RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                    .strokeBorder(
                        focusedField == field ? AppTheme.accent : AppTheme.separator.opacity(0.5),
                        lineWidth: focusedField == field ? 2 : 1
                    )
            }
            .shadow(color: focusedField == field ? AppTheme.accent.opacity(0.1) : .clear, radius: 8, y: 4)
            .animation(AnimationConstants.express, value: focusedField)
        }
    }
}

#Preview {
    LoginView()
        .environment(AuthManager.shared)
}
