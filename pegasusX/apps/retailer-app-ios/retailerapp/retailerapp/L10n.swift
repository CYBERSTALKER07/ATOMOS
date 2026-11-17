import Foundation
import Observation

enum RetailerLanguage: String, CaseIterable, Identifiable, Sendable {
    case uz
    case ru
    case en

    static let defaultLanguage: RetailerLanguage = .uz
    static let storageKey = "retailer_language"

    var id: String { rawValue }

    var nativeName: String {
        switch self {
        case .uz: "O'zbekcha"
        case .ru: "Русский"
        case .en: "English"
        }
    }

    var codeLabel: String {
        rawValue.uppercased()
    }

    var locale: Locale {
        Locale(identifier: rawValue)
    }

    static func resolve(_ raw: String?) -> RetailerLanguage {
        let normalized = (raw ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .replacingOccurrences(of: "_", with: "-")
        if normalized.hasPrefix("ru") { return .ru }
        if normalized.hasPrefix("en") { return .en }
        if normalized.hasPrefix("uz") { return .uz }
        return .defaultLanguage
    }
}

enum RetailerWorkspaceMode: String, CaseIterable, Identifiable, Sendable {
    case fast
    case full

    static let storageKey = "retailer_workspace_mode"

    var id: String { rawValue }
}

@Observable
@MainActor
final class RetailerPreferencesStore {
    static let shared = RetailerPreferencesStore()

    private let defaults: UserDefaults
    private let baseBundle: Bundle
    private var cachedBundles: [RetailerLanguage: Bundle] = [:]

    var currentLanguage: RetailerLanguage
    var workspaceModeOverride: RetailerWorkspaceMode?

    init(defaults: UserDefaults = .standard, bundle: Bundle = .main) {
        self.defaults = defaults
        self.baseBundle = bundle
        self.currentLanguage = RetailerLanguage.resolve(defaults.string(forKey: RetailerLanguage.storageKey))
        if let rawMode = defaults.string(forKey: RetailerWorkspaceMode.storageKey),
           let parsed = RetailerWorkspaceMode(rawValue: rawMode) {
            self.workspaceModeOverride = parsed
        } else {
            self.workspaceModeOverride = nil
        }
    }

    var activeBundle: Bundle {
        bundle(for: currentLanguage)
    }

    func setLanguage(_ language: RetailerLanguage) {
        currentLanguage = language
        defaults.set(language.rawValue, forKey: RetailerLanguage.storageKey)
        if defaults === UserDefaults.standard {
            defaults.set([language.rawValue], forKey: "AppleLanguages")
        }
    }

    func setWorkspaceMode(_ mode: RetailerWorkspaceMode?) {
        workspaceModeOverride = mode
        if let mode {
            defaults.set(mode.rawValue, forKey: RetailerWorkspaceMode.storageKey)
        } else {
            defaults.removeObject(forKey: RetailerWorkspaceMode.storageKey)
        }
    }

    func bundle(for language: RetailerLanguage) -> Bundle {
        if let cached = cachedBundles[language] {
            return cached
        }
        let searchBundles = [baseBundle] + Bundle.allBundles.filter { $0 != baseBundle }
        let codeCandidates = [language.rawValue, "\(language.rawValue)-Latn"]
        for container in searchBundles {
            for code in codeCandidates {
                if let path = container.path(forResource: code, ofType: "lproj"),
                   let localized = Bundle(path: path) {
                    cachedBundles[language] = localized
                    return localized
                }
            }
        }
        return baseBundle
    }

    func localizedString(_ key: String, language: RetailerLanguage? = nil) -> String {
        let targetLanguage = language ?? currentLanguage
        let missing = "__PEGASUS_MISSING_L10N__"
        let localizedBundle = bundle(for: targetLanguage)
        let candidate = localizedBundle.localizedString(forKey: key, value: missing, table: nil)
        if candidate != missing {
            return candidate
        }
        if targetLanguage != .uz {
            let uzFallback = bundle(for: .uz).localizedString(forKey: key, value: missing, table: nil)
            if uzFallback != missing {
                return uzFallback
            }
        }
        let mainCandidate = baseBundle.localizedString(forKey: key, value: missing, table: nil)
        return mainCandidate == missing ? key : mainCandidate
    }
}

/// Runtime localization formatter backed by `RetailerPreferencesStore` so language switches update immediately.
@MainActor
enum L10n {
    static func string(_ key: String) -> String {
        RetailerPreferencesStore.shared.localizedString(key)
    }

    static func format(_ key: String, _ args: CVarArg...) -> String {
        let format = RetailerPreferencesStore.shared.localizedString(key)
        return String(format: format, locale: RetailerPreferencesStore.shared.currentLanguage.locale, arguments: args)
    }
}
