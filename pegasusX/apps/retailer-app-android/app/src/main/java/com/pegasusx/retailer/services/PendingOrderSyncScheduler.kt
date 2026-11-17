package com.pegasusx.retailer.services

import android.content.Context
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager

fun interface PendingOrderSyncEnqueuer {
    fun enqueue()
}

class DefaultPendingOrderSyncEnqueuer(
    private val context: Context,
) : PendingOrderSyncEnqueuer {
    override fun enqueue() {
        PendingOrderSyncScheduler.enqueue(context)
    }
}

object PendingOrderSyncScheduler {
    fun enqueue(context: Context) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()
        val request = OneTimeWorkRequestBuilder<PendingOrderSyncWorker>()
            .setConstraints(constraints)
            .build()
        WorkManager.getInstance(context)
            .enqueueUniqueWork(
                PendingOrderSyncWorker.WORK_NAME,
                ExistingWorkPolicy.APPEND_OR_REPLACE,
                request,
            )
    }
}
