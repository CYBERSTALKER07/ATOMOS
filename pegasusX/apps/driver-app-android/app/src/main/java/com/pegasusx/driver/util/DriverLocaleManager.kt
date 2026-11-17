package com.pegasusx.driver.util

import android.content.Context
import android.content.res.Configuration
import java.util.Locale

data class DriverLanguageOption(
    val code: String,
    val displayName: String,
)

/**
 * Manages persisted driver app language preference (`uz` default, `ru`, `en`)
 * without requiring external localization dependencies.
 */
object DriverLocaleManager {
    private const val PREFS_NAME = "pegasus_driver_prefs"
    private const val KEY_LOCALE = "app_locale"
    const val DEFAULT_LOCALE = "uz"

    val supportedLanguages: List<DriverLanguageOption> = listOf(
        DriverLanguageOption("uz", "O'zbekcha"),
        DriverLanguageOption("ru", "Русский"),
        DriverLanguageOption("en", "English"),
    )

    fun normalizeLocaleTag(tag: String?): String {
        val clean = tag.orEmpty().trim().lowercase(Locale.US)
        return if (supportedLanguages.any { it.code == clean }) clean else DEFAULT_LOCALE
    }

    fun displayName(tag: String?): String {
        val code = normalizeLocaleTag(tag)
        return supportedLanguages.firstOrNull { it.code == code }?.displayName ?: "O'zbekcha"
    }

    fun getSavedLocale(context: Context): String {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return normalizeLocaleTag(prefs.getString(KEY_LOCALE, DEFAULT_LOCALE))
    }

    fun saveLocale(context: Context, tag: String): String {
        val normalized = normalizeLocaleTag(tag)
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY_LOCALE, normalized)
            .apply()
        Locale.setDefault(Locale.forLanguageTag(normalized))
        return normalized
    }

    fun applyProcessLocale(context: Context): String {
        val tag = getSavedLocale(context)
        Locale.setDefault(Locale.forLanguageTag(tag))
        return tag
    }

    fun wrapContext(base: Context): Context {
        val tag = getSavedLocale(base)
        val locale = Locale.forLanguageTag(tag)
        Locale.setDefault(locale)
        val config = Configuration(base.resources.configuration)
        config.setLocale(locale)
        return base.createConfigurationContext(config)
    }
}
