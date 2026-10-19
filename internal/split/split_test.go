package split

import (
	"strings"
	"testing"

	"github.com/rahmankhanjohn/ocost-splitter/internal/allocation"
)

const sample = `{
  "data": [
    {
      "window": "2026-10-19T00:00:00Z",
      "total": 1000,
      "idle": 80,
      "shared": 120,
      "other": 0,
      "namespaces": {
        "default":    {"total": 120, "cpuCoreHours": 200, "ramGiBHours": 400},
        "payments":   {"total": 380, "cpuCoreHours": 800, "ramGiBHours": 1600},
        "kube-system":{"total": 150, "cpuCoreHours": 120, "ramGiBHours": 300},
        "reporting":  {"total": 60,  "cpuCoreHours": 90,  "ramGiBHours": 180},
        "staging":    {"total": 90,  "cpuCoreHours": 60,  "ramGiBHours": 110}
      }
    }
  ]
}`

func load(t *testing.T) *allocation.Allocation {
	t.Helper()
	a, err := allocation.Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return a
}

func totalFor(r *Result, name string) float64 {
	for _, line := range r.Lines {
		if line.Namespace == name {
			return line.Total
		}
	}
	return 0
}

func TestEvenSplitDistributesTheWholePool(t *testing.T) {
	r := Split(load(t), MethodEven, nil)
	if r.Allocated != 200.0 {
		t.Errorf("allocated = %v, want 200", r.Allocated)
	}
	if r.Remaining != 0 {
		t.Errorf("remaining = %v, want 0", r.Remaining)
	}
	// 200 split over 5 namespaces is 40 each
	if got := totalFor(r, "default"); got != 160.0 {
		t.Errorf("default total = %v, want 160", got)
	}
}

func TestUsageSplitFollowsConsumption(t *testing.T) {
	r := Split(load(t), MethodUsage, nil)
	// total usage is 200+400+800+1600+120+300+90+180+60+110 = 3860
	// payments usage 2400 of 3860, pool 200
	expected := 380.0 + 200.0*2400.0/3860.0
	got := totalFor(r, "payments")
	if diff := got - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("payments total = %v, want about %v", got, expected)
	}
}

func TestDirectMethodLeavesThePoolUnattributed(t *testing.T) {
	r := Split(load(t), MethodDirect, nil)
	if r.Allocated != 0 {
		t.Errorf("allocated = %v, want 0", r.Allocated)
	}
	if r.Remaining != 200 {
		t.Errorf("remaining = %v, want 200", r.Remaining)
	}
	if len(r.Notes) == 0 {
		t.Error("expected a note explaining that no split was applied")
	}
}

func TestWeightsMethodUsesTheWeightMap(t *testing.T) {
	weights := map[string]float64{"payments": 900, "default": 100}
	r := Split(load(t), MethodWeights, weights)
	// only two namespaces have weights, 900 and 100
	if got := totalFor(r, "payments"); got != 380+180 {
		t.Errorf("payments total = %v, want 560", got)
	}
	if got := totalFor(r, "reporting"); got != 60 {
		t.Errorf("reporting total = %v, want 60, namespaces without weight get nothing", got)
	}
	if r.Remaining != 0 {
		t.Errorf("remaining = %v, want 0", r.Remaining)
	}
}

func TestWeightsWithoutFileFallsBackToEven(t *testing.T) {
	r := Split(load(t), MethodWeights, nil)
	if r.Allocated != 200 {
		t.Errorf("allocated = %v, want 200", r.Allocated)
	}
	if len(r.Notes) == 0 {
		t.Error("expected a note about the fallback")
	}
}

func TestSharesSumToOneHundred(t *testing.T) {
	r := Split(load(t), MethodUsage, nil)
	sum := 0.0
	for _, line := range r.Lines {
		sum += line.Share
	}
	if diff := sum - 100; diff > 0.01 || diff < -0.01 {
		t.Errorf("share sum = %v, want 100", sum)
	}
}

func TestLinesAreSortedByTotalDescending(t *testing.T) {
	r := Split(load(t), MethodUsage, nil)
	for i := 1; i < len(r.Lines); i++ {
		if r.Lines[i-1].Total < r.Lines[i].Total {
			t.Error("lines are not sorted by total descending")
		}
	}
}
