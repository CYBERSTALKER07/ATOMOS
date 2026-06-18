pluginManagement {
    repositories {
        google {
            content {
                includeGroupByRegex("com\\.android.*")
                includeGroupByRegex("com\\.google.*")
                includeGroupByRegex("androidx.*")
            }
        }
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "PegasusXSupplier"
include(":app")
include(":barcode-scanner")
include(":mobile-design")
project(":barcode-scanner").projectDir = file("../../packages/mobile-android-barcode-scanner")
project(":mobile-design").projectDir = file("../../packages/mobile-android-design")
