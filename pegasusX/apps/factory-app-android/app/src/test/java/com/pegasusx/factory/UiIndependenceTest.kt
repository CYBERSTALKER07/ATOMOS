package com.pegasusx.factory

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class UiIndependenceTest {

    @Test
    fun settingsGradleMustNotIncludeSharedMobileModules() {
        val rootDir = File(".").canonicalFile.let {
            if (it.name == "app") it.parentFile else it
        }
        val settingsFile = File(rootDir, "settings.gradle.kts")
        val content = settingsFile.readText()
        assertFalse(
            "settings.gradle.kts must not include :mobile-design",
            content.contains(":mobile-design")
        )
        assertFalse(
            "settings.gradle.kts must not include :mobile-kit",
            content.contains(":mobile-kit")
        )
    }

    @Test
    fun appBuildGradleMustNotDependOnSharedMobileModules() {
        val rootDir = File(".").canonicalFile.let {
            if (it.name == "app") it.parentFile else it
        }
        val buildFile = File(rootDir, "app/build.gradle.kts")
        val content = buildFile.readText()
        assertFalse(
            "app/build.gradle.kts must not depend on project(\":mobile-design\")",
            content.contains("project(\":mobile-design\")")
        )
        assertFalse(
            "app/build.gradle.kts must not depend on project(\":mobile-kit\")",
            content.contains("project(\":mobile-kit\")")
        )
    }

    @Test
    fun sourceCodeMustNotImportFromSharedMobileModules() {
        val rootDir = File(".").canonicalFile.let {
            if (it.name == "app") it.parentFile else it
        }
        val srcDir = File(rootDir, "app/src/main/java")
        val kotlinFiles = srcDir.walkTopDown().filter { it.extension == "kt" }.toList()

        for (file in kotlinFiles) {
            val text = file.readText()
            assertFalse(
                "File ${file.name} must not import from com.pegasus.design",
                text.contains("import com.pegasus.design") || text.contains("com.pegasus.design.")
            )
            assertFalse(
                "File ${file.name} must not import from com.pegasusx.mobilekit",
                text.contains("import com.pegasusx.mobilekit") || text.contains("com.pegasusx.mobilekit.")
            )
        }
    }

    @Test
    fun localNativeUiComponentsAndTokensMustExist() {
        val rootDir = File(".").canonicalFile.let {
            if (it.name == "app") it.parentFile else it
        }
        val uiDir = File(rootDir, "app/src/main/java/com/pegasusx/factory/ui")
        assertTrue(File(uiDir, "theme/Tokens.kt").exists())
        assertTrue(File(uiDir, "design/PulseHonesty.kt").exists())
        assertTrue(File(uiDir, "design/StatusStack.kt").exists())
    }
}
