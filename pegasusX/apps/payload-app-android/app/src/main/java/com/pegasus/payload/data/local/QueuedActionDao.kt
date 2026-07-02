package com.pegasus.payload.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import kotlinx.coroutines.flow.Flow

@Dao
interface QueuedActionDao {
    @Query("SELECT COUNT(*) FROM queued_actions WHERE endpoint LIKE '%' || :path || '%'")
    fun countByEndpointFlow(path: String): Flow<Int>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(action: QueuedActionEntity)

    @Query("SELECT * FROM queued_actions ORDER BY timestamp ASC")
    suspend fun getAll(): List<QueuedActionEntity>

    @Query("DELETE FROM queued_actions WHERE id = :id")
    suspend fun deleteById(id: String)
    
    @Query("SELECT COUNT(*) FROM queued_actions")
    suspend fun count(): Int
}
