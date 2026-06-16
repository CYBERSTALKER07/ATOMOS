package com.pegasus.payload.ui

import android.content.Context
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import com.pegasus.payload.BuildConfig
import com.pegasus.payload.data.repository.AuthRepository
import com.pegasus.payload.data.remote.PayloadApi
import com.pegasus.payload.service.AutoUpdater
import com.pegasus.payload.ui.auth.LoginScreen
import com.pegasus.payload.ui.components.ClientPolicyBanner
import com.pegasus.payload.ui.home.HomeScreen
import com.pegasus.payload.ui.inbound.InboundReturnsScreen
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * Top-level composable. Routes between [LoginScreen] (unauth) and [HomeScreen]
 * (auth). Client-policy banner is global across login + authenticated shells.
 */
@Composable
fun PayloadRoot(viewModel: RootViewModel = hiltViewModel()) {
    val session by viewModel.session.collectAsStateWithLifecycle()
    val clientPolicyMessage by viewModel.clientPolicyMessage.collectAsStateWithLifecycle()
    var showInboundReturns by remember { mutableStateOf(false) }

    LaunchedEffect(session) {
        viewModel.loadClientPolicy()
    }

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        Column(modifier = Modifier.fillMaxSize()) {
            ClientPolicyBanner(clientPolicyMessage)
            when {
                session == null -> LoginScreen()
                showInboundReturns -> InboundReturnsScreen(onBack = { showInboundReturns = false })
                else -> HomeScreen(
                    onLogout = viewModel::logout,
                    onInboundReturns = { showInboundReturns = true },
                )
            }
        }
    }
}

@HiltViewModel
class RootViewModel @Inject constructor(
    @ApplicationContext context: Context,
    private val authRepository: AuthRepository,
    private val api: PayloadApi,
) : ViewModel() {
    val session = authRepository.session

    private val autoUpdater = AutoUpdater(context.applicationContext)
    private val _clientPolicyMessage = MutableStateFlow<String?>(null)
    val clientPolicyMessage: StateFlow<String?> = _clientPolicyMessage.asStateFlow()

    init {
        loadClientPolicy()
    }

    fun loadClientPolicy() {
        viewModelScope.launch {
            runCatching {
                val resp = api.getClientPolicy(
                    platform = "android",
                    version = BuildConfig.VERSION_NAME,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    val policy = resp.body()!!
                    _clientPolicyMessage.value = if (policy.outdated || policy.forceUpdate) {
                        buildString {
                            append(if (policy.forceUpdate) "Update required" else "Update available")
                            if (policy.minimumVersion.isNotBlank()) {
                                append(" — minimum version ${policy.minimumVersion}")
                            }
                            policy.deferReason?.takeIf { it.isNotBlank() }?.let { append(". $it") }
                        }
                    } else {
                        null
                    }
                    if (policy.outdated || policy.forceUpdate) {
                        autoUpdater.checkForUpdates(BuildConfig.VERSION_CODE)
                    }
                }
            }
        }
    }

    fun logout() {
        viewModelScope.launch { authRepository.logout() }
    }

    override fun onCleared() {
        autoUpdater.cleanup()
        super.onCleared()
    }
}
