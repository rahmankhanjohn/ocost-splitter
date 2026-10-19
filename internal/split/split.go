// Package split allocates the unattributed cost of a shared cluster across the
// namespaces that share it, using a method that can be written down and
// defended.
package split

import (
	"fmt"
	"sort"

	"github.com/rahmankhanjohn/ocost-splitter/internal/allocation"
)

// Method names the way the unattributed cost was shared out.
type Method string

const (
	// MethodEven splits the pool equally between namespaces.
	MethodEven Method = "even"
	// MethodUsage splits the pool in proportion to measured usage.
	MethodUsage Method = "usage"
	// MethodDirect leaves the pool unattributed, which is the honest baseline.
	MethodDirect Method = "direct"
	// MethodWeights splits the pool using an external weight map, such as
	// revenue or headcount.
	MethodWeights Method = "weights"
)

// Line is one namespace after the split.
type Line struct {
	Namespace string  `json:"namespace"`
	Direct    float64 `json:"direct"`
	Allocated float64 `json:"allocated"`
	Total     float64 `json:"total"`
	Share     float64 `json:"share_percent"`
}

// Result is the outcome of a split.
type Result struct {
	Method       Method   `json:"method"`
	Window       string   `json:"window"`
	ClusterTotal float64  `json:"cluster_total"`
	Direct       float64  `json:"direct"`
	Pool         float64  `json:"unattributed_pool"`
	Allocated    float64  `json:"allocated"`
	Remaining    float64  `json:"remaining"`
	Lines        []Line   `json:"lines"`
	Notes        []string `json:"notes"`
}

// Split divides the unattributed pool across namespaces.
//
// weights is optional and used by MethodWeights. A namespace missing from the
// weight map gets no allocation and is reported in the notes.
func Split(a *allocation.Allocation, method Method, weights map[string]float64) *Result {
	result := &Result{
		Method:       method,
		Window:       a.Window,
		ClusterTotal: a.Total,
		Direct:       a.Direct(),
		Pool:         a.Unattributed(),
	}

	names := make([]string, 0, len(a.Namespaces))
	for name := range a.Namespaces {
		names = append(names, name)
	}
	sort.Strings(names)

	allocations := map[string]float64{}
	switch method {
	case MethodDirect:
		result.Notes = append(result.Notes, "no split applied, the pool stays unattributed")
	case MethodUsage:
		allocations = usageWeights(a, names, result)
	case MethodWeights:
		allocations = externalWeights(a, names, weights, result)
	default:
		allocations = evenWeights(names)
	}

	allocated := 0.0
	// weights are relative, turn them into money
	totalWeight := 0.0
	for _, weight := range allocations {
		totalWeight += weight
	}
	if totalWeight > 0 && result.Pool > 0 {
		for name, weight := range allocations {
			allocations[name] = weight / totalWeight * result.Pool
		}
	}
	for _, name := range names {
		ns := a.Namespaces[name]
		share := allocations[name]
		allocated += share
		result.Lines = append(result.Lines, Line{
			Namespace: name,
			Direct:    ns.Total,
			Allocated: share,
			Total:     ns.Total + share,
		})
	}
	result.Allocated = allocated
	result.Remaining = result.Pool - allocated

	if result.ClusterTotal > 0 {
		for i := range result.Lines {
			result.Lines[i].Share = result.Lines[i].Total / result.ClusterTotal * 100
		}
	}
	sort.Slice(result.Lines, func(i, j int) bool {
		return result.Lines[i].Total > result.Lines[j].Total
	})
	return result
}

func evenWeights(names []string) map[string]float64 {
	weights := map[string]float64{}
	if len(names) == 0 {
		return weights
	}
	for _, name := range names {
		weights[name] = 1
	}
	return weights
}

func usageWeights(a *allocation.Allocation, names []string, result *Result) map[string]float64 {
	weights := map[string]float64{}
	total := 0.0
	for _, name := range names {
		usage := a.Namespaces[name].Usage()
		weights[name] = usage
		total += usage
	}
	if total == 0 {
		result.Notes = append(result.Notes,
			"no usage data in this window, fell back to an even split")
		return evenWeights(names)
	}
	return weights
}

func externalWeights(a *allocation.Allocation, names []string, weights map[string]float64, result *Result) map[string]float64 {
	if len(weights) == 0 {
		result.Notes = append(result.Notes, "no weight file supplied, fell back to an even split")
		return evenWeights(names)
	}
	chosen := map[string]float64{}
	total := 0.0
	missing := []string{}
	for _, name := range names {
		weight, ok := weights[name]
		if !ok || weight <= 0 {
			missing = append(missing, name)
			continue
		}
		chosen[name] = weight
		total += weight
	}
	if len(missing) > 0 {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"no weight for %d namespace(s), they receive nothing: %s",
			len(missing), joinNames(missing)))
	}
	if total == 0 {
		result.Notes = append(result.Notes, "weights summed to zero, fell back to an even split")
		return evenWeights(names)
	}
	return chosen
}

func joinNames(names []string) string {
	out := ""
	for i, name := range names {
		if i > 0 {
			out += ", "
		}
		out += name
	}
	return out
}
