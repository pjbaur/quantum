# Plan: P4 — Remaining Subsystem Concerns

Spec: `specs/architecture/code_quality_assessment.md`, section "Priority 4:
Remaining Subsystem Concerns (queued 2026-08-25)" (items 4.1–4.8). The spec
is the binding authority; this plan maps each item to one task.

## Global Constraints

- AGENTS.md binds all tasks: smallest change that solves the problem; no
  renames of exported symbols; no new dependencies (stdlib only); `gofmt`
  on edited files; `go vet` + `go test` on affected packages (prefer
  `go test ./...`); new behavior requires tests; prefer table-driven tests;
  do not weaken assertions.
- Typed errors: follow the existing taxonomy in `quantum/errortypes.go`
  (structured error types with diagnostic fields). No bare `fmt.Errorf`
  for new error conditions in library packages.
- CLI output strings are pinned by ADR-0006 contract tests in
  `cmd/quantum/main_test.go` — demo/banner output must not change except
  where a task explicitly says so.
- Commit style: conventional commits (`feat:`, `fix:`, `docs:`, `test:`,
  `refactor:`, `ci:`), matching existing history.
- Go 1.25 toolchain; CI also runs staticcheck — keep `staticcheck ./...`
  clean.

## Task 1: Make `quantum all` safe when piped (spec item 4.1)

`cmd/quantum/main.go` `runDemos`, case `"all"` (lines 104–138): six
unguarded `fmt.Scanln()` calls with discarded errors ("Press Enter to
continue to ..." pauses). Piped/CI invocation of `quantum all` hangs
forever on the first pause.

Required changes:

1. Add a `-no-pause` boolean flag (default false) to the existing flag set
   in `main.go`, documented in the usage/help output alongside existing
   flags: when set, the `all` demo runs straight through with no pauses.
2. Auto-disable pauses when stdin is not a terminal: use
   `os.Stdin.Stat()` and check `mode & os.ModeCharDevice == 0` (stdlib
   only — no `x/term`). Piped `quantum all` must complete without any
   interaction.
3. Stop discarding the `Scanln` error: extract the repeated
   pause-and-prompt into one small helper (e.g. `pause(w io.Writer, next
   string)` or method on existing structure — follow local style). On
   `Scanln` error (e.g. EOF when stdin closes mid-run), disable all
   subsequent pauses instead of spamming prompts — do not abort the demos.
4. Interactive behavior (TTY, no flag) must stay byte-identical: same
   prompt strings, same ordering.
5. Tests: extend `cmd/quantum/main_test.go` (ADR-0006 contract tests):
   - `quantum all` with stdin from a pipe (or `-no-pause`) completes
     without hanging and exits 0, with all section banners present.
   - `-no-pause` suppresses the "Press Enter" prompt strings.
   - Existing contract tests keep passing.
   Note the demos are slow (~6s package test); reuse the existing test
   harness pattern in `main_test.go` rather than inventing a new one.

## Task 2: Untrack the committed pprof binaries (spec item 4.2)

Tracked binaries: `CHANGES/profiles/ws1-circuit-cpu.pprof` and
`CHANGES/profiles/ws1-circuit-mem.pprof`. `README.md` (Benchmarking
section, ~lines 125–137) documents a workflow that regenerates them into
the tracked path.

Required changes:

1. `git rm --cached CHANGES/profiles/*.pprof` (keep local files on disk).
2. Add `CHANGES/profiles/` (or `*.pprof` under it) to `.gitignore`.
3. Update the README Benchmarking section so the documented
   `-cpuprofile`/`-memprofile` workflow writes to the now-ignored path and
   note that profiles are local artifacts, not tracked.
4. No Go code changes. Verify `git status` shows the pprof files as
   untracked-and-ignored afterward.

## Task 3: Wire `MaxGateQubits` into `circuit.checkCapabilities` (spec item 4.3)

`quantum.BackendCapabilities` has two methods; only `SupportsGateQubits`
is consulted (`circuit/circuit.go:173`). `MaxGateQubits` has zero non-test
callers. Ruling (recorded in ledger): wire it in — do NOT deprecate.

Required changes:

1. In `circuit/circuit.go` capability pre-check (the loop containing line
   173), also verify each gate's required qubit count against
   `caps.MaxGateQubits()`; on violation return the same typed error class
   the existing check uses (`UnsupportedOperationError` — match existing
   fields/message style, including which gate and counts are reported).
   Keep `SupportsGateQubits` as the primary check; `MaxGateQubits` is a
   hard upper bound.
2. Doc comment on the check (or on the interface) stating both halves of
   the contract are enforced.
3. Tests in `circuit/`: table-driven test with a stub backend whose
   `SupportsGateQubits` returns true but `MaxGateQubits` returns a bound
   smaller than the gate — Execute must fail fast with the typed error
   before any state mutation. Also cover the passing boundary
   (gate size == MaxGateQubits).

## Task 4: Contract for uncomparable states in `validateIndependentStates` (spec item 4.4)

`circuit/parallel.go:16` keys `seen` map by `quantum.QuantumState`
interface values. An implementation with an uncomparable dynamic type
(e.g. a struct value containing a slice, rather than a pointer) panics at
runtime. Doc comment is silent. Ruling: document AND guard (a documented
panic is still a panic).

Required changes:

1. Extend `validateIndependentStates`'s doc comment: states must have
   comparable dynamic types (pointer implementations, as all in-repo
   backends are).
2. Cold-path guard before map insertion: `reflect.TypeOf(exec.State).Comparable()`
   check; on failure return a typed error following the
   `quantum/errortypes.go` taxonomy (add a small structured type, e.g.
   `UncomparableStateError` with the execution index and type name, if no
   existing type fits — check the taxonomy first).
3. Table-driven test: a deliberately uncomparable `QuantumState`
   implementation returns the typed error (not a panic); pointer states
   still validate as before.
4. Also document the comparability requirement wherever
   `ExecuteAllParallel`'s user-facing contract lives (doc comment on the
   exported function).

