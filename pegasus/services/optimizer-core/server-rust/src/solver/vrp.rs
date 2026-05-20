use std::collections::HashMap;
use std::time::{Duration, Instant};

use crate::pb::{Int64Row, SolverStatus, VehicleRoute, VrpRequest, VrpResponse};

const DISTANCE_FALLBACK_PENALTY: i64 = 1_000_000;

pub fn solve(request: &VrpRequest) -> VrpResponse {
    let drop_count = request.drop_off_node_uuids.len();
    let matrix_size = (drop_count + 1) as i32;
    if drop_count == 0 {
        return VrpResponse {
            meta: request.meta.clone(),
            feasible: true,
            timed_out: false,
            objective_cost_scaled: 0,
            routes: Vec::new(),
            unassigned_node_uuids: Vec::new(),
            warnings: Vec::new(),
            status: SolverStatus::Optimal as i32,
            matrix_size,
        };
    }

    let mut warnings = Vec::new();
    if request.vehicles.is_empty() {
        return VrpResponse {
            meta: request.meta.clone(),
            feasible: false,
            timed_out: false,
            objective_cost_scaled: 0,
            routes: Vec::new(),
            unassigned_node_uuids: request.drop_off_node_uuids.clone(),
            warnings: vec!["no vehicles provided for route calculation".to_string()],
            status: SolverStatus::ModelInvalid as i32,
            matrix_size,
        };
    }

    let expected_matrix_size = drop_count + 1;
    if request.distance_matrix_scaled.len() < expected_matrix_size
        || request
            .distance_matrix_scaled
            .iter()
            .any(|row| row.values.len() < expected_matrix_size)
    {
        warnings.push(
            "distance_matrix_scaled is incomplete for depot+dropoff topology; using penalty fallback for missing edges"
                .to_string(),
        );
    }

    let started_at = Instant::now();
    let time_limit = (request.solver_time_limit_ms > 0)
        .then_some(Duration::from_millis(request.solver_time_limit_ms as u64));
    let mut timed_out = false;

    let demand_by_node: HashMap<&str, i64> = request
        .node_demands
        .iter()
        .map(|demand| (demand.node_uuid.as_str(), demand.demand_scaled.max(0)))
        .collect();

    let time_window_by_node: HashMap<&str, (i64, i64)> = request
        .node_time_windows
        .iter()
        .map(|window| {
            (
                window.node_uuid.as_str(),
                (
                    window.start_time_window_scaled,
                    window.end_time_window_scaled,
                ),
            )
        })
        .collect();

    let mut assigned = vec![false; drop_count];
    let mut routes = Vec::new();
    let mut objective_cost_scaled = 0_i64;
    let mut warned_missing_edge = false;

    for vehicle in &request.vehicles {
        if reached_time_limit(started_at, time_limit) {
            timed_out = true;
            break;
        }

        let mut remaining_capacity = vehicle.capacity_scaled.max(0);
        let mut current_time = vehicle.start_time_window_scaled;
        let vehicle_end_window = vehicle.end_time_window_scaled;
        let mut current_matrix_index = 0_usize;
        let mut route_nodes = Vec::new();
        let mut route_cost_scaled = 0_i64;
        let mut load_scaled = 0_i64;

        loop {
            let mut best_candidate: Option<(usize, i64, i64)> = None;

            for (drop_idx, drop_uuid) in request.drop_off_node_uuids.iter().enumerate() {
                if assigned[drop_idx] {
                    continue;
                }

                let demand_scaled = *demand_by_node.get(drop_uuid.as_str()).unwrap_or(&0);
                if demand_scaled > remaining_capacity {
                    continue;
                }

                let node_matrix_index = drop_idx + 1;
                let distance_scaled = edge_cost(
                    &request.distance_matrix_scaled,
                    current_matrix_index,
                    node_matrix_index,
                    &mut warned_missing_edge,
                );

                let arrival_time = current_time.saturating_add(distance_scaled);
                let (start_window, end_window) = time_window_by_node
                    .get(drop_uuid.as_str())
                    .copied()
                    .unwrap_or((i64::MIN / 4, i64::MAX / 4));
                if arrival_time > end_window {
                    continue;
                }

                let service_start = arrival_time.max(start_window);
                if service_start > vehicle_end_window {
                    continue;
                }

                let candidate = (drop_idx, distance_scaled, service_start);
                match best_candidate {
                    Some((best_idx, best_distance, _))
                        if distance_scaled > best_distance
                            || (distance_scaled == best_distance
                                && request.drop_off_node_uuids[drop_idx]
                                    > request.drop_off_node_uuids[best_idx]) => {}
                    _ => {
                        best_candidate = Some(candidate);
                    }
                }
            }

            let Some((drop_idx, distance_scaled, service_start)) = best_candidate else {
                break;
            };

            let drop_uuid = request.drop_off_node_uuids[drop_idx].clone();
            let demand_scaled = *demand_by_node.get(drop_uuid.as_str()).unwrap_or(&0);

            assigned[drop_idx] = true;
            route_nodes.push(drop_uuid);
            remaining_capacity = remaining_capacity.saturating_sub(demand_scaled);
            load_scaled = load_scaled.saturating_add(demand_scaled);
            route_cost_scaled = route_cost_scaled.saturating_add(distance_scaled);
            current_time = service_start;
            current_matrix_index = drop_idx + 1;

            if reached_time_limit(started_at, time_limit) {
                timed_out = true;
                break;
            }
        }

        if !route_nodes.is_empty() {
            let return_to_depot = edge_cost(
                &request.distance_matrix_scaled,
                current_matrix_index,
                0,
                &mut warned_missing_edge,
            );
            route_cost_scaled = route_cost_scaled.saturating_add(return_to_depot);

            objective_cost_scaled = objective_cost_scaled.saturating_add(route_cost_scaled);
            routes.push(VehicleRoute {
                vehicle_uuid: vehicle.vehicle_uuid.clone(),
                driver_uuid: vehicle.driver_uuid.clone(),
                ordered_node_uuids: route_nodes,
                load_scaled,
                route_cost_scaled,
            });
        }

        if timed_out {
            break;
        }
    }

    if warned_missing_edge {
        warnings.push(
            "one or more distance edges were missing; applied deterministic fallback penalty"
                .to_string(),
        );
    }
    if timed_out {
        warnings.push("solver_time_limit_ms reached during route construction".to_string());
    }

    let unassigned_node_uuids: Vec<String> = request
        .drop_off_node_uuids
        .iter()
        .enumerate()
        .filter_map(|(index, uuid)| (!assigned[index]).then_some(uuid.clone()))
        .collect();

    if timed_out && !request.return_best_effort {
        let assigned_count = assigned.iter().filter(|is_assigned| **is_assigned).count();
        return VrpResponse {
            meta: request.meta.clone(),
            feasible: false,
            timed_out: true,
            objective_cost_scaled: 0,
            routes: Vec::new(),
            unassigned_node_uuids: request.drop_off_node_uuids.clone(),
            warnings,
            status: if assigned_count > 0 {
                SolverStatus::Feasible as i32
            } else {
                SolverStatus::Infeasible as i32
            },
            matrix_size,
        };
    }

    let assigned_count = assigned.iter().filter(|is_assigned| **is_assigned).count();
    let status = if timed_out || assigned_count > 0 {
        SolverStatus::Feasible as i32
    } else {
        SolverStatus::Infeasible as i32
    };

    VrpResponse {
        meta: request.meta.clone(),
        feasible: unassigned_node_uuids.is_empty(),
        timed_out,
        objective_cost_scaled,
        routes,
        unassigned_node_uuids,
        warnings,
        status,
        matrix_size,
    }
}

