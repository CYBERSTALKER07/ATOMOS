use std::collections::HashMap;
use std::time::{Duration, Instant};

use crate::pb::{Assignment, CpsatRequest, CpsatResponse};

#[derive(Clone)]
struct ManifestNode {
    manifest_id: String,
    original_index: usize,
    required_capacity_scaled: i64,
    priority_score_scaled: i64,
    eligible_factory_indices: Vec<usize>,
}

#[derive(Clone)]
struct BestSolution {
    objective_score_scaled: i64,
    assignments: Vec<Option<usize>>,
}

impl BestSolution {
    fn new(size: usize) -> Self {
        Self {
            objective_score_scaled: i64::MIN,
            assignments: vec![None; size],
        }
    }

    fn capture_if_better(&mut self, score: i64, assignments: &[Option<usize>]) {
        if score > self.objective_score_scaled {
            self.objective_score_scaled = score;
            self.assignments.clone_from_slice(assignments);
        }
    }
}

pub fn solve(request: &CpsatRequest) -> CpsatResponse {
    if request.manifest_requirements.is_empty() {
        return CpsatResponse {
            meta: request.meta.clone(),
            feasible: true,
            timed_out: false,
            objective_score_scaled: 0,
            assignments: Vec::new(),
            unassigned_manifest_ids: Vec::new(),
            warnings: Vec::new(),
        };
    }

    let mut warnings = Vec::new();
    let mut factory_indices: HashMap<String, usize> = HashMap::new();
    let mut factory_uuids: Vec<String> = Vec::new();
    let mut capacities: Vec<i64> = Vec::new();

    for slot in &request.factory_slots {
        if slot.factory_node_uuid.is_empty() {
            continue;
        }
        if let Some(existing_index) = factory_indices.get(&slot.factory_node_uuid).copied() {
            capacities[existing_index] =
                capacities[existing_index].saturating_add(slot.slot_capacity_scaled.max(0));
            continue;
        }
        let index = factory_uuids.len();
        factory_indices.insert(slot.factory_node_uuid.clone(), index);
        factory_uuids.push(slot.factory_node_uuid.clone());
        capacities.push(slot.slot_capacity_scaled.max(0));
    }

    if factory_uuids.is_empty() {
        return CpsatResponse {
            meta: request.meta.clone(),
            feasible: false,
            timed_out: false,
            objective_score_scaled: 0,
            assignments: request
                .manifest_requirements
                .iter()
                .map(|manifest| Assignment {
                    manifest_id: manifest.manifest_id.clone(),
                    factory_node_uuid: String::new(),
                    assigned: false,
                })
                .collect(),
            unassigned_manifest_ids: request
                .manifest_requirements
                .iter()
                .map(|manifest| manifest.manifest_id.clone())
                .collect(),
            warnings: vec!["no factory slots provided for constraint resolution".to_string()],
        };
    }

    let mut manifests = Vec::with_capacity(request.manifest_requirements.len());
    let mut no_eligible_count = 0_i64;

    for (index, manifest) in request.manifest_requirements.iter().enumerate() {
        let mut eligible = Vec::new();
        for factory_uuid in &manifest.eligible_factory_node_uuids {
            if let Some(factory_index) = factory_indices.get(factory_uuid).copied() {
                if !eligible.contains(&factory_index) {
                    eligible.push(factory_index);
                }
            }
        }
        eligible.sort_unstable();
        if eligible.is_empty() {
            no_eligible_count += 1;
        }

        manifests.push(ManifestNode {
            manifest_id: manifest.manifest_id.clone(),
            original_index: index,
            required_capacity_scaled: manifest.required_capacity_scaled.max(0),
            priority_score_scaled: manifest.priority_score_scaled,
            eligible_factory_indices: eligible,
        });
    }

    if no_eligible_count > 0 {
        warnings.push(format!(
            "{} manifest requirement(s) have no eligible factory slots",
            no_eligible_count
        ));
    }

    manifests.sort_by(|left, right| {
        left.eligible_factory_indices
            .len()
            .cmp(&right.eligible_factory_indices.len())
            .then(right.priority_score_scaled.cmp(&left.priority_score_scaled))
            .then(left.manifest_id.cmp(&right.manifest_id))
    });

    let mut suffix_upper_bound = vec![0_i64; manifests.len() + 1];
    for index in (0..manifests.len()).rev() {
        suffix_upper_bound[index] = suffix_upper_bound[index + 1]
            .saturating_add(manifests[index].priority_score_scaled.max(0));
    }

    let started_at = Instant::now();
    let time_limit = (request.solver_time_limit_ms > 0)
        .then_some(Duration::from_millis(request.solver_time_limit_ms as u64));

    let mut best_solution = BestSolution::new(manifests.len());
    let mut current_assignments = vec![None; manifests.len()];
    let mut remaining_capacities = capacities;
    let mut timed_out = false;

    search_optimal_assignments(
        0,
        0,
        &manifests,
        &suffix_upper_bound,
        &mut current_assignments,
        &mut remaining_capacities,
        &mut best_solution,
        started_at,
        time_limit,
        &mut timed_out,
    );

    if timed_out {
        warnings.push("solver_time_limit_ms reached during constraint search".to_string());
    }

    if timed_out && !request.return_best_effort {
        return CpsatResponse {
            meta: request.meta.clone(),
            feasible: false,
            timed_out: true,
            objective_score_scaled: 0,
            assignments: request
                .manifest_requirements
                .iter()
                .map(|manifest| Assignment {
                    manifest_id: manifest.manifest_id.clone(),
                    factory_node_uuid: String::new(),
                    assigned: false,
                })
                .collect(),
            unassigned_manifest_ids: request
                .manifest_requirements
                .iter()
                .map(|manifest| manifest.manifest_id.clone())
                .collect(),
            warnings,
        };
    }

    let mut assigned_factories_by_original = vec![None; request.manifest_requirements.len()];
    for (sorted_index, manifest) in manifests.iter().enumerate() {
        assigned_factories_by_original[manifest.original_index] = best_solution.assignments
            [sorted_index]
            .map(|factory_index| factory_uuids[factory_index].clone());
    }

    let mut assignments = Vec::with_capacity(request.manifest_requirements.len());
    let mut unassigned_manifest_ids = Vec::new();
    for (index, manifest) in request.manifest_requirements.iter().enumerate() {
        if let Some(factory_uuid) = assigned_factories_by_original[index].clone() {
            assignments.push(Assignment {
                manifest_id: manifest.manifest_id.clone(),
                factory_node_uuid: factory_uuid,
                assigned: true,
            });
            continue;
        }
        assignments.push(Assignment {
            manifest_id: manifest.manifest_id.clone(),
            factory_node_uuid: String::new(),
            assigned: false,
        });
        unassigned_manifest_ids.push(manifest.manifest_id.clone());
    }

    let objective_score_scaled = if best_solution.objective_score_scaled == i64::MIN {
        0
    } else {
        best_solution.objective_score_scaled
    };

    CpsatResponse {
        meta: request.meta.clone(),
        feasible: unassigned_manifest_ids.is_empty(),
        timed_out,
        objective_score_scaled,
        assignments,
        unassigned_manifest_ids,
        warnings,
    }
}

