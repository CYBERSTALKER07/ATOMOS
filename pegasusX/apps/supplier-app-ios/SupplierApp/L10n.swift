import Foundation

/// Localized format helper for dotted catalog keys in Localizable.strings.
/// Catalog values use positional `%1$@`-style specs generated from `{name}` placeholders.
enum L10n {
    private static let bundle: Bundle = {
        let tokenBundle = Bundle(for: L10nBundleToken.self)
        if tokenBundle.path(forResource: "en", ofType: "lproj") != nil {
            return tokenBundle
        }
        if let host = Bundle.allBundles.first(where: { $0.path(forResource: "en", ofType: "lproj") != nil }) {
            return host
        }
        return Bundle.main
    }()

    private static let englishBundle: Bundle? = {
        guard let enPath = bundle.path(forResource: "en", ofType: "lproj") else { return nil }
        return Bundle(path: enPath)
    }()

    static func format(_ key: String, _ args: CVarArg...) -> String {
        var format = bundle.localizedString(forKey: key, value: nil, table: nil)
        if format == key, let fallback = englishBundle {
            format = fallback.localizedString(forKey: key, value: key, table: nil)
        }
        guard !args.isEmpty else { return format }
        return String(format: format, locale: Locale.current, arguments: args)
    }
}

private final class L10nBundleToken {}
