import java.util.Properties
import org.gradle.api.GradleException
import org.gradle.api.tasks.Exec

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("com.google.dagger.hilt.android")
    id("org.jetbrains.kotlin.plugin.serialization")
    id("com.google.devtools.ksp")
}

// Read dev.host from local.properties (falls back to emulator address)
val localProps = Properties().also { props ->
    val f = rootProject.file("local.properties")
    if (f.exists()) props.load(f.inputStream())
}
val devHost: String = localProps.getProperty("dev.host", "10.0.2.2")
val mapsApiKey: String = localProps.getProperty("MAPS_API_KEY", "")
val prodApiBaseUrl: String = localProps.getProperty("prod.api.base.url", "https://api.pegasus.uz")
val prodWsBaseUrl: String = localProps.getProperty("prod.ws.base.url", "wss://api.pegasus.uz")
val quicktypeBinary: String = localProps.getProperty("quicktype.path", "quicktype")

val contractsSchemaFile = rootProject.file("../../contracts/events.schema.json")
val backendGoDir = rootProject.file("../../apps/backend-go")
val generatedWsModelFile = rootProject.file(
    "app/src/main/java/com/pegasusx/retailer/generated/contracts/PegasusWSEventEnvelope.kt"
)
val wsCodegenEnabled = localProps.getProperty("retailer.ws.codegen", "false") == "true"

fun assertCommandAvailable(command: String) {
    val process = ProcessBuilder("sh", "-c", "command -v \"$command\" >/dev/null 2>&1").start()
    val exitCode = process.waitFor()
    if (exitCode != 0) {
        throw GradleException(
            "quicktype not found (binary: $command). Install quicktype or set quicktype.path in local.properties",
        )
    }
}

val generateEventSchema by tasks.registering(Exec::class) {
    group = "codegen"
    description = "Generate websocket JSON schema from backend Go contracts"
    workingDir = backendGoDir
    commandLine(
        "go",
        "run",
        "./cmd/gen-contracts",
        "-source",
        "events/events.go",
        "-mode",
        "json-schema",
        "-schema-out",
        contractsSchemaFile.absolutePath,
        "-pretty=true",
    )
    outputs.file(contractsSchemaFile)
}

val generateWsEventModels by tasks.registering(Exec::class) {
    group = "codegen"
    description = "Generate Kotlin websocket contract models from shared schema"
    dependsOn(generateEventSchema)

    inputs.file(contractsSchemaFile)
    outputs.file(generatedWsModelFile)

    doFirst {
        generatedWsModelFile.parentFile.mkdirs()
        assertCommandAvailable(quicktypeBinary)
    }

    commandLine(
        quicktypeBinary,
        "--lang",
        "kotlin",
        "--src-lang",
        "schema",
        "--src",
        contractsSchemaFile.absolutePath,
        "--package",
        "com.pegasusx.retailer.generated.contracts",
        "--framework",
        "kotlinx",
        "--top-level",
        "PegasusWSEventEnvelope",
        "--out",
        generatedWsModelFile.absolutePath,
    )
}

if (wsCodegenEnabled) {
    tasks.named("preBuild") {
        dependsOn(generateWsEventModels)
    }
}

android {
    namespace = "com.pegasusx.retailer"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.pegasusx.retailer"
        minSdk = 26
        targetSdk = 35
        versionCode = 1
        versionName = "1.0.0"

        manifestPlaceholders["MAPS_API_KEY"] = mapsApiKey
        buildConfigField("String", "BASE_URL", "\"http://$devHost:8180/\"")
        buildConfigField("String", "WS_URL", "\"ws://$devHost:8180/\"")
    }


    // distribution: enterprise = website CDN OTA; store = Play Store (no CDN APK install)
    flavorDimensions += "distribution"
    productFlavors {
        create("enterprise") {
            dimension = "distribution"
            isDefault = true
            buildConfigField("String", "DISTRIBUTION_CHANNEL", "\"enterprise\"")
            buildConfigField("boolean", "ENABLE_CDN_OTA", "true")
            buildConfigField("String", "STORE_LISTING_URL", "\"\"")
        }
        create("store") {
            dimension = "distribution"
            buildConfigField("String", "DISTRIBUTION_CHANNEL", "\"production\"")
            buildConfigField("boolean", "ENABLE_CDN_OTA", "false")
            buildConfigField(
                "String",
                "STORE_LISTING_URL",
                "\"https://play.google.com/store/apps/details?id=com.pegasusx.retailer\"",
            )
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            buildConfigField("String", "BASE_URL", "\"${prodApiBaseUrl.trimEnd('/')}/\"")
            buildConfigField("String", "WS_URL", "\"${prodWsBaseUrl.trimEnd('/')}/\"")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
    
    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
            excludes += "META-INF/DEPENDENCIES"
            // Exclude duplicate class issues
            excludes += "META-INF/spring.*"
        }
    }
}

