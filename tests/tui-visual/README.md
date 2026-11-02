# Real native terminal captures

This development harness runs the production `tui.New` model and the actual
Bubble Tea renderer in a disposable POSIX PTY. Its only backend is the existing
same-package `fakeBackend`, seeded with disposable data. It never runs the
production CLI, reads operator configuration, calls a gateway, or touches a real
service. HTML contains an xterm terminal, with no reconstruction of the app UI.
Generated output is evidence of rendering, **not visual acceptance**.

Run from the repository/worktree root (Go 1.26.1, Python 3.10+ on POSIX,
Node 18+, npm and Playwright's Linux libraries are required):

```sh
python3 tests/tui-visual/capture.py bootstrap
python3 -m unittest discover -s tests/tui-visual -p 'test_*.py'
python3 tests/tui-visual/run_fault_tests.py
python3 tests/tui-visual/capture.py capture --scenario dashboard --output artifacts/native-dashboard
python3 tests/tui-visual/capture.py batch --batch screens --sizes all --themes all --output artifacts/native-screens
python3 tests/tui-visual/capture.py batch --batch all --sizes all --themes all --plan --output artifacts/native-required-plan
python3 tests/tui-visual/verify.py artifacts/native-dashboard/manifest.json
```

Bootstrap installs the committed lockfile's exact xterm.js 5.5.0, Playwright
1.56.1 and JetBrains Mono 5.2.8 into `.cache/tui-visual/tools`, and downloads
Playwright's pinned Chromium into `.cache/tui-visual/browsers`. npm downloads,
Go scratch, Chromium profiles and temporary files stay under `.cache/tui-visual`.
No production dependency or `go.mod` change is needed. Bootstrap is the only
network-dependent step; later captures use bundled local assets and block all
browser network requests. Do not install packages globally.

Every capture rebuilds the guarded test binary with `GOMAXPROCS=2` and `-p 2`.
The build tag `visualcapture` and `ONELINE_VISUAL_CAPTURE=1` guard the helper.
Regular Go tests do not activate it. To run the surrounding TUI suite:

```sh
mkdir -p .cache/tui-visual/tmp
GOMAXPROCS=2 GOTMPDIR="$PWD/.cache/tui-visual/tmp" TMPDIR="$PWD/.cache/tui-visual/tmp" \
  go test -p 2 -count=1 -tags=visualcapture ./internal/tui
```

The actual guarded helper is exercised by `capture`/`batch` through the PTY.
It intentionally skips when invoked by an ordinary test run without the guard.
`run_fault_tests.py` creates eleven focused real captures in disposable local
cache: parity, Unicode, Help first/last, three validation refusals, two empty
searches and Review last/acknowledged. It runs Python process/verifier faults
and Node renderer/export/semantic faults against those positive controls.
Bootstrap must already be installed. Evidence faults otherwise skip without
`A2_TEST_MANIFEST`; the original process tests still run independently. This
test setup needs no retained PNG fixtures from another checkout and no matrix.

## Scenarios and explicit coverage

`python3 tests/tui-visual/capture.py list` prints the supported inventory and
uncovered states. The screen names are `dashboard`, `models`, `review`, `jobs`,
`gateway`, `service`, `repair`, `ssh`, `utilities`, `settings`. Modal scenarios
are `help`, `palette`, `discard`, `quit`, `ssh-confirm`, `monitor-install`,
`monitor-uninstall`. Editor scenarios are `edit-filter`, `edit-model`,
`edit-budget`, `edit-compaction`, `edit-endpoint`, `edit-credential`, `edit-key`.

`--batch screens` covers ten screens; `--batch all` adds the seven real modal
and seven real editor variants. `--sizes all --themes all` requests 150 screen
captures or 360 total baseline captures: 120x40, 100x30, 80x24, 60x20, 40x12,
each in dark/light/mono. `--plan` emits pending rows without building or
claiming a capture. Use a new output directory for every run; existing manifests
are protected against accidental overwrite. All output must stay within this
worktree's ignored `artifacts/` or `.cache/` directories.

A3 adds **More**, the eighth modal, through the semantic `nav:More` ID. The
rejected baseline does not have this menu. `--scenario more` supports `baseline`
and `last-item` setup (real Down keys through the bounded menu) for integration.
Both fail closed on the old baseline. The `last-item` outcome guard still requires
the integrated More model/catalog contract; it cannot be certified on this
branch. `--batch integrated` requests all 25
surfaces (375 baseline rows), while `--batch all` preserves the original 24.
The manifest lists this extension and its pending coverage explicitly. Plan it
on either branch without pretending the old app has it:

```sh
python3 tests/tui-visual/capture.py capture --scenario more --sizes all --themes all --plan --output artifacts/more-first-plan
python3 tests/tui-visual/capture.py capture --scenario more --variant last-item --sizes all --themes all --plan --output artifacts/more-last-plan
```

Single captures also support explicit `--state baseline|empty|missing|unsupported|drift|unicode|parity`,
`--color truecolor|256|16|none` (`none` also injects `NO_COLOR`), and `--ascii`.
States seed the fake snapshot; they do not assert that every combination is
meaningful. For example, an exact Review requiring an editable client fails
explicitly with the `empty` or `unsupported` state. It is never silently replaced
by a fabricated screen. Native ASCII currently sets its option without changing
all production decorations; this harness does not manufacture an ASCII layout.

`--state parity` uses the approved browser's six-client composition: Codex,
Claude Code, OpenCode, Pi, Grok Build and Hermes. Only Codex is marked supported;
the other adapters retain truthful unsupported/read-only flags, and Pi is not
installed. The selected Codex uses gpt-6.1-sol / Standard and a 500000-token
budget. New fixtures show `Session only` as the public credential status;
injected dummy provenance and the complete public client flags live in the
manifest. The previously captured three-client baseline is preserved.

```sh
python3 tests/tui-visual/capture.py capture --scenario dashboard --state parity --output artifacts/native-parity-dashboard
```

Additional `--variant` values are deliberately bounded:

| Scenario | Variants beyond `baseline` |
| --- | --- |
| Dashboard, palette | `no-results` |
| Help | `last-page` |
| Review | `last-page`, `acknowledged` |
| Budget, compaction, endpoint editors | `invalid` |

```sh
python3 tests/tui-visual/capture.py capture --scenario review --variant last-page --sizes 40x12 --themes mono --output artifacts/native-review-last
python3 tests/tui-visual/capture.py capture --scenario palette --variant no-results --output artifacts/native-palette-empty
python3 tests/tui-visual/capture.py capture --scenario dashboard --state unicode --color 256 --output artifacts/native-unicode
python3 tests/tui-visual/capture.py capture --scenario dashboard --color none --ascii --output artifacts/native-no-color
```

First-run loading, actual asynchronous work/cancel/failure, all service outcomes,
discovery permutations, handoff/return, real apply receipts and the complete
save/cancel/caret/focus-return matrix remain open, explicitly listed in each
manifest. The baseline Repair screen has no prepared brief; baseline Jobs has
one fake success receipt. These are states of existing screens, not new screens.

## Capture, readiness and provenance

Fixture setup calls production commands/controls by semantic ID and drains the
real model's fake-backend commands before starting the renderer. This gives a
stable snapshot, not a recording of the whole interactive journey. The PTY
helper first probes both Tab/Shift+Tab orders on private copies of the real model
and chooses an order that restores the production View and draft. Clamped modal
boundaries can require Shift+Tab followed by Tab. The initial readiness write
announces this bounded key plan; the driver sends both real keys through Bubble
Tea's input decoder. The helper checks their exact delivery, reasserts the
requested semantic outcome and requires the final production View to equal the
initial rendered View. It sends F12 to a fixture-only wrapper for normal clean
exit. No production key binding is added.

The wrapper delegates `Update`/`View` to the production model and sets an
invisible window title for each observed input. An output writer adds an ignored
OSC readiness marker **after** the corresponding renderer write completes.
The driver waits on those markers with a deadline, not an arbitrary sleep. The
active ANSI stream ends at the final marker, before alt-screen cleanup. Exit
cleanup is stored separately and checked for alternate-screen leave, cursor
restore, process success and exact termios restoration. The full transcript
retains both streams. On a deadline the disposable child process group is killed.

xterm parses the ANSI bytes, including cursor motion, Unicode and alternate
screen state, then paints the PNG after local font loading and its render event.
The complete styled cell buffer is also written as `.frame.json`. Latin regular,
bold and italic font files, terminal JS/CSS and their hashes are recorded.
Non-Latin glyphs still depend on Chromium's host fallback fonts. A Latin bundled
font does not certify CJK glyph fidelity on other hosts.

`manifest.json` records the source SHA/branch/diff hash, fixture helper and binary
hashes, exact command, size/theme/state/variant, source profile without endpoint,
input keys, setup actions, actual font/cell/pixel metrics and theme colors.
Each row links SHA-256 hashes for ANSI, cleanup, full transcript, cell frame and
PNG. `index.html` is an ignored, local contact sheet linking the actual PNGs;
open individual full-size PNGs for review. All rows start `not-inspected`,
and no generation step sets visual PASS or asserts a matched mockup.

New captures also export the exact production `View.Content` and assert the
requested screen/modal/editor, fixture state, dimensions and theme before and
after PTY input. Shared export/verifier gates compare that View's parsed text
(normalizing trailing padding), replay captured ANSI to every styled cell, and
check redundant text against cells. Validation schema 3 also exports the actual
before/after semantic outcomes: Help/Review page numbers, acknowledgment, search
queries/counts and editor values/refusals. Invalid edits record the real Enter
attempt, retained editor and before/after full-draft hashes without exposing the
dummy endpoint. A separate copy calls the production validator to check refusal;
no layout or validation rules are reimplemented. The shared verifier independently
checks client-state predicates, visible page markers, refusal text, invalid input,
unchanged draft hashes and empty searches. Unicode requires the actual fixture
name, including its CJK/accented text.

PNG decoding requires opaque native pixels, rejects transparency chunks and
checks contrasting pixels in at least 90% of text-bearing cells, rejecting valid
transparent, solid and missing-body exports. This is paint sanity, not OCR or
visual acceptance. The PNG and frame
are published only after these gates pass. Legacy captures without a saved
production View remain verifiable for ANSI/cell/text, scene metadata, geometry,
privacy and paint consistency; they are explicitly reported as legacy rather
than given new View proof retroactively. Schema 2 captures retain their View
proof but are explicitly reported as legacy for distinguishing semantic outcomes.
Legacy schema 1/2 rows without recorded clients explicitly report client
observations unavailable; recorded optional clients still undergo state checks.
Schema 3 requires both client and before/after semantic observations, including
for an empty-client request. Historical files, source records and hashes are not
rewritten or upgraded.

Focused compatibility tests can use an unchanged original probe without building
or capturing anything (set `A2_TEST_MANIFEST` to an existing schema-3 positive):

```sh
A2_TEST_MANIFEST=artifacts/tui-native-fix2/unicode/manifest.json A2_LEGACY_MANIFEST=artifacts/tui-native-probe/manifest.json node --test tests/tui-visual/legacy.test.mjs
```

The portable fault runner includes the schema-boundary tests against its fresh
fixture. The actual historical-row control runs when `A2_LEGACY_MANIFEST` is set;
it never synthesizes or hydrates the old manifest.

Partial render resume freezes Chromium/font/xterm and capture-code fingerprints.
A changed or incomplete environment record is refused before any manifest write;
failed new work cannot relabel already-rendered rows. Fully completed historical
manifests are left untouched. Use a new output directory for a changed renderer.

Source provenance separates checkout HEAD from production, fixture and tool
revision/input hashes and dirtiness. Relevant tracked and untracked Go inputs,
testdata and capture-tool inputs are hashed; even ignored Go source is included.
Generated cache, artifact, worktree and workspace-report roots are excluded.
The original probe has an additive provenance annotation identifying its
untracked helper; its original source record and artifact hashes are preserved.

Images, generated manifests and operational `.superpowers/` reports are ignored
local evidence and do not travel with these commits. Regenerate them on another
checkout. Only the harness code, usage and guarded helper belong in the tracked
change set; do not force-add workspace reports.

Dummy key input is injected into the hidden editor and stays masked. The dummy
endpoint uses the reserved `.invalid` host. Raw ANSI/transcripts, fixture
metadata and visible terminal cells are checked for that endpoint, the injected
secret and URL strings before publishing their artifacts. PNGs contain only
those checked cells; binary PNG-string searching would not validate text
secrecy. This is a fixed-fixture safety gate, not general-purpose redaction for
arbitrary host data. **Never replace the fake backend with a live backend.**

The endpoint editor's baseline intentionally clears its input before the first
render because the production editor can expose its local value; the recorded
setup action makes this explicit. Endpoint privacy and a comparable filled
editor state are renderer work for A3. No other production TUI file is owned by
this harness.

## Browser mockups are separate evidence

The coordinator's pinned approved references are local artifacts in its checkout
under `artifacts/tui-reference-baseline/`; a separate worktree may not have them.
A4 owns the six
missing screen references. This harness does not modify site files or call the
browser specimen native output. A reference-only capture aid for the original
four browser screens is available against a local server:

```sh
# In another terminal, from this worktree; choose a free local port.
python3 -m http.server 38627 --bind 127.0.0.1 --directory site
node tests/tui-visual/render.mjs preview http://127.0.0.1:38627/tui-preview.html artifacts/browser-reference dark dashboard
```

It waits for `document.fonts.ready` and records `kind: browser-mockup`, URL,
capture-tool revision/dirty state and image hash in a separate `reference.json`.
Fetched HTML/CSS/JS/font resources are hashed from their actual response bytes.
The served reference revision, dirty state and untracked files are marked unknown:
a loopback server can serve another checkout, so tool HEAD does not identify it.
These response hashes identify the fetched inputs, without claiming a verified
reference git revision. The older smoke record is annotated with this limitation.
It captures the
browser's existing illustrative fixture, which is **not identical** to the
native fake snapshot. A visual reviewer must match fixture/state/size and pin
both revisions before making acceptance claims. Preview capture does not close
native/reference gaps or certify the six new A4 screens.

## Integration seams and portability

A3 may change production rendering/navigation without editing
`internal/tui/visual_capture_test.go`; A2 owns that helper. Setup couples to
`New`, same-package `fakeBackend`, model/command/control IDs, `press`/`run`,
Review pagination, production Help page markers and editor state. Help text
clipping/wrapping is a separate production UI concern. Recheck the key plan and
before/after page numbers when integrating A3, and rerun More first/last coverage
only after its semantic outcome guard is implemented against the reviewed model.
Missing semantic IDs fail closed. It uses
IDs rather than positional focus indices or copied layout text. The renderer
barrier depends on Bubble Tea v2.1.1 emitting the complete window-title/frame in
one output Write. The metric recorder reads xterm 5.5.0's private render-service
dimensions in addition to DOM measurements. Revalidate both on dependency
upgrades.

The PTY driver is POSIX-only. Linux captures do not certify native Windows or
macOS appearance, key delivery or process cleanup. Playwright may download an
Ubuntu fallback on this Arch host; no package manager or OS-library installer is
invoked automatically. Exact PNG reproducibility requires matching pinned
Chromium, fonts, metrics and host glyph fallback. The manifest records enough
data to distinguish a changed app from a changed capture environment.
