# Native foundation delivery ledger

Plan: [native foundation](../plans/native-foundation.md).
Baseline: `5769d423bbe5c698c5ef559da5d17204e23f99ce`.
Acceptance source: `94feea622814bc677cf95267096ac4872bce830e`.
Mode: subagent-driven implementation in isolated worktrees, durable briefs and
reports, independent subsystem reviews, controller integration and native gates.

## Delivered scope

The native Go executable connects the approved colored terminal interface to
Codex inspection, settings, reviewed application, receipts and launch/return.
It includes shortcut help, command search, draft preservation, service status,
explicit observer actions and Windows SSH handoff. Exact model identity, default
request tier and available tiers remain distinct. Unsupported adapters and client
versions retain an explanation rather than accepting writes.

Protected transactions preserve foreign settings, refuse drift and unsafe native
metadata, and support explicit recovery. Keys use a foreground session, an
existing environment reference or Windows Credential Manager. Application startup
does not install services, modify clients, run repair agents or make inference
requests. See [development use and limits](../native-app.md).

## Review and corrections

- Subsystem reviews covered transactions, native adapter/application boundaries,
  credentials/catalog, UI behavior, CLI and observer authority.
- The [whole-branch review](native-foundation-final-review.md) identified unsafe
  foreground shutdown, Linux inode-policy metadata loss, systemd wire decoding,
  uncertain service-state display and persisted-key selection. All were accepted
  and corrected in the [final fix pass](native-foundation-final-fix-report.md).
- The [scoped final review](native-foundation-final-rereview.md) approved those
  corrections at `154b35e`, plus the CLI process, redaction, JSON-identity, output
  failure and Darwin pipe corrections.
- Actual terminal acceptance then reproduced raw terminal modes after SIGTERM;
  native Windows CI also repeatedly found staging sharing failures. These were
  reopened as acceptance defects. The [acceptance fix](native-foundation-acceptance-fix-report.md)
  extracts one shared process owner for launches, configuration calls and version
  probes, restores the exact pre-child terminal state, and reports staging cleanup
  failures while preserving substituted directories. Integration: `bbfbe43`.
- The real rendered-TUI SIGTERM regression was added separately at `94feea6`.
  Its original failure and successful snapshot probe are retained in the final
  fix report. Final integrated gates and scoped acceptance review follow below.

## Controller decisions

1. Deliver the approved first native slice while retaining wider production gates.
2. Name the executable `oneline-ccc` because `ccc` already names another tool.
3. Pin the verified Go/Bubble Tea toolchain and use Codex's native configuration
   API rather than rewriting unrelated TOML with a separate serializer.
4. Keep automatic download/update unavailable until authenticated distribution is
   established; provide source and CI development artifacts in the meantime.
5. Accept and independently review concrete credential, transaction, UI, CLI and
   native protocol defects rather than retaining test-only assumptions.
6. Retain the existing private workflow workspace during continuation; publish
   sanitized review records alongside this ledger.
7. Continue focused acceptance corrections after the single final fix pass because
   actual terminal and native Windows failures still affected usability.
8. Put terminal restoration in the shared process owner before Tea resumes,
   avoiding a competing outer terminal-reset mechanism.

These are reversible implementation/distribution choices. An incorrect decision
costs a scoped follow-up or another platform gate; broader capabilities are not
advertised as complete.

## Final gates

Fresh controller execution on the acceptance source:

| Gate | Result |
| --- | --- |
| Full normal suite with real isolated native Codex/catalog | PASS: 643 test/subtest events, four conditional skips, 12 packages |
| Full race suite with the same native fixtures | PASS: 641 test/subtest events, four conditional skips, 12 packages |
| `go vet -p 2 ./...` | PASS |
| Cgo-disabled standalone Linux build | PASS |
| Preview state tests and diff whitespace | PASS: nine state tests |
| Actual sandboxed terminal setup journey | PASS: help, command search, masked key, review/apply/receipt, launch/return and five terminal sizes |
| SIGTERM during actual native Codex handoff | PASS: child reaped and original terminal modes restored exactly |
| [Native CI 41451803057](https://github.com/clankercode/oneline-ccc/actions/runs/41451803057) | PASS: all nine jobs, including Windows/macOS/Linux native tests and six OS/architecture builds |

The four Linux conditional skips are the unsupported-platform branch, two
inapplicable rollback fault positions and the Windows-only observer refusal test.
The normal suite includes the rendered-PTY regression; its two helper events are
absent from the race suite for the dependency reason retained below.

The [independent acceptance review](native-foundation-acceptance-rereview.md)
approved Standards and Spec for the frozen candidate, with no new material
findings. Main integration, local installation and Pages publication follow this
verified source; their workflow results remain visible in repository Actions.

## Retained limits and follow-up

- Mutation supports native Codex 0.166.0. Additional clients, versions, utilities,
  automatic installation/updates and automated repair remain on the production
  plan; no capability is inferred from discovery alone.
- Persistent native credentials initially support Windows only. Linux/macOS
  foreground session/environment credentials remain available.
- Native observer session lifecycle and Windows console terminal modes require
  their own disposable-session acceptance. Windows observation is unavailable.
- The actual rendered-PTY SIGTERM test runs without race instrumentation because
  that fixture exposes a pinned Bubble Tea/cancelreader reader-close race. The
  headless lifecycle and shared ownership remain covered by the race suite.
- Source/CI artifacts are development distribution. Signing, trusted bootstrap
  and automatic binary updates retain their release gates.

The [verification record](../verification/native-foundation.md) distinguishes
native configuration, terminal, observer, distribution and inference evidence.
