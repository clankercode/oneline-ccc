# Native remediation checkpoint

Recorded 2026-11-03. Integrated terminal source: `418283a`.
Latest verified composition checkpoint: `889ace7`.

**Native visual and interaction acceptance remains open.** This checkpoint
records the reviewed navigation and terminal corrections. The rejected
appearance has not been certified against the mockups; the shared presentation
work and full screen-by-screen visual campaign are still in progress.

## Integrated corrections

| Work | Integrated commits | Bounded evidence |
| --- | --- | --- |
| Directional navigation and focus regions | `6e48755`, `da17aa0` | Local arrows, horizontal tabs, F6/Shift+F6, direct section switching, visible More; visit-specific Back frames, stable discovered-model identity, Help page boundaries and contextual credential hints; [independent re-review](../review/native-navigation-rereview.md) |
| CLI terminal fixtures and navigation instructions | `ce63417`, `24b71d2` | Fixtures now follow the actual region contract while retaining mutation lifetime and dirty-exit guards; [integration and documentation review](../review/native-navigation-integration-review.md) |
| Interactive no-color mode | `cbbaf77`, `d9a7c5b`, `c1440cf` | Real Linux PTYs exercise entry, help, palette, staged editor focus/masking, drafts and safe exit; actual redirected stdin/stdout remain plain; [independent review](../review/native-no-color-rereview.md) |
| Terminal input lifetime | `b8dab03`, `6c50a7a` | Cancellation-aware event dispatch and joined decoder/raw reader before Close on Quit, Kill and terminal release; [independent re-review and provenance](../review/native-terminal-input-rereview.md) |
| Startup and error ownership | `28576af`, `418283a` | Deferred cleanup for partial setup, returned errors and panics; serialized resource ownership; observed cancellation checked before Init outside the ownership lock; [review](../review/native-startup-review.md) and [resolved cancellation re-review](../review/native-startup-rereview.md). Failed-reader constructor descriptors and native platform proof remain open |
| Secondary screen, layer and editor references | Reviewed source `46b50cd`, verifier `f285ff8`; source/docs integration follows `1d75e8b` | Six secondary design targets, eight layers and seven editors; 14 integrated browser journeys pass. [Reference verification, independent reviews and pending CI wiring](tui-native-references.md); these are design targets, not native acceptance |
| Native capture integrity and semantics | `cb936e6` | Reviewed opacity, state/outcome, provenance, cleanup and legacy-proof checks; controller fault, real-legacy and tagged TUI race gates pass. [Evidence and remaining integration/type/coverage limits](tui-native-captures.md); no visual PASS |
| Shared shell and client composition | `cc76758` through `1ee8fc5` | Multiline clients, list/detail layout, action/default-tier hierarchy and compact draft/filter visibility; [initial source review](../review/native-composition-review.md), [resolved compact-state rereview](../review/native-composition-rereview.md) and [interim actual-image review](../review/native-tui-composition-vision.md). Other bodies and full visual acceptance remain open |
| Help wrapped-row pagination | `40ba57b` | [Independent readability review](../review/native-help-readability-review.md) approves measured viewport and boundary/resize coverage. Useful grouped shortcut design remains open |
| Concise Models fields and Advanced disclosure | `7ec3738` through `c37ca30` | [Independent source/geometry review](../review/native-models-composition-review.md) approves actual keyboard paths, explicit choices and retained review/attestation guards. Sampled native full images and crops show improved fields/summary and compact Review access; chooser/Advanced/discovery and full visual acceptance remain open |
| Grouped, contextual shortcut Help | `df9404d`, `889ace7` | [Independent source/geometry review](../review/native-help-design-review.md) approves useful first-page bindings, contextual explanations and measured pagination. Sampled native first/last pages and crops were compared with reference bodies; all-origin and final integrated acceptance remains open |
| Prepared-frame capture association | `257468a` | Original independent reviewer approves the old-UI R1 correction. Composed multiline production-row integration and new Models variants are a separate candidate under review; no final composed-fixture compatibility or visual PASS is inferred |

The terminal correction uses licensed, pinned module mirrors because the checked
upstream revisions did not contain the required dispatch/join fix. The isolated
[patch](../../third_party/terminal-input.patch) and
[verification/update procedure](../../third_party/README.md) preserve an auditable
comparison with the original public archives. Owners of custom non-cancelable
readers must unblock their reads before shutdown can finish; the foreground path
uses native cancelable terminal readers.

## Executed controller gates

All commands use checkout-local temporary paths, `GOMAXPROCS=2` and `-p 2`.
The integrated `6c50a7a` tree passed:

```sh
go test -p 2 -count=1 ./...
go test -race -p 2 -count=1 ./...
go vet -p 2 ./...
python3 third_party/verify.py /path/to/go/pkg/mod/cache/download
```

The normal and race suites include the enabled, actually rendered Linux
raw-child/SIGTERM regression, queued-event cancellation/Quit/release cases, and
the new real no-color PTY/redirect tests. The offline provenance check compared
all 188 original archive files plus exactly the committed correction. The
controller also observed the focused queued-input/rendered race gate passing
on the reviewed task branch before integration.

After the startup/error correction, the controller ran those same four commands
again on integrated `418283a`: full Go tests, full race tests, vet and offline
archive-plus-patch verification all passed. The real Linux startup PTY cases now
cover errors after raw-mode acquisition, concurrent cancellation, partial setup
and model panics, reader errors, pre-canceled entry and normal quit. Dependency
tests separately pin renderer-start cancellation, reentrant Init/Kill and input
joining. This is Linux/runtime and shared-source evidence; it does not add native
Windows/macOS failure injection or a new live/pinned Codex session.

