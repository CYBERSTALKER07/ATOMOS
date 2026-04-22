package com.thelab.driver.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import com.thelab.driver.data.model.OrderEntity
import com.thelab.driver.data.model.PendingMutationEntity
import com.thelab.driver.data.model.RouteManifestEntity
import kotlinx.coroutines.flow.Flow

@Dao
interface OrderDao {

    @Query("SELECT * FROM orders ORDER BY createdAt DESC")
    fun observeAll(): Flow<List<OrderEntity>>

    @Query("SELECT * FROM orders WHERE state IN ('LOADED', 'IN_TRANSIT', 'ARRIVING', 'ARRIVED', 'AWAITING_PAYMENT') ORDER BY createdAt ASC")
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

    @Query("SELECT * FROM pending_mutations ORDER BY createdAt ASC")
    suspend fun getAll(): List<PendingMutationEntity>

    @Query("SELECT COUNT(*) FROM pending_mutations")
    fun observeCount(): Flow<Int>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(mutation: PendingMutationEntity)

    @Query("DELETE FROM pending_mutations WHERE id = :id")
    suspend fun deleteById(id: String)
}
