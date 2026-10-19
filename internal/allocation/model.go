// Package allocation models OpenCost allocation data and loads it from a file
// or from a running OpenCost API.
package allocation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Namespace holds the cost and usage of one namespace in an allocation window.
type Namespace struct {
	Total     float64 `json:"total"`
	Idle      float64 `json:"idle"`
	Shared    float64 `json:"shared"`
	Other     float64 `json:"other"`
	Compute   float64 `json:"compute"`
	Storage   float64 `json:"storage"`
	Network   float64 `json:"network"`
	CPUCoreHr float64 `json:"cpuCoreHours"`
	RAMGiBHr  float64 `json:"ramGiBHours"`
	GPUHr     float64 `json:"gpuHours"`
}

// Allocation is one allocation window returned by OpenCost.
type Allocation struct {
	Window     string                     `json:"window"`
	Total      float64                    `json:"total"`
	Idle       float64                    `json:"idle"`
	Shared     float64                    `json:"shared"`
	Other      float64                    `json:"other"`
	Compute    float64                    `json:"compute"`
	Storage    float64                    `json:"storage"`
	Network    float64                    `json:"network"`
	Namespaces map[string]*Namespace      `json:"namespaces"`
	Nodes      map[string]json.RawMessage `json:"nodes"`
}

// Response is the envelope OpenCost uses for allocation data.
type Response struct {
	Data []Allocation `json:"data"`
}

// Unattributed returns the cost that belongs to no single namespace: shared,
// idle and other. This is the hard part of a shared cluster.
func (a *Allocation) Unattributed() float64 {
	return a.Shared + a.Idle + a.Other
}

// Direct returns the cost attributed to namespaces before any split.
func (a *Allocation) Direct() float64 {
	total := 0.0
	for _, ns := range a.Namespaces {
		total += ns.Total
	}
	return total
}

// Usage returns a usage weight for a namespace, CPU core hours plus RAM GiB
// hours. Used when splitting by consumption rather than by count.
func (n *Namespace) Usage() float64 {
	return n.CPUCoreHr + n.RAMGiBHr
}

// Parse reads an OpenCost allocation document.
func Parse(r io.Reader) (*Allocation, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var response Response
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse allocation json: %w", err)
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("allocation document contains no data windows")
	}
	latest := response.Data[len(response.Data)-1]
	if latest.Namespaces == nil {
		latest.Namespaces = map[string]*Namespace{}
	}
	return &latest, nil
}

// LoadFile reads an allocation document from disk.
func LoadFile(path string) (*Allocation, error) {
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return Parse(handle)
}

// Client reads allocation data from a running OpenCost instance.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client for the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// endpoints are the paths OpenCost serves allocation data on, newest first.
var endpoints = []string{"/model/allocation", "/allocation"}

// Fetch retrieves the latest allocation window.
func (c *Client) Fetch() (*Allocation, error) {
	var lastErr error
	for _, path := range endpoints {
		url := c.BaseURL + path
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/json")
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("%s returned 404", url)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s returned %s", url, resp.Status)
			continue
		}
		return Parse(strings.NewReader(string(body)))
	}
	return nil, fmt.Errorf("could not read allocation data: %w", lastErr)
}

// LoadWeights reads a simple name,weight CSV used for revenue or headcount
// based splitting. Lines may be "name,value" and blank lines are ignored.
func LoadWeights(path string) (map[string]float64, error) {
	handle, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	weights := map[string]float64{}
	scanner := bufio.NewScanner(handle)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}
		weights[strings.TrimSpace(parts[0])] = value
	}
	return weights, scanner.Err()
}