#[allow(clippy::too_many_arguments)]
fn search_optimal_assignments(
    index: usize,
    current_score: i64,
    manifests: &[ManifestNode],
    suffix_upper_bound: &[i64],
    current_assignments: &mut [Option<usize>],
    remaining_capacities: &mut [i64],
    best_solution: &mut BestSolution,
    started_at: Instant,
    time_limit: Option<Duration>,
    timed_out: &mut bool,
) {
    if let Some(limit) = time_limit {
        if started_at.elapsed() >= limit {
            *timed_out = true;
            return;
        }
    }

    if index == manifests.len() {
        best_solution.capture_if_better(current_score, current_assignments);
        return;
    }

    let optimistic_bound = current_score.saturating_add(suffix_upper_bound[index]);
    if optimistic_bound <= best_solution.objective_score_scaled {
        return;
    }

    let manifest = &manifests[index];
    for factory_index in &manifest.eligible_factory_indices {
        if remaining_capacities[*factory_index] < manifest.required_capacity_scaled {
            continue;
        }

        remaining_capacities[*factory_index] =
            remaining_capacities[*factory_index].saturating_sub(manifest.required_capacity_scaled);
        current_assignments[index] = Some(*factory_index);

        search_optimal_assignments(
            index + 1,
            current_score.saturating_add(manifest.priority_score_scaled),
            manifests,
            suffix_upper_bound,
            current_assignments,
            remaining_capacities,
            best_solution,
            started_at,
            time_limit,
            timed_out,
        );

        current_assignments[index] = None;
        remaining_capacities[*factory_index] =
            remaining_capacities[*factory_index].saturating_add(manifest.required_capacity_scaled);

        if *timed_out {
            return;
        }
    }

    search_optimal_assignments(
        index + 1,
        current_score,
        manifests,
        suffix_upper_bound,
        current_assignments,
        remaining_capacities,
        best_solution,
        started_at,
        time_limit,
        timed_out,
    );
}

