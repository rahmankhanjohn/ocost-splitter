// Command ocost-splitter splits the unattributed cost of a shared Kubernetes
// cluster across the namespaces that share it.
//
// The problem it solves: in a multi-tenant cluster, shared cost lands on no
// namespace, so cost per tenant cannot be stated. This reads the allocation
// data OpenCost already produces and applies a split method that can be
// written down and defended.
//
// Examples:
//
//	ocost-splitter --file allocation.json --method usage
//	ocost-splitter --url http://opencost:9090 --method even
//	ocost-splitter --file allocation.json --method weights --weights revenue.csv --report
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rahmankhanjohn/ocost-splitter/internal/allocation"
	"github.com/rahmankhanjohn/ocost-splitter/internal/split"
)

const toolURL = "https://github.com/rahmankhanjohn/ocost-splitter"

func main() {
	var (
		file        = flag.String("file", "", "allocation json file to read instead of calling the API")
		url         = flag.String("url", "http://localhost:9090", "OpenCost base URL")
		method      = flag.String("method", "usage", "split method: even, usage, weights or direct")
		weightsFile = flag.String("weights", "", "csv of namespace,weight for the weights method")
		jsonOut     = flag.Bool("json", false, "machine readable output")
		report      = flag.Bool("report", false, "one paste ready line")
		maxUnattr   = flag.Float64("max-unattributed", 0, "fail if unattributed cost exceeds this share of the total, for CI")
	)
	flag.Parse()

	var (
		alloc *allocation.Allocation
		err   error
	)
	if *file != "" {
		alloc, err = allocation.LoadFile(*file)
	} else {
		alloc, err = allocation.NewClient(*url).Fetch()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read allocation data: %v\n", err)
		os.Exit(2)
	}

	weights := map[string]float64{}
	if *weightsFile != "" {
		weights, err = allocation.LoadWeights(*weightsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not read weights file: %v\n", err)
			os.Exit(2)
		}
	}

	chosen := split.Method(strings.ToLower(*method))
	switch chosen {
	case split.MethodEven, split.MethodUsage, split.MethodDirect, split.MethodWeights:
	default:
		fmt.Fprintf(os.Stderr, "unknown method %q, use even, usage, weights or direct\n", *method)
		os.Exit(2)
	}

	result := split.Split(alloc, chosen, weights)

	switch {
	case *jsonOut:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(2)
		}
	case *report:
		fmt.Println(reportLine(result))
	default:
		fmt.Print(render(result))
	}

	if *maxUnattr > 0 && result.ClusterTotal > 0 {
		share := result.Remaining / result.ClusterTotal * 100
		if share > *maxUnattr {
			fmt.Fprintf(os.Stderr, "unattributed cost is %.1f%% of the total, above the %.1f%% limit\n",
				share, *maxUnattr)
			os.Exit(1)
		}
	}
}

func render(r *split.Result) string {
	var b strings.Builder
	b.WriteString("OpenCost shared cost split\n")
	b.WriteString(strings.Repeat("=", 70) + "\n")
	b.WriteString(fmt.Sprintf("  Method          %s\n", r.Method))
	b.WriteString(fmt.Sprintf("  Window          %s\n", r.Window))
	b.WriteString(fmt.Sprintf("  Cluster total   %s\n", money(r.ClusterTotal)))
	b.WriteString(fmt.Sprintf("  Direct          %s\n", money(r.Direct)))
	b.WriteString(fmt.Sprintf("  Shared pool     %s\n", money(r.Pool)))
	b.WriteString(strings.Repeat("-", 70) + "\n")
	fmt.Fprintf(&b, "  %-32s %12s %12s %12s %8s\n", "NAMESPACE", "DIRECT", "ALLOCATED", "TOTAL", "SHARE")
	for _, line := range r.Lines {
		fmt.Fprintf(&b, "  %-32s %12s %12s %12s %7.1f%%\n",
			truncate(line.Namespace, 32), money(line.Direct), money(line.Allocated),
			money(line.Total), line.Share)
	}
	b.WriteString(strings.Repeat("-", 70) + "\n")
	b.WriteString(fmt.Sprintf("  %-32s %12s %12s %12s\n", "POOL REMAINING", "", money(r.Remaining), ""))
	b.WriteString("\n")
	for _, note := range r.Notes {
		b.WriteString("  note: " + note + "\n")
	}
	if len(r.Notes) == 0 {
		b.WriteString("  Every dollar of the shared pool is attributed.\n")
	}
	b.WriteString("\n")
	b.WriteString("  Write the method down, agree it with finance, apply it the same way every month.\n")
	b.WriteString("  A different method each month is not a method.\n")
	return b.String()
}

func reportLine(r *split.Result) string {
	parts := []string{
		fmt.Sprintf("OpenCost shared cost split (%s): %s of %s cluster cost attributed to %d namespaces, %s of shared pool still unattributed.",
			r.Method, money(r.ClusterTotal-r.Remaining), money(r.ClusterTotal), len(r.Lines), money(r.Remaining)),
		fmt.Sprintf("Calculated %s.", time.Now().Format("2006-01-02")),
		fmt.Sprintf("Tool: %s", toolURL),
	}
	return strings.Join(parts, " ")
}

func money(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func truncate(text string, width int) string {
	if len(text) <= width {
		return text
	}
	return text[:width-1] + "~"
}
