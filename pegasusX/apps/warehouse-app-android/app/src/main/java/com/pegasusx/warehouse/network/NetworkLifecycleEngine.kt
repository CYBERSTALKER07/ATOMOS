package com.pegasusx.warehouse.network

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlin.random.Random

enum class NetworkState {
    CONNECTED,
    CONNECTING,
    OFFLINE,
    FLAPPING,
    CAPTIVE_BLOCKED
}

class NetworkLifecycleEngine(
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
) {
    private val _state = MutableStateFlow(NetworkState.CONNECTED)
    val state: StateFlow<NetworkState> = _state.asStateFlow()

    private val _isOnline = MutableStateFlow(true)
    val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()

    private val _syncEpoch = MutableStateFlow(0)
    val syncEpoch: StateFlow<Int> = _syncEpoch.asStateFlow()

    private val transitionTimestamps = mutableListOf<Long>()
    private var flappingJob: Job? = null
    private var reconnectJitterJob: Job? = null
    private var isTesting: Boolean = false

    @Volatile
    private var lastRawOnline: Boolean = true

    fun initialize(context: Context) {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()

        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                if (!isTesting) {
                    recordTransition(online = true)
                }
            }

            override fun onLost(network: Network) {
                if (!isTesting) {
                    recordTransition(online = false)
                }
            }

            override fun onUnavailable() {
                if (!isTesting) {
                    recordTransition(online = false)
                }
            }
        }

        try {
            cm.registerNetworkCallback(request, callback)
        } catch (e: Exception) {
            // Unit tests or missing runtime context
        if (e is kotlinx.coroutines.CancellationException) throw e
        }
    }

    @Synchronized
    fun recordTransition(online: Boolean) {
        val now = System.currentTimeMillis()
        lastRawOnline = online

        // Retain transitions within the last 30 seconds
        transitionTimestamps.removeAll { now - it >= 30_000L }
        transitionTimestamps.add(now)

        if (transitionTimestamps.size >= 3) {
            _state.value = NetworkState.FLAPPING
            _isOnline.value = false
            reconnectJitterJob?.cancel()
            flappingJob?.cancel()
            flappingJob = scope.launch {
                delay(15_000L)
                synchronized(this@NetworkLifecycleEngine) {
                    val currentOnline = lastRawOnline
                    _state.value = if (currentOnline) NetworkState.CONNECTED else NetworkState.OFFLINE
                    _isOnline.value = currentOnline
                    if (currentOnline) {
                        _syncEpoch.value += 1
                    }
                }
            }
            return
        }

        flappingJob?.cancel()
        if (online) {
            _state.value = NetworkState.CONNECTED
            _isOnline.value = true
            reconnectJitterJob?.cancel()
            val jitterMs = Random.nextLong(200L, 801L)
            reconnectJitterJob = scope.launch {
                delay(jitterMs)
                _syncEpoch.value += 1
            }
        } else {
            reconnectJitterJob?.cancel()
            _state.value = NetworkState.OFFLINE
            _isOnline.value = false
        }
    }

    suspend fun triggerSyncEpoch() {
        _syncEpoch.value += 1
    }

    suspend fun waitForOnline() {
        while (!_isOnline.value) {
            delay(200L)
        }
    }

    // Testing hooks
    fun resetForTesting() {
        isTesting = true
        flappingJob?.cancel()
        flappingJob = null
        reconnectJitterJob?.cancel()
        reconnectJitterJob = null
        synchronized(this) {
            transitionTimestamps.clear()
            lastRawOnline = true
            _isOnline.value = true
            _state.value = NetworkState.CONNECTED
            _syncEpoch.value = 0
        }
    }

    fun setOnlineForTesting(online: Boolean) {
        isTesting = true
        recordTransition(online)
    }

    companion object {
        val shared = NetworkLifecycleEngine()
    }
}
