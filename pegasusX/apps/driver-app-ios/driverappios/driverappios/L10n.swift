import Foundation

/// Localized format helper for dotted catalog keys in Localizable.strings.
/// Catalog values use positional `%1$@`-style specs generated from `{name}` placeholders.
enum L10n {
    static func format(_ key: String, _ args: CVarArg...) -> String {
        let format = String(localized: String.LocalizationValue(stringLiteral: key))
        return String(format: format, locale: Locale.current, arguments: args)
    }
}