dependencies {
    // Core
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.core:core-splashscreen:1.0.1")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.activity:activity-compose:1.9.3")

    // Compose BOM
    implementation(platform("androidx.compose:compose-bom:2024.12.01"))
    implementation(project(":mobile-design"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material3:material3-window-size-class")
    implementation("androidx.compose.material3:material3-adaptive-navigation-suite")
    implementation("androidx.compose.material3.adaptive:adaptive:1.0.0")
    implementation("androidx.compose.material3.adaptive:adaptive-layout:1.0.0")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.compose.animation:animation")

    // Navigation
    implementation("androidx.navigation:navigation-compose:2.8.5")

    // Hilt DI
    implementation("com.google.dagger:hilt-android:2.59.2")
    ksp("com.google.dagger:hilt-compiler:2.59.2")
    ksp("androidx.hilt:hilt-compiler:1.2.0")
    implementation("androidx.hilt:hilt-navigation-compose:1.2.0")
    implementation("androidx.hilt:hilt-work:1.2.0")
    implementation("androidx.work:work-runtime-ktx:2.10.0")

    // Networking
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    implementation("com.jakewharton.retrofit:retrofit2-kotlinx-serialization-converter:1.0.0")

    // Image Loading
    implementation("io.coil-kt:coil-compose:2.7.0")

    // Charts (Vico — Jetpack Compose)
    implementation("com.patrykandpatrick.vico:compose-m3:2.1.2")
    implementation("com.patrykandpatrick.vico:compose:2.1.2")
    implementation("com.patrykandpatrick.vico:core:2.1.2")

    // Geospatial
    implementation("com.uber:h3:4.1.1")

    // Room (Offline-first)
    val roomVersion = "2.7.0-alpha11"
    implementation("androidx.room:room-runtime:$roomVersion")
    implementation("androidx.room:room-ktx:$roomVersion")
    ksp("androidx.room:room-compiler:$roomVersion")

    // DataStore (Preferences)
    implementation("androidx.datastore:datastore-preferences:1.1.1")

    // Google Maps
    implementation("com.google.maps.android:maps-compose:6.2.1")

    // QR generation for retailer delivery confirmation overlays
    implementation("com.google.zxing:core:3.5.3")

    // Barcode scanning removed from ecosystem scope — see docs/BARCODE_SCANNING.md
    // To reinstate: re-add CameraX 1.4.1 + com.google.mlkit:barcode-scanning:17.3.0
    // and add android.permission.CAMERA to AndroidManifest.xml.

    // Security (Encrypted token storage)
    implementation("androidx.security:security-crypto:1.1.0-alpha06")

    // Location (FusedLocationProviderClient)
    implementation("com.google.android.gms:play-services-location:21.3.0")
    implementation("com.google.accompanist:accompanist-permissions:0.36.0")

    // Firebase
    implementation(platform("com.google.firebase:firebase-bom:33.7.0"))
    implementation("com.google.firebase:firebase-messaging-ktx")
    implementation("com.google.firebase:firebase-auth-ktx")

    // Debug
    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")

    // Unit Tests
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    
    // H3 Geospatial Indexing
    implementation("com.uber:h3:4.1.1")
}

configurations.all {
    exclude(group = "io.swagger", module = "swagger-parser-safe-url-resolver")
    exclude(group = "commons-logging", module = "commons-logging")
}
