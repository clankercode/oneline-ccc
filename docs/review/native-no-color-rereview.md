Spec verdict: APPROVED for the scoped I1/R9 correction. Missing: none within this correction. Extra: none. Misunderstood: none.

Quality verdict: APPROVED. I1/R9 ADDRESSED. No new Critical, Important, or Minor findings in `0e7ada2..c3985a8`.

## Scope and identity

Reviewed the complete re-review brief, original independent review contract and I1 finding, exact saved fix diff, actual changed source, Fix round1 report appendix, all six `.cache/a5b-r1-*.log` outputs, and retained fault-overlay sources/maps/manifest. Review worktree: `/home/xertrov/src/oneline-ccc/.worktrees/native-no-color`.

HEAD is `c3985a841296ad82afc88b3880553cd009fe5d57`, with parent `0e7ada200bee2e47cb76df202837eb0e379c065c`. The assigned worktree is clean. The only tracked change is `cmd/oneline-ccc/no_color_terminal_linux_test.go`: 24 insertions, 2 deletions. `git diff --check 0e7ada2 c3985a8` passes. The saved `A5b-fix1-review.diff` matches the actual diff with seven context lines; both SHA-256 values are `3573de42b1e1ffe3dd8c24dae06e2846085e1b7dc64b2458d7e7e22a12a1302e`.

## I1/R9 disposition and checked-and-fine items

- **Observable barriers before Escape:** test lines 451–452 separately send Gateway Tab and Enter and await focused labels. Lines 456–460 send ordinary dummy-key bytes, then Tab, await `> [ ] Persist using backend credential storage`, check masking, separately send Shift+Tab, await `> API key: [hidden]`, and check masking again. Escape is sent only at line 461, after both checks have completed. The persistence focus marker is absent from the preceding key-focused state, and the key focus marker is absent from the persistence-focused state. Each transition therefore requires observed terminal changes through the existing bounded `await` helper.
- **Live editor assertions:** `assertKeyHidden` at lines 389–400 reconstructs the observed terminal cells, requires the editor heading and expected focused row, and rejects the dummy key in the live screen, complete raw output, and ANSI-stripped output. It neither manufactures a matching view nor replaces the production renderer.
- **Claim stays within observable evidence:** source comments at lines 453–455 and the report appendix at lines 90–94 expressly reserve exact-value acceptance/retention. Successful output establishes the two still-open editor focus states after sending ordinary bytes, with no observed dummy-value exposure, before cancellation. No buffer inspection, SaveKey call, vault access, or stored-value claim was added.
- **Existing guarantees preserved:** the exact diff changes only the new masking helper and staged input sequence. Production entry/OS environment lookup, real PTY setup, renderer and foreground lifetime, deadline-based reads, kill/wait/restore/close cleanup, backend counters and isolation remain unchanged. Raw-mode checks, whole-termios restoration before cleanup, alternate-screen entry/exit, final complete-transcript masking, color-SGR rejection, dirty-draft Ctrl+C/Escape retention, guarded normal exit, and separate real redirected stdin/stdout/plain cases remain active. No sleeps, additional skips, fixed top-level tab counts, or relaxed assertions were introduced.

## Fault and success evidence correspondence

The retained `.cache/a5b-r1-probe/old-overlay.json` substitutes exactly two files: the injected `panels.go` and the old burst test. `staged-overlay.json` substitutes only that same injected `panels.go`, leaving the current staged test in use. Both retain the production renderer and fixture boundaries.

The injected source differs from real `internal/tui/panels.go` only by a comment and a conditional appending the entered value to the key label when `len(e.value) > 0 && e.focus == 1` (probe lines 113–116). This is a deliberate populated-editor exposure fault. The actual production source is unchanged from `cbbaf77`.

Independently recalculated SHA-256 values match the manifest and relevant Git versions:

| Artifact | Verified SHA-256 |
| --- | --- |
| Production `panels.go`, also `cbbaf77:internal/tui/panels.go` | `9d7573df63c9f6362f49ade0ccc9141a31e8c392fa5ce412e391da0111023269` |
| Injected probe `panels.go` | `f847dfb073f88220657a0e74e24e67c6a47b3b774ce32e009c48bca38cf1c437` |
| Retained old test, also `0e7ada2:cmd/oneline-ccc/no_color_terminal_linux_test.go` | `52b675e0cb5fea1e563cc425cfb3864eec97f026f9f4cc1b01622d7ffbad234b` |
| Current staged test | `3b3d337ecba8b1fb7dfaf8b5565e7a0591d88aa43cf4a2c658d462e68034e2e5` |

Saved outputs agree with the exact commands and outcomes recorded in the Fix round1 appendix:

| Output under the assigned worktree | Observed result |
| --- | --- |
| `.cache/a5b-r1-old-fault.log` | Both NO_COLOR values PASS despite the injected fault; package 0.853s. |
| `.cache/a5b-r1-staged-fault-red.log` | Both values FAIL at current test line 458 with `dummy key visible before credential cancellation`; package 0.382s. This is the first new live-editor masking assertion, before Escape. |
| `.cache/a5b-r1-green.log` | All focused terminal, redirect/plain, detector and reconstruction cases PASS without overlays; 0.963s. |
| `.cache/a5b-r1-race.log` | The same focused cases PASS without overlays; 7.155s, no race diagnostic. |
| `.cache/a5b-r1-cmd.log` | CLI package PASS without overlays; 1.420s. |
| `.cache/a5b-r1-initial-green.log` | Earlier focused success, 0.966s; consistent with the later success evidence. |

The old-test PASS and revised-test failure use the same retained exposure fault. The staged failure is caused by observed credential exposure at the intended barrier, rather than timeout, missing focus, old-entry fallback, or an unrelated harness failure. The ordinary success path must pass both focus barriers and both masking assertions before it can reach the remaining journey and normal-exit checks. These artifacts directly address the original coalescing blind spot. The report correctly avoids claiming that the old burst misses the fault under every possible schedule.

## New regressions

None found in this scoped test-only correction. No production source changed, and no previously reviewed lifecycle/color/isolation/backend assertion was removed or weakened.

## Cannot verify / limits

- These execution results are inspected saved artifacts; this independent re-review did not rerun tests or fault probes. The report records process exit statuses; the retained plain-text logs show the corresponding PASS/FAIL results. Separate raw PTY transcripts, process inventories and termios dumps are not retained here.
- Exact credential-value acceptance/retention is intentionally unasserted by the successful constant-mask UI path. This limitation is explicit and does not reopen the corrected renderer-observation issue.
- Main's subsequent More/focus-region integration, native visual/all-screen acceptance, broader product acceptance, and native Windows/macOS terminal behavior remain outside this bounded approval. Coordinator integration/retesting remains separate.

All checks were read-only. No source/Git mutations, runtime probes, suite reruns, agents, operator-state access, live calls, or notifications were performed. This designated ignored report was written last and is the only file written by the re-review.

=== DONE ===
