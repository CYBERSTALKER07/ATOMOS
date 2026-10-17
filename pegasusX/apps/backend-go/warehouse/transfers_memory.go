package warehouse

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"os"
	"strings"
)

type memoryTransferRow struct {
	TransferID    string
	FactoryID     string
	SupplierID    string
	State         string
	TotalVolumeVU float64
	Notes         string
}

func (s *Service) memoryTransfersEnabled() bool {
	return s != nil && s.spannerClient == nil
}

func (s *Service) ensureMemoryTransferStoreLocked() {
	if s.internalTransfers == nil {
		s.internalTransfers = make(map[string]memoryTransferRow)
	}
}

func (s *Service) memoryFactoryID() string {
	if id := strings.TrimSpace(os.Getenv("FACTORY_DEMO_ID")); id != "" {
		return id
	}
	return "factory-demo-1"
}

func (s *Service) memoryResolveWarehouseFactory(_ context.Context, warehouseID string) (factoryID, supplierID string, err error) {
	_ = warehouseID
	supplierID = strings.TrimSpace(s.seedSupplierID)
	if supplierID == "" {
		supplierID = "supplier-demo"
	}
	return s.memoryFactoryID(), supplierID, nil
}

func (s *Service) memoryCreateEmergencyTransfer(
	whID string,
	factoryID string,
	supplierID string,
	totalVolumeVU float64,
	notes string,
) memoryTransferRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMemoryTransferStoreLocked()
	transferID := uuid.NewString()
	row := memoryTransferRow{
		TransferID:    transferID,
		FactoryID:     factoryID,
		SupplierID:    supplierID,
		State:         "APPROVED",
		TotalVolumeVU: totalVolumeVU,
		Notes:         strings.TrimSpace(notes),
	}
	s.internalTransfers[transferID] = row
	_ = whID
	return row
}

func (s *Service) memoryForceReceiveTransfer(
	factoryID string,
	supplierID string,
	totalVolumeVU float64,
	notes string,
) memoryTransferRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMemoryTransferStoreLocked()
	transferID := uuid.NewString()
	row := memoryTransferRow{
		TransferID:    transferID,
		FactoryID:     factoryID,
		SupplierID:    supplierID,
		State:         "RECEIVED",
		TotalVolumeVU: totalVolumeVU,
		Notes:         strings.TrimSpace(notes),
	}
	s.internalTransfers[transferID] = row
	return row
}

func (s *Service) memoryReceiveTransfer(ops *auth.WarehouseOps, transferID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMemoryTransferStoreLocked()
	row, ok := s.internalTransfers[transferID]
	if !ok {
		return errTransferNotFound
	}
	if ops != nil && ops.SupplierID != "" && row.SupplierID != ops.SupplierID {
		return errTransferForbidden
	}
	if _, ok := receiveableTransferStates[strings.ToUpper(row.State)]; !ok {
		return fmt.Errorf("%w: %s", errInvalidTransfer, row.State)
	}
	row.State = "RECEIVED"
	s.internalTransfers[transferID] = row
	return nil
}

func (s *Service) memoryUpsertTransferLocked(row memoryTransferRow) {
	s.ensureMemoryTransferStoreLocked()
	s.internalTransfers[row.TransferID] = row
}

func (s *Service) ensureMemoryDemoReceiveTransferLocked() {
	s.ensureMemoryTransferStoreLocked()
	const demoID = "ssmr-wh-transfer-receive"
	if _, exists := s.internalTransfers[demoID]; exists {
		return
	}
	s.internalTransfers[demoID] = memoryTransferRow{
		TransferID:    demoID,
		FactoryID:     s.memoryFactoryID(),
		SupplierID:    strings.TrimSpace(s.seedSupplierID),
		State:         "IN_TRANSIT",
		TotalVolumeVU: 12,
	}
}
