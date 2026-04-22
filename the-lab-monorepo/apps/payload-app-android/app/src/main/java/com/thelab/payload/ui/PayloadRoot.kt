package com.thelab.payload.ui

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import com.thelab.payload.data.repository.AuthRepository
import com.thelab.payload.ui.auth.LoginScreen
import com.thelab.payload.ui.home.HomeScreen
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
    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        if (session == null) {
            LoginScreen()
        } else {
            HomeScreen(onLogout = viewModel::logout)
        }
    }
}

@HiltViewModel
class RootViewModel @Inject constructor(
    private val authRepository: AuthRepository,
) : ViewModel() {
    val session = authRepository.session

    fun logout() {
        viewModelScope.launch { authRepository.logout() }
    }
}
