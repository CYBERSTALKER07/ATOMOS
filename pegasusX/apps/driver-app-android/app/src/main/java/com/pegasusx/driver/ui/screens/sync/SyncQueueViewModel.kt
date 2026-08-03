package com.pegasusx.driver.ui.screens.sync

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.local.PendingMutationDao
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.services.OfflineSyncScheduler
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

@HiltViewModel
class SyncQueueViewModel @Inject constructor(
    private val dao: PendingMutationDao,
    @ApplicationContext private val appContext: Context,
) : ViewModel() {

    val pending: StateFlow<List<PendingMutationEntity>> = dao.observePending()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    val dead: StateFlow<List<PendingMutationEntity>> = dao.observeDead()
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    private val _flushing = MutableStateFlow(false)
    val flushing: StateFlow<Boolean> = _flushing.asStateFlow()

    private val _statusMessage = MutableStateFlow<String?>(null)
    val statusMessage: StateFlow<String?> = _statusMessage.asStateFlow()

    fun flushNow() {
        viewModelScope.launch {
            _flushing.value = true
            _statusMessage.value = "Flush scheduled"
            OfflineSyncScheduler.enqueue(appContext)
            _flushing.value = false
        }
    }

    fun dismissDead() {
        viewModelScope.launch {
            dao.clearDead()
            _statusMessage.value = "Dead-letter cleared"
        }
    }
}
