use crate::optimizer_core::{OptimizeVrpRequest, OptimizeVrpResponse, SolverStatus, VehicleRoute};
use crate::scaling;
use log::{info, warn};
use std::time::{Instant, Duration};

pub const DEFAULT_SOLVER_TIME_LIMIT_MS: u64 = 2_000;
pub const MAX_SOLVER_TIME_LIMIT_MS: u64 = 60_000;
pub const MIN_SOLVER_TIME_LIMIT_MS: u64 = 1;

/// Solves VRP deterministically using a greedy nearest-neighbor heuristic with capacity and time windows.
pub fn solve(req: OptimizeVrpRequest) -> OptimizeVrpResponse {
    let start_time = Instant::now();
    let time_limit_ms = if req.solver_time_limit_ms <= 0 {
        DEFAULT_SOLVER_TIME_LIMIT_MS
    } else {
        (req.solver_time_limit_ms as u64).clamp(MIN_SOLVER_TIME_LIMIT_MS, MAX_SOLVER_TIME_LIMIT_MS)
    };
    let time_limit = Duration::from_millis(time_limit_ms);

    let actual_elements = req.distance_matrix_km.len();
    if req.matrix_size <= 0 {
        warn!("Invalid non-positive matrix_size: {}", req.matrix_size);
        let structured_err = serde_json::json!({
            "code": "INVALID_DISTANCE_MATRIX_DIMENSIONS",
            "message": "Distance matrix size must be positive.",
            "expected_elements": 0,
            "received_elements": actual_elements,
            "nodes_count": req.matrix_size,
        });
        return OptimizeVrpResponse {
            status: SolverStatus::ModelInvalid as i32,
            timed_out: false,
            matrix_size: req.matrix_size,
            objective_cost_scaled: 0,
            routes: vec![],
            unassigned_node_uuids: req.drop_off_node_uuids.clone(),
            warnings: vec![
                structured_err.to_string(),
                format!("INVALID_DISTANCE_MATRIX_DIMENSIONS: non-positive matrix_size {}", req.matrix_size),
            ],
        };
    }

    let matrix_size = req.matrix_size as usize;
    let expected_elements = match matrix_size.checked_mul(matrix_size) {
        Some(v) => v,
        None => {
            warn!("Distance matrix size overflow for matrix_size={}", matrix_size);
            return OptimizeVrpResponse {
                status: SolverStatus::ModelInvalid as i32,
                timed_out: false,
                matrix_size: req.matrix_size,
                objective_cost_scaled: 0,
                routes: vec![],
                unassigned_node_uuids: req.drop_off_node_uuids.clone(),
                warnings: vec!["INVALID_DISTANCE_MATRIX_DIMENSIONS: matrix_size^2 overflow".to_string()],
            };
        }
    };

    if actual_elements != expected_elements {
        warn!(
            "Distance matrix dimension mismatch: expected {} elements, got {}",
            expected_elements, actual_elements
        );
        let structured_err = serde_json::json!({
            "code": "INVALID_DISTANCE_MATRIX_DIMENSIONS",
            "message": "Distance matrix size mismatch: expected N*N elements.",
            "expected_elements": expected_elements,
            "received_elements": actual_elements,
            "nodes_count": matrix_size,
        });
        return OptimizeVrpResponse {
            status: SolverStatus::ModelInvalid as i32,
            timed_out: false,
            matrix_size: req.matrix_size,
            objective_cost_scaled: 0,
            routes: vec![],
            unassigned_node_uuids: req.drop_off_node_uuids.clone(),
            warnings: vec![
                structured_err.to_string(),
                format!(
                    "INVALID_DISTANCE_MATRIX_DIMENSIONS: Distance matrix dimension mismatch: received length {}, expected matrix_size^2 ({} * {} = {} elements).",
                    actual_elements, matrix_size, matrix_size, expected_elements
                ),
            ],
        };
    }

    let n_nodes = req.drop_off_node_uuids.len() + 1; // 1 for depot
    if matrix_size != n_nodes {
        warn!("Matrix size mismatch: expected {}, got {}", n_nodes, matrix_size);
        let structured_err = serde_json::json!({
            "code": "INVALID_DISTANCE_MATRIX_NODES_MISMATCH",
            "message": format!("Matrix size ({}) does not match nodes count ({})", matrix_size, n_nodes),
            "expected_nodes": n_nodes,
            "matrix_size": matrix_size,
        });
        return OptimizeVrpResponse {
            status: SolverStatus::ModelInvalid as i32,
            timed_out: false,
            matrix_size: req.matrix_size,
            objective_cost_scaled: 0,
            routes: vec![],
            unassigned_node_uuids: req.drop_off_node_uuids.clone(),
            warnings: vec![
                structured_err.to_string(),
                "Matrix size does not match nodes count".to_string(),
            ],
        };
    }

    let node_to_matrix_idx: std::collections::HashMap<&str, usize> = req
        .drop_off_node_uuids
        .iter()
        .enumerate()
        .map(|(idx, uuid)| (uuid.as_str(), idx + 1))
        .collect();

    let node_demand_map: std::collections::HashMap<&str, f64> = req
        .node_demands
        .iter()
        .map(|d| (d.node_uuid.as_str(), d.demand_vu))
        .collect();

    let node_tw_map: std::collections::HashMap<&str, (f64, f64)> = req
        .node_time_windows
        .iter()
        .map(|tw| (tw.node_uuid.as_str(), (tw.start_window_hours, tw.end_window_hours)))
        .collect();

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
        let mut best_cost_f64 = 0.0;

        loop {
            if start_time.elapsed() >= time_limit {
                timed_out = true;
                break;
            }

            let mut best_next_node_idx = None;
            let mut best_cost = f64::MAX;
            let mut best_unassigned_idx = 0;

            for (i, node_uuid) in unassigned.iter().enumerate() {
                // Determine node indices: lookup invariant matrix index from map
                let node_idx = *node_to_matrix_idx.get(node_uuid.as_str()).unwrap_or(&(i + 1));

                // Fetch demand in O(1)
                let demand = *node_demand_map.get(node_uuid.as_str()).unwrap_or(&0.0);

                // Fetch time window in O(1)
                let default_tw = (0.0, 24.0);
                let (start_tw, end_tw) = *node_tw_map.get(node_uuid.as_str()).unwrap_or(&default_tw);

                let distance_cost = req.distance_matrix_km[current_node_idx * matrix_size + node_idx];

                // Time heuristic: assume distance_cost roughly correlates to hours (e.g. 50 km/h) -> dist / 50.0
                let travel_time = distance_cost / 50.0;
                let arrival_time = current_time + travel_time;
                let service_time = arrival_time.max(start_tw); // Wait if arriving early

                // Feasibility checks
                if current_load + demand <= vehicle.capacity_vu
                    && service_time <= end_tw
                    && service_time <= vehicle.end_window_hours
                    && distance_cost < best_cost
                {
                    best_cost = distance_cost;
                    best_next_node_idx = Some(node_idx);
                    best_unassigned_idx = i;
                }
            }

            if let Some(next_node_idx) = best_next_node_idx {
                // Commit to the node
                let node_uuid = unassigned.remove(best_unassigned_idx);
                let demand = *node_demand_map.get(node_uuid.as_str()).unwrap_or(&0.0);

                ordered_nodes.push(node_uuid);
                current_load += demand;
                best_cost_f64 += best_cost;

                let travel_time = best_cost / 50.0;
                current_time += travel_time;
                current_node_idx = next_node_idx;
            } else {
                break; // No more feasible nodes for this vehicle
            }
        }

        // Return to depot for initial cost calc
        if current_node_idx != 0 {
            let return_cost = req.distance_matrix_km[current_node_idx * matrix_size];
            best_cost_f64 += return_cost;
            route_cost_scaled = scaling::to_solver_int(best_cost_f64);
        }

        // Local Search (2-opt) to improve the route
        let mut improved = true;
        while improved && start_time.elapsed() < time_limit && ordered_nodes.len() > 2 {
            improved = false;
            let n = ordered_nodes.len();

            for i in 0..n - 1 {
                if start_time.elapsed() >= time_limit {
                    break;
                }
                for k in i + 1..n {
                    // Try reversing the segment from i to k
                    let mut new_route = ordered_nodes.clone();
                    new_route[i..=k].reverse();

                    // Evaluate feasibility and cost of new_route
                    let mut valid = true;
                    let mut eval_time = vehicle.start_window_hours;
                    let mut eval_cost = 0.0;
                    let mut eval_curr_idx = 0; // Depot

                    for node_uuid in &new_route {
                        let Some(&node_idx) = node_to_matrix_idx.get(node_uuid.as_str()) else {
                            valid = false;
                            break;
                        };
                        let dist = req.distance_matrix_km[eval_curr_idx * matrix_size + node_idx];
                        eval_cost += dist;

                        let travel_time = dist / 50.0;
                        eval_time += travel_time;

                        let default_tw = (0.0, 24.0);
                        let (start_tw, end_tw) = *node_tw_map.get(node_uuid.as_str()).unwrap_or(&default_tw);

                        eval_time = eval_time.max(start_tw);

                        if eval_time > end_tw || eval_time > vehicle.end_window_hours {
                            valid = false;
                            break;
                        }
                        eval_curr_idx = node_idx;
                    }

                    if valid {
                        // Return to depot
                        let return_cost = req.distance_matrix_km[eval_curr_idx * matrix_size];
                        eval_cost += return_cost;

                        // Original cost of this route
                        let _original_cost_scaled = route_cost_scaled; // This includes return to depot

                        if eval_cost < best_cost_f64 {
                            // It's strictly better!
                            ordered_nodes = new_route;
                            best_cost_f64 = eval_cost;
                            improved = true;
                            break;
                        }
                    }
                }
                if improved {
                    break;
                }
            }
        }

        route_cost_scaled = scaling::to_solver_int(best_cost_f64);
        total_cost += route_cost_scaled;

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

    let status = if routes.is_empty() && !unassigned.is_empty() {
        SolverStatus::Infeasible
    } else {
        // Nearest-neighbor greedy never proves optimality — report HEURISTIC (P2-2).
        SolverStatus::Heuristic
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
