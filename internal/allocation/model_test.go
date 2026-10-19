package allocation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `{
  "data": [
    {
      "window": "2026-10-19T00:00:00Z",
      "total": 1000,
      "idle": 80,
      "shared": 120,
      "other": 0,
      "compute": 800,
      "storage": 200,
      "network": 0,
      "namespaces": {
        "default":    {"total": 120, "cpuCoreHours": 200, "ramGiBHours": 400},
        "payments":   {"total": 380, "cpuCoreHours": 800, "ramGiBHours": 1600},
        "kube-system":{"total": 150, "cpuCoreHours": 120, "ramGiBHours": 300},
        "reporting":  {"total": 60,  "cpuCoreHours": 90,  "ramGiBHours": 180},
        "staging":    {"total": 90,  "cpuCoreHours": 60,  "ramGiBHours": 110}
      },
      "nodes": {}
    }
  ]
}`

func TestParseReadsTheLatestWindow(t *testing.T) {
	a, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if a.Window != "2026-10-19T00:00:00Z" {
		t.Errorf("window = %q", a.Window)
	}
	if a.Total != 1000 {
		t.Errorf("total = %v", a.Total)
	}
	if len(a.Namespaces) != 5 {
		t.Errorf("namespaces = %d, want 5", len(a.Namespaces))
	}
}

func TestUnattributedIsSharedPlusIdlePlusOther(t *testing.T) {
	a, _ := Parse(strings.NewReader(sample))
	if got := a.Unattributed(); got != 200 {
		t.Errorf("unattributed = %v, want 200", got)
	}
}

func TestDirectSumsNamespaceTotals(t *testing.T) {
	a, _ := Parse(strings.NewReader(sample))
	if got := a.Direct(); got != 800 {
		t.Errorf("direct = %v, want 800", got)
	}
}

func TestUsageAddsCpuAndRam(t *testing.T) {
	a, _ := Parse(strings.NewReader(sample))
	if got := a.Namespaces["payments"].Usage(); got != 2400 {
		t.Errorf("usage = %v, want 2400", got)
	}
}

func TestParseRejectsEmptyDocument(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"data": []}`)); err == nil {
		t.Error("expected an error for an empty document")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse(strings.NewReader("not json")); err == nil {
		t.Error("expected an error for invalid json")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allocation.json")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a.Total != 1000 {
		t.Errorf("total = %v", a.Total)
	}
}

func TestLoadWeightsParsesCsv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weights.csv")
	content := "# comment\npayments,900\ndefault,100\n\nbroken,notanumber\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	weights, err := LoadWeights(path)
	if err != nil {
		t.Fatalf("load weights: %v", err)
	}
	if weights["payments"] != 900 {
		t.Errorf("payments = %v", weights["payments"])
	}
	if _, ok := weights["broken"]; ok {
		t.Error("expected unparsable rows to be skipped")
	}
}
