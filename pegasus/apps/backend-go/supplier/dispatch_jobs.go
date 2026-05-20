package supplier

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/optimizationjobs"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	contract "optimizercontract"
)

type dispatchJobActiveListResponse struct {
	Jobs     []dispatchJobSummary `json:"jobs"`
	Source   string               `json:"source"`
	Degraded bool                 `json:"degraded"`
}

type dispatchJobSummary struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	SolverType  string `json:"solver_type"`
	RequestedAt string `json:"requested_at"`
	UpdatedAt   string `json:"updated_at"`
	Ready       bool   `json:"ready"`
}

type dispatchJobProjectionResponse struct {
	JobID           string                    `json:"job_id"`
	Status          string                    `json:"status"`
	SolverType      string                    `json:"solver_type"`
	Ready           bool                      `json:"ready"`
	RequestedAt     string                    `json:"requested_at"`
	UpdatedAt       string                    `json:"updated_at"`
	CompletedAt     string                    `json:"completed_at,omitempty"`
	FailureCode     string                    `json:"failure_code,omitempty"`
	FailureMessage  string                    `json:"failure_message,omitempty"`
	TimedOut        bool                      `json:"timed_out,omitempty"`
	MatrixSize      int32                     `json:"matrix_size,omitempty"`
	ObjectiveCostKM float64                   `json:"objective_cost_km,omitempty"`
	Warnings        []string                  `json:"warnings"`
	Depot           *dispatchProjectionDepot  `json:"depot,omitempty"`
	Routes          []dispatchProjectionRoute `json:"routes"`
	Unassigned      []dispatchProjectionStop  `json:"unassigned"`
}

