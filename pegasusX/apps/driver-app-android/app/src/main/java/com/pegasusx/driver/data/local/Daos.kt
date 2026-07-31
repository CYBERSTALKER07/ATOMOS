package com.pegasusx.driver.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import com.pegasusx.driver.data.model.OrderEntity
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.RouteManifestEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface OrderDao {

    @Query("SELECT * FROM orders ORDER BY createdAt DESC")
    fun observeAll(): Flow<List<OrderEntity>>

    @Query("SELECT * FROM orders WHERE state IN ('LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT', 'PENDING_CASH_COLLECTION', 'FISCALIZING', 'FISCAL_FAILED') ORDER BY createdAt ASC")
    fun observeActive(): Flow<List<OrderEntity>>

    @Query("SELECT * FROM orders WHERE id = :orderId")
    suspend fun getById(orderId: String): OrderEntity?

    @Query("SELECT * FROM orders WHERE state = :state")
    suspend fun getByState(state: String): List<OrderEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertAll(orders: List<OrderEntity>)

    @Query("UPDATE orders SET state = :newState, updatedAt = :updatedAt WHERE id = :orderId")
    suspend fun updateState(orderId: String, newState: String, updatedAt: String): Int

    @Query("DELETE FROM orders")
    suspend fun clearAll(): Int
}

@Dao
interface RouteManifestDao {

    @Query("SELECT * FROM route_manifest WHERE date = :date")
    suspend fun getForDate(date: String): RouteManifestEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(manifest: RouteManifestEntity)

    @Query("DELETE FROM route_manifest")
    suspend fun clearAll(): Int
}

@Dao
interface PendingMutationDao {

    @Query("SELECT * FROM pending_mutations WHERE status = 'PENDING' ORDER BY priority ASC, createdAt ASC")
    suspend fun getPending(): List<PendingMutationEntity>

    @Query("SELECT * FROM pending_mutations WHERE status = 'DEAD' ORDER BY createdAt DESC")
    suspend fun getDead(): List<PendingMutationEntity>

    @Query("SELECT * FROM pending_mutations ORDER BY priority ASC, createdAt ASC")
    suspend fun getAll(): List<PendingMutationEntity>

    @Query("SELECT COUNT(*) FROM pending_mutations WHERE status = 'PENDING'")
    fun observePendingCount(): Flow<Int>

    @Query("SELECT COUNT(*) FROM pending_mutations WHERE status = 'PENDING'")
    fun observeCount(): Flow<Int>

    @Query("SELECT * FROM pending_mutations WHERE status = 'PENDING' ORDER BY priority ASC, createdAt ASC")
    fun observePending(): Flow<List<PendingMutationEntity>>

    @Query("SELECT * FROM pending_mutations WHERE status = 'DEAD' ORDER BY createdAt DESC")
    fun observeDead(): Flow<List<PendingMutationEntity>>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(mutation: PendingMutationEntity)

    @Query("DELETE FROM pending_mutations WHERE id = :id")
    suspend fun deleteById(id: String)

    @Query("DELETE FROM pending_mutations WHERE idempotencyKey = :key")
    suspend fun deleteByIdempotencyKey(key: String)

    @Query(
        """
        UPDATE pending_mutations
        SET attemptCount = attemptCount + 1, lastError = :error
        WHERE id = :id
        """,
    )
    suspend fun recordAttempt(id: String, error: String)

    @Query(
        """
        UPDATE pending_mutations
        SET status = 'DEAD', lastError = :error, attemptCount = attemptCount + 1
        WHERE id = :id
        """,
    )
    suspend fun markDead(id: String, error: String)

    @Query("DELETE FROM pending_mutations WHERE status = 'DEAD'")
    suspend fun clearDead()
}

@Dao
interface TelemetryDao {

    @Query("SELECT * FROM telemetry_locations ORDER BY timestamp ASC")
    suspend fun getAll(): List<com.pegasusx.driver.data.model.TelemetryLocationEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(location: com.pegasusx.driver.data.model.TelemetryLocationEntity)

    @Query("DELETE FROM telemetry_locations WHERE id IN (:ids)")
    suspend fun deleteByIds(ids: List<String>)
}
