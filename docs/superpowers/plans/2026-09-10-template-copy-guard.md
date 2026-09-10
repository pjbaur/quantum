# Template Copy Guard (Backlog Item 21) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backlog item 21 by making `parameterized.Template` refuse `AddParamGate`, `AddGate`, and `Bind` on a by-value copy taken after a declaration, so a copy can no longer desync the original's `ParamNames` from its `ParamStepCounts` or let `Bind` accept an incomplete binding.

**Architecture:** `Template` gains an unexported `addr *Template` field in the style of `strings.Builder`: the two writers set it to the receiver on the first accepted declaration, and all three methods that read `seen` or append to a slice compare it with the receiver first, returning one package-level error on a mismatch. The accessors (`NumQubits`, `ParamNames`, `ParamStepCounts`) are untouched, since they read only what a copy holds by value; `NewTemplate` and `Bind` never write `addr`, so a copy taken before any declaration stays an independent template and `Bind` stays a concurrent-safe read. The `Template` doc comment states the contract.

**Tech Stack:** Go standard library only. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`

## Classification

**Standard.** One new unexported field, one unexported helper, a guard at the top of three existing methods, and a one-line pin in two of them; no signature, exported type, or cross-package change. One task, one test cycle.

## Global Constraints

- Never edit a failing test to make code pass. The moved red test is the item's acceptance test; its setup, the original's declarations, the names-versus-counts loop, and the `Bind` assertion are copied verbatim. The one rewritten expectation (a declaration on the copy must error, not succeed) is ruled in the spec's "Red test rulings" and is written out in full in Step 2; make no other change to it. If it fails after implementation, the implementation is wrong. Report it.
- Error style in `parameterized/`: typed package-local errors (`MissingParameterError`, `UnknownParameterError`, `InvalidParameterValueError`) for `Bind`-time binding problems; untyped errors for caller bugs (`fmt.Errorf` for a nil factory, a nil gate, an empty target list). This plan adds one untyped, unexported package-level error, `errCopiedTemplate`, and no type (spec Decision 2). Its exact text is `Template copied by value after a declaration; a declared Template is used in place or through a pointer, never by copy`.
- The refusal is an error, never a panic (spec Decision 2). No method of `Template` panics on any receiver this plan can produce.
- Guard placement: `checkNotCopied` is the first statement of `AddParamGate`, `AddGate`, and `Bind`, before every argument check (spec Decision 3). The pin `t.addr = t` sits in `AddParamGate` and `AddGate` only, immediately after the `checkTargets` call succeeds and before the first mutation. `Bind` and `NewTemplate` never write `addr`.
- `NumQubits`, `ParamNames`, `ParamStepCounts`, `checkTargets`, `NewTemplate`, and every exported signature are not edited. The existing bodies of `AddParamGate`, `AddGate`, and `Bind` after the inserted lines are not edited.
- No `unsafe`, no `noescape` trick (spec Decision 3).
- Doc-comment voice: explain the why, cite conventions (`NewTemplate`, `strings.Builder`, `Bind`), no filler. Every comment in this plan is verbatim; do not reword it.
- Item 22's test stays tagged and must still fail. `TestRedDeclaredTargetsAreNotAliasedToCallerSlice` and its `// Backlog item 22:` comment stay in `parameterized/backlog_red_test.go` under `//go:build redtests`, unedited; the plan removes only the item 21 test, its comment, and the `errors` import that only it used. The file's header comment is unchanged. Do not delete the file.
- No new dependencies. Go standard library only.
- The task ends with `gofmt -l .` printing nothing, `go vet ./...`, `staticcheck ./...`, `go build ./...`, `go test ./...` green, `go test -race ./parameterized ./algorithm` green, `go vet -tags redtests ./parameterized ./algorithm` compiling, then a commit whose message ends with a blank line and the trailers `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11`.
- Do not edit `.superpowers/backlog/` or `docs/enhancement-backlog-2026-08-27.md`; the run lead ticks the item after the QA gate.

---

### Task 1: Refuse `AddParamGate`, `AddGate`, and `Bind` on a by-value copy of a declared `Template`

