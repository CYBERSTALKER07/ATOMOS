import Foundation
import Observation

struct DriverLanguageOption: Identifiable, Equatable, Sendable {
    let code: String
    let displayName: String

    var id: String { code }
}

@Observable
@MainActor
final class DriverLocaleManager {
    static let shared = DriverLocaleManager()

    nonisolated static let defaultLanguage = "uz"
    nonisolated static let storageKey = "driver_app_locale"

    nonisolated static let supportedLanguages: [DriverLanguageOption] = [
        DriverLanguageOption(code: "uz", displayName: "O'zbekcha"),
        DriverLanguageOption(code: "ru", displayName: "Русский"),
        DriverLanguageOption(code: "en", displayName: "English"),
    ]

    private let defaults: UserDefaults

    private(set) var currentCode: String

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        let saved = defaults.string(forKey: Self.storageKey)
        self.currentCode = Self.normalizeCode(saved)
        if saved == nil {
            defaults.set(self.currentCode, forKey: Self.storageKey)
            defaults.set([self.currentCode], forKey: "AppleLanguages")
        }
    }

    nonisolated static func normalizeCode(_ raw: String?) -> String {
        let cleaned = (raw ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        if let prefix = cleaned.split(separator: "-").first.map(String.init),
           supportedLanguages.contains(where: { $0.code == prefix }) {
            return prefix
        }
        return supportedLanguages.contains(where: { $0.code == cleaned }) ? cleaned : defaultLanguage
    }

    var currentDisplayName: String {
        Self.supportedLanguages.first(where: { $0.code == currentCode })?.displayName ?? "O'zbekcha"
    }

    var locale: Locale {
        Locale(identifier: currentCode)
    }

    func setLanguage(_ code: String) {
        let normalized = Self.normalizeCode(code)
        currentCode = normalized
        defaults.set(normalized, forKey: Self.storageKey)
        defaults.set([normalized], forKey: "AppleLanguages")
    }

    nonisolated static func currentLocalizedBundle(
        defaults: UserDefaults = .standard,
        overrideCode: String? = nil
    ) -> Bundle {
        let code = normalizeCode(overrideCode ?? defaults.string(forKey: storageKey))
        if let path = Bundle.main.path(forResource: code, ofType: "lproj"),
           let bundle = Bundle(path: path) {
            return bundle
        }
        return .main
    }
}

/// Localized format helper for dotted catalog keys in Localizable.strings.
/// Resolves strings from the active `DriverLocaleManager` `.lproj` bundle (`uz` default, `ru`, `en`).
enum L10n {
    static func string(_ key: String, localeCode: String? = nil) -> String {
        let bundle = DriverLocaleManager.currentLocalizedBundle(overrideCode: localeCode)
        let resolved = bundle.localizedString(forKey: key, value: nil, table: nil)
        if resolved != key {
            return resolved
        }
        if let enPath = Bundle.main.path(forResource: "en", ofType: "lproj"),
           let enBundle = Bundle(path: enPath) {
            return enBundle.localizedString(forKey: key, value: key, table: nil)
        }
        return resolved
    }

    static func format(_ key: String, _ args: CVarArg...) -> String {
        let format = string(key)
        let code = DriverLocaleManager.normalizeCode(
            UserDefaults.standard.string(forKey: DriverLocaleManager.storageKey)
        )
        return String(format: format, locale: Locale(identifier: code), arguments: args)
    }
}
