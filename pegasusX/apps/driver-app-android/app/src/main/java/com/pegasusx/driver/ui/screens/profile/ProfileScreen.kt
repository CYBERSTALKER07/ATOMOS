package com.pegasusx.driver.ui.screens.profile

import android.app.Activity
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.R
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.components.StaggeredAppear
import com.pegasusx.driver.ui.screens.manifest.ManifestViewModel
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.screens.profile.components.*
import com.pegasusx.driver.util.DriverLocaleManager

@Composable
fun ProfileScreen(
    viewModel: ManifestViewModel,
    onOfflineVerifier: () -> Unit = {},
    onSyncQueue: () -> Unit = {},
    onFleetDvir: () -> Unit = {},
    onMidRouteRescue: () -> Unit = {},
    onYardGate: () -> Unit = {},
    pendingCount: Int = 0,
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    var showEndSession by remember { mutableStateOf(false) }
    var showLanguageDialog by remember { mutableStateOf(false) }
    var activeLocale by remember { mutableStateOf(DriverLocaleManager.getSavedLocale(context)) }

    LaunchedEffect(Unit) {
        viewModel.loadEarningsAndHistory()
    }

    if (showLanguageDialog) {
        AlertDialog(
            onDismissRequest = { showLanguageDialog = false },
            title = { Text(stringResource(R.string.settings_language_title)) },
            text = {
                Column {
                    Text(
                        stringResource(R.string.settings_language_subtitle),
                        color = lab.fgSecondary,
                        modifier = Modifier.padding(bottom = 12.dp),
                    )
                    DriverLocaleManager.supportedLanguages.forEach { option ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    activeLocale = DriverLocaleManager.saveLocale(context, option.code)
                                    showLanguageDialog = false
                                    (context as? Activity)?.recreate()
                                }
                                .padding(vertical = 6.dp),
                        ) {
                            RadioButton(
                                selected = activeLocale == option.code,
                                onClick = {
                                    activeLocale = DriverLocaleManager.saveLocale(context, option.code)
                                    showLanguageDialog = false
                                    (context as? Activity)?.recreate()
                                },
                            )
                            Text(option.displayName, modifier = Modifier.padding(start = 8.dp))
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showLanguageDialog = false }) {
                    Text(stringResource(android.R.string.cancel))
                }
            },
        )
    }

    if (showEndSession) {
        EndSessionSheet(
            hasActiveOrders = viewModel.hasActiveOrders,
            isEnding = state.isEndingSession,
            error = state.endSessionError,
            onEndSession = { reason, note -> viewModel.endSession(reason, note) },
            onDismiss = { showEndSession = false }
        )
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg),
        contentAlignment = Alignment.TopCenter
    ) {
        Column(
            modifier = Modifier
                .widthIn(max = 640.dp)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = PegasusSpacing.s16)
                .padding(bottom = 100.dp)
        ) {
        // MARK: - Header
        ProfileHeader()

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Driver Card
        StaggeredAppear(index = 0) {
            DriverCard(
                orders = state.orders,
                hasActiveRoute = state.orders.any {
                    it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING
                }
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Quick Actions
        StaggeredAppear(index = 1) {
            QuickActions(
                onEndSession = { showEndSession = true },
                onOfflineVerifier = onOfflineVerifier,
                onSyncQueue = onSyncQueue,
                onFleetDvir = onFleetDvir,
                onMidRouteRescue = onMidRouteRescue,
                onYardGate = onYardGate,
                onSettings = { showLanguageDialog = true },
                currentLanguageLabel = DriverLocaleManager.displayName(activeLocale),
                pendingCount = pendingCount,
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Ride History
        StaggeredAppear(index = 2) {
            HistorySection(historyRows = state.historyRows)
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Stats
        StaggeredAppear(index = 3) {
            StatsSection(earnings = state.earnings)
        }
        }
    }
}
