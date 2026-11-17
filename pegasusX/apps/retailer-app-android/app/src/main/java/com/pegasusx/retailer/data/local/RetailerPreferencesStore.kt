package com.pegasusx.retailer.data.local

import android.app.LocaleManager
import android.content.Context
import android.content.res.Configuration
import android.os.Build
import android.os.LocaleList
import androidx.core.app.LocaleManagerCompat
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import java.util.Locale
import javax.inject.Singleton
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.map

enum class RetailerWorkspaceMode(val wireValue: String) {
    FAST("fast"),
    FULL("full"),
    ;

    companion object {
        fun fromWire(raw: String?): RetailerWorkspaceMode? {
            return when (raw?.trim()?.lowercase()) {
                "fast" -> FAST
                "full" -> FULL
                else -> null
            }
        }
    }
}

enum class RetailerLanguage(
    val code: String,
    val nativeLabel: String,
) {
    UZ("uz", "O'zbekcha"),
    RU("ru", "Русский"),
    EN("en", "English"),
    ;

    companion object {
        val DEFAULT: RetailerLanguage = UZ

        fun fromCode(raw: String?): RetailerLanguage {
            val normalized = raw?.trim()?.lowercase()?.substringBefore('-')?.substringBefore('_')
            return entries.firstOrNull { it.code == normalized } ?: DEFAULT
        }
    }
}

interface RetailerPreferencesStore {
    val workspaceModeFlow: StateFlow<RetailerWorkspaceMode?>
    val languageFlow: StateFlow<RetailerLanguage>

    fun getWorkspaceMode(): RetailerWorkspaceMode?
    fun setWorkspaceMode(mode: RetailerWorkspaceMode)
    fun getLanguage(): RetailerLanguage
    fun setLanguage(language: RetailerLanguage)
}

class DefaultRetailerPreferencesStore(context: Context) : RetailerPreferencesStore {
    private val prefs = context.getSharedPreferences(RetailerLocaleHelper.PREFS_NAME, Context.MODE_PRIVATE)

    private val _workspaceModeFlow = MutableStateFlow(
        RetailerWorkspaceMode.fromWire(prefs.getString(RetailerLocaleHelper.KEY_WORKSPACE_MODE, null)),
    )
    override val workspaceModeFlow: StateFlow<RetailerWorkspaceMode?> = _workspaceModeFlow.asStateFlow()

    private val _languageFlow = MutableStateFlow(
        RetailerLanguage.fromCode(prefs.getString(RetailerLocaleHelper.KEY_LANGUAGE, RetailerLanguage.DEFAULT.code)),
    )
    override val languageFlow: StateFlow<RetailerLanguage> = _languageFlow.asStateFlow()

    override fun getWorkspaceMode(): RetailerWorkspaceMode? = _workspaceModeFlow.value

    override fun setWorkspaceMode(mode: RetailerWorkspaceMode) {
        prefs.edit().putString(RetailerLocaleHelper.KEY_WORKSPACE_MODE, mode.wireValue).commit()
        _workspaceModeFlow.value = mode
    }

    override fun getLanguage(): RetailerLanguage = _languageFlow.value

    override fun setLanguage(language: RetailerLanguage) {
        prefs.edit().putString(RetailerLocaleHelper.KEY_LANGUAGE, language.code).commit()
        _languageFlow.value = language
    }
}

class InMemoryRetailerPreferencesStore(
    initialLanguage: RetailerLanguage = RetailerLanguage.DEFAULT,
    initialWorkspaceMode: RetailerWorkspaceMode? = null,
) : RetailerPreferencesStore {
    private val _workspaceModeFlow = MutableStateFlow(initialWorkspaceMode)
    override val workspaceModeFlow: StateFlow<RetailerWorkspaceMode?> = _workspaceModeFlow.asStateFlow()

    private val _languageFlow = MutableStateFlow(initialLanguage)
    override val languageFlow: StateFlow<RetailerLanguage> = _languageFlow.asStateFlow()

    override fun getWorkspaceMode(): RetailerWorkspaceMode? = _workspaceModeFlow.value

    override fun setWorkspaceMode(mode: RetailerWorkspaceMode) {
        _workspaceModeFlow.value = mode
    }

    override fun getLanguage(): RetailerLanguage = _languageFlow.value

    override fun setLanguage(language: RetailerLanguage) {
        _languageFlow.value = language
    }
}

