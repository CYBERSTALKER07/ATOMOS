package com.pegasusx.enterprise.ui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.enterprise.theme.SwissCADTheme
import com.pegasusx.enterprise.viewmodel.EnterpriseViewModel

class MainActivity : ComponentActivity() {

    private val viewModel: EnterpriseViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        setContent {
            SwissCADTheme {
                val uiState by viewModel.uiState.collectAsStateWithLifecycle()
                val snackbarHostState = remember { SnackbarHostState() }

                LaunchedEffect(uiState.userNotice) {
                    uiState.userNotice?.let { notice ->
                        snackbarHostState.showSnackbar(notice)
                        viewModel.clearNotice()
                    }
                }

                EnterpriseDashboardScreen(
                    uiState = uiState,
                    onSearchQueryChanged = viewModel::onSearchQueryChanged,
                    onCategorySelected = viewModel::onCategorySelected,
                    onStatusFilterSelected = viewModel::onStatusFilterSelected,
                    onSortOptionSelected = viewModel::onSortOptionSelected,
                    onRefresh = { viewModel.loadData(isPullToRefresh = true) },
                    onInquiryClick = viewModel::onInquirySelected,
                    onQuickStatusAdvance = viewModel::onStatusAdvanced,
                    snackbarHostState = snackbarHostState
                )

                uiState.activeInquiry?.let { req ->
                    RequestDetailSheet(
                        request = req,
                        isSendingReply = uiState.isSendingReply,
                        onDismiss = viewModel::onInquiryDismissed,
                        onStatusChange = { newStatus ->
                            viewModel.onStatusAdvanced(req.id, newStatus)
                        },
                        onDelete = {
                            viewModel.onDeleteInquiry(req.id)
                        },
                        onSendReply = { reqId, recipient, subject, message ->
                            viewModel.onSendReply(reqId, recipient, subject, message)
                        }
                    )
                }
            }
        }
    }
}
