package com.pegasus.payload.ui

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import com.pegasus.payload.data.repository.AuthRepository
import com.pegasus.payload.data.remote.PayloadApi
import com.pegasus.payload.ui.auth.LoginScreen
import com.pegasus.payload.ui.home.HomeScreen
import com.pegasus.payload.ui.inbound.InboundReturnsScreen
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * Top-level composable. Routes between [LoginScreen] (unauth) and [HomeScreen]
 * (auth). Auth state is sourced from [AuthRepository.session].
 */
@Composable
fun PayloadRoot(viewModel: RootViewModel = hiltViewModel()) {
    val session by viewModel.session.collectAsStateWithLifecycle()
    var showInboundReturns by remember { mutableStateOf(false) }
    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        if (session == null) {
            LoginScreen()
        } else if (showInboundReturns) {
            InboundReturnsScreen(onBack = { showInboundReturns = false })
        } else {
            HomeScreen(
                onLogout = viewModel::logout,
                onInboundReturns = { showInboundReturns = true },
            )
        }
    }
}

@HiltViewModel
class RootViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    val api: PayloadApi,
) : ViewModel() {
    val session = authRepository.session

    fun logout() {
        viewModelScope.launch { authRepository.logout() }
    }
}
