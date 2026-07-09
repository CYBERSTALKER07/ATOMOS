package com.pegasusx.retailer.ui.components

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane

@Composable
fun PegasusEmptyState(
    icon: ImageVector,
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    PegasusStatePane(
        kind = PegasusStateKind.Empty,
        headline = title,
        body = message,
        modifier = modifier,
        actionLabel = actionLabel,
        onAction = onAction
    )
}
