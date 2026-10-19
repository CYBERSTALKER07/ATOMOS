package com.pegasus.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

// UI Re-exports
typealias PegasusSpacing = com.pegasus.design.ui.PegasusSpacing
typealias PegasusNeutralColors = com.pegasus.design.ui.PegasusNeutralColors
typealias PulseHonesty = com.pegasus.design.ui.PulseHonesty
typealias PegasusRailGroup = com.pegasus.design.ui.PegasusRailGroup
typealias PegasusRailItem = com.pegasus.design.ui.PegasusRailItem
typealias PegasusStateKind = com.pegasus.design.ui.PegasusStateKind
typealias PegasusRuntimeTone = com.pegasus.design.ui.PegasusRuntimeTone
typealias PegasusMotionTokens = com.pegasus.design.ui.PegasusMotionTokens
typealias StatusStackMode = com.pegasus.design.ui.StatusStackMode
typealias StatusStackRow = com.pegasus.design.ui.StatusStackRow
typealias StatusStackModel = com.pegasus.design.ui.StatusStackModel

val ORDER_STATUS_FUNNEL = com.pegasus.design.ui.ORDER_STATUS_FUNNEL
val MANIFEST_STATES = com.pegasus.design.ui.MANIFEST_STATES
val TRUCK_DUTY_STATUSES = com.pegasus.design.ui.TRUCK_DUTY_STATUSES
val FACTORY_TRANSFER_STATES = com.pegasus.design.ui.FACTORY_TRANSFER_STATES
val FACTORY_VEHICLE_STATES = com.pegasus.design.ui.FACTORY_VEHICLE_STATES
val FACTORY_DRIVER_DUTY = com.pegasus.design.ui.FACTORY_DRIVER_DUTY

fun canonicalizeOrderStatus(status: String): String =
    com.pegasus.design.ui.canonicalizeOrderStatus(status)

fun incrementOrderStatusCount(counts: Map<String, Int>?, status: String): Map<String, Int> =
    com.pegasus.design.ui.incrementOrderStatusCount(counts, status)

fun statusStackModel(
    dictionary: List<String> = com.pegasus.design.ui.ORDER_STATUS_FUNNEL,
    counts: Map<String, Int>?,
    available: Boolean = true,
): com.pegasus.design.ui.StatusStackModel =
    com.pegasus.design.ui.statusStackModel(dictionary, counts, available)

@Composable
fun StatusStack(
    counts: Map<String, Int>?,
    modifier: Modifier = Modifier,
    dictionary: List<String> = com.pegasus.design.ui.ORDER_STATUS_FUNNEL,
    available: Boolean = true,
    source: String? = null,
    onSelect: ((String) -> Unit)? = null,
) {
    com.pegasus.design.ui.StatusStack(
        counts = counts,
        modifier = modifier,
        dictionary = dictionary,
        available = available,
        source = source,
        onSelect = onSelect,
    )
}

@Composable
fun PegasusLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
) {
    com.pegasus.design.ui.PegasusLoadingState(
        title = title,
        body = body,
        modifier = modifier,
    )
}

@Composable
fun PegasusStatePane(
    kind: com.pegasus.design.ui.PegasusStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    com.pegasus.design.ui.PegasusStatePane(
        kind = kind,
        headline = headline,
        body = body,
        modifier = modifier,
        actionLabel = actionLabel,
        onAction = onAction,
    )
}

@Composable
fun PegasusRuntimeBanner(
    tone: com.pegasus.design.ui.PegasusRuntimeTone,
    message: String,
    modifier: Modifier = Modifier,
    onRetry: (() -> Unit)? = null,
) {
    com.pegasus.design.ui.PegasusRuntimeBanner(
        tone = tone,
        message = message,
        modifier = modifier,
        onRetry = onRetry,
    )
}

@Composable
fun PegasusMonochromeTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    typography: Typography = Typography(),
    content: @Composable () -> Unit,
) {
    com.pegasus.design.ui.PegasusMonochromeTheme(
        darkTheme = darkTheme,
        dynamicColor = dynamicColor,
        typography = typography,
        content = content,
    )
}

@Composable
fun PackBanner(pack: com.pegasus.design.network.MarketPack?, modifier: Modifier = Modifier) {
    com.pegasus.design.ui.PackBanner(pack = pack, modifier = modifier)
}

@Composable
fun PegasusCollapsibleRail(
    appTitle: String,
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    groups: List<com.pegasus.design.ui.PegasusRailGroup>,
    selectedItemId: String?,
    onItemSelected: (com.pegasus.design.ui.PegasusRailItem) -> Unit,
    modifier: Modifier = Modifier,
    footer: (@Composable () -> Unit)? = null,
) {
    com.pegasus.design.ui.PegasusCollapsibleRail(
        appTitle = appTitle,
        isExpanded = isExpanded,
        onToggleExpanded = onToggleExpanded,
        groups = groups,
        selectedItemId = selectedItemId,
        onItemSelected = onItemSelected,
        modifier = modifier,
        footer = footer,
    )
}

@Composable
fun SourceChip(source: String, modifier: Modifier = Modifier) {
    com.pegasus.design.ui.SourceChip(source = source, modifier = modifier)
}

// Network Re-exports
typealias MarketPack = com.pegasus.design.network.MarketPack
typealias MarketPackBinder = com.pegasus.design.network.MarketPackBinder
typealias MarketPackStore = com.pegasus.design.network.MarketPackStore
typealias CellPinInterceptor = com.pegasus.design.network.CellPinInterceptor
typealias PackMapCenter = com.pegasus.design.network.PackMapCenter

fun sessionMapCenter(): com.pegasus.design.network.PackMapCenter? =
    com.pegasus.design.network.sessionMapCenter()

fun packCurrency(pack: com.pegasus.design.network.MarketPack?, fallback: String = ""): String =
    com.pegasus.design.network.packCurrency(pack, fallback)

fun sessionPackCurrency(): String =
    com.pegasus.design.network.sessionPackCurrency()