**Sizing estimate:** about 1,250 lines of existing code to read (`parameterized/parameterized.go` 218, `parameterized/parameterized_test.go` 445, `parameterized/backlog_red_test.go` 101, `strings/builder.go` in `$(go env GOROOT)/src` lines 1-60 (about 60), `algorithm/vqe.go` lines 10-30 and 305-311 (about 30), the spec in full (about 290), `CHANGELOG.md` lines 1-170); 2 non-test files modified (`parameterized/parameterized.go` +58/-5, `CHANGELOG.md` +20) plus 2 test files (`parameterized/parameterized_test.go` +122, `parameterized/backlog_red_test.go` -40); net diff about +155 lines; one test cycle. Under a quarter of a context window.

**Files:**
- Modify: `parameterized/parameterized.go:7-14` (the import block: add `errors`), `parameterized/parameterized.go:46-61` (the `Template` doc comment and struct: add the `addr` field, then the new error value and helper after the struct), `parameterized/parameterized.go:105-131` (`AddParamGate`: doc comment, the check as first statement, the pin after `checkTargets`), `parameterized/parameterized.go:133-153` (`AddGate`: doc comment, the check as first statement, the pin after `checkTargets`), `parameterized/parameterized.go:155-159` (`Bind`: doc comment and the check as first statement). Line numbers are at `d52e18f`, before any edit; later anchors shift as earlier edits land, so match on text.
- Modify: `CHANGELOG.md` (insert one entry at the end of `### Fixed` under `## Unreleased`, directly before the first `### Breaking`, line 168 at `d52e18f`)
- Test: `parameterized/parameterized_test.go` (append three tests at the end of the file after `TestAddNoTargetsErrorRendersEmptyGateNameVisibly`, which ends at line 445; no import change: `errors`, `strings`, `gates`, and `parameterized` are already imported)
- Test: `parameterized/backlog_red_test.go:17` (delete the `"errors"` import line) and `parameterized/backlog_red_test.go:26-64` (delete the item 21 comment, test, and the blank line after it, so the `// Backlog item 22:` comment directly follows the import block plus one blank line)

**Interfaces:**
- Consumes (existing, unchanged):
  - `type Template struct { numQubits int; steps []step; paramOrder []string; seen map[string]bool }` in `parameterized/parameterized.go:54-61` (gains the unexported field `addr *Template` as its first field)
  - `func NewTemplate(numQubits int) *Template` in `parameterized/parameterized.go:64` (not modified; never sets `addr`)
  - `func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error` in `parameterized/parameterized.go:112` (signature unchanged; gains the check and the pin)
  - `func (t *Template) AddGate(gate quantum.Gate, targets ...int) error` in `parameterized/parameterized.go:142` (signature unchanged; gains the check and the pin)
  - `func (t *Template) Bind(values Params) (*circuit.Circuit, error)` in `parameterized/parameterized.go:159` (signature unchanged; gains the check only)
  - `func (t *Template) checkTargets(targets []int) error` in `parameterized/parameterized.go:96` (not modified; the pin goes after its successful return)
  - `func wrapEvaluationError(where string, err error) error` in `algorithm/vqe.go:308` (wraps any `Bind` error as `*InvalidVQEInputError` with `Err` set; not modified; how the new error reaches `VQE` callers)
  - `strings.Builder`'s `addr *Builder` field and `copyCheck` method in `$(go env GOROOT)/src/strings/builder.go` (the precedent; read only)
  - `gates.NewHadamard()`, `parameterized.Ry`, `parameterized.Rx`, `parameterized.Params`, `parameterized.MissingParameterError`
- Produces (new, unexported):
  - field `addr *Template` on `Template`
  - `var errCopiedTemplate error` in `parameterized/parameterized.go`
  - `func (t *Template) checkNotCopied() error` in `parameterized/parameterized.go`
  - No later task. For item 22 (a later plan): the two `append(t.steps, step{... targets: targets})` lines are byte for byte what item 19 left them, one line below the new `t.addr = t` pin in each method; item 22's copy of `targets` goes into those lines or just above them without touching the pin or the check.

- [ ] **Step 1: Read the context**

