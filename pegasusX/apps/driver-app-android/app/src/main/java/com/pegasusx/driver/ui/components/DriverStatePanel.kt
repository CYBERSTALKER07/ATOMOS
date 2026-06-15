package com.pegasusx.driver.ui.components

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector

/** @see DriverLoadingState */
@Composable
fun DriverLoadingStatePanel(
    title: String,
    message: String,
    modifier: Modifier = Modifier,
) {
    DriverLoadingState(
        title = title,
        body = message,
        modifier = modifier,
        shimmerLines = true,
    )
}

/** @see DriverStatePane */
@Composable
fun DriverStatePanel(
    icon: ImageVector,
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    DriverStatePane(
        kind = DriverStateKind.Route,
        headline = title,
        body = message,
        modifier = modifier,
        actionLabel = actionLabel,
        onAction = onAction,
        iconOverride = icon,
        usePegasusCard = true,
    )
}
