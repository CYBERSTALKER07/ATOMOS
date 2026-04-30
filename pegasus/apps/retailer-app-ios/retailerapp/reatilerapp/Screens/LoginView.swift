import SwiftUI
import CoreLocation
import MapKit

struct LoginView: View {
    @Environment(AuthManager.self) private var auth
    @State private var phone = "+998"
    @State private var password = ""
    @State private var isLoginMode = true

    // Registration fields
    @State private var storeName = ""
    @State private var ownerName = ""
    @State private var addressText = ""
    @State private var taxId = ""
    @State private var receivingWindowOpen = ""
    @State private var receivingWindowClose = ""
    @State private var accessType = ""           // STREET_PARKING | ALLEYWAY | LOADING_DOCK
    @State private var storageCeilingHeightText = ""

    // Location state
    @State private var latitude: Double = 0.0
    @State private var longitude: Double = 0.0
    @State private var locationLabel = ""
    @State private var showLocationPicker = false
    @State private var locating = false

    @State private var locationManager = LocationManager()
    @FocusState private var focusedField: Field?
    @State private var logoScale: Double = 0.5
    @State private var formOpacity: Double = 0

    enum Field { case phone, password, storeName, ownerName, address, taxId, windowOpen, windowClose, ceilingHeight }

    var body: some View {
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

                        // Logo
                        VStack(spacing: AppTheme.spacingMD) {
                            ZStack {
                                Circle()
                                    .fill(AppTheme.accentGradient)
                                    .frame(width: 80, height: 80)
                                    .shadow(color: AppTheme.accent.opacity(0.3), radius: 16, y: 8)

                                Image(systemName: "storefront.fill")
                                    .font(.system(size: 32, weight: .semibold))
                                    .foregroundStyle(.white)
                            }
                            .scaleEffect(logoScale)

                            Text("The Lab")
                                .font(.system(.largeTitle, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)

                            Text("Retailer Portal")
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        .padding(.bottom, AppTheme.spacingHuge)

                        // Form
                        VStack(spacing: AppTheme.spacingXL) {
                            formField(
                                label: "Phone Number",
                                icon: "phone",
                                text: Binding(
                                    get: { phone },
                                    set: { newValue in
                                        if newValue.hasPrefix("+998") || newValue == "+99" || newValue == "+9" || newValue == "+" {
                                            phone = newValue
                                        }
                                    }
                                ),
                                placeholder: "+998 XX XXX XX XX",
                                field: .phone,
                                isSecure: false,
                                keyboard: .phonePad
                            )

                            formField(
                                label: "Password",
                                icon: "lock",
                                text: $password,
                                placeholder: "Enter your password",
                                field: .password,
                                isSecure: true,
                                keyboard: .default
                            )

                            // ── Registration fields ──
                            if !isLoginMode {
                                formField(
                                    label: "Store Name",
                                    icon: "building.2",
                                    text: $storeName,
                                    placeholder: "Your store name",
                                    field: .storeName,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                formField(
                                    label: "Owner Name",
                                    icon: "person",
                                    text: $ownerName,
                                    placeholder: "Full name of owner",
                                    field: .ownerName,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                formField(
                                    label: "Store Address",
                                    icon: "mappin.and.ellipse",
                                    text: $addressText,
                                    placeholder: "Street address, city",
                                    field: .address,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                // ── Location picker buttons ──
                                HStack(spacing: AppTheme.spacingSM) {
                                    Button {
                                        showLocationPicker = true
                                    } label: {
                                        HStack(spacing: 6) {
                                            Image(systemName: "map")
                                                .font(.system(size: 14, weight: .medium))
                                            Text("Open Map")
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

                                    Button {
                                        if locationManager.authorizationStatus == .notDetermined {
                                            locationManager.requestPermission()
                                            return
                                        }
                                        locating = true
                                        locationManager.requestLocation()
                                        Task {
                                            // Wait briefly for location
                                            try? await Task.sleep(for: .seconds(2))
                                            if let loc = locationManager.lastLocation {
                                                latitude = loc.latitude
                                                longitude = loc.longitude
                                                locationLabel = String(format: "%.5f, %.5f", loc.latitude, loc.longitude)
                                            }
                                            locating = false
                                        }
                                    } label: {
                                        HStack(spacing: 6) {
                                            if locating {
                                                ProgressView()
                                                    .scaleEffect(0.7)
                                                    .tint(AppTheme.textPrimary)
                                            } else {
                                                Image(systemName: "location.fill")
                                                    .font(.system(size: 14, weight: .medium))
                                            }
                                            Text(locating ? "Locating..." : "Share Location")
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
                                    .disabled(locating)
                                }

                                if !locationLabel.isEmpty {
                                    Text("Location: \(locationLabel)")
                                        .font(.system(.caption2, design: .rounded))
                                        .foregroundStyle(AppTheme.textTertiary)
                                }

                                formField(
                                    label: "Tax ID (optional)",
                                    icon: "doc.text",
                                    text: $taxId,
                                    placeholder: "INN / Tax identification",
                                    field: .taxId,
                                    isSecure: false,
                                    keyboard: .default
                                )

                                // ── Logistics Details ──
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("Receiving Window")
                                        .font(.system(.caption, design: .rounded, weight: .medium))
                                        .foregroundStyle(AppTheme.textTertiary)
                                    HStack(spacing: 8) {
                                        formField(
                                            label: "Opens (HH:MM)",
                                            icon: "clock",
                                            text: $receivingWindowOpen,
                                            placeholder: "09:00",
                                            field: .windowOpen,
                                            isSecure: false,
                                            keyboard: .numbersAndPunctuation
                                        )
                                        formField(
                                            label: "Closes (HH:MM)",
                                            icon: "clock.badge.xmark",
                                            text: $receivingWindowClose,
                                            placeholder: "18:00",
                                            field: .windowClose,
                                            isSecure: false,
                                            keyboard: .numbersAndPunctuation
                                        )
                                    }
                                }

                                VStack(alignment: .leading, spacing: 8) {
                                    Text("Loading Access Type")
                                        .font(.system(.caption, design: .rounded, weight: .medium))
                                        .foregroundStyle(AppTheme.textTertiary)
                                    HStack(spacing: 6) {
                                        ForEach([("STREET_PARKING", "Street"), ("ALLEYWAY", "Alley"), ("LOADING_DOCK", "Dock")], id: \.0) { value, label in
                                            Button {
                                                accessType = accessType == value ? "" : value
                                            } label: {
                                                Text(label)
                                                    .font(.system(.caption, design: .rounded, weight: .semibold))
                                                    .padding(.horizontal, 12)
                                                    .padding(.vertical, 8)
                                                    .background(accessType == value ? AppTheme.accent : AppTheme.cardBackground)
                                                    .foregroundStyle(accessType == value ? Color.white : AppTheme.textPrimary)
                                                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                                                    .overlay {
                                                        RoundedRectangle(cornerRadius: AppTheme.radiusButton)
                                                            .strokeBorder(accessType == value ? AppTheme.accent : AppTheme.separator.opacity(0.5), lineWidth: 1)
                                                    }
                                            }
                                        }
                                    }
                                }

                                formField(
                                    label: "Storage Ceiling Height cm (optional)",
                                    icon: "arrow.up.to.line",
                                    text: $storageCeilingHeightText,
                                    placeholder: "e.g. 300",
                                    field: .ceilingHeight,
                                    isSecure: false,
                                    keyboard: .decimalPad
                                )
                            }
                        }
                        .padding(.horizontal, AppTheme.spacingXL)
                        .opacity(formOpacity)

                        // Error
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
                        }

                        // Primary action
                        Button {
                            focusedField = nil
                            Task {
                                if isLoginMode {
                                    await auth.login(phone: phone, password: password)
                                } else {
                                    await auth.register(
                                        phone: phone,
                                        password: password,
                                        storeName: storeName,
                                        ownerName: ownerName,
                                        addressText: addressText,
                                        latitude: latitude,
                                        longitude: longitude,
                                        taxId: taxId.isEmpty ? nil : taxId,
                                        receivingWindowOpen: receivingWindowOpen.isEmpty ? nil : receivingWindowOpen,
                                        receivingWindowClose: receivingWindowClose.isEmpty ? nil : receivingWindowClose,
                                        accessType: accessType.isEmpty ? nil : accessType,
                                        storageCeilingHeightCM: Double(storageCeilingHeightText)
                                    )
                                }
                            }
                        } label: {
                            HStack(spacing: AppTheme.spacingSM) {
                                if auth.isLoading {
                                    ProgressView()
                                        .tint(.white)
                                        .scaleEffect(0.8)
                                } else {
                                    Text(isLoginMode ? "Sign In" : "Create Account")
                                    Image(systemName: "arrow.right")
                                        .font(.system(size: 14, weight: .semibold))
                                }
                            }
                            .font(.system(.headline, design: .rounded))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 16)
                            .background(AppTheme.accentGradient)
                            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                            .shadow(color: AppTheme.accent.opacity(isFormValid ? 0.35 : 0), radius: 12, y: 6)
                        }
                        .disabled(!isFormValid || auth.isLoading)
                        .opacity(isFormValid ? 1 : 0.5)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.top, AppTheme.spacingXXL)
                        .opacity(formOpacity)

                        // Toggle mode
                        Button {
                            withAnimation(.easeInOut(duration: 0.3)) {
                                isLoginMode.toggle()
                            }
                            auth.errorMessage = nil
                        } label: {
                            Text(isLoginMode ? "Don't have an account? Sign Up" : "Already have an account? Sign In")
                                .font(.system(.subheadline, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        .padding(.top, AppTheme.spacingMD)

                        Spacer(minLength: AppTheme.spacingHuge)
                    }
                    .frame(minHeight: geo.size.height)
                }
                .scrollIndicators(.hidden)
            }
        }
        .onAppear {
            withAnimation(AnimationConstants.hero) {
                logoScale = 1.0
            }
            withAnimation(.easeOut(duration: 0.6).delay(0.3)) {
                formOpacity = 1.0
            }
            focusedField = .phone
        }
        .sheet(isPresented: $showLocationPicker) {
            LocationPickerView(
                initialLatitude: latitude != 0 ? latitude : 41.2995,
                initialLongitude: longitude != 0 ? longitude : 69.2401
            ) { lat, lng, display in
                latitude = lat
                longitude = lng
                locationLabel = display
            }
        }
    }

    // MARK: - Form Field

    private func formField(label: String, icon: String, text: Binding<String>, placeholder: String, field: Field, isSecure: Bool, keyboard: UIKeyboardType) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            Text(label)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textTertiary)
                .textCase(.uppercase)

            HStack(spacing: AppTheme.spacingMD) {
                Image(systemName: icon)
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(focusedField == field ? AppTheme.accent : AppTheme.textTertiary)
                    .frame(width: 20)

                if isSecure {
                    SecureField(placeholder, text: text)
                        .font(.system(.body, design: .rounded))
                        .focused($focusedField, equals: field)
                } else {
                    TextField(placeholder, text: text)
                        .font(.system(.body, design: .rounded))
                        .focused($focusedField, equals: field)
                        .keyboardType(keyboard)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
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

    private var isFormValid: Bool {
        guard phone.count >= 13, !password.isEmpty else { return false }
        if !isLoginMode {
            return !storeName.isEmpty && !ownerName.isEmpty && !addressText.isEmpty
        }
        return true
    }
}

#Preview {
    LoginView()
        .environment(AuthManager.shared)
}
