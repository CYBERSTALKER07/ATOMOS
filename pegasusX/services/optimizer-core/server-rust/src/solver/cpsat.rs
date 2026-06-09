use crate::optimizer_core::{OptimizeCpsatRequest, OptimizeCpsatResponse, SolverStatus, Assignment};
use crate::scaling;
use log::{info, warn};
use std::time::{Instant, Duration};

/// Solves factory slot assignment deterministically using a greedy heuristic.
pub fn solve(req: OptimizeCpsatRequest) -> OptimizeCpsatResponse {
    let start_time = Instant::now();
    let time_limit = Duration::from_millis(req.solver_time_limit_ms as u64);
    
    let mut unassigned = Vec::new();
    let mut assignments = Vec::new();
    let mut total_score = 0.0;
    let mut timed_out = false;

    // Track remaining capacity for each factory slot
    let mut factory_capacities: std::collections::HashMap<String, f64> = req.factory_slots.iter()
        .map(|s| (s.factory_node_uuid.clone(), s.slot_capacity))
        .collect();

    // Sort manifest requirements by "bang for buck": priority_score / required_capacity
    let mut requirements = req.manifest_requirements.clone();
    requirements.sort_by(|a, b| {
        let ratio_a = a.priority_score / a.required_capacity.max(0.001);
        let ratio_b = b.priority_score / b.required_capacity.max(0.001);
        ratio_b.partial_cmp(&ratio_a).unwrap_or(std::cmp::Ordering::Equal)
    });

    let mut current_assignments = std::collections::HashMap::new();

    // 1. Initial Greedy Construction
    for req_manifest in &requirements {
        if start_time.elapsed() >= time_limit {
            timed_out = true;
            break;
        }

        let mut best_factory = None;
        let mut best_remaining_cap = -1.0;

        for factory_id in &req_manifest.eligible_factory_node_uuids {
            if let Some(&cap) = factory_capacities.get(factory_id) {
                if cap >= req_manifest.required_capacity {
                    // Tie-breaker: pick factory with most remaining capacity to balance load
                    if cap > best_remaining_cap {
                        best_remaining_cap = cap;
                        best_factory = Some(factory_id.clone());
                    }
                }
            }
        }

        if let Some(factory_id) = best_factory {
            *factory_capacities.get_mut(&factory_id).unwrap() -= req_manifest.required_capacity;
            current_assignments.insert(req_manifest.manifest_id.clone(), factory_id);
            total_score += req_manifest.priority_score;
        }
    }

    // 2. Local Search (Swap unassigned with assigned if it improves total priority)
    if !timed_out {
        let mut improved = true;
        while improved && start_time.elapsed() < time_limit {
            improved = false;
            
            for unassigned_req in requirements.iter().filter(|r| !current_assignments.contains_key(&r.manifest_id)) {
                if start_time.elapsed() >= time_limit {
                    break;
                }

                // Look for an assigned manifest to swap with
                let mut best_swap = None;
                let mut best_gain = 0.0;

                for assigned_req in requirements.iter().filter(|r| current_assignments.contains_key(&r.manifest_id)) {
                    let factory_id = current_assignments.get(&assigned_req.manifest_id).unwrap();
                    
                    // Can the unassigned request go to this factory?
                    if !unassigned_req.eligible_factory_node_uuids.contains(factory_id) {
                        continue;
                    }

                    // Check if swapping is feasible and profitable
                    let cap = *factory_capacities.get(factory_id).unwrap();
                    let freed_cap = assigned_req.required_capacity;
                    let required_cap = unassigned_req.required_capacity;

                    if cap + freed_cap >= required_cap {
                        let gain = unassigned_req.priority_score - assigned_req.priority_score;
                        if gain > best_gain {
                            best_gain = gain;
                            best_swap = Some((assigned_req.manifest_id.clone(), factory_id.clone()));
                        }
                    }
                }

                if let Some((swap_out_id, factory_id)) = best_swap {
                    // Execute swap
                    let swap_out_req = requirements.iter().find(|r| r.manifest_id == swap_out_id).unwrap();
                    
                    *factory_capacities.get_mut(&factory_id).unwrap() += swap_out_req.required_capacity;
                    *factory_capacities.get_mut(&factory_id).unwrap() -= unassigned_req.required_capacity;
                    
                    current_assignments.remove(&swap_out_id);
                    current_assignments.insert(unassigned_req.manifest_id.clone(), factory_id);
                    
                    total_score += best_gain;
                    improved = true;
                    break; // Restart loop
                }
            }
        }
    }

    // Prepare response
    for req_manifest in &requirements {
        if let Some(factory_id) = current_assignments.get(&req_manifest.manifest_id) {
            assignments.push(Assignment {
                manifest_id: req_manifest.manifest_id.clone(),
                factory_node_uuid: factory_id.clone(),
                assigned: true,
            });
        } else {
            unassigned.push(req_manifest.manifest_id.clone());
            assignments.push(Assignment {
                manifest_id: req_manifest.manifest_id.clone(),
                factory_node_uuid: String::new(),
                assigned: false,
            });
        }
    }

    let status = if unassigned.is_empty() {
        SolverStatus::Optimal
    } else {
        SolverStatus::Feasible
    };

    info!("CPSAT solved with status {:?}, unassigned: {}", status, unassigned.len());

    OptimizeCpsatResponse {
        status: status as i32,
        timed_out,
        matrix_size: 0, // N/A for CPSAT
        objective_score_scaled: scaling::to_solver_int(total_score),
        assignments,
        unassigned_manifest_ids: unassigned,
        warnings: vec![],
    }
}
