// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "PegasusKit",
    platforms: [
        .iOS(.v17),
        .macOS(.v14),
    ],
    products: [
        .library(name: "PegasusKit", targets: ["PegasusKit"]),
    ],
    targets: [
        .target(
            name: "PegasusKit",
            path: "Sources/PegasusKit"
        ),
        .testTarget(
            name: "PegasusKitTests",
            dependencies: ["PegasusKit"],
            path: "Tests/PegasusKitTests"
        ),
    ]
)