1. `parameterized/parameterized.go` in full. Note the `Template` type at lines 46-61 (four fields, no self-pointer), `NewTemplate` at 63-66 (`&Template{numQubits: numQubits}`), `checkTargets` at 96-103, `AddParamGate` at 112-131 (the nil-factory check, the no-target check, `checkTargets`, the `seen` allocation, the `seen` lookup and `paramOrder` append, the `steps` append), `AddGate` at 142-153 (nil-gate check, no-target check, `checkTargets`, the `steps` append), and `Bind` at 159-189 (the `paramOrder` loop, the `seen` read per key of `values`, `circuit.New`, the `steps` loop). The three methods that read `seen` or append are the three you guard.
2. `$(go env GOROOT)/src/strings/builder.go` lines 1-60: the `addr *Builder` field, and `copyCheck`, which sets `addr` on first use and panics on `b.addr != b`. You copy the field and the comparison, not the panic and not the `abi.NoEscape` cast (spec Decisions 2 and 3).
3. `algorithm/vqe.go` lines 10-30 (`evaluate` returns `Bind`'s error as is) and 305-311 (`wrapEvaluationError` wraps it as `InvalidVQEInputError`). Nothing there changes; this is how the new error reaches a `VQE` caller.
4. `parameterized/backlog_red_test.go` lines 26-64: the test you are moving. Its final assertions (the names-versus-counts loop and the `Bind` check) are the acceptance criteria; its middle expectation (that `b.AddParamGate` succeeds) is the one the spec rules against. Lines 65-101 are item 22, which stays.
5. `parameterized/parameterized_test.go` lines 1-16 (imports: `errors`, `strings`, `gates`, `parameterized` are all present) and lines 396-445 (`TestAddNoTargetsErrorNamesTheGateAndDeclaresNothing` and `TestAddNoTargetsErrorRendersEmptyGateNameVisibly`, the `strings.Contains` style you follow; the file ends after the latter, and you append there).
6. The spec, in full: Decision 1 says why independence is impossible in Go and why the `strings.Builder` guard is the fit; Decision 2 says why the refusal is an error and what its text is; Decision 3 says which methods check, where the pin goes, and why `Bind` never pins; "Red test rulings" says exactly which expectation of the moved test changes and why.

- [ ] **Step 2: Move the red test, trim the red file's imports, and add the two new tests**

In `parameterized/backlog_red_test.go`, replace the import block (lines 16-24), that is, exactly

```go
import (
	"errors"
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)
```

with

```go
import (
	"testing"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/parameterized"
	"github.com/pjbaur/quantum/state"
)
```

Then delete exactly this block (lines 26-64: the item 21 comment, the function, and the blank line after it), so that the `// Backlog item 22:` comment directly follows the closing parenthesis of the import block plus one blank line:

```go
// Backlog item 21: copying a Template by value after a declaration
// desyncs ParamNames from ParamStepCounts.
//
// Template copied by value after AddParamGate × shared seen map, private
// paramOrder slice → a declaration added to one copy is counted in the
// other copy's ParamStepCounts (seen is shared) but absent from its
// ParamNames (paramOrder is not), so Bind on the stale copy accepts
// values that omit the name it still counts.
func TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent(t *testing.T) {
	a := *parameterized.NewTemplate(2)
	if err := a.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	b := a
	if err := b.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	if err := a.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	names := a.ParamNames()
	counts := a.ParamStepCounts()
	for name := range counts {
		found := false
		for _, n := range names {
			if n == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("a.ParamStepCounts() = %v lists %q but a.ParamNames() = %q lacks it", counts, name, names)
		}
	}
	var missing *parameterized.MissingParameterError
	if _, err := a.Bind(parameterized.Params{"theta": 0.1}); !errors.As(err, &missing) {
		t.Fatalf("a.Bind without phi: err = %v, want MissingParameterError", err)
	}
}

```

The `errors` import goes because only the item 21 test used it; item 22's test uses `testing`, `circuit`, `gates`, `parameterized`, and `state`. The header comment at the top of the file is unchanged.

In `parameterized/parameterized_test.go`, append to the end of the file (after the closing brace of `TestAddNoTargetsErrorRendersEmptyGateNameVisibly`). The first function is the moved test: same setup, same declarations on `a`, same names-versus-counts loop, same `Bind` assertion, renamed to the package's style; the `b.AddParamGate` expectation is inverted per the spec's "Red test rulings" (a declaration on the copy must now error), and the leading comment is rewritten to describe the contract. The second pins the shape of the refusal; the third pins that a copy taken before any accepted declaration is independent.

```go

// TestCopyAfterDeclarationKeepsNamesAndCountsConsistent pins that a
// Template copied by value after a declaration cannot desync the original
// (backlog item 21). Before the fix the copy shared the original's seen
// map but had its own paramOrder header, so a name declared on the copy
// was already known to the original: its next declaration of that name
// was counted by ParamStepCounts, absent from ParamNames, and never
// demanded by Bind. A declaration on the copy is now refused, so the
// original's names and counts agree and Bind still demands every name it
// counts.
func TestCopyAfterDeclarationKeepsNamesAndCountsConsistent(t *testing.T) {
	a := *parameterized.NewTemplate(2)
	if err := a.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	b := a
	if err := b.AddParamGate("phi", parameterized.Rx, 1); err == nil {
		t.Fatal("AddParamGate on a by-value copy taken after a declaration succeeded, want an error: the copy shares the original's bookkeeping")
	}
	if err := a.AddParamGate("phi", parameterized.Rx, 1); err != nil {
		t.Fatal(err)
	}
	names := a.ParamNames()
	counts := a.ParamStepCounts()
	for name := range counts {
		found := false
		for _, n := range names {
			if n == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("a.ParamStepCounts() = %v lists %q but a.ParamNames() = %q lacks it", counts, name, names)
		}
	}
	var missing *parameterized.MissingParameterError
	if _, err := a.Bind(parameterized.Params{"theta": 0.1}); !errors.As(err, &missing) {
		t.Fatalf("a.Bind without phi: err = %v, want MissingParameterError", err)
	}
}

// TestCopiedTemplateIsRefusedByAddAndBind pins the shape of the refusal:
// AddParamGate, AddGate, and Bind on a by-value copy taken after a
// declaration each return the copy error before any other check, the
// accessors on the copy still describe the template as it was when
// copied, and the original is untouched and fully usable.
func TestCopiedTemplateIsRefusedByAddAndBind(t *testing.T) {
	orig := parameterized.NewTemplate(2)
	if err := orig.AddParamGate("theta", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	c := *orig
	const want = "copied by value"
	if err := c.AddParamGate("phi", parameterized.Rx, 1); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("AddParamGate on a copy: err = %v, want an error mentioning %q", err, want)
	}
	// The copy check precedes the argument checks: a nil factory on a copy
	// is reported as the copy, not as the factory.
	if err := c.AddParamGate("phi", nil, 1); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("AddParamGate(nil factory) on a copy: err = %v, want the copy error first", err)
	}
	if err := c.AddGate(gates.NewHadamard(), 1); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("AddGate on a copy: err = %v, want an error mentioning %q", err, want)
	}
	if _, err := c.Bind(parameterized.Params{"theta": 0.1}); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Bind on a copy: err = %v, want an error mentioning %q", err, want)
	}
	// The accessors read only what the copy holds by value.
	if got := c.NumQubits(); got != 2 {
		t.Fatalf("NumQubits() on a copy = %d, want 2", got)
	}
	if names := c.ParamNames(); len(names) != 1 || names[0] != "theta" {
		t.Fatalf("ParamNames() on a copy = %q, want [%q]", names, "theta")
	}
	if counts := c.ParamStepCounts(); len(counts) != 1 || counts["theta"] != 1 {
		t.Fatalf("ParamStepCounts() on a copy = %v, want map[theta:1]", counts)
	}
	// The original is untouched by the refused calls and still usable.
	if names := orig.ParamNames(); len(names) != 1 || names[0] != "theta" {
		t.Fatalf("ParamNames() on the original after refused calls on a copy = %q, want [%q]", names, "theta")
	}
	if err := orig.AddGate(gates.NewHadamard(), 1); err != nil {
		t.Fatalf("AddGate on the original: %v", err)
	}
	if _, err := orig.Bind(parameterized.Params{"theta": 0.1}); err != nil {
		t.Fatalf("Bind on the original: %v", err)
	}
}

// TestCopyBeforeDeclarationIsIndependent pins the other half of the
// contract: a Template copied before any declaration is accepted shares
// nothing with its source, so both go on as separate templates. A
// rejected declaration does not count; it leaves the template untouched,
// so a copy taken after one is independent too.
func TestCopyBeforeDeclarationIsIndependent(t *testing.T) {
	fresh := *parameterized.NewTemplate(2)
	twin := fresh
	if err := fresh.AddParamGate("alpha", parameterized.Ry, 0); err != nil {
		t.Fatal(err)
	}
	if err := twin.AddParamGate("beta", parameterized.Rx, 1); err != nil {
		t.Fatalf("AddParamGate on a copy taken before any declaration: %v, want success", err)
	}
	if names := fresh.ParamNames(); len(names) != 1 || names[0] != "alpha" {
		t.Fatalf("fresh.ParamNames() = %q, want [%q]", names, "alpha")
	}
	if names := twin.ParamNames(); len(names) != 1 || names[0] != "beta" {
		t.Fatalf("twin.ParamNames() = %q, want [%q]", names, "beta")
	}
	if _, err := twin.Bind(parameterized.Params{"beta": 0.2}); err != nil {
		t.Fatalf("twin.Bind: %v", err)
	}

	rejected := parameterized.NewTemplate(2)
	if err := rejected.AddParamGate("gamma", nil, 0); err == nil {
		t.Fatal("nil factory accepted, want an error")
	}
	after := *rejected
	if err := after.AddParamGate("gamma", parameterized.Ry, 0); err != nil {
		t.Fatalf("AddParamGate on a copy taken after only a rejected declaration: %v, want success", err)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run:

```bash
go test ./parameterized -run '^TestCop' -v 2>&1 | grep -E '^\s*(--- |parameterized_test)|^(ok|FAIL)'
go vet -tags redtests ./parameterized
```

Expected: `TestCopyAfterDeclarationKeepsNamesAndCountsConsistent` fails with `AddParamGate on a by-value copy taken after a declaration succeeded, want an error: the copy shares the original's bookkeeping`. `TestCopiedTemplateIsRefusedByAddAndBind` fails with `AddParamGate on a copy: err = <nil>, want an error mentioning "copied by value"`. `TestCopyBeforeDeclarationIsIndependent` passes (the code already honors that half of the contract). The `go vet` under the `redtests` tag compiles cleanly with the trimmed import block.

- [ ] **Step 4: Implement the guard**

Six edits in `parameterized/parameterized.go`, in file order.

(a) Replace the import block (lines 7-14), that is, exactly

```go
import (
	"fmt"
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)
```

with

```go
import (
	"errors"
	"fmt"
	"math"

	"github.com/pjbaur/quantum/circuit"
	"github.com/pjbaur/quantum/gates"
	"github.com/pjbaur/quantum/quantum"
)
```

(b) Replace the `Template` doc comment and struct (lines 46-61), that is, exactly

```go
// Template is a circuit recipe with named parameter holes. The zero value
// is ready to use and is the template NewTemplate(0) returns: it declares
// nothing, rejects a declaration with an out-of-range target or with none,
// and, once Bind's own parameter checks pass, Bind fails as circuit.New
// does for a qubit count of zero. No method panics on it; NewTemplate is
// how a template for a positive qubit count is made, not a precondition
// of the methods. A Template is safe for concurrent reads after all Add
// calls complete.
type Template struct {
	numQubits  int
	steps      []step
	paramOrder []string
	// seen is allocated by AddParamGate on the first declaration, the only
	// place it is written, so the zero value needs no constructor.
	seen map[string]bool
}
```

with

```go
// Template is a circuit recipe with named parameter holes. The zero value
// is ready to use and is the template NewTemplate(0) returns: it declares
// nothing, rejects a declaration with an out-of-range target or with none,
// and, once Bind's own parameter checks pass, Bind fails as circuit.New
// does for a qubit count of zero. No method panics on it; NewTemplate is
// how a template for a positive qubit count is made, not a precondition
// of the methods.
//
// A Template is used in place or through a pointer, as NewTemplate returns
// it. Copying a Template by value after a declaration has been accepted is
// not supported: the copy shares the original's name-tracking map and the
// backing arrays of its step and name lists but not their slice headers,
// so declarations on the two would drift apart or overwrite each other.
// Like strings.Builder, a Template records the receiver of its first
// accepted declaration, and AddParamGate, AddGate, and Bind on a copy taken
// after that return an error, before any other check, instead of touching
// the shared state; NumQubits, ParamNames, and ParamStepCounts on such a
// copy describe the template as it was when copied. A copy taken before any
// declaration is accepted shares nothing and is an independent template.
// A Template is safe for concurrent reads after all Add calls complete.
type Template struct {
	// addr is the receiver of the first accepted declaration, set only by
	// AddParamGate and AddGate, so a by-value copy taken after that can be
	// told from the original, as strings.Builder does. Nil until then: a
	// copy of a template with no accepted declaration shares nothing.
	addr       *Template
	numQubits  int
	steps      []step
	paramOrder []string
	// seen is allocated by AddParamGate on the first declaration, the only
	// place it is written, so the zero value needs no constructor.
	seen map[string]bool
}

// errCopiedTemplate is what AddParamGate, AddGate, and Bind return on a
// Template copied by value after an accepted declaration; see Template.
var errCopiedTemplate = errors.New("Template copied by value after a declaration; a declared Template is used in place or through a pointer, never by copy")

// checkNotCopied returns errCopiedTemplate when t is a by-value copy of a
// Template that had already accepted a declaration when it was copied.
// Such a copy shares seen and the slices' backing arrays with the
// original, so a write through either, or Bind's read of seen on the
// copy, could desync the two; the three methods that touch seen or append
// call this first.
func (t *Template) checkNotCopied() error {
	if t.addr != nil && t.addr != t {
		return errCopiedTemplate
	}
	return nil
}
```

`NewTemplate`, which follows, is not edited.

(c) Replace the tail of `AddParamGate`'s doc comment and its opening statements, that is, exactly

```go
// it is what Bind and ParamStepCounts key on. At least one target is
// required: a declaration with none is rejected here, with an error naming
// the parameter, rather than declared, counted, and left for
// circuit.AddGate to reject at Bind.
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if factory == nil {
```

with

```go
// it is what Bind and ParamStepCounts key on. At least one target is
// required: a declaration with none is rejected here, with an error naming
// the parameter, rather than declared, counted, and left for
// circuit.AddGate to reject at Bind. On a Template copied by value after
// an accepted declaration the call is refused before any of these checks
// (see Template).
func (t *Template) AddParamGate(name string, factory Factory, targets ...int) error {
	if err := t.checkNotCopied(); err != nil {
		return err
	}
	if factory == nil {
```

(d) Inside `AddParamGate`, replace the statements between the `checkTargets` call and the `seen` allocation, that is, exactly

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	if t.seen == nil {
		t.seen = make(map[string]bool)
	}
```

with

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.addr = t
	if t.seen == nil {
		t.seen = make(map[string]bool)
	}
```

This block occurs once in the file (only `AddParamGate` allocates `seen`).

(e) Replace the tail of `AddGate`'s doc comment and its opening statements, that is, exactly

```go
// panics when this method calls gate.Name() to name the gate in the
// no-target error.
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if gate == nil {
```

with

```go
// panics when this method calls gate.Name() to name the gate in the
// no-target error. On a Template copied by value after an accepted
// declaration the call is refused before any of these checks (see
// Template).
func (t *Template) AddGate(gate quantum.Gate, targets ...int) error {
	if err := t.checkNotCopied(); err != nil {
		return err
	}
	if gate == nil {
```

Then, inside `AddGate`, replace the statements between the `checkTargets` call and the `steps` append, that is, exactly

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.steps = append(t.steps, step{gate: gate, targets: targets})
```

with

```go
	if err := t.checkTargets(targets); err != nil {
		return err
	}
	t.addr = t
	t.steps = append(t.steps, step{gate: gate, targets: targets})
```

This block occurs once in the file (the `step{gate: gate, ...}` literal is `AddGate`'s alone).

(f) Replace `Bind`'s doc comment and its opening statement, that is, exactly

```go
// Bind materializes the template into a circuit using values. Every declared
// parameter must be present and finite; unknown names are rejected so typos
// fail loudly instead of silently ignoring an angle.
func (t *Template) Bind(values Params) (*circuit.Circuit, error) {
	for _, name := range t.paramOrder {
```

with

```go
// Bind materializes the template into a circuit using values. Every declared
// parameter must be present and finite; unknown names are rejected so typos
// fail loudly instead of silently ignoring an angle. On a Template copied
// by value after an accepted declaration the call is refused before any of
// these checks (see Template); the refusal is a read, so Bind stays safe
// to call concurrently once all Add calls complete.
func (t *Template) Bind(values Params) (*circuit.Circuit, error) {
	if err := t.checkNotCopied(); err != nil {
		return nil, err
	}
	for _, name := range t.paramOrder {
```

`Bind` never writes `addr`. Everything after the inserted lines in each of the three methods, `checkTargets`, `NewTemplate`, `NumQubits`, `ParamNames`, and `ParamStepCounts` are not edited.

- [ ] **Step 5: Run the tests to verify they pass**

Run:

```bash
go test ./parameterized -v 2>&1 | grep -E '^(--- |ok|FAIL)'
```

Expected: every test in the package reports `--- PASS`, seventeen in all, including `TestCopyAfterDeclarationKeepsNamesAndCountsConsistent`, `TestCopiedTemplateIsRefusedByAddAndBind`, `TestCopyBeforeDeclarationIsIndependent`, both `TestZeroValueTemplate` tests (every call on the same variable, so `addr` is either nil or the receiver), and `TestParamStepCounts` (whose table holds pointers), then `ok`.

- [ ] **Step 6: Record the change in the CHANGELOG**

In `CHANGELOG.md`, under `## Unreleased`, `### Fixed`, insert a new entry directly before the first `### Breaking` heading (line 168 at `d52e18f`). The three lines immediately above the insertion point are the end of item 20's entry:

```markdown
chosen contract, `algorithm/backlog_red_numeric_test.go` is removed. Design:
`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`.

### Breaking
```

Replace that with:

```markdown
chosen contract, `algorithm/backlog_red_numeric_test.go` is removed. Design:
`docs/superpowers/specs/2026-09-09-parameter-shift-angle-bound-design.md`.

#### `parameterized.Template` refuses a by-value copy taken after a declaration

A `Template` copied by value after a declaration shared the original's
name-tracking map but not its slice headers, so a name declared on the
copy was already "known" to the original: the original's next declaration
of that name was counted by `ParamStepCounts` but never listed by
`ParamNames`, and `Bind` accepted values that omitted it, binding the gate
at angle 0. Go cannot make such copies independent (the two also share the
backing arrays of their step lists, so a declaration on one can overwrite
one on the other), so a `Template` is now documented as used in place or
through a pointer, as `NewTemplate` returns it, and, like
`strings.Builder`, records the receiver of its first accepted declaration:
`AddParamGate`, `AddGate`, and `Bind` on a copy taken after that return an
error, before any other check, instead of touching the shared state.
`NumQubits`, `ParamNames`, and `ParamStepCounts` on such a copy still
describe the template as it was when copied; a copy taken before any
declaration is accepted is an independent template; no method panics.
Nothing in the module copies a `Template` by value, so no caller changes.
Design: `docs/superpowers/specs/2026-09-10-template-copy-guard-design.md`.

### Breaking
```

- [ ] **Step 7: Full verification**

Run:

```bash
gofmt -l .
go vet ./...
go vet -tags redtests ./parameterized ./algorithm
staticcheck ./...
go build ./...
go test ./...
go test -race ./parameterized ./algorithm
go test -tags redtests ./parameterized ./algorithm -run '^TestRed' 2>&1 | grep -E '^(--- FAIL|ok|FAIL)'
go run ./cmd/quantum -demo qaoa | grep -A1 'VQE from'
git status --short
```

Expected: `gofmt -l` prints nothing; vet (both), staticcheck, build, the full suite, and the race run are clean; the red run lists exactly one failure, `--- FAIL: TestRedDeclaredTargetsAreNotAliasedToCallerSlice` (item 22, untouched by design), an `ok ... [no tests to run]` line for `algorithm`, and no `TestRedCopyAfterDeclaration` line; the demo prints `VQE from the symmetric start: energy = -1.0000, expected cut = 2.0000` and then `  iterations accepted: 29, energy evaluations: 378`; `git status` lists exactly the four files named under **Files** and nothing else. If item 22's test passes, a second red test fails, or the demo lines differ, stop and report: something outside this item changed.

- [ ] **Step 8: Commit**

```bash
git add parameterized/parameterized.go parameterized/parameterized_test.go parameterized/backlog_red_test.go CHANGELOG.md
git commit -m "fix(parameterized): refuse a Template copied by value after a declaration

A value copy of a Template shares the seen map and the backing arrays of
steps and paramOrder with the original but owns its own slice headers,
so a name declared on the copy was already \"seen\" by the original: its
next declaration of that name was counted by ParamStepCounts, missing
from ParamNames, and never demanded by Bind, which bound the gate at
angle 0. Go cannot make such copies independent, so Template now records
the receiver of its first accepted declaration in an unexported addr
field, as strings.Builder does, and AddParamGate, AddGate, and Bind, the
three methods that read seen or append, return errCopiedTemplate on a
copy before any other check. The writers pin after checkTargets so a
rejected declaration leaves the template untouched; Bind only reads, so
it stays safe to call concurrently; the accessors are unchanged since
they read only what a copy holds by value. The Template doc comment
states the contract.

TestRedCopyAfterDeclarationKeepsNamesAndCountsConsistent moves into the
regular suite as TestCopyAfterDeclarationKeepsNamesAndCountsConsistent
with its declaration-on-the-copy expectation inverted per the spec; the
red file keeps item 22's test and drops the errors import only item 21's
used. Backlog item 21.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01XMPFmYskZD8uji6xSh5x11"
```

If the commit fails with `.git/index.lock` present, another agent is committing; wait a few seconds and run the `git commit` again unchanged.

---

## Self-Review

- **Spec coverage.** Decision 1 (copy-unsafe, `strings.Builder`-style runtime guard, contract on the type): Step 4b's doc comment, `addr` field, and helper. Decision 2 (error, not panic; one unexported untyped value with the exact text): Step 4b's `errCopiedTemplate` and the Global Constraints line; pinned by the `strings.Contains` checks in `TestCopiedTemplateIsRefusedByAddAndBind` (Step 2). Decision 3 (checked in `AddParamGate`, `AddGate`, `Bind` as the first statement; pinned by the two writers after `checkTargets`; `Bind` and `NewTemplate` never pin; accessors untouched; no `noescape`): Steps 4c through 4f place the check and the pin exactly there, `NewTemplate` and the accessors are not edited, and the Global Constraints forbid `unsafe`; pinned by the nil-factory-on-a-copy ordering, the accessor checks, and the rejected-declaration copy in Step 2's tests; the race run in Step 7 covers `Bind`'s read-only check. Rulings: undeclared copy independent (third test), `AddGate` pins too (the `t.addr = t` in Step 4e), refused calls change nothing and the original is untouched (second test), error precedence (second test's nil-factory case), zero value (Step 5 notes both `TestZeroValueTemplate` tests still pass). Effect on callers: Step 7 runs the full suite, the race run, and the QAOA demo. Backward compatibility: CHANGELOG entry under `### Fixed` (Step 6); no ADR or spec amendment (none planned, by design). Red test rulings: the moved test keeps its setup, the original's declarations, the loop, and the `Bind` assertion byte for byte and inverts the one ruled expectation (Step 2); the `errors` import leaves the red file with it. Testing: all three tests in Step 2 with the two failure messages in Step 3; the red-tag expectation is in Step 7.
- **Placeholder scan.** No TBD/TODO; every code and doc step carries its full text; every command names its expected result.
- **Type consistency.** `addr *Template`, `errCopiedTemplate error`, and `checkNotCopied() error` are named identically in Step 4b, Steps 4c through 4f, the Interfaces block, and the commit message. `AddParamGate(name string, factory Factory, targets ...int) error`, `AddGate(gate quantum.Gate, targets ...int) error`, and `Bind(values Params) (*circuit.Circuit, error)` match `parameterized/parameterized.go`. `errors.As` on `*parameterized.MissingParameterError` in the moved test matches the pointer-receiver `Error()` in the package. `gates.NewHadamard()`, `parameterized.Ry`, `parameterized.Rx`, `strings.Contains`, and `errors.As` are all available through the test file's existing imports. The test names cited in Steps 3, 5, and 7 match the functions written in Step 2.
- **Prototype.** Every code and test block above was applied to a scratch copy of the repository at `d52e18f` in this exact sequence: Step 2 red (both failure messages as stated in Step 3, the third test passing, `go vet -tags redtests` clean), Step 4 green (seventeen `--- PASS` lines as stated in Step 5), Step 6 anchor matched exactly once, then `gofmt -l`, `go vet` (plain and `-tags redtests` on both packages), `staticcheck`, `go build`, `go test ./...`, `go test -race ./parameterized ./algorithm`, the red-tag run (exactly item 22 failing, `algorithm` with no tests to run), and the QAOA demo lines, all as stated in Step 7. The diff was +58/-5 on `parameterized/parameterized.go`, +20 on `CHANGELOG.md`, +122 on `parameterized/parameterized_test.go`, and -40 on `parameterized/backlog_red_test.go`. `go build -gcflags=-m` confirmed the receiver of `AddParamGate` and `AddGate` now escapes (`leaking param: t`), as spec Decision 3 records.
