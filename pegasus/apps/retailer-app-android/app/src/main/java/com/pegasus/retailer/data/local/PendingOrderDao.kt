package com.pegasus.retailer.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query

@Dao
interface PendingOrderDao {
    @Insert
    suspend fun insert(order: PendingOrderEntity)

    @Query("SELECT * FROM pending_orders ORDER BY createdAt ASC")
    suspend fun getAll(): List<PendingOrderEntity>

    @Query("DELETE FROM pending_orders WHERE id = :id")
    suspend fun deleteById(id: Long): Int

    @Query("UPDATE pending_orders SET retryCount = retryCount + 1, lastError = :lastError WHERE id = :id")
    suspend fun incrementRetry(id: Long, lastError: String): Int
}
