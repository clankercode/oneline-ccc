# Rerun defaults implementation report

Frozen commit: 5cf745b3b33ed6b515266e39ce7cdba68e77dd8a. Controller implemented the urgent script path while independent subagent planned TUI. Scope: script, shared/core tests, rerun tests, README/site copy and plan. No real user config/key reads, inference or SSH changes.

Original seven focused cases:1pass/6fail, missing required defaults and missing saved User key (.cache/rerun-red.log). New behavior then focused7/0; added compatibility cases brought focused11/0 (.cache/rerun-focused-final.log). Those added cases exercise existing guards and preference boundaries; they are not falsely claimed as previously RED. First full run89/2 surfaced existing secret-bearing profile refusal leaving a staging root; upfront profile validation fixed it, focused old regressions2/0 and final full95/0.

Final gates run from this worktree:
- /home/xertrov/src/oneline-ccc/.cache/powershell/pwsh -NoProfile -File tests/run.ps1 ->95passed/0failed, .cache/rerun-core-final.log.
- same pwsh -NoProfile -File tests/ssh.ps1 ->87passed/0failed, .cache/rerun-ssh-final.log.
- same pwsh -NoProfile -File tests/native-smoke.ps1 ->PASS bare codex loads LLMP default; .cache/rerun-native-final.log.
- git diff --check and pending-file private-host/credential-shape scan passed.

Design: active oneline_llmp user layer supplies defaults, otherwise exact owned profile; exact generated TOML fast-path recognition only, arbitrary TOML read through private isolated Codex user-layer RPC. Only input fields extracted, no inline auth imported. Snapshots captured before prompts. Enter keeps saved URL/model/tier/context/custom compaction. Key secure prompt keeps Process then Windows User saved key without showing it; noninteractive reuse and explicit args remain. Changing route does not inherit tier/catalog. Installer-owned catalog refresh retains caps and current installed version. DryRun skips native staging, uses recognized managed defaults or explicit values. Optional SSH launcher remains independent and unchanged.

Cleanup delta: existing required two successful reaps remains default; read-only one-process staging explicitly passes requiredStops1, with the same private root/config-deletion/completeness checks. No broad new parser or third-party dependency.

Remaining gates: independent fresh review, native Windows PS5.1/PS7 and real SSH/public bootstrap. Normalized manual main TOML may receive managed-key formatting on first apply; byte-idempotence claim is scoped to unchanged installer-generated config. No production-readiness claim for TUI implementation; separate planning only.

=== DONE ===
