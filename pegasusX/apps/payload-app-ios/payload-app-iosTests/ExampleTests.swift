import XCTest
@testable import payload_app_ios

final class ExampleTests: XCTestCase {
    func testManifestDecodesTotalVolumeVuWireKey() throws {
        let json = """
        {
            "manifest_id": "mf-901",
            "truck_id": "trk-7",
            "state": "LOADING",
            "total_volume_vu": 24.5,
            "max_volume_vu": 40.0,
            "stop_count": 6,
            "region_code": "TAS-01"
        }
        """.data(using: .utf8)
        let data = try XCTUnwrap(json)
        let decoded = try JSONDecoder().decode(Manifest.self, from: data)
        XCTAssertEqual(decoded.manifestId, "mf-901")
        XCTAssertEqual(decoded.truckId, "trk-7")
        XCTAssertEqual(decoded.state, "LOADING")
        XCTAssertEqual(decoded.usedVolumeVu, 24.5)
        XCTAssertEqual(decoded.maxVolumeVu, 40.0)
        XCTAssertEqual(decoded.stopCount, 6)
        XCTAssertEqual(decoded.regionCode, "TAS-01")
        XCTAssertTrue(decoded.matchesTruck("trk-7"))
        XCTAssertFalse(decoded.matchesTruck("trk-other"))
    }

    func testHandoffPathResolverRoutesAndFallbacks() {
        XCTAssertEqual(HandoffPathResolver.resolve("/fleet"), .truckList)
        XCTAssertEqual(HandoffPathResolver.resolve("/manifests"), .truckList)
        XCTAssertEqual(HandoffPathResolver.resolve("/manifests/mf-42?tab=items"), .manifestDetail("mf-42"))
        XCTAssertEqual(HandoffPathResolver.resolve("https://portal.pegasusx.app/dispatch/mf-99"), .manifestDetail("mf-99"))
        XCTAssertEqual(HandoffPathResolver.resolve("/orders/ord-123"), .orderDetail("ord-123"))
        XCTAssertEqual(HandoffPathResolver.resolve("/orders"), .unresolved)
        XCTAssertEqual(HandoffPathResolver.resolve("   "), .unresolved)
        XCTAssertEqual(HandoffPathResolver.resolve("/unknown/route"), .unresolved)
    }
}
