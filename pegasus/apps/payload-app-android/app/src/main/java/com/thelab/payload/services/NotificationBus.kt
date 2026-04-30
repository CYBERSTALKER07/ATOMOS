package com.pegasus.payload.services

import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Process-wide bus for "user tapped a notification that should deep-link into
 * the notifications panel". [MainActivity] emits on every intent carrying the
 * [EXTRA_OPEN_NOTIFICATIONS] flag; HomeViewModel collects and toggles the panel.
 */
@Singleton
class NotificationBus @Inject constructor() {
    private val _openPanel = MutableSharedFlow<Unit>(
        replay = 1,
        extraBufferCapacity = 1,
        onBufferOverflow = BufferOverflow.DROP_OLDEST,
    )
    val openPanel: SharedFlow<Unit> = _openPanel.asSharedFlow()

    fun requestOpenPanel() {
        _openPanel.tryEmit(Unit)
    }

    companion object {
        const val EXTRA_OPEN_NOTIFICATIONS = "open_notifications"
        const val CHANNEL_ID = "payload_default"
        const val CHANNEL_NAME = "Payload alerts"
    }
}
