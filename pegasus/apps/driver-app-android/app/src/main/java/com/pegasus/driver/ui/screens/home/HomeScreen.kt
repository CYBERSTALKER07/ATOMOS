package com.pegasus.driver.ui.screens.home

import android.content.Intent
import android.net.Uri
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Map
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.ShieldMoon
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material3.Button
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasus.driver.data.model.Order
import com.pegasus.driver.data.model.OrderState
import com.pegasus.driver.data.remote.TokenHolder
import com.pegasus.driver.services.TelemetryService
import com.pegasus.driver.ui.components.PegasusCard
import com.pegasus.driver.ui.components.StaggeredAppear
import com.pegasus.driver.ui.components.StatusPill
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import com.pegasus.driver.ui.screens.manifest.ManifestViewModel
import com.pegasus.driver.ui.theme.PegasusSpacing
import com.pegasus.driver.ui.theme.LocalPegasusColors
import com.pegasus.driver.ui.theme.MotionTokens
import com.pegasus.driver.ui.theme.formattedAmount
import com.pegasus.driver.ui.theme.pressable
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Locale

@Composable
fun HomeScreen(
    viewModel: ManifestViewModel,
    onOpenMap: () -> Unit,
    onScanQR: () -> Unit,
    onNotificationsClick: () -> Unit = {},
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current

    if (state.isLoading) {
        HomeShimmer(lab = lab)
        return
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = PegasusSpacing.s16)
            .padding(bottom = 100.dp)
    ) {
        // MARK: - Greeting + Notification Bell
        StaggeredAppear(index = 0) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Top,
            ) {
                Box(modifier = Modifier.weight(1f)) {
                    GreetingSection()
                }
                IconButton(onClick = onNotificationsClick) {
                    Icon(
                        Icons.Outlined.Notifications,
                        contentDescription = "Notifications",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Status Chips
        StaggeredAppear(index = 1) {
            StatusChips(
                hasActiveRoute = state.orders.any {
                    it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING
                },
                isReturning = state.isReturning
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Vehicle Info Card
        val vehicle = TokenHolder.vehicleType
        val plate = TokenHolder.licensePlate
        val vClass = TokenHolder.vehicleClass
        val vu = TokenHolder.maxVolumeVU
        if (!vehicle.isNullOrBlank() || !plate.isNullOrBlank()) {
            StaggeredAppear(index = 2) {
                VehicleInfoCard(
                    truckId = vehicle ?: "—",
                    licensePlate = plate ?: "—",
                    vehicleClass = vClass ?: "",
                    maxVolumeVU = vu
                )
            }
            Spacer(modifier = Modifier.height(PegasusSpacing.s20))
        }

        // MARK: - Transit Control
        StaggeredAppear(index = 3) {
            if (state.isReturning) {
                ReturningToWarehouseCard(
                    onNavigate = { viewModel.state.value }, // warehouse coords from backend; fallback to depot
                    onArrived = { viewModel.returnComplete() }
                )
            } else {
                TransitControlCard(
                    orders = state.orders,
                    onDepart = { viewModel.departRoute() }
                )
            }
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Today Summary
        StaggeredAppear(index = 4) {
            TodaySummaryCard(orders = state.orders)
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Open Map CTA
        StaggeredAppear(index = 5) {
            MapButton(
                pendingCount = state.orders.count {
                    it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
                },
                onOpenMap = onOpenMap
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Quick Actions
        StaggeredAppear(index = 6) {
            QuickActionsSection(
                onScanQR = onScanQR,
                hasArrivedOrder = state.orders.any { it.state == OrderState.ARRIVED }
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Recent Activity
        StaggeredAppear(index = 7) {
            RecentActivitySection(
                completedOrders = state.orders.filter { it.state == OrderState.COMPLETED }
            )
        }
    }
}

// ── Greeting ──

@Composable
private fun GreetingSection() {
    val colorScheme = MaterialTheme.colorScheme
    val typography = MaterialTheme.typography
    val driverName = TokenHolder.driverName ?: "Driver"

    Column(
        modifier = Modifier.padding(
            top = 60.dp,
            start = PegasusSpacing.s4,
            end = PegasusSpacing.s4
        )
    ) {
        Text(
            text = greetingText(),
            style = typography.labelSmall.copy(
                fontWeight = FontWeight.Black,
                fontFamily = FontFamily.Monospace,
                letterSpacing = 1.2.sp,
            ),
            color = colorScheme.onSurfaceVariant,
        )
        Spacer(modifier = Modifier.height(6.dp))
        Text(
            text = driverName,
            style = typography.headlineLarge.copy(fontWeight = FontWeight.Bold),
            color = colorScheme.onSurface,
        )
    }
}

private fun greetingText(): String {
    val hour = Calendar.getInstance().get(Calendar.HOUR_OF_DAY)
    return when (hour) {
        in 5..11 -> "GOOD MORNING"
        in 12..16 -> "GOOD AFTERNOON"
        in 17..20 -> "GOOD EVENING"
        else -> "GOOD NIGHT"
    }
}

// ── Status Chips ──

@Composable
private fun StatusChips(hasActiveRoute: Boolean, isReturning: Boolean) {
    val lab = LocalPegasusColors.current
    val plate = TokenHolder.licensePlate ?: "—"

    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        StatusChip(
            icon = Icons.Default.LocalShipping,
            label = plate,
            active = true
        )
        when {
            isReturning -> StatusChip(
                icon = Icons.Default.Home,
                label = "Returning",
                active = true
            )
            hasActiveRoute -> StatusChip(
                icon = Icons.Default.Sync,
                label = "On Route",
                active = true
            )
            else -> StatusChip(
                icon = Icons.Default.ShieldMoon,
                label = "Idle",
                active = false
            )
        }
    }
}

@Composable
private fun StatusChip(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    active: Boolean
) {
    val colorScheme = MaterialTheme.colorScheme
    val tint = if (active) colorScheme.onSurface else colorScheme.onSurfaceVariant
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        modifier = Modifier
            .clip(MaterialTheme.shapes.small)
            .background(colorScheme.surfaceContainerLow)
            .padding(horizontal = 12.dp, vertical = 8.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = tint,
            modifier = Modifier.size(16.dp)
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium.copy(
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Monospace,
            ),
            color = tint,
        )
    }
}

// ── Vehicle Info Card ──

@Composable
private fun VehicleInfoCard(
    truckId: String,
    licensePlate: String,
    vehicleClass: String,
    maxVolumeVU: Double
) {
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
                    .size(48.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(lab.separator),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.LocalShipping,
                    contentDescription = null,
                    tint = lab.fg,
                    modifier = Modifier.size(24.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = truckId,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg
                )
                val subtitle = buildString {
                    append(licensePlate)
                    if (vehicleClass.isNotBlank()) {
                        append(" · $vehicleClass · ${maxVolumeVU.toInt()} VU")
                    }
                }
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary
                )
            }

            StatusPill(label = "ASSIGNED", color = lab.success)
        }
    }
}

// ── Transit Control Card ──

@Composable
private fun TransitControlCard(
    orders: List<Order>,
    onDepart: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    val loadedOrders = orders.filter { it.state == OrderState.LOADED }
    val inTransitOrders = orders.filter {
        it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING
    }

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            when {
                inTransitOrders.isNotEmpty() -> {
                    // Active transit
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        PulsingDot(color = lab.live)
                        Text(
                            text = "IN TRANSIT",
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Black,
                            fontFamily = FontFamily.Monospace,
                            color = lab.live
                        )
                        Spacer(modifier = Modifier.weight(1f))
                        Text(
                            text = "${inTransitOrders.size} deliveries",
                            style = MaterialTheme.typography.bodySmall,
                            fontWeight = FontWeight.Medium,
                            color = lab.fgTertiary
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "Telemetry active — drive safely",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgTertiary
                    )
                }
                loadedOrders.isNotEmpty() -> {
                    // Ready to depart
                    Text(
                        text = "READY TO DEPART",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Black,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fg
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = "${loadedOrders.size} orders loaded",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgTertiary
                    )
                    Spacer(modifier = Modifier.height(14.dp))
                    Button(
                        onClick = {
                            val intent =
                                Intent(context, TelemetryService::class.java).apply {
                                    action = TelemetryService.ACTION_START
                                }
                            context.startForegroundService(intent)
                            onDepart()
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(PegasusSpacing.s48),
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Default.LocalShipping,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp)
                            )
                            Text(
                                text = "START TRANSIT",
                                style = MaterialTheme.typography.labelLarge,
                                fontWeight = FontWeight.Black,
                                fontFamily = FontFamily.Monospace,
                            )
                        }
                    }
                }
                else -> {
                    // No orders
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Schedule,
                            contentDescription = null,
                            tint = lab.fgTertiary,
                            modifier = Modifier.size(18.dp)
                        )
                        Text(
                            text = "No orders loaded yet",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.Medium,
                            color = lab.fgTertiary
                        )
                    }
                }
            }
        }
    }
}

