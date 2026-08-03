package com.pegasusx.retailer.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface PendingPosSaleDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(sale: PendingPosSaleEntity)

    @Query("SELECT * FROM pending_pos_sales ORDER BY createdAt ASC")
    suspend fun getAll(): List<PendingPosSaleEntity>

    @Query("SELECT * FROM pending_pos_sales WHERE status IN ('PENDING','FAILED','SYNCING') ORDER BY createdAt ASC")
    suspend fun getActive(): List<PendingPosSaleEntity>

    @Query("SELECT COUNT(*) FROM pending_pos_sales WHERE sessionId = :sessionId AND status IN ('PENDING','FAILED','SYNCING')")
    suspend fun countActiveForSession(sessionId: String): Int

    @Query("DELETE FROM pending_pos_sales WHERE id = :id")
    suspend fun delete(id: String)

    @Query(
        "UPDATE pending_pos_sales SET status = :status, retryCount = :retryCount, lastError = :lastError, " +
            "serverSaleId = :serverSaleId, serverReceiptNumber = :serverReceipt WHERE id = :id",
    )
    suspend fun updateStatus(
        id: String,
        status: String,
        retryCount: Int,
        lastError: String?,
        serverSaleId: String?,
        serverReceipt: String?,
    )
}
