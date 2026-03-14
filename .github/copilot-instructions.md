# Copilot Instructions

## Language & Style

- Write idiomatic Go — follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `gofmt`/`goimports` formatting. Never argue about style that `gofmt` already decides.
- Prefer short, clear variable names scoped tightly (`r` for a request in a handler, `ctx` for context).
- Use named return values only when they improve documentation; avoid naked returns.

## Error Handling

- Always handle errors explicitly. Never use `_` to discard an error unless you add a comment explaining why.
- Wrap errors with `fmt.Errorf("doing X: %w", err)` to build useful error chains.
- Prefer returning errors over panicking. Reserve `panic` for truly unrecoverable situations.

## Package Design

- Keep packages small and focused. Name them with short, lowercase, single-word names.
- Avoid `package utils` or `package common` — be specific.
- Export only what consumers need. Start unexported; promote to exported when required.

## Concurrency

- Don't start goroutines without a clear shutdown/cancellation strategy (use `context.Context`).
- Prefer channels for communication and `sync` primitives for state protection.
- Document goroutine ownership and lifecycle.

## Testing

- Place tests in the same package (`_test.go`) for white-box tests, or `package foo_test` for black-box tests.
- Use table-driven tests where appropriate.
- Use `t.Helper()` in test helpers so failure locations are reported correctly.
- Prefer the standard `testing` package; avoid heavy assertion libraries.

## Project Layout

- Follow the standard Go project layout: `cmd/`, `internal/`, `pkg/` (only if truly public).
- Keep `main.go` thin — it should wire dependencies and call into `internal/` packages.

## Dependencies

- Prefer the standard library. Only add third-party dependencies when they provide significant value.
- Pin dependency versions via `go.sum`.

## Documentation

- Write doc comments on all exported identifiers (`// FuncName does …`).
- Start doc comments with the identifier name.
