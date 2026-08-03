package com.pegasusx.retailer.ui.components

import androidx.compose.runtime.Composable
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.MediaUploadService
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.Order
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject

@HiltViewModel
class FileClaimDepsViewModel @Inject constructor(
    val api: PegasusApi,
    val mediaUpload: MediaUploadService,
) : ViewModel()

/** Hilt-backed host for [FileClaimSheet]. */
@Composable
fun FileClaimHost(
    order: Order,
    onDismiss: () -> Unit,
    viewModel: FileClaimDepsViewModel = hiltViewModel(),
) {
    FileClaimSheet(
        order = order,
        api = viewModel.api,
        mediaUpload = viewModel.mediaUpload,
        onDismiss = onDismiss,
    )
}
