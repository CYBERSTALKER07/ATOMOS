// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "PegasusMobileCore",
    platforms: [
        .iOS(.v16)
    ],
    products: [
        .library(name: "PegasusNetworking", targets: ["PegasusNetworking"]),
        .library(name: "PegasusUIKit", targets: ["PegasusUIKit"]),
        .library(name: "PegasusLiveActivities", targets: ["PegasusLiveActivities"])
    ],
    dependencies: [],
    targets: [
        .target(
            name: "PegasusNetworking",
            dependencies: []),
        .target(
            name: "PegasusUIKit",
            dependencies: []),
        .target(
            name: "PegasusLiveActivities",
            dependencies: ["PegasusUIKit"])
    ]
)