// ── Returning to Warehouse Card ──

@Composable
private fun ReturningToWarehouseCard(
    onNavigate: () -> Unit,
    onArrived: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    // Dynamic warehouse coords from backend (fallback to Tashkent depot)
    val depotLat = TokenHolder.warehouseLat.takeIf { it != 0.0 } ?: 41.2995
    val depotLng = TokenHolder.warehouseLng.takeIf { it != 0.0 } ?: 69.2401
    val warehouseLabel = TokenHolder.warehouseName ?: "Warehouse"

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                PulsingDot(color = lab.warning)
                Text(
                    text = "RETURNING TO WAREHOUSE",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    color = lab.warning
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "All deliveries completed. Return to warehouse to finish shift.",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium,
                color = lab.fgTertiary
            )
            Spacer(modifier = Modifier.height(16.dp))

            // Navigate to warehouse
            FilledTonalButton(
                onClick = {
                    val uri = Uri.parse("google.navigation:q=$depotLat,$depotLng&mode=d")
                    val intent = Intent(Intent.ACTION_VIEW, uri).apply {
                        setPackage("com.google.android.apps.maps")
                    }
                    if (intent.resolveActivity(context.packageManager) != null) {
                        context.startActivity(intent)
                    } else {
                        val webUri = Uri.parse("https://www.google.com/maps/dir/?api=1&destination=$depotLat,$depotLng&travelmode=driving")
                        context.startActivity(Intent(Intent.ACTION_VIEW, webUri))
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) {
                Icon(
                    imageVector = Icons.Default.Navigation,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Navigate to $warehouseLabel", style = MaterialTheme.typography.labelLarge)
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Arrived at warehouse
            Button(
                onClick = onArrived,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(PegasusSpacing.s48),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Home,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = "ARRIVED AT WAREHOUSE",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Black,
                        fontFamily = FontFamily.Monospace,
                    )
                }
            }
        }
    }
}