interface RetailerSessionContext {
    fun getUserId(): String?
    fun getUserName(): String?
    fun getPreferredToken(): String?
}

interface RetailerRealtimeHub {
    fun connect()
    val promotionChangedEvents: kotlinx.coroutines.flow.Flow<Unit>
    val reconnectEvents: kotlinx.coroutines.flow.Flow<Unit>
}

object NoopRetailerRealtimeHub : RetailerRealtimeHub {
    override fun connect() = Unit
    override val promotionChangedEvents: kotlinx.coroutines.flow.Flow<Unit> = emptyFlow()
    override val reconnectEvents: kotlinx.coroutines.flow.Flow<Unit> = emptyFlow()
}

object RetailerLocaleHelper {
    const val PREFS_NAME = "pegasus_retailer_prefs"
    const val KEY_LANGUAGE = "retailer_language"
    const val KEY_WORKSPACE_MODE = "retailer_workspace_mode"

    fun readSavedLanguage(context: Context): RetailerLanguage {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val raw = prefs.getString(KEY_LANGUAGE, null)
        if (!raw.isNullOrBlank()) {
            return RetailerLanguage.fromCode(raw)
        }
        val appLocales = runCatching { LocaleManagerCompat.getApplicationLocales(context) }.getOrNull()
        val firstTag = if (appLocales != null && !appLocales.isEmpty) appLocales[0]?.language else null
        return if (!firstTag.isNullOrBlank()) RetailerLanguage.fromCode(firstTag) else RetailerLanguage.DEFAULT
    }

    fun wrapBaseContext(newBase: Context): Context {
        val lang = readSavedLanguage(newBase)
        return createLocalizedContext(newBase, lang)
    }

    fun createLocalizedContext(base: Context, language: RetailerLanguage): Context {
        val locale = Locale.forLanguageTag(language.code)
        Locale.setDefault(locale)
        val config = Configuration(base.resources.configuration)
        config.setLocale(locale)
        config.setLocales(LocaleList(locale))
        config.setLayoutDirection(locale)
        return base.createConfigurationContext(config)
    }

    fun applyApplicationLocale(context: Context, language: RetailerLanguage) {
        val locale = Locale.forLanguageTag(language.code)
        Locale.setDefault(locale)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            runCatching {
                context.getSystemService(LocaleManager::class.java)?.applicationLocales =
                    LocaleList.forLanguageTags(language.code)
            }
        }
    }
}

@Module
@InstallIn(SingletonComponent::class)
object RetailerPreferencesModule {
    @Provides
    @Singleton
    fun provideRetailerPreferencesStore(
        @ApplicationContext context: Context,
    ): RetailerPreferencesStore = DefaultRetailerPreferencesStore(context)

    @Provides
    @Singleton
    fun provideRetailerSessionContext(
        tokenManager: TokenManager,
    ): RetailerSessionContext = object : RetailerSessionContext {
        override fun getUserId(): String? = tokenManager.getUserId()
        override fun getUserName(): String? = tokenManager.getUserName()
        override fun getPreferredToken(): String? = tokenManager.getPreferredToken()
    }

    @Provides
    @Singleton
    fun provideRetailerRealtimeHub(
        ws: com.pegasusx.retailer.data.api.RetailerWebSocket,
    ): RetailerRealtimeHub = object : RetailerRealtimeHub {
        override fun connect() = ws.connect()
        override val promotionChangedEvents: kotlinx.coroutines.flow.Flow<Unit> =
            ws.events.filter { it.type == "PROMOTION_CHANGED" }.map { Unit }
        override val reconnectEvents: kotlinx.coroutines.flow.Flow<Unit> = ws.reconnects
    }
}
