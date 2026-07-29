package com.pegasusx.factory.ui.screens.loadingbay.components

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable

@Composable
fun LoadingBayControls(
    dispatching: Boolean,
    onClick: () -> Unit
) {
    ExtendedFloatingActionButton(
        text = { Text(if (dispatching) "Dispatching…" else "Batch Dispatch") },
        icon = { Icon(Icons.Default.LocalShipping, null) },
        onClick = onClick,
    )
}
