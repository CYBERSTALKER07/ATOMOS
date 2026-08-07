package com.pegasusx.driver.ui.screens.home

import androidx.compose.ui.res.stringResource

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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.model.PulseEvent
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.services.TelemetryService
import com.pegasusx.driver.ui.components.DriverLoadingState
import com.pegasusx.driver.ui.components.DriverSectionTitle
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import com.pegasusx.driver.ui.components.DriverTodayKpiCard
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.PulseStrip
import com.pegasusx.driver.ui.components.StaggeredAppear
import com.pegasusx.driver.ui.components.StatusPill
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import com.pegasusx.driver.ui.screens.home.components.FactorySupplyCard
import com.pegasusx.driver.ui.screens.home.components.MapButton
import com.pegasusx.driver.ui.screens.home.components.QuickActionsSection
import com.pegasusx.driver.ui.screens.home.components.RecentActivitySection
import com.pegasusx.driver.ui.screens.home.components.ReturningToWarehouseCard
import com.pegasusx.driver.ui.screens.home.components.TodaySummaryCard
import com.pegasusx.driver.ui.screens.home.components.TransitControlCard
import com.pegasusx.driver.ui.screens.home.components.VehicleInfoCard
import com.pegasusx.driver.ui.screens.manifest.ManifestViewModel
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.MotionTokens
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.ui.theme.pressable
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Locale

@Composable
fun HomeScreen(
    api: DriverApi,
    viewModel: ManifestViewModel,
    onOpenMap: () -> Unit,
    onScanQR: () -> Unit,
    onOfflineVerify: () -> Unit = {},
    onResumeCashCollection: (orderId: String, amount: Long) -> Unit = { _, _ -> },
    onNotificationsClick: () -> Unit = {},
    onOpenSupplyTransfers: () -> Unit = {},
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current
    var returnLines by remember { mutableStateOf<List<com.pegasusx.driver.data.model.ReturnGoodsLine>>(emptyList()) }
    var returnUnits by remember { mutableStateOf(0L) }
    var pulseEvents by remember { mutableStateOf<List<PulseEvent>>(emptyList()) }
    var pulseLoading by remember { mutableStateOf(true) }
    var showRescueSheet by remember { mutableStateOf(false) }

    if (showRescueSheet) {
        RequestRescueSheet(
            api = api,
            onDismiss = { showRescueSheet = false },
        )
    }

    LaunchedEffect(Unit) {
        pulseLoading = true
        try {
            val response = api.getPulse()
            pulseEvents = response.events
        } catch (_: Exception) {
            pulseEvents = emptyList()
        } finally {
            pulseLoading = false
        }
    }

    LaunchedEffect(state.isReturning) {
        if (!state.isReturning) {
            returnLines = emptyList()
            returnUnits = 0
            return@LaunchedEffect
        }
        try {
            val resp = api.getReturnGoods()
            returnLines = resp.items
            returnUnits = resp.totalUnits
        } catch (_: Exception) { }
    }

    if (state.isLoading) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(lab.bg)
                .padding(horizontal = PegasusSpacing.s16, vertical = PegasusSpacing.s24),
            contentAlignment = Alignment.Center,
        ) {
            DriverLoadingState(
                title = stringResource(R.string.mobile_driver_ui_loading_your_route),
                body = "Checking manifest assignments, vehicle profile, and delivery status.",
                compact = true,
            )
        }
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
                        contentDescription = stringResource(R.string.portal_nav_notifications),
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s16))

        StaggeredAppear(index = 1) {
            PulseStrip(
                events = pulseEvents,
                loading = pulseLoading,
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Status Chips
        StaggeredAppear(index = 2) {
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
            StaggeredAppear(index = 3) {
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
        StaggeredAppear(index = 4) {
            if (state.isReturning) {
                ReturningToWarehouseCard(
                    returnLines = returnLines,
                    totalUnits = returnUnits,
                    onNavigate = { viewModel.state.value },
                    onArrived = { viewModel.returnComplete() },
                    showCashRecon = state.showCashRecon,
                    declaredCashMinor = state.declaredCashMinor,
                    onDeclaredCashChange = viewModel::updateDeclaredCash,
                    onSubmitCashRecon = { viewModel.submitCashReconciliation() },
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
        StaggeredAppear(index = 5) {
            TodaySummaryCard(orders = state.orders)
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Open Map CTA
        StaggeredAppear(index = 6) {
            MapButton(
                pendingCount = state.orders.count {
                    it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
                },
                onOpenMap = onOpenMap
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        if (state.pendingCollections.isNotEmpty()) {
            StaggeredAppear(index = 7) {
                val pending = state.pendingCollections.first()
                PegasusCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .pressable {
                            onResumeCashCollection(pending.orderId, pending.amount)
                        },
                ) {
                    Column(modifier = Modifier.padding(PegasusSpacing.s16)) {
                        Text(
                            text = stringResource(R.string.mobile_driver_ui_pending_cash_collection),
                            fontWeight = FontWeight.Bold,
                            color = lab.fg,
                        )
                        Text(
                            text = stringResource(R.string.mobile_driver_ui_order_takelast_formattedamount, pending.orderId.takeLast(6), pending.amount.formattedAmount()),
                            fontSize = 13.sp,
                            color = lab.fgTertiary,
                        )
                    }
                }
            }
            Spacer(modifier = Modifier.height(PegasusSpacing.s20))
        }

        // MARK: - Factory supply (factory-scoped drivers)
        if (TokenHolder.isFactoryScopedDriver()) {
            StaggeredAppear(index = 7) {
                FactorySupplyCard(onOpenSupplyTransfers = onOpenSupplyTransfers)
            }
            Spacer(modifier = Modifier.height(PegasusSpacing.s20))
        }

        // MARK: - Quick Actions
        StaggeredAppear(index = 8) {
            QuickActionsSection(
                onScanQR = onScanQR,
                onOfflineVerify = onOfflineVerify,
                onRequestRescue = { showRescueSheet = true },
                hasArrivedOrder = state.orders.any { it.state == OrderState.ARRIVED }
            )
        }

        Spacer(modifier = Modifier.height(PegasusSpacing.s20))

        // MARK: - Recent Activity
        StaggeredAppear(index = 9) {
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
                label = stringResource(R.string.mobile_driver_ui_returning),
                active = true
            )
            hasActiveRoute -> StatusChip(
                icon = Icons.Default.Sync,
                label = stringResource(R.string.mobile_driver_ui_on_route),
                active = true
            )
            else -> StatusChip(
                icon = Icons.Default.ShieldMoon,
                label = stringResource(R.string.mobile_driver_ui_idle),
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
