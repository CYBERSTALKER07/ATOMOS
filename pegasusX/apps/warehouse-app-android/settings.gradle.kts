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

rootProject.name = "PegasusWarehouse"
include(":app")
include(":barcode-scanner")
include(":mobile-design")
include(":mobile-kit")
project(":barcode-scanner").projectDir = file("../../packages/mobile-android-barcode-scanner")
project(":mobile-design").projectDir = file("../../packages/mobile-android-design")
project(":mobile-kit").projectDir = file("../../packages/mobile-android-kit")
