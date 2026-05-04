package com.pegasus.driver.ui.screens.profile

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
import com.pegasus.driver.data.model.Order
import com.pegasus.driver.data.model.OrderState
import com.pegasus.driver.data.remote.TokenHolder
import com.pegasus.driver.ui.components.PegasusCard
import com.pegasus.driver.ui.components.StaggeredAppear
import com.pegasus.driver.ui.components.StatusPill
import com.pegasus.driver.ui.screens.manifest.ManifestViewModel
import com.pegasus.driver.ui.theme.PegasusSpacing
import com.pegasus.driver.ui.theme.LocalPegasusColors
import com.pegasus.driver.ui.theme.formattedAmount
import com.pegasus.driver.ui.theme.pressable
import androidx.compose.material3.MaterialTheme

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
        Column(
            modifier = Modifier.padding(
                top = 60.dp,
                start = PegasusSpacing.s4,
                end = PegasusSpacing.s4
            )
        ) {
            Text(
                text = "DRIVER",
                style = MaterialTheme.typography.labelSmall.copy(
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    letterSpacing = 1.2.sp,
                ),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(modifier = Modifier.height(6.dp))
            Text(
                text = "Profile",
                style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.onSurface,
            )
        }

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

// ── Driver Card ──

@Composable
private fun DriverCard(orders: List<Order>, hasActiveRoute: Boolean) {
    val lab = LocalPegasusColors.current
    val driverName = TokenHolder.driverName ?: "Driver"
    val driverId = TokenHolder.userId ?: "—"
    val truckId = TokenHolder.vehicleType ?: "—"
    val plate = TokenHolder.licensePlate ?: "—"
    val completedCount = orders.count { it.state == OrderState.COMPLETED }

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(14.dp)
            ) {
                // Avatar
                Box(
                    modifier = Modifier
                        .size(52.dp)
                        .clip(CircleShape)
                        .background(lab.fg),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = driverName.take(1).uppercase(),
                        fontSize = 22.sp,
                        fontWeight = FontWeight.Bold,
                        color = lab.buttonFg
                    )
                }

                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = driverName,
                        fontSize = 17.sp,
                        fontWeight = FontWeight.Bold,
                        color = lab.fg
                    )
                    Text(
                        text = driverId,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fgSecondary
                    )
                }

                StatusPill(
                    label = if (hasActiveRoute) "ON DUTY" else "IDLE",
                    color = if (hasActiveRoute) lab.success else lab.fgSecondary
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Info tiles
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                InfoTile(
                    label = "Truck",
                    value = truckId,
                    icon = Icons.Default.LocalShipping,
                    modifier = Modifier.weight(1f)
                )
                InfoTile(
                    label = "Plate",
                    value = plate,
                    icon = Icons.Default.DirectionsCar,
                    modifier = Modifier.weight(1f)
                )
                InfoTile(
                    label = "Completed",
                    value = "$completedCount",
                    icon = Icons.Default.CheckCircle,
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
private fun InfoTile(
    label: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier
) {
    val lab = LocalPegasusColors.current
    val isDark = isSystemInDarkTheme()
    Column(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(lab.fg.copy(alpha = 0.03f))
            .padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = lab.fgSecondary,
            modifier = Modifier.size(14.dp)
        )
        Text(
            text = value,
            fontSize = 14.sp,
            fontWeight = FontWeight.Bold,
            fontFamily = FontFamily.Monospace,
            color = lab.fg,
            textAlign = TextAlign.Center
        )
        Text(
            text = label,
            fontSize = 10.sp,
            fontWeight = FontWeight.Medium,
            color = lab.fgTertiary
        )
    }
}

// ── Quick Actions ──

@Composable
private fun QuickActions(onEndSession: () -> Unit = {}) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        ActionRow(
            icon = Icons.Default.ShieldMoon,
            title = "Offline Verifier",
            subtitle = "Hash manifest protocol",
            onClick = {}
        )
        ActionRow(
            icon = Icons.Default.Sync,
            title = "Sync Queue",
            subtitle = "Upload pending deliveries",
            onClick = {}
        )
        ActionRow(
            icon = Icons.Default.Settings,
            title = "Settings",
            subtitle = "App configuration",
            onClick = {}
        )
        ActionRow(
            icon = Icons.AutoMirrored.Filled.ExitToApp,
            title = "End Session",
            subtitle = "Go offline and sign out",
            destructive = true,
            onClick = onEndSession
        )
    }
}

@Composable
private fun ActionRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    destructive: Boolean = false,
    onClick: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val tint = if (destructive) lab.destructive else lab.fg

    PegasusCard(modifier = Modifier.pressable(onClick = onClick)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(RoundedCornerShape(10.dp))
                    .background(tint.copy(alpha = 0.06f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = tint,
                    modifier = Modifier.size(15.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = tint
                )
                Text(
                    text = subtitle,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgSecondary
                )
            }

            Icon(
                imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = lab.fgTertiary,
                modifier = Modifier.size(11.dp)
            )
        }
    }
}

// ── History Section ──

@Composable
private fun HistorySection(completedOrders: List<Order>) {
    val lab = LocalPegasusColors.current

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = "Completed Rides",
            fontSize = 17.sp,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s8)
        )

        if (completedOrders.isEmpty()) {
            PegasusCard {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 30.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Schedule,
                        contentDescription = null,
                        tint = lab.fgTertiary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = "No completed rides yet",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgSecondary
                    )
                }
            }
        } else {
            completedOrders.forEachIndexed { index, order ->
                StaggeredAppear(index = index) {
                    HistoryRow(order)
                }
            }
        }
    }
}

@Composable
private fun HistoryRow(order: Order) {
    val lab = LocalPegasusColors.current

    PegasusCard {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(lab.success.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.CheckCircle,
                    contentDescription = null,
                    tint = lab.success,
                    modifier = Modifier.size(13.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = order.id,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fg
                )
                Text(
                    text = "${order.retailerName} · ${order.totalAmount.formattedAmount()}",
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgSecondary
                )
            }

            StatusPill(label = "DELIVERED", color = lab.success)
        }
    }
}

// ── Stats Section ──

@Composable
private fun StatsSection(completedOrders: List<Order>) {
    val lab = LocalPegasusColors.current
    val totalValue = completedOrders.sumOf { it.totalAmount }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = "Session Stats",
            fontSize = 17.sp,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s8)
        )

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            StatCard(
                title = "Total Value",
                value = if (totalValue > 0) totalValue.formattedAmount() else "—",
                icon = Icons.Default.DirectionsCar,
                modifier = Modifier.weight(1f)
            )
            StatCard(
                title = "Avg Distance",
                value = "—",
                icon = Icons.Default.LocationOn,
                modifier = Modifier.weight(1f)
            )
        }
    }
}

@Composable
private fun StatCard(
    title: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier
) {
    val lab = LocalPegasusColors.current
    PegasusCard(modifier = modifier) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = lab.fgSecondary,
                modifier = Modifier.size(14.dp)
            )
            Text(
                text = value,
                fontSize = 15.sp,
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Monospace,
                color = lab.fg,
                maxLines = 1
            )
            Text(
                text = title,
                fontSize = 11.sp,
                fontWeight = FontWeight.Medium,
                color = lab.fgTertiary
            )
        }
    }
}
