# Contributing to ocost-splitter

Thanks for taking the time to contribute.

## What this project is

A single static Go binary that splits the unattributed cost of a shared Kubernetes cluster across namespaces, using OpenCost allocation data and one of four split methods. MIT licence.

## Getting started

```
git clone https://github.com/rahmankhanjohn/ocost-splitter.git
cd ocost-splitter
go test ./...
```

All tests should pass on Go 1.24 or newer. No install step.

## Rules of the codebase

1. **No dependencies.** Standard library only. If a feature needs a third-party package, it does not go in this repo.
2. **Static binary.** No runtime dependencies. The binary drops into a distroless container.
2. **Single file.** `cmd/ocost-splitter/main.go` is the whole tool. Keep it that way.
3. **Every feature needs a test.** Add a test in `test_lab_status.py` that fails without your change.
4. **Exit codes matter.** 0 = pass, 1 = fail, 2 = error. Do not break this contract.
5. **Plain English in output.** Every number carries a plain sentence explaining what it means. No jargon without a gloss.
6. **No em dashes.** Use periods or commas.

## How to contribute

1. Fork the repository.
2. Create a branch: `git checkout -b feat/your-change`.
3. Make your change. Add or update tests.
4. Run the full suite: `go test ./...`.
5. Run the smoke test: `go build -o ocost-splitter ./cmd/ocost-splitter/ && go test ./...`.
6. Open a pull request with a short description of what you changed and why.

## What we are looking for

- Additional checks and validation rules
- Better heuristics for edge cases
- Additional output formats
- Documentation improvements
- Bug reports with reproduction steps

## What we are not looking for

- New dependencies
- Rewriting the tool in another language
- Features that require elevated privileges

## Reporting bugs

Open a GitHub issue with:
- The command you ran
- The output you got
- The output you expected
- Your Go version and operating system

## Code of conduct

Be direct, be specific, be kind. This is a tool for operators who are already having a hard quarter. Don't make them work harder to use it.
