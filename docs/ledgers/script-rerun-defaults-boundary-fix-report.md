# Reopened capability-boundary task

Accepted RR1: a stale Ultrafast true flag survived native arbitrary-TOML merges
after a route change or explicit tier restriction; a later defaults read promoted
it as current capability. Added two integrated sequence tests and watched0/2 RED
in .cache/rerun-boundary-red.log. Native merge now always synchronizes this owned
flag, writing true when enabled or null to delete it when disabled. Other feature
keys remain untouched. Sequence tests parse both main/named TOML with Python,
assert absence after change and after later omitted-input rerun, and retain the
unrelated comment. Full core101/0 GREEN in .cache/rerun-boundary-core-green.log.

Ruling R6: material scoped-review regression reopens the affected task, rather
than treating the prior final-pass limit as permission to ship it. This is a
bounded three-file correction, no unrelated source work or SSH change.

RR2 is an integration dependency: docs/plans/tui-production-plan.md is already
committed/pushed on main4ee2553. Merge both branches then verify the link exists;
no link or TUI-content rewrite required. The feature branch alone lacks that
separate planning commit, as the reviewer correctly observed.

Current HEAD is frozen for targeted independent re-review. Prior native feature
run41404807893 passed both editions99/0 core+87/0 SSH and real SSH. Deployment was
explicitly rejected because feat/rerun-defaults is not an allowed Pages branch;
no public script update is claimed. Final main workflow must test101 cases and
verify deployment/public bytes. Actual operator SSH startup incident remains open
pending requested Windows logs; no root cause or guessed SSH fix is claimed.

=== DONE ===