## Task 5: Bloch vectors for multi-qubit states (spec item 4.5)

`visualization.BlochVectorFromQubit` handles only single `qubit.Qubit`s.
`internal/density` has `ReducedBlochVector` but no bridge from a state
vector. Goal: Bloch export for qubits inside arbitrary multi-qubit pure
states.

Required changes:

1. `internal/density`: add `FromState(s quantum.QuantumState) (*State, error)`
   (or equivalent constructor) building the density matrix ρ = |ψ⟩⟨ψ| from
   the state's amplitudes via the existing amplitude accessors on
   `quantum.QuantumState`. Validate qubit count bounds consistently with
   the package's existing constructors (per ADR-0004, return typed errors,
   never coerce).
2. `visualization`: add an exported entry point, e.g.
   `BlochVectorFromState(s quantum.QuantumState, target int) (BlochVector, error)`
   (match the existing Bloch API's naming/return conventions in
   `visualization/bloch.go`), implemented via `density.FromState` +
   `ReducedBlochVector`. Validate `target` range with a typed error.
3. Tests (table-driven):
   - Product states: |0⟩⊗|+⟩ gives target-0 vector (0,0,1) and target-1
     vector (1,0,0) within tolerance.
   - Entangled: Bell pair (|00⟩+|11⟩)/√2 → reduced vector of EACH qubit is
     the zero vector (maximally mixed) within tolerance.
   - Agreement: for a single-qubit state, `BlochVectorFromState` matches
     `BlochVectorFromQubit` for the same amplitudes.
   - Error paths: target out of range; nil/unsupported state if the
     accessor contract requires it.
4. Works for both dense and sparse backends if both expose the needed
   amplitude accessor; if the accessor is missing on one backend, follow
   the established capability-error pattern (`UnsupportedOperationError`)
   rather than type-switching on concrete types.

## Task 6: Record density's deliberate scope (spec item 4.6)

`internal/density` implements fixed-arity `ApplySingleQubitGate` etc.,
not `quantum.QuantumState`. Ruling (recorded in ledger): do NOT implement
the interface now — document the deliberate scope instead. Rationale:
measurement/collapse semantics on density matrices is real design work
with no current consumer; the package now has two consumers (noise demo,
Task 5 Bloch bridge) in its current shape.

Required changes:

1. New ADR `docs/adr/0007-density-backend-scope.md` following the existing
   ADR format (check `docs/adr/README.md` index conventions and update the
   index): status Accepted; decision that `internal/density` is a
   noise-demo and analysis backend (Kraus channels, reduced Bloch vectors)
   and deliberately does NOT implement `quantum.QuantumState`; consequences
   (cannot join circuit abstraction; revisit trigger = a concrete consumer
   needing density-matrix circuits/measurement).
2. Package doc comment on `internal/density/state.go` stating the scope in
   one or two lines and pointing at the ADR.
3. If Task 5 landed `FromState`, reference it as the state-vector bridge.
   Docs only — no behavior change.

## Task 7: Extract shared example banner code (spec item 4.7)

`internal/examples/utils.go` holds little while `=====` banner/section
blocks are hand-inlined in `internal/examples/bell.go` (~257–265),
`internal/examples/hadamard.go` (~201–219), `internal/examples/tgate.go`
(~234–243), and `cmd/quantum/main.go` (the "QUANTUM COMPUTING IN GO /
ALL DEMONSTRATIONS" and "ALL DEMONSTRATIONS COMPLETED" blocks in the
`all` case). Locate the blocks by content — Task 1 may have shifted
main.go line numbers.

Required changes:

1. Add shared banner/section helper(s) to `internal/examples/utils.go`
   (e.g. `PrintBanner(title ...string)` — design to fit ALL existing call
   sites, including multi-line titles and the different `=` widths if they
   differ; check actual strings first).
2. Replace the inlined blocks in the four files with helper calls.
   Output must remain BYTE-IDENTICAL — ADR-0006 contract tests pin CLI
   strings, and demo output is compared by eye; do not "normalize" widths
   or spacing.
3. Verification: run the CLI contract tests; additionally capture
   `go run ./cmd/quantum <demo>` output before/after (or rely on existing
   tests where they pin these strings) and state in the report how
   byte-identity was verified.
4. `cmd/quantum` may call the helper only if `internal/examples` is
   already an import; it is (demos are invoked from main.go).

## Task 8: Small-sweep leftovers (spec item 4.8)

Three independent mechanical items, batched:

1. **(a)** Test `visualization.DefaultStateViewOptions` — the package's
   only 0%-covered function. Table-driven or direct: assert each default
   field value in the returned `StateViewOptions`.
2. **(b)** Add to `docs/deprecation-policy-v2.md` the same versioning note
   `docs/MIGRATION-v2.md:5-10` carries (blockquote "Versioning note
   (2026-08-25)": "v2" names the API generation, not a Go module major
   version; releases tagged v0.x; module path unchanged). Adapt wording
   minimally to fit the policy doc's context.
3. **(c)** Trim CI's triple suite execution in
   `.github/workflows/ci.yml`: currently `go test ./...` (plain), then
   `go test -race ./...`, then `go test ./... -coverprofile=...` — three
   full suite runs per matrix leg. Drop the plain `Go test` step; keep
   race and coverage (they jointly cover everything the plain run does).
   Keep step names/structure otherwise intact.
   Ruling (recorded in ledger): pre-commit hooks are NOT added — decision
   recorded, not implemented.
