# ocost-splitter

![Tests](https://github.com/rahmankhanjohn/ocost-splitter/actions/workflows/test.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![XERJ Corpus Hub contributor](https://hub.xerj.org/contributors/rahmankhanjohn/badge.svg)
![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8.svg)
![Static binary](https://img.shields.io/badge/static--binary-yes-brightgreen.svg)
![Tests: 11](https://img.shields.io/badge/tests-11%20passing-brightgreen.svg)
![CNCF Candidate](https://img.shields.io/badge/CNCF-contribution%20candidate-blue.svg)
![GitHub stars](https://img.shields.io/github/stars/rahmankhanjohn/ocost-splitter?style=social)
![Last commit](https://img.shields.io/github/last-commit/rahmankhanjohn/ocost-splitter)

Split the unattributed cost of a shared Kubernetes cluster across the namespaces that share it.

In a multi-tenant cluster, shared cost lands on no namespace. DNS, ingress, monitoring, the control plane fee, cross-namespace transfer. No tag splits it, so cost per tenant cannot be stated, and the honest number in every report is a guess.

This reads the allocation data [OpenCost](https://opencost.io) already produces and applies a split method that can be written down and defended. Four methods, one number, one report line.

## Why Go

Single static binary. No runtime, no dependency tree, drops into a distroless container or a CI image. And the CNCF track: OpenCost, Kubernetes, Cilium and the rest of the ecosystem are Go, so this is also the code shape an upstream contribution would take.

## Build

```
go build -o ocost-splitter ./cmd/ocost-splitter
```

Windows binary:

```
go build -o ocost-splitter.exe ./cmd/ocost-splitter
```

Linux, no CGO, fully static:

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ocost-splitter ./cmd/ocost-splitter
```

## Use

Against a running OpenCost:

```
ocost-splitter --url http://opencost:9090 --method usage
```

Against a saved allocation file:

```
ocost-splitter --file allocation.json --method usage
```

With external weights, for example revenue or headcount:

```
ocost-splitter --file allocation.json --method weights --weights revenue.csv
```

One paste-ready line for a status report:

```
ocost-splitter --file allocation.json --method usage --report
```

## Methods

| Method | What it does | When to use it |
|--------|--------------|----------------|
| `usage` | Proportional to measured CPU and RAM consumption | Default. Defensible, tracks real consumption |
| `even` | Equal share per namespace | Simple, and honest when consumption data is thin |
| `weights` | Proportional to an external weight map, revenue or headcount | When finance wants value, not consumption |
| `direct` | No split, the pool stays unattributed | The baseline to compare the others against |

Run `direct` first. It shows what the cluster looks like with no assumptions, and every other method is an improvement on it.

## Example

```
$ ocost-splitter --file allocation.json --method usage
OpenCost shared cost split
======================================================================
  Method          usage
  Window          2026-10-19T00:00:00Z
  Cluster total   1000.00
  Direct          800.00
  Shared pool     200.00
----------------------------------------------------------------------
  NAMESPACE                        DIRECT    ALLOCATED        TOTAL    SHARE
  payments                        380.00     124.35       504.35   50.4%
  kube-system                     150.00      20.73       170.73   17.1%
  default                         120.00      31.09       151.09   15.1%
  staging                          90.00      13.47       103.47   10.3%
  reporting                        60.00      10.35        70.35    7.0%
----------------------------------------------------------------------
  POOL REMAINING                               0.00

  Every dollar of the shared pool is attributed.

  Write the method down, agree it with finance, apply it the same way every month.
  A different method each month is not a method.
```

## CI use

Fail a build when too much cost stays unattributed:

```
ocost-splitter --url http://opencost:9090 --method usage --max-unattributed 5
```

Exit code 1 when the unattributed share is above the limit, 0 otherwise, 2 on error.

## The part that is not code

The tool does not decide the method. A split method is a financial policy, and the policy is what finance signs. Use the tool to make the consequences of each method visible, then write the chosen one down and apply it the same way every month.

A different method each month is not a method.

## Tests

```
go test ./...
go vet ./...
```

## Where this comes from

Written while preparing *Kubernetes Cost: The 30 Percent No Tag Can Split*, and built to be upstreamable to OpenCost rather than to sit next to it.

## Recognition

Indexed on the [XERJ Corpus Hub](https://hub.xerj.org/contributors/rahmankhanjohn) - the open reference memory AI coding agents search before they write code.

## License

MIT
