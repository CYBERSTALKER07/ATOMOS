package com.pegasusx.driver.ui.screens.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.ShieldMoon
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.StaggeredAppear
import com.pegasusx.driver.ui.components.StatusPill
import com.pegasusx.driver.ui.screens.manifest.ManifestViewModel
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.ui.theme.pressable
import androidx.compose.material3.MaterialTheme
import com.pegasusx.driver.ui.screens.profile.components.*

@Composable
fun ProfileScreen(viewModel: ManifestViewModel) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current
    var showEndSession by remember { mutableStateOf(false) }

    if (showEndSession) {
        EndSessionSheet(
            hasActiveOrders = viewModel.hasActiveOrders,
            isEnding = state.isEndingSession,
            error = state.endSessionError,
            onEndSession = { reason, note -> viewModel.endSession(reason, note) },
            onDismiss = { showEndSession = false }
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
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
            QuickActions(onEndSession = { showEndSession = true })
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Ride History
        StaggeredAppear(index = 2) {
            HistorySection(
                completedOrders = state.orders.filter { it.state == OrderState.COMPLETED }
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s24))

        // MARK: - Stats
        StaggeredAppear(index = 3) {
            StatsSection(
                completedOrders = state.orders.filter { it.state == OrderState.COMPLETED }
            )
        }
    }
}