type dispatchProjectionDepot struct {
	NodeUUID string  `json:"node_uuid"`
	Label    string  `json:"label,omitempty"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type dispatchProjectionRoute struct {
	RouteID      string                   `json:"route_id,omitempty"`
	ManifestID   string                   `json:"manifest_id,omitempty"`
	VehicleUUID  string                   `json:"vehicle_uuid"`
	DriverUUID   string                   `json:"driver_uuid"`
	DriverName   string                   `json:"driver_name,omitempty"`
	VehicleType  string                   `json:"vehicle_type,omitempty"`
	VehicleClass string                   `json:"vehicle_class,omitempty"`
	CapacityVU   float64                  `json:"capacity_vu"`
	LoadVU       float64                  `json:"load_vu"`
	RouteCostKM  float64                  `json:"route_cost_km"`
	Stops        []dispatchProjectionStop `json:"stops"`
}

type dispatchProjectionStop struct {
	Sequence             int     `json:"sequence,omitempty"`
	NodeUUID             string  `json:"node_uuid"`
	OrderID              string  `json:"order_id,omitempty"`
	RetailerID           string  `json:"retailer_id,omitempty"`
	RetailerName         string  `json:"retailer_name,omitempty"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	Amount               int64   `json:"amount,omitempty"`
	DemandVU             float64 `json:"demand_vu"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

func HandleActiveDispatchJobs(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		requestedSupplierID := chi.URLParam(r, "supplierID")
		if requestedSupplierID != "" && requestedSupplierID != claims.ResolveSupplierID() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		result, err := listActiveDispatchJobs(ctx, client, claims.ResolveSupplierID())
		if err != nil {
			slog.ErrorContext(ctx, "dispatch job active list failed", "supplier_id", claims.ResolveSupplierID(), "trace_id", telemetry.TraceIDFromContext(ctx), "err", err)
			http.Error(w, `{"error":"dispatch job list failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func HandleDispatchJobRoutes(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		requestedSupplierID := chi.URLParam(r, "supplierID")
		if requestedSupplierID != "" && requestedSupplierID != claims.ResolveSupplierID() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		jobID := chi.URLParam(r, "jobID")
		if jobID == "" {
			http.NotFound(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		job, found, err := optimizationjobs.GetByID(ctx, client, claims.ResolveSupplierID(), jobID)
		if err != nil {
			slog.ErrorContext(ctx, "dispatch job projection read failed", "supplier_id", claims.ResolveSupplierID(), "job_id", jobID, "trace_id", telemetry.TraceIDFromContext(ctx), "err", err)
			http.Error(w, `{"error":"dispatch job projection failed"}`, http.StatusInternalServerError)
			return
		}
		if !found {
			http.NotFound(w, r)
			return
		}

		projection, err := buildDispatchJobProjection(job)
		if err != nil {
			slog.ErrorContext(ctx, "dispatch job projection decode failed", "supplier_id", claims.ResolveSupplierID(), "job_id", jobID, "trace_id", telemetry.TraceIDFromContext(ctx), "err", err)
			http.Error(w, `{"error":"dispatch job projection is unavailable"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projection)
	}
}

func listActiveDispatchJobs(ctx context.Context, client *spanner.Client, supplierID string) (dispatchJobActiveListResponse, error) {
	traceID := telemetry.TraceIDFromContext(ctx)
	if cache.GetClient() == nil {
		summaries, err := optimizationjobs.ListActiveSummariesBySupplier(ctx, client, supplierID, 100)
		if err != nil {
			return dispatchJobActiveListResponse{}, err
		}
		return buildActiveDispatchJobResponse(summaries, "spanner_fallback", true), nil
	}

	jobIDs, err := cache.ListSupplierOptimizationJobsActive(ctx, supplierID)
	if err != nil {
		slog.WarnContext(ctx, "dispatch jobs redis lookup degraded to spanner", "supplier_id", supplierID, "trace_id", traceID, "err", err)
		summaries, fallbackErr := optimizationjobs.ListActiveSummariesBySupplier(ctx, client, supplierID, 100)
		if fallbackErr != nil {
			return dispatchJobActiveListResponse{}, fallbackErr
		}
		return buildActiveDispatchJobResponse(summaries, "spanner_fallback", true), nil
	}
	if len(jobIDs) == 0 {
		return dispatchJobActiveListResponse{Jobs: []dispatchJobSummary{}, Source: "redis_index", Degraded: false}, nil
	}

	summaries, err := optimizationjobs.ListSummariesByJobIDs(ctx, client, supplierID, jobIDs)
	if err != nil {
		return dispatchJobActiveListResponse{}, err
	}

	byID := make(map[string]optimizationjobs.Summary, len(summaries))
	for _, summary := range summaries {
		byID[summary.JobID] = summary
	}

	staleIDs := make([]string, 0)
	active := make([]optimizationjobs.Summary, 0, len(jobIDs))
	for _, jobID := range jobIDs {
		summary, ok := byID[jobID]
		if !ok || !summary.Status.IsActive() {
			staleIDs = append(staleIDs, jobID)
			continue
		}
		active = append(active, summary)
	}

	for _, staleID := range staleIDs {
		if err := cache.RemoveSupplierOptimizationJobActive(ctx, supplierID, staleID); err != nil {
			slog.WarnContext(ctx, "dispatch jobs active-set stale cleanup failed", "supplier_id", supplierID, "job_id", staleID, "trace_id", traceID, "err", err)
		}
	}

	return buildActiveDispatchJobResponse(active, "redis_index", false), nil
}

func buildActiveDispatchJobResponse(summaries []optimizationjobs.Summary, source string, degraded bool) dispatchJobActiveListResponse {
	sort.Slice(summaries, func(left int, right int) bool {
		return summaries[left].UpdatedAt.After(summaries[right].UpdatedAt)
	})

	jobs := make([]dispatchJobSummary, 0, len(summaries))
	for _, summary := range summaries {
		jobs = append(jobs, dispatchJobSummary{
			JobID:       summary.JobID,
			Status:      string(summary.Status),
			SolverType:  string(summary.SolverType),
			RequestedAt: summary.RequestedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt:   summary.UpdatedAt.UTC().Format(time.RFC3339Nano),
			Ready:       !summary.Status.IsActive(),
		})
	}
	if jobs == nil {
		jobs = []dispatchJobSummary{}
	}

	return dispatchJobActiveListResponse{Jobs: jobs, Source: source, Degraded: degraded}
}

func buildDispatchJobProjection(job optimizationjobs.Job) (dispatchJobProjectionResponse, error) {
	response := dispatchJobProjectionResponse{
		JobID:          job.JobID,
		Status:         string(job.Status),
		SolverType:     string(job.SolverType),
		Ready:          false,
		RequestedAt:    job.RequestedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      job.UpdatedAt.UTC().Format(time.RFC3339Nano),
		FailureCode:    job.FailureCode,
		FailureMessage: job.FailureMessage,
		Warnings:       []string{},
		Routes:         []dispatchProjectionRoute{},
		Unassigned:     []dispatchProjectionStop{},
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	data, err := decodeDispatchJobVRP(job)
	if err != nil {
		return dispatchJobProjectionResponse{}, err
	}
	if data == nil {
		return response, nil
	}

	response.Ready = true
	response.TimedOut = data.solved.TimedOut
	response.MatrixSize = data.solved.MatrixSize
	response.Warnings = append([]string{}, data.solved.Warnings...)

	if data.solved.VRP == nil {
		return response, nil
	}

	response.ObjectiveCostKM = float64(data.solved.VRP.ObjectiveCostScaled) / data.scaleFactor

	if data.envelope.VRP != nil && data.envelope.VRP.Depot != nil {
		response.Depot = &dispatchProjectionDepot{
			NodeUUID: data.envelope.VRP.Depot.NodeUUID,
			Label:    data.envelope.VRP.Depot.Label,
			Lat:      data.envelope.VRP.Depot.Lat,
			Lng:      data.envelope.VRP.Depot.Lng,
		}
	}

	for _, route := range buildDispatchJobRoutePlans(job, data) {
		projectionRoute := dispatchProjectionRoute{
			RouteID:      route.RouteID,
			ManifestID:   route.ManifestID,
			VehicleUUID:  route.VehicleUUID,
			DriverUUID:   route.DriverID,
			DriverName:   route.DriverName,
			VehicleType:  route.VehicleType,
			VehicleClass: route.VehicleClass,
			CapacityVU:   route.CapacityVU,
			LoadVU:       route.LoadVU,
			RouteCostKM:  route.RouteCostKM,
			Stops:        append([]dispatchProjectionStop{}, route.Stops...),
		}
		response.Routes = append(response.Routes, projectionRoute)
	}

	nodesByID := make(map[string]contract.VRPNodeProjectionPayload, len(data.envelope.VRP.Nodes))
	for _, node := range data.envelope.VRP.Nodes {
		nodesByID[node.NodeUUID] = node
	}
	for _, nodeUUID := range data.solved.VRP.UnassignedNodeUUIDs {
		response.Unassigned = append(response.Unassigned, buildProjectionStop(nodesByID[nodeUUID], nodeUUID, 0))
	}

	return response, nil
}

func buildProjectionStop(node contract.VRPNodeProjectionPayload, nodeUUID string, sequence int) dispatchProjectionStop {
	return dispatchProjectionStop{
		Sequence:             sequence,
		NodeUUID:             nodeUUID,
		OrderID:              node.OrderID,
		RetailerID:           node.RetailerID,
		RetailerName:         node.RetailerName,
		Lat:                  node.Lat,
		Lng:                  node.Lng,
		Amount:               node.Amount,
		DemandVU:             node.DemandVU,
		ReceivingWindowOpen:  node.ReceivingWindowOpen,
		ReceivingWindowClose: node.ReceivingWindowClose,
	}
}
