package com.pegasusx.driver.ui.screens.offline

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.StatusRed

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OfflineVerifierScreen(
    onBack: () -> Unit,
    viewModel: OfflineVerifierViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Offline Verifier") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("Hash Manifest Protocol")
            if (state.isSyncing) {
                CircularProgressIndicator()
            } else {
                Text("Cached orders: ${state.orderCount}")
                state.syncedAt?.let { Text("Manifest date: $it") }
            }
            state.error?.let { Text(it, color = StatusRed) }
            Button(
                onClick = { viewModel.syncManifest() },
                enabled = !state.isSyncing,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (state.isSyncing) "Syncing…" else "Sync Manifest")
            }
        }
    }
}
