package com.pegasus.payload.data.repository

import com.pegasus.payload.data.local.SecureStore
import com.pegasus.payload.data.model.LoginRequest
import com.pegasus.payload.data.model.LoginResponse
import com.pegasus.payload.data.remote.PayloadApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * AuthRepository — owns login + cold-start session restore + logout.
 *
 * Surfaces a [session] StateFlow so composables can route on auth state without
 * polling SecureStore directly. Session is restored eagerly from secure prefs
 * on first construction (Hilt @Singleton).
 */
@Singleton
class AuthRepository @Inject constructor(
    private val api: PayloadApi,
    private val secureStore: SecureStore,
) {

    data class Session(
        val token: String,
        val workerId: String,
        val supplierId: String,
        val name: String,
        val warehouseId: String,
        val warehouseName: String,
    )

    private val _session = MutableStateFlow<Session?>(restore())
    val session: StateFlow<Session?> = _session.asStateFlow()

    val isAuthenticated: Boolean get() = _session.value != null

    private fun restore(): Session? {
        val token = secureStore.token ?: return null
        return Session(
            token = token,
            workerId = "", // worker_id not persisted in current Expo flow; reserved for future
            supplierId = secureStore.supplierId.orEmpty(),
            name = secureStore.name.orEmpty(),
            warehouseId = secureStore.warehouseId.orEmpty(),
            warehouseName = secureStore.warehouseName.orEmpty(),
        )
    }

    suspend fun login(phone: String, pin: String): Result<Session> = runCatching {
        val resp: LoginResponse = api.login(LoginRequest(phone = phone, pin = pin))
        secureStore.token = resp.token
        secureStore.name = resp.name
        secureStore.supplierId = resp.supplierId
        secureStore.warehouseId = resp.warehouseId
        secureStore.warehouseName = resp.warehouseName
        secureStore.firebaseToken = resp.firebaseToken
        val s = Session(
            token = resp.token,
            workerId = resp.workerId,
            supplierId = resp.supplierId,
            name = resp.name,
            warehouseId = resp.warehouseId,
            warehouseName = resp.warehouseName,
        )
        _session.value = s
        s
    }

    fun logout() {
        secureStore.clear()
        _session.value = null
    }
}
