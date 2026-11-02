# Native application foundation

Approved direction: [native production plan](tui-production-plan.md) and the
[usable visual specimen](../design/tui-usability.md). This is the first working
native slice; broader adapters and platform release gates remain on that plan.

Executable: `oneline-ccc` (`ccc` is already an unrelated installed command).
Go 1.26.1; Bubble Tea v2.1.1; Lip Gloss v2.0.6. No inference or operator
configuration changes during development. Tests use explicit disposable roots.

1. Establish domain/application ports, module pins and stage-zero evidence.
2. Implement protected, drift-aware multi-file configuration transactions with
   exact backups, rollback and interruption recovery, independently reviewed.
3. Reuse the pinned native Codex configuration API in an isolated staging root;
   preserve foreign TOML, populate default and named profile, validate before
   commit. Other detected clients remain inspectable until their own adapters.
4. Build the actual terminal shell and client → settings → review → apply →
   receipt path, with help, command palette, safe drafts and source-labelled state.
5. Add explicit session/vault credential delivery, bounded inventory/metadata
   requests and native launch/return. Optional observer lifecycle uses native
   user backends; unavailable operations retain useful reasons.
6. Integrate native tests, Linux PTY proof, six cross-builds and native CI on all
   three OSes; independent final review/fix. Install the tested local executable,
   publish the verified source/build surface and notify the operator.

Acceptance: real fixture files/effective Codex configuration change only after
review/apply; reruns retain settings; stale/invalid review or failed validation
changes nothing; safe drafts survive navigation; keys never enter files/diffs/
logs; root, app store and service remain separately owned. Clear labels separate
configuration, authentication and inference. Default tier and availability stay
distinct. Optional service/SSH/install/update/repair must not run on startup.

Each worker gets disjoint files, a disk brief/report and independent review.
The controller handles critical-path interfaces, integration and publication.