fn edge_cost(matrix: &[Int64Row], from: usize, to: usize, warned_missing_edge: &mut bool) -> i64 {
    if let Some(cost) = matrix.get(from).and_then(|row| row.values.get(to)).copied() {
        return cost.max(0);
    }
    *warned_missing_edge = true;
    DISTANCE_FALLBACK_PENALTY
}

fn reached_time_limit(started_at: Instant, time_limit: Option<Duration>) -> bool {
    match time_limit {
        Some(limit) => started_at.elapsed() >= limit,
        None => false,
    }
}

#[cfg(test)]
mod tests {
    use crate::pb::{Int64Row, NodeDemand, NodeTimeWindow, Vehicle, VrpRequest};

    use super::solve;

    #[test]
    fn builds_capacity_constrained_route() {
        let response = solve(&VrpRequest {
            meta: None,
            depot_node_uuid: "depot".to_string(),
            drop_off_node_uuids: vec!["a".to_string(), "b".to_string()],
            distance_matrix_scaled: vec![
                Int64Row {
                    values: vec![0, 2, 3],
                },
                Int64Row {
                    values: vec![2, 0, 2],
                },
                Int64Row {
                    values: vec![3, 2, 0],
                },
            ],
            vehicles: vec![Vehicle {
                vehicle_uuid: "v1".to_string(),
                driver_uuid: "d1".to_string(),
                capacity_scaled: 10,
                start_time_window_scaled: 0,
                end_time_window_scaled: 100,
            }],
            node_demands: vec![
                NodeDemand {
                    node_uuid: "a".to_string(),
                    demand_scaled: 3,
                },
                NodeDemand {
                    node_uuid: "b".to_string(),
                    demand_scaled: 2,
                },
            ],
            node_time_windows: vec![
                NodeTimeWindow {
                    node_uuid: "a".to_string(),
                    start_time_window_scaled: 0,
                    end_time_window_scaled: 100,
                },
                NodeTimeWindow {
                    node_uuid: "b".to_string(),
                    start_time_window_scaled: 0,
                    end_time_window_scaled: 100,
                },
            ],
            solver_time_limit_ms: 0,
            return_best_effort: true,
        });

        assert!(response.feasible);
        assert!(response.unassigned_node_uuids.is_empty());
        assert_eq!(response.routes.len(), 1);
        assert_eq!(response.routes[0].ordered_node_uuids, vec!["a", "b"]);
        assert_eq!(response.routes[0].route_cost_scaled, 7);
    }
}
