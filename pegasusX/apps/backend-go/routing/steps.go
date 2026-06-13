package routing

import (
	"fmt"
	"strings"
)

// RouteStep is one turn-by-turn navigation segment from OSRM.
type RouteStep struct {
	Instruction string  `json:"instruction"`
	DistanceM   float64 `json:"distance_m"`
	DurationS   float64 `json:"duration_s"`
	Maneuver    string  `json:"maneuver,omitempty"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

type osrmStep struct {
	Name     string       `json:"name"`
	Distance float64      `json:"distance"`
	Duration float64      `json:"duration"`
	Maneuver osrmManeuver `json:"maneuver"`
	Mode     string       `json:"mode"`
}

type osrmManeuver struct {
	Type     string    `json:"type"`
	Modifier string    `json:"modifier"`
	Location []float64 `json:"location"`
}

type osrmLeg struct {
	Steps []osrmStep `json:"steps"`
}

func parseOSRMSteps(legs []osrmLeg) []RouteStep {
	if len(legs) == 0 {
		return nil
	}
	raw := legs[0].Steps
	steps := make([]RouteStep, 0, len(raw))
	for _, step := range raw {
		if strings.EqualFold(step.Mode, "ferry") {
			continue
		}
		lat, lng := maneuverLocation(step.Maneuver)
		maneuver := formatManeuverLabel(step.Maneuver.Type, step.Maneuver.Modifier)
		steps = append(steps, RouteStep{
			Instruction: formatStepInstruction(step.Maneuver.Type, step.Maneuver.Modifier, step.Name),
			DistanceM:   step.Distance,
			DurationS:   step.Duration,
			Maneuver:    maneuver,
			Lat:         lat,
			Lng:         lng,
		})
	}
	return steps
}

func maneuverLocation(maneuver osrmManeuver) (lat, lng float64) {
	if len(maneuver.Location) < 2 {
		return 0, 0
	}
	return maneuver.Location[1], maneuver.Location[0]
}

func formatManeuverLabel(maneuverType, modifier string) string {
	maneuverType = strings.TrimSpace(maneuverType)
	modifier = strings.TrimSpace(modifier)
	switch {
	case modifier != "" && maneuverType != "":
		return strings.ToUpper(modifier) + " " + strings.ToUpper(maneuverType)
	case maneuverType != "":
		return strings.ToUpper(maneuverType)
	default:
		return ""
	}
}

func formatStepInstruction(maneuverType, modifier, street string) string {
	maneuverType = strings.ToLower(strings.TrimSpace(maneuverType))
	modifier = strings.ToLower(strings.TrimSpace(modifier))
	street = strings.TrimSpace(street)

	var action string
	switch {
	case maneuverType == "depart":
		action = "Head"
		if modifier != "" {
			action += " " + modifier
		}
	case maneuverType == "arrive":
		action = "Arrive"
	case modifier != "":
		action = "Turn " + modifier
	default:
		action = maneuverType
		if action != "" {
			action = strings.ToUpper(action[:1]) + action[1:]
		}
	}
	if street != "" {
		return fmt.Sprintf("%s onto %s", action, street)
	}
	return action
}