// ── Today Summary ──

@Composable
private fun TodaySummaryCard(orders: List<Order>) {
    val lab = LocalPegasusColors.current
    val pending = orders.count {
        it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
    }
    val completed = orders.count { it.state == OrderState.COMPLETED }
    val revenue = orders
        .filter { it.state == OrderState.COMPLETED }
        .sumOf { it.totalAmount }

    val todayDate = remember {
        SimpleDateFormat("dd MMM yyyy", Locale.getDefault())
            .format(Calendar.getInstance().time)
            .uppercase()
    }

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Today",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg
                )
                Text(
                    text = todayDate,
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Medium,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary
                )
            }
            Spacer(modifier = Modifier.height(14.dp))

            Row(modifier = Modifier.fillMaxWidth()) {
                SummaryTile(
                    value = "$pending",
                    label = "Pending",
                    icon = Icons.Default.Schedule,
                    modifier = Modifier.weight(1f)
                )
                VerticalDivider()
                SummaryTile(
                    value = "$completed",
                    label = "Done",
                    icon = Icons.Default.CheckCircle,
                    modifier = Modifier.weight(1f)
                )
                VerticalDivider()
                SummaryTile(
                    value = if (revenue > 0) revenue.formattedAmount().replace("", "") else "—",
                    label = "Revenue",
                    icon = Icons.Default.DirectionsCar,
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
private fun SummaryTile(
    value: String,
    label: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    modifier: Modifier = Modifier
) {
    val lab = LocalPegasusColors.current
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = lab.fgTertiary,
            modifier = Modifier.size(16.dp)
        )
        Text(
            text = value,
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            fontFamily = FontFamily.Monospace,
            color = lab.fg,
            maxLines = 1
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.Medium,
            color = lab.fgTertiary
        )
    }
}

@Composable
private fun VerticalDivider() {
    val lab = LocalPegasusColors.current
    Box(
        modifier = Modifier
            .width(0.5.dp)
            .height(36.dp)
            .background(lab.separator)
    )
}

// ── Map Button ──

@Composable
private fun MapButton(pendingCount: Int, onOpenMap: () -> Unit) {
    val lab = LocalPegasusColors.current
    PegasusCard(modifier = Modifier.pressable(onClick = onOpenMap)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(RoundedCornerShape(14.dp))
                    .background(lab.fg),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.Map,
                    contentDescription = null,
                    tint = lab.buttonFg,
                    modifier = Modifier.size(18.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "Open Map",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg
                )
                Text(
                    text = "$pendingCount deliveries waiting",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgSecondary
                )
            }

            Icon(
                imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                contentDescription = null,
                tint = lab.fgTertiary,
                modifier = Modifier.size(18.dp)
            )
        }
    }
}

// ── Quick Actions ──

@Composable
private fun QuickActionsSection(onScanQR: () -> Unit, hasArrivedOrder: Boolean = false) {
    val lab = LocalPegasusColors.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(
            text = "Quick Actions",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s4)
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            ActionTile(
                icon = Icons.Default.QrCodeScanner,
                label = "Scan QR",
                modifier = Modifier.weight(1f),
                enabled = hasArrivedOrder,
                onClick = onScanQR
            )
            ActionTile(
                icon = Icons.Default.ShieldMoon,
                label = "Offline\nVerify",
                modifier = Modifier.weight(1f),
                onClick = {}
            )
            ActionTile(
                icon = Icons.Default.Sync,
                label = "Sync",
                modifier = Modifier.weight(1f),
                onClick = {}
            )
        }
    }
}

