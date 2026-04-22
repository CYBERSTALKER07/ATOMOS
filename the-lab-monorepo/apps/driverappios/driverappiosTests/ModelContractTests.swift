//
//  ModelContractTests.swift
//  driverappiosTests
//

import Testing
import Foundation
@testable import driverappios

// MARK: - RouteManifest Tests

struct RouteManifestTests {

    @Test func isValid_futureExpiry_returnsTrue() {
        let manifest = RouteManifest(
            driver_id: "DRV-001",
            date: "2026-04-12",
            expires_at: Date().timeIntervalSince1970 + 86400,
            hashes: [:]
        )
        #expect(manifest.isValid == true)
    }

    @Test func isValid_pastExpiry_returnsFalse() {
        let manifest = RouteManifest(
            driver_id: "DRV-001",
            date: "2026-04-12",
            expires_at: Date().timeIntervalSince1970 - 3600,
            hashes: [:]
        )
        #expect(manifest.isValid == false)
    }

    @Test func isValid_epochZero_returnsFalse() {
        let manifest = RouteManifest(
            driver_id: "DRV-001",
            date: "2026-04-12",
            expires_at: 0,
            hashes: [:]
        )
        #expect(manifest.isValid == false)
    }

    @Test func hashes_containsExpectedOrders() {
        let manifest = RouteManifest.mock
        #expect(manifest.hashes.count == 3)
        #expect(manifest.hashes["ORD-TASH-0056"] != nil)
        #expect(manifest.hashes["ORD-TASH-0057"] != nil)
        #expect(manifest.hashes["ORD-TASH-0058"] != nil)
    }
}

// MARK: - SHA256 Helper Tests

struct SHA256HelperTests {

    @Test func sha256Hex_emptyString() {
        let hash = sha256Hex("")
        // SHA-256 of empty string is well-known
        #expect(hash == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
    }

    @Test func sha256Hex_knownInput() {
        let hash = sha256Hex("hello")
        #expect(hash == "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
    }

    @Test func sha256Hex_deterministic() {
        let a = sha256Hex("test-token")
        let b = sha256Hex("test-token")
        #expect(a == b)
    }

    @Test func sha256Hex_differentInputs_differentOutputs() {
        let a = sha256Hex("token-a")
        let b = sha256Hex("token-b")
        #expect(a != b)
    }

    @Test func sha256Hex_length_is64() {
        let hash = sha256Hex("anything")
        #expect(hash.count == 64, "SHA-256 hex digest should be 64 characters")
    }

    @Test func sha256Hex_lowercaseHex() {
        let hash = sha256Hex("test")
        let isLowerHex = hash.allSatisfy { $0.isHexDigit && ($0.isNumber || $0.isLowercase) }
        #expect(isLowerHex == true, "Hash should be lowercase hex")
    }
}

// MARK: - QRPayload Tests

struct QRPayloadTests {

    @Test func qrPayload_fields() {
        let qr = QRPayload(order_id: "ORD-001", token: "tok_abc")
        #expect(qr.order_id == "ORD-001")
        #expect(qr.token == "tok_abc")
    }

    @Test func qrPayload_jsonRoundtrip() throws {
        let original = QRPayload(order_id: "ORD-RT", token: "tok_rt")
        let data = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(QRPayload.self, from: data)
        #expect(decoded.order_id == original.order_id)
        #expect(decoded.token == original.token)
    }
}

// MARK: - Mission Tests

struct MissionTests {

    @Test func mission_id_isOrderId() {
        let mission = Mission.mockMissions[0]
        #expect(mission.id == mission.order_id)
    }

    @Test func mission_mockCount() {
        #expect(Mission.mockMissions.count == 3)
    }

    @Test func mission_gateways_allValid() {
        let validGateways = Set(["CASH", "PAYME", "CLICK", "UZCARD"])
        for m in Mission.mockMissions {
            #expect(validGateways.contains(m.gateway), "Unexpected gateway: \(m.gateway)")
        }
    }
}

// MARK: - OrderLineItem Computed Props

struct OrderLineItemTests {

    @Test func lineTotal_multipliesQuantityByPrice() {
        let item = OrderLineItem(productId: "P1", productName: "Water", quantity: 10, unitPrice: 5000)
        #expect(item.lineTotal == 50000)
    }

    @Test func lineTotal_singleUnit() {
        let item = OrderLineItem(productId: "P2", productName: "Juice", quantity: 1, unitPrice: 12000)
        #expect(item.lineTotal == 12000)
    }

    @Test func lineTotal_zeroQuantity() {
        let item = OrderLineItem(productId: "P3", productName: "Milk", quantity: 0, unitPrice: 8000)
        #expect(item.lineTotal == 0)
    }

    @Test func id_isProductId() {
        let item = OrderLineItem(productId: "PROD-ABC", productName: "Test", quantity: 1, unitPrice: 1000)
        #expect(item.id == "PROD-ABC")
    }
}
