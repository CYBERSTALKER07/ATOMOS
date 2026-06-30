package com.pegasus.retailer.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface CartCacheDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(cache: CartCacheEntity)

    @Query("SELECT * FROM cart_cache WHERE retailerId = :retailerId LIMIT 1")
    suspend fun getForRetailer(retailerId: String): CartCacheEntity?

    @Query("DELETE FROM cart_cache WHERE retailerId = :retailerId")
    suspend fun clear(retailerId: String)
}
