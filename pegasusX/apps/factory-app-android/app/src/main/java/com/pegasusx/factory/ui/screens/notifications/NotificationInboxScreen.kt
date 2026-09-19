package com.pegasusx.factory.ui.screens.notifications

import androidx.compose.ui.res.stringResource

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.pegasusx.factory.data.model.MarkNotificationsReadRequest
import com.pegasusx.factory.data.model.NotificationItem
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotificationInboxScreen(
    api: FactoryApi,
    onBack: (() -> Unit)? = null,
) {
    var items by remember { mutableStateOf<List<NotificationItem>>(emptyList()) }
    var unreadCount by remember { mutableStateOf(0) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getNotifications(limit = 50, offset = 0)
                if (resp.isSuccessful && resp.body() != null) {
                    val page = resp.body()!!
                    items = page.notifications
                    unreadCount = page.unreadCount
                } else {
                    error = when (resp.code()) {
                        503 -> "Notifications unavailable on this stack"
                        else -> "Failed to load notifications (${resp.code()})"
                    }
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        if (unreadCount > 0) "Notifications ($unreadCount)" else "Notifications",
                        fontWeight = FontWeight.Bold,
                    )
                },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = stringResource(R.string.portal_page_orders_action_refresh))
                    }
                    if (unreadCount > 0) {
                        TextButton(
                            onClick = {
                                scope.launch {
                                    api.markNotificationsRead(MarkNotificationsReadRequest(markAll = true))
                                    load()
                                }
                            },
                        ) {
                            Text("Mark all read")
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            modifier = Modifier.fillMaxSize().padding(innerPadding),
        ) {
            when {
                loading && items.isEmpty() -> item(span = { GridItemSpan(maxLineSpan) }) {
                    PegasusLoadingState(
                        title = stringResource(R.string.mobile_factory_ui_loading_notifications),
                        body = "Fetching your latest alerts and messages."
                    )
                }
                error != null && items.isEmpty() -> item(span = { GridItemSpan(maxLineSpan) }) {
                    PegasusStatePane(
                        kind = PegasusStateKind.Error,
                        headline = "Unable to load notifications",
                        body = error!!,
                        actionLabel = "Retry",
                        onAction = { load() }
                    )
                }
                items.isEmpty() -> item(span = { GridItemSpan(maxLineSpan) }) {
                    PegasusStatePane(
                        kind = PegasusStateKind.Empty,
                        headline = "No notifications",
                        body = "You're all caught up."
                    )
                }
                else -> {
                    items(items, key = { it.id }) { item ->
                        ListItem(
                            headlineContent = {
                                Text(
                                    item.title.ifBlank { item.type.ifBlank { "Notification" } },
                                    fontWeight = if (item.readAt == null) FontWeight.SemiBold else FontWeight.Normal,
                                )
                            },
                            supportingContent = {
                                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                    if (item.body.isNotBlank()) {
                                        Text(item.body, style = MaterialTheme.typography.bodySmall)
                                    }
                                    if (item.createdAt.isNotBlank()) {
                                        Text(
                                            item.createdAt,
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                    }
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        HorizontalDivider()
                    }
                }
            }
        }
    }
}