@Composable
private fun ActionTile(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    onClick: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val alpha = if (enabled) 1f else 0.35f
    PegasusCard(modifier = modifier.pressable(onClick = { if (enabled) onClick() })) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(PegasusSpacing.s48)
                    .clip(CircleShape)
                    .background(lab.separator.copy(alpha = alpha)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = lab.fg.copy(alpha = alpha),
                    modifier = Modifier.size(20.dp)
                )
            }
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = lab.fgSecondary.copy(alpha = alpha),
                lineHeight = MaterialTheme.typography.labelMedium.lineHeight,
                maxLines = 2
            )
        }
    }
}

// ── Recent Activity ──

@Composable
private fun RecentActivitySection(completedOrders: List<Order>) {
    val lab = LocalPegasusColors.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(
            text = "Recent",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s4)
        )

        if (completedOrders.isEmpty()) {
            PegasusCard {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Schedule,
                        contentDescription = null,
                        tint = lab.fgTertiary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = "No deliveries yet",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgSecondary
                    )
                }
            }
        } else {
            completedOrders.take(3).forEach { order ->
                PegasusCard {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(PegasusSpacing.s12),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(CircleShape)
                                .background(lab.success.copy(alpha = 0.12f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.CheckCircle,
                                contentDescription = null,
                                tint = lab.success,
                                modifier = Modifier.size(16.dp)
                            )
                        }

                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = order.id,
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace,
                                color = lab.fg
                            )
                            Text(
                                text = order.totalAmount.formattedAmount(),
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Medium,
                                color = lab.fgSecondary
                            )
                        }

                        Text(
                            text = order.retailerName,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = lab.fgTertiary
                        )
                    }
                }
            }
        }
    }
}

// ── Pulsing Dot ──

@Composable
private fun PulsingDot(color: androidx.compose.ui.graphics.Color) {
    val transition = rememberInfiniteTransition(label = "pulse")
    val alpha by transition.animateFloat(
        initialValue = 1f,
        targetValue = 0.3f,
        animationSpec = infiniteRepeatable(
            animation = tween(MotionTokens.DurationExtraLong4),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_alpha"
    )
    Box(
        modifier = Modifier
            .size(8.dp)
            .alpha(alpha)
            .clip(CircleShape)
            .background(color)
    )
}

// ── Shimmer skeleton shown while orders are loading ──────────────────────────
@Composable
private fun HomeShimmer(lab: com.pegasus.driver.ui.theme.PegasusColors) {
    val transition = rememberInfiniteTransition(label = "shimmer")
    val shimmerAlpha by transition.animateFloat(
        initialValue = 0.25f,
        targetValue = 0.60f,
        animationSpec = infiniteRepeatable(tween(MotionTokens.DurationExtraLong4), RepeatMode.Reverse),
        label = "shimmer_alpha"
    )
    val shimmerColor = lab.fgTertiary.copy(alpha = shimmerAlpha)

    @Composable
    fun ShimmerBlock(width: Float = 1f, height: Int = 20, corner: Int = 8) {
        Box(
            modifier = Modifier
                .fillMaxWidth(fraction = width)
                .height(height.dp)
                .clip(RoundedCornerShape(corner.dp))
                .background(shimmerColor)
        )
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = PegasusSpacing.s16)
            .padding(bottom = 100.dp),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.s20),
    ) {
        Spacer(Modifier.height(PegasusSpacing.s20))
        // Greeting skeleton
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            ShimmerBlock(width = 0.4f, height = 16)
            ShimmerBlock(width = 0.65f, height = 24)
        }
        // Status chips skeleton
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            repeat(3) { ShimmerBlock(width = 0.26f, height = 32, corner = 16) }
        }
        // Vehicle card skeleton
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(88.dp)
                .clip(RoundedCornerShape(16.dp))
                .background(lab.card)
                .padding(16.dp)
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                ShimmerBlock(width = 0.5f, height = 14)
                ShimmerBlock(width = 0.35f, height = 14)
            }
        }
        // Transit card skeleton
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(120.dp)
                .clip(RoundedCornerShape(16.dp))
                .background(lab.card)
        )
        // Summary card skeleton
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(80.dp)
                .clip(RoundedCornerShape(16.dp))
                .background(lab.card)
        )
        // Map button skeleton
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp)
                .clip(RoundedCornerShape(16.dp))
                .background(shimmerColor)
        )
    }
}