The worker's retained RED/GREEN evidence reproduced queued input surviving the
old 500 ms fallback, then passed 90 queued-input scenarios and 30 rendered
raw-child SIGTERM repetitions under the race detector. Dependency tests include
deterministic completion barriers and blocked event delivery. The no-color test
fix includes a controlled renderer leak fault: the former burst test passed it,
and the staged test correctly failed before Escape. Exact typed-value acceptance
is not established by a constant hidden label and is not claimed.

The pinned native Codex 0.166.0 adapter, app and engine journey also passed after
navigation integration at `24b71d2`, using explicit executable/catalog fixture
paths and disposable roots. That run predates the terminal dependency integration
and is not an updated native client handoff test. No inference was requested.
Six static builds passed on the reviewed terminal-fix branch; integrated hosted
CI and native terminal evidence have their own status.

After composition and Help integration, the first normal gate found a stale
Linux PTY expectation for the former `| Draft` separator. Test-only `b8d7d09`
follows the new explicit pending-draft message without removing staged-state,
masking, dirty-exit or terminal-restoration checks. The focused two-value
`NO_COLOR` journey and fresh full Go normal/race/vet gates all passed on this
integrated tree. Ten wide/dark base-screen captures pass the evidence verifier;
the coordinator inspected every individual image and two relevant crops.
The resulting [visual findings](../review/native-tui-composition-vision.md) keep
the page-body and broader acceptance gates explicitly open.

After integrating Models and grouped Help, fresh full Go normal/race/vet gates
all pass at `889ace7`. The Models main path uses three field selectors and
Advanced disclosure; normal Tab traversal does not walk every hidden option.
The independently reviewed full keyboard journey still retains staging,
attestations, Review/Back, acknowledgment, Apply and reconciliation checks.

The corrected native console fixture candidate `e6fee8e` passed hosted run
[41536849674](https://github.com/clankercode/oneline-ccc/actions/runs/41536849674):
Windows ConPTY, macOS PTY, Linux and all six static builds. This is evidence for
that exact candidate's input/masking/restoration journeys, not physical-terminal
appearance or all-screen acceptance. A later combined candidate `30d2ffc` in
[41537608013](https://github.com/clankercode/oneline-ccc/actions/runs/41537608013)
fails both Windows interactive resize cases: its expected endpoint caret window
does not match the new inset viewport. Original-author diagnosis must distinguish
a stale expectation from input loss or replay defects while retaining full-value,
masking and restoration checks. Its approval remains open; the earlier green
run cannot stand in for this integrated gate. The same combined run also
times out in macOS fixture teardown after its failed resize assertion. The
source-only corrections `2e1ab10` and `5ea1e6a` retain the exact complete-value
oracle and bound/drain the single-waiter cleanup; independent review and a new
actual native run remain pending.

## Integrated page, layer and resource checkpoint

All ten page bodies and the overlay/editor compositions have bounded independent
source approval and are integrated through `0cfc82c`. Availability corrections
start unavailable SSH/Utilities on Back and enforce inspected absent-monitor
installation prerequisites on both Service and Dashboard. Models retains chooser
context in the tiny viewport. Wrapped dialog guidance and complete control labels
are covered at heights 12–14. None of those source checks is a pixel approval.

The normal/race/vet gates pass at `e070d7d` using one CPU and the reviewed shared
verification slot. They took 21.011, 128.929 and 0.861 seconds respectively;
[resource measurements](tui-test-resources.md) explain costs and limits. Real
terminal fixtures follow the composed UI without removing masking, mutation,
shutdown or restoration assertions. The Linux no-color oracle still needs the
explicit sticky transient-exposure check retained in backlog item 19.

The exact hosted candidate `5ea1e6a` in run `41545126866` passed Linux, Windows
and six static builds. Darwin console normal/race passed; Darwin RPC race cleanup
still failed. Reviewed resize/teardown corrections therefore have native console
evidence, while the remaining RPC lifetime diagnostic and production fix stay
open. The isolated diagnostic candidate must not be confused with a product fix.

Fresh PNG generation remains pending the browser-lifecycle correction. Review
found that browser leader exit alone can hide a surviving descendant. The More
evidence predicate also needs to follow real inset menu windows while retaining
the full destination contract; its fresh ANSI-only and historical-image checks
support the correction, with independent review and fresh PNG checks still open.

## Remaining acceptance

- Shared shell and client Dashboard composition are integrated with bounded
  source/geometry approval and interim image inspection. They, every other
  screen, editor and overlay still need the complete alternating review against
  pinned matching-state mockups on the exact final installed revision.
- Reviewed capture tooling and composed multiline prepared-row/Models fixture
  association are integrated. The minimum-size Models title fix has source
  approval; its fresh PNGs remain pending the browser-lifecycle correction.
  Reviewed secondary references are available; compare every native
  surface with those targets. Existing captures retain their original provenance.
- Wrapped-row Help and useful grouped shortcuts are corrected in source.
  Final integrated and per-origin captures remain acceptance work.
- Bounded startup/error cleanup is integrated. Failed-reader constructor-private
  descriptors, native platform failures and uncooperative external I/O remain
  outside that approval; this does not certify every panic/callback combination.
- Windows/macOS console input, restoration, appearance and shortcuts still need
  actual native evidence. Static binaries and hosted unit tests do not close it.
- Client installation, additional adapters, release/bootstrap and the other
  product requirements remain in the [backlog](../plans/remaining-items-so-far.md).

The [campaign](../plans/product-completion-campaign.md) and backlog remain the
execution and acceptance authority. Source fixes and trustworthy artifacts
support visual review; they do not substitute for it.
