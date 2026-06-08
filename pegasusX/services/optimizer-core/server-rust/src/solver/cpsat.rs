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

    // Sort manifest requirements by priority score (descending)
    let mut requirements = req.manifest_requirements.clone();
    requirements.sort_by(|a, b| b.priority_score.partial_cmp(&a.priority_score).unwrap_or(std::cmp::Ordering::Equal));

    for req_manifest in requirements {
        if start_time.elapsed() >= time_limit {
            timed_out = true;
            unassigned.push(req_manifest.manifest_id.clone());
            continue;
        }

        let mut assigned = false;

        // Try to assign to an eligible factory with enough capacity
        for factory_id in &req_manifest.eligible_factory_node_uuids {
            if let Some(capacity) = factory_capacities.get_mut(factory_id) {
                if *capacity >= req_manifest.required_capacity {
                    *capacity -= req_manifest.required_capacity;
                    assignments.push(Assignment {
                        manifest_id: req_manifest.manifest_id.clone(),
                        factory_node_uuid: factory_id.clone(),
                        assigned: true,
                    });
                    total_score += req_manifest.priority_score;
                    assigned = true;
                    break;
                }
            }
        }

        if !assigned {
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
