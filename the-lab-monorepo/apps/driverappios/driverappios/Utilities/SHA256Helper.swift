//
//  SHA256Helper.swift
//  driverappios
//

import CryptoKit
import Foundation

/// Returns the SHA-256 hex digest of a UTF-8 string.
func sha256Hex(_ input: String) -> String {
    let digest = SHA256.hash(data: Data(input.utf8))
    return digest.map { String(format: "%02x", $0) }.joined()
}