#[cfg(test)]
mod tests {
    use crate::pb::{CpsatRequest, FactorySlot, ManifestRequirement};

    use super::solve;

    #[test]
    fn assigns_manifests_with_capacity_constraints() {
        let response = solve(&CpsatRequest {
            meta: None,
            factory_slots: vec![
                FactorySlot {
                    factory_node_uuid: "f1".to_string(),
                    slot_capacity_scaled: 5,
                },
                FactorySlot {
                    factory_node_uuid: "f2".to_string(),
                    slot_capacity_scaled: 3,
                },
            ],
            manifest_requirements: vec![
                ManifestRequirement {
                    manifest_id: "m1".to_string(),
                    required_capacity_scaled: 3,
                    priority_score_scaled: 10,
                    eligible_factory_node_uuids: vec!["f1".to_string(), "f2".to_string()],
                },
                ManifestRequirement {
                    manifest_id: "m2".to_string(),
                    required_capacity_scaled: 3,
                    priority_score_scaled: 8,
                    eligible_factory_node_uuids: vec!["f1".to_string(), "f2".to_string()],
                },
                ManifestRequirement {
                    manifest_id: "m3".to_string(),
                    required_capacity_scaled: 2,
                    priority_score_scaled: 6,
                    eligible_factory_node_uuids: vec!["f1".to_string()],
                },
            ],
            solver_time_limit_ms: 100,
            return_best_effort: true,
            num_search_workers: 1,
        });

        assert!(response.feasible);
        assert_eq!(response.objective_score_scaled, 24);
        assert!(response.unassigned_manifest_ids.is_empty());
        assert_eq!(response.assignments.len(), 3);
    }

    #[test]
    fn marks_manifests_without_eligibility_unassigned() {
        let response = solve(&CpsatRequest {
            meta: None,
            factory_slots: vec![FactorySlot {
                factory_node_uuid: "f1".to_string(),
                slot_capacity_scaled: 2,
            }],
            manifest_requirements: vec![ManifestRequirement {
                manifest_id: "m1".to_string(),
                required_capacity_scaled: 2,
                priority_score_scaled: 5,
                eligible_factory_node_uuids: vec!["missing".to_string()],
            }],
            solver_time_limit_ms: 0,
            return_best_effort: true,
            num_search_workers: 1,
        });

        assert!(!response.feasible);
        assert_eq!(response.unassigned_manifest_ids, vec!["m1".to_string()]);
        assert!(!response.warnings.is_empty());
    }
}
