//
//  EANBarcode.swift
//  Shared EAN/GTIN normalization for catalog and return-gate flows.
//

import Foundation

enum EANBarcode {
    /// Strips non-digits and validates GTIN length + checksum (EAN-8/12/13/14).
    static func normalize(_ raw: String) -> String? {
        let digits = raw.filter(\.isNumber)
        guard !digits.isEmpty else { return nil }
        guard [8, 12, 13, 14].contains(digits.count) else { return nil }
        guard validGtinChecksum(digits) else { return nil }
        return digits
    }

    private static func validGtinChecksum(_ code: String) -> Bool {
        let chars = Array(code)
        guard chars.count >= 8 else { return false }
        var sum = 0
        let n = chars.count
        for i in 0..<(n - 1) {
            guard let d = chars[i].wholeNumberValue else { return false }
            let posFromRight = n - 1 - i
            sum += (posFromRight % 2 == 1) ? d * 3 : d
        }
        guard let check = chars[n - 1].wholeNumberValue else { return false }
        return (10 - (sum % 10)) % 10 == check
    }
}
