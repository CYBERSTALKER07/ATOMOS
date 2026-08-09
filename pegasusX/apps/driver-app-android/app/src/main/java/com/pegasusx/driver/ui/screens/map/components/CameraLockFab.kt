package com.pegasusx.driver.ui.screens.map.components

import androidx.compose.ui.res.stringResource

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.driver.R

@Composable
fun CameraLockFab(
    isCameraLocked: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    FloatingActionButton(
        onClick = onClick,
        modifier = modifier,
        containerColor = if (isCameraLocked) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
        contentColor = if (isCameraLocked) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface
    ) {
        Icon(Icons.Default.MyLocation, contentDescription = stringResource(R.string.mobile_driver_ui_lock_camera))
    }
}
