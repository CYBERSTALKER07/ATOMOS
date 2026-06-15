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

val localProps = Properties().also { props ->
    val f = rootProject.file("local.properties")
    if (f.exists()) props.load(f.inputStream())
}
val devHost: String = localProps.getProperty("dev.host", "10.0.2.2")
val prodApiBaseUrl: String = localProps.getProperty("prod.api.base.url", "https://api.pegasus.uz")
val quicktypeBinary: String = localProps.getProperty("quicktype.path", "quicktype")

val contractsSchemaFile = rootProject.file("../../contracts/events.schema.json")
val backendGoDir = rootProject.file("../../apps/backend-go")
val generatedWsModelFile = rootProject.file(
    "app/src/main/java/com/pegasusx/factory/generated/contracts/PegasusWSEventEnvelope.kt"
)
val wsCodegenEnabled = localProps.getProperty("factory.ws.codegen", "false") == "true"

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
        "com.pegasusx.factory.generated.contracts",
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
    namespace = "com.pegasusx.factory"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.pegasusx.factory"
        minSdk = 26
        targetSdk = 35
        versionCode = 1
        versionName = "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"

        buildConfigField("String", "API_BASE_URL", "\"http://$devHost:8180\"")
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            buildConfigField("String", "API_BASE_URL", "\"$prodApiBaseUrl\"")
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
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2024.12.01")
    implementation(composeBom)

    // Compose
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.navigation:navigation-compose:2.8.5")

    // Hilt
    implementation("com.google.dagger:hilt-android:2.59.2")
    ksp("com.google.dagger:hilt-android-compiler:2.59.2")
    implementation("androidx.hilt:hilt-navigation-compose:1.2.0")

    // Networking
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    implementation("com.squareup.retrofit2:converter-kotlinx-serialization:2.11.0")

    // Core
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.core:core-splashscreen:1.0.1")
    implementation("androidx.security:security-crypto:1.1.0-alpha06")

    // Debug
    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")

    // Unit tests
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
}
