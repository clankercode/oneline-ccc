# Native TUI capture infrastructure

Recorded: 2026-11-03. Reviewed task source: `4b096c1`, including the opacity and
semantic fixes at `86c3486`. This infrastructure supports the
[P0 comparison campaign](../plans/remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation);
it does not establish visual acceptance.

The [harness](../../tests/tui-visual/README.md) builds a guarded fixture around
production `tui.New`, drives actual Bubble Tea rendering through a disposable
Linux/POSIX PTY, and replays captured ANSI through pinned xterm/Chromium/font
assets. Its backend uses fixture data. It never starts the production CLI or
accesses operator configuration, keys, gateways or services. Browser HTML hosts
the terminal renderer; it does not recreate the application's interface.

```sh
python3 tests/tui-visual/capture.py bootstrap
python3 tests/tui-visual/capture.py capture --scenario dashboard --state parity --output artifacts/dashboard
python3 tests/tui-visual/verify.py artifacts/dashboard/manifest.json
```

Bootstrap downloads pinned development tools; subsequent captures use local
assets. Build/browser/temp state stays in checkout-local `.cache/tui-visual`.
The harness records revision and input hashes, dirty/untracked inputs, fixture
state, terminal cells, font/renderer identities, actual delivered key order and
before/after semantic outcomes. PNG, ANSI, text and cell artifacts must agree.
Resuming cannot silently replace the original renderer provenance. Reference
asset hashes remain separate from the capture-tool revision.

## Controller integration gates

On the integrated source after terminal correction `418283a`, the controller ran:

```sh
python3 tests/tui-visual/run_fault_tests.py
go test -race -p 2 -count=1 -tags=visualcapture ./internal/tui
```

The fault runner passed **15 Python tests and 57 Node tests**; one optional test
for a retained historical artifact skipped because a fresh checkout has none.
An explicit follow-up against the unchanged original probe passed all **10
legacy/current-schema checks**, with no skips. The tagged TUI race suite passed.
The guarded live helper also ran through eleven actual PTY captures in the
fault runner; an ordinary tagged test invocation alone does not activate it.

Negative controls include genuine glyph-bearing PNGs reencoded with transparent
or partial alpha, blank/header-only/wrong-target captures, matching-hash artifact
corruption, incorrect Unicode/page/validation outcomes, draft drift, both real
key orders, failed-leader descendant cleanup and changed resume provenance.
Schema 1/2 artifacts retain weaker proof when observations were never recorded;
schema 3 requires current observations. Historical bytes were preserved.

The [semantic re-review](../review/native-capture-semantic-review.md) and
[legacy re-review](../review/native-capture-legacy-rereview.md) retain independent
findings, dispositions and precise execution limits.

## Open evidence requirements

- Validate client-observation types; the inherited empty-state predicate can
  accept an object where only an array or explicit `null` is valid.
- Add the integrated More-menu first/last semantic guard and confirm real Help
  boundary input against the reviewed production navigation.
- Capture focused-but-unselected pending drafts and matching active filters so
  compact state-fidelity fixes receive actual native visual review.
- Recapture the exact final integrated TUI after its source fixes and review;
  provisional development images remain provisional.
- Complete every screen/editor/layer and state in the backlog matrix, alternating
  readable native/mockup PNG inspection and independent source/geometry review
  until both pass the same revision. Native Windows/macOS appearance and actual
  terminal behavior require separate evidence.

Integrity checks, semantic checks and passing fixtures cannot approve the design
or substitute for final installed-build comparisons.
