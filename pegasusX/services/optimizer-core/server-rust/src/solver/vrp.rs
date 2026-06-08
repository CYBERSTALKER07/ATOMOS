use crate::optimizer_core::{OptimizeVrpRequest, OptimizeVrpResponse, SolverStatus, VehicleRoute};
use crate::scaling;
use log::{info, warn};
use std::time::{Instant, Duration};

/// Solves VRP deterministically using a greedy nearest-neighbor heuristic with capacity and time windows.
pub fn solve(req: OptimizeVrpRequest) -> OptimizeVrpResponse {
    let start_time = Instant::now();
    let time_limit = Duration::from_millis(req.solver_time_limit_ms as u64);
    
    let matrix_size = req.matrix_size as usize;
    let n_nodes = req.drop_off_node_uuids.len() + 1; // 1 for depot
    
    if matrix_size != n_nodes {
        warn!("Matrix size mismatch: expected {}, got {}", n_nodes, matrix_size);
        return OptimizeVrpResponse {
            status: SolverStatus::ModelInvalid as i32,
            timed_out: false,
            matrix_size: req.matrix_size,
            objective_cost_scaled: 0,
            routes: vec![],
            unassigned_node_uuids: req.drop_off_node_uuids.clone(),
            warnings: vec!["Matrix size does not match nodes count".to_string()],
        };
    }

    let mut unassigned = req.drop_off_node_uuids.clone();
    let mut routes = Vec::new();
    let mut total_cost = 0;
    let mut timed_out = false;

    // Simple greedy implementation: for each vehicle, start at depot, visit nearest feasible unassigned node
    for vehicle in &req.vehicles {
        let mut current_node_idx = 0; // Depot is always index 0
        let mut current_time = vehicle.start_window_hours;
        let mut current_load = 0.0;
        let mut ordered_nodes = Vec::new();
        let mut route_cost_scaled = 0;

        loop {
            if start_time.elapsed() >= time_limit {
                timed_out = true;
                break;
            }

            let mut best_next_node_idx = None;
            let mut best_cost = f64::MAX;
            let mut best_unassigned_idx = 0;

            for (i, node_uuid) in unassigned.iter().enumerate() {
                // Determine node indices: node_uuid corresponds to index i + 1 in distance matrix
                let node_idx = i + 1;
                
                // Fetch demand
                let demand = req.node_demands.iter().find(|d| d.node_uuid == *node_uuid).map_or(0.0, |d| d.demand_vu);
                
                // Fetch time window
                let default_tw = (0.0, 24.0);
                let (start_tw, end_tw) = req.node_time_windows.iter().find(|tw| tw.node_uuid == *node_uuid).map_or(default_tw, |tw| (tw.start_window_hours, tw.end_window_hours));

                let distance_cost = req.distance_matrix_km[current_node_idx * matrix_size + node_idx];
                
                // Time heuristic: assume distance_cost roughly correlates to hours (e.g. 50 km/h) -> dist / 50.0
                let travel_time = distance_cost / 50.0; 
                let arrival_time = current_time + travel_time;
                let service_time = arrival_time.max(start_tw); // Wait if arriving early

                // Feasibility checks
                if current_load + demand <= vehicle.capacity_vu && service_time <= end_tw && service_time <= vehicle.end_window_hours {
                    if distance_cost < best_cost {
                        best_cost = distance_cost;
                        best_next_node_idx = Some(node_idx);
                        best_unassigned_idx = i;
                    }
                }
            }

            if let Some(next_node_idx) = best_next_node_idx {
                // Commit to the node
                let node_uuid = unassigned.remove(best_unassigned_idx);
                let demand = req.node_demands.iter().find(|d| d.node_uuid == *node_uuid).map_or(0.0, |d| d.demand_vu);
                
                ordered_nodes.push(node_uuid);
                current_load += demand;
                route_cost_scaled += scaling::to_solver_int(best_cost);
                total_cost += scaling::to_solver_int(best_cost);
                
                let travel_time = best_cost / 50.0;
                current_time += travel_time;
                current_node_idx = next_node_idx;
            } else {
                break; // No more feasible nodes for this vehicle
            }
        }

        // Return to depot
        if current_node_idx != 0 {
            let return_cost = req.distance_matrix_km[current_node_idx * matrix_size + 0];
            route_cost_scaled += scaling::to_solver_int(return_cost);
            total_cost += scaling::to_solver_int(return_cost);
        }

        if !ordered_nodes.is_empty() {
            routes.push(VehicleRoute {
                vehicle_uuid: vehicle.vehicle_uuid.clone(),
                driver_uuid: vehicle.driver_uuid.clone(),
                ordered_node_uuids: ordered_nodes,
                load_scaled: scaling::to_solver_int(current_load),
                route_cost_scaled,
            });
        }

        if timed_out || unassigned.is_empty() {
            break;
        }
    }

    let status = if unassigned.is_empty() {
        SolverStatus::Optimal
    } else {
        SolverStatus::Feasible
    };

    info!("VRP solved with status {:?}, unassigned: {}", status, unassigned.len());

    OptimizeVrpResponse {
        status: status as i32,
        timed_out,
        matrix_size: req.matrix_size,
        objective_cost_scaled: total_cost,
        routes,
        unassigned_node_uuids: unassigned,
        warnings: vec![],
    }
}
