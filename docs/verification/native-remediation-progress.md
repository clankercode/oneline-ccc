# Native remediation checkpoint

Recorded 2026-11-03. Integrated terminal source: `418283a`.

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

## Remaining acceptance

- New shared shell and client Dashboard composition, then every other screen,
  editor and overlay, still need the mandatory alternating visual/source review
  against pinned mockups on the exact final installed revision.
- Reviewed capture tooling is integrated; the inherited observation-type guard,
  integrated More semantics and compact draft/filter fixture coverage remain
  follow-ups. Reviewed secondary references are available; compare every native
  surface with those targets. Existing captures retain their original provenance.
- Narrow Help title/current-page visibility is still a concrete readability case.
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
