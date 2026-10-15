// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "EnterpriseAppApple",
    platforms: [
        .iOS(.v17),
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "EnterpriseAppApple",
            targets: ["EnterpriseAppApple"]
        )
    ],
    dependencies: [],
    targets: [
        .target(
            name: "EnterpriseAppApple",
            dependencies: [],
            path: "Sources",
            resources: [
                .process("Resources")
            ]
        )
    ]
)
