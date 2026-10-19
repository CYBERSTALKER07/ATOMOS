// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "EnterpriseDesktopApp",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(
            name: "EnterpriseDesktopApp",
            targets: ["EnterpriseDesktopApp"]
        )
    ],
    dependencies: [],
    targets: [
        .executableTarget(
            name: "EnterpriseDesktopApp",
            dependencies: [],
            path: "Sources",
            resources: [
                .process("Resources")
            ]
        )
    ]
)
