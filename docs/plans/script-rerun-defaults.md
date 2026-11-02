# Preserve LLMP defaults on installer reruns

Goal: Enter keeps current LLMP URL/model/tier/context, saved credentials remain hidden and reusable, and an unchanged rerun can reach the optional SSH offer without resetting preferences. Script changes are isolated on feat/rerun-defaults; TUI planning is a disjoint documentation task.

Input precedence: explicit nonempty command arguments; active oneline_llmp user configuration; otherwise the installer-owned named LLMP profile; otherwise first-run prompts/defaults. Do not import other providers' root model/context/tier or credentials. A process LLMP_API_KEY takes precedence over the Windows User LLMP_API_KEY. Interactive secure-key entry accepts Enter to keep that saved key; a missing key remains required. Noninteractive runs reuse existing LLMP values when available and refuse missing required inputs.

Read existing byte snapshots before prompting and retain them for concurrent-change guards. Recognize exact generated TOML without native side effects. For valid arbitrary/main TOML, use isolated private native config/read staging, never a regex TOML parser. Extract only the owned provider's input fields, never inline credentials. DryRun must not create staging or mutate files; use safely recognized managed defaults or require explicit inputs when arbitrary TOML needs native parsing.

Keep the prior compaction limit when context stays the same; recompute90% when its context changes unless explicitly overridden. Model/catalog/tier capabilities belong to a route: inherit catalog/tier only when endpoint and model are unchanged. Refresh an installer-generated catalog against the selected native release while preserving its tier restrictions; retain an explicit external catalog as an external source. Keep existing main/profile/config-key-PATH rollback behavior and backup byte fidelity.

Task1: add failing integrated tests for omitted arguments, Enter defaults, process/User hidden key reuse, active-main precedence, other-provider isolation, explicit overrides, compaction behavior and dry-run preservation; implement bounded helpers and prompt flow; run complete core/SSH/native gates; independent review/fix.

Task2: validate actual Windows PS5.1/PS7 and native SSH/public bootstrap, push/deploy, compare hosted bytes, give the operator one test command including SSH. No real key or inference in development.

Task3 (parallel): subagent produces the TUI production/aesthetic plan only; separate review; notify with attn when ready. No TUI implementation in this phase.

Review focus: secrets never printed; empty input cannot erase an existing key; current active LLMP settings beat a stale named profile; unrelated provider roots never become LLMP defaults; context changes cannot retain an invalid old compaction limit; inherited catalogs preserve restrictions and match the selected native version; same config snapshots still guard publication; unsafe/incomplete native staging cleanup remains fatal.

Workflow: ultra-code-workflows with file briefs/reports, test RED->GREEN, independent task review, fresh whole-change review and verified try-it deployment. Controller implements this urgent script path; planning/review seats are independent. User's requested prompt behavior supplies the design authorization, so routine implementation proceeds without another approval gate.
