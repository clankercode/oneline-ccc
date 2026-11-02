# TUI, background lifecycle and release stack

Checked 2026-11-02 against public primary sources over HTTPS. V = verified source statement/code; D = proposed decision; A = assumption; U = not established. No package installation, compilation, native service test, credential access or inference occurred. Versions must be pinned after a compatibility spike; links to moving branches are evidence for this dated review, not immutable release inputs. [Prior project research](existing-projects.md) supplies the adoption baseline; this note does not repeat the proxy survey.

## Verified candidates and consequential limits

| Area | Primary evidence / verified fact | Consequence / remaining check |
| --- | --- | --- |
| Bubble Tea | [README](https://github.com/charmbracelet/bubbletea/blob/main/README.md), [module](https://github.com/charmbracelet/bubbletea/blob/main/go.mod): V current module is `charm.land/bubbletea/v2`; message/update/view model and cell renderer are documented | D default TUI framework. Pin a released version and matching component APIs after a native spike; do not paste v1 tutorials into v2 |
| Lip Gloss | [README](https://github.com/charmbracelet/lipgloss/blob/main/README.md): V current examples import `charm.land/lipgloss/v2`, with terminal color-profile support | D semantic tokens/layout styling behind TUI boundary; color downsampling does not prove contrast/accessibility |
| tview/tcell | [tview](https://github.com/rivo/tview), [tcell](https://github.com/gdamore/tcell): V rich form/table/list widgets; tcell documents Unicode and Linux/macOS/Windows workflows | Viable rival for widget-heavy UI. Native resize/focus/screen-reader behavior still requires tests; no fabricated version pin |
| Cross-platform service library | [kardianos/service](https://github.com/kardianos/service): V install/uninstall/start/stop/run and Windows/Linux/launchd support; documents backend limitations | Useful candidate for later SCM/system-service support. Generic support does not establish desired per-user privileges/session lifetime |
| Native lifecycle | [systemd units](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html), [Apple launchd](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html), [Windows task contexts](https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks): V distinct user/session/security models | D native systemd user / LaunchAgent / current-user logon task adapters; foreground fallback. Task Scheduler is not SCM |
| Windows SCM | [service accounts](https://learn.microsoft.com/en-us/windows/win32/services/service-user-accounts), [per-user services](https://learn.microsoft.com/en-us/windows/application-management/per-user-services-in-windows): V service identity differs from interactive user, with distinct account/template semantics | Do not elevate the whole app or assume an SCM service can unlock a user's vault. SCM OpenSSH is separate; own app SCM backend deferred |
| Credentials | [go-keyring README](https://github.com/zalando/go-keyring), [Darwin source](https://github.com/zalando/go-keyring/blob/master/keyring_darwin.go), [Windows source](https://github.com/zalando/go-keyring/blob/master/keyring_windows.go): V Windows vault, Linux Secret Service/DBus, macOS `/usr/bin/security`; Darwin Set currently uses `security -i` and stdin, not a password argument | D candidate vault adapter, not unconditional endorsement. Audit chosen release for subprocess timeout/cleanup and size limits; unavailable/locked vault needs session-only mode. No “encryption isolates same-user agents” claim |
| Go packaging | [cgo](https://pkg.go.dev/cmd/cgo), [Darwin runtime source](https://github.com/golang/go/blob/master/src/runtime/sys_darwin.go): V cgo is a separate facility; Darwin runtime calls OS libc/system frameworks | D `CGO_ENABLED=0` release builds; check actual binary dependencies. “Self-contained Go executable” is viable; literal fully static macOS is not established and must not be promised |
| Release automation | [GoReleaser Go builder](https://goreleaser.com/customization/builds/builders/go/), [signing](https://goreleaser.com/customization/sign/sign/): V OS/architecture build matrix and signing integration are documented | D GoReleaser packaging with authenticated manifests/checksums and provenance. Signer identity, key custody, Windows Authenticode/macOS notarization availability and bootstrap trust are U |
| Metadata | [models.dev README](https://github.com/anomalyco/models.dev/blob/dev/README.md): V public `api.json`, evolving source schema, explicit `canonical_model_id` for applicable generated provider entries | Separate remote response contract from repository schema. Pin parser fixtures/schema checks; exact gateway availability/capacity/tier must come from gateway or explicit operator evidence |

## IGC: TUI choice

Context: Go-only manager with ten views, keyboard flows and isolated side effects. G1 = render interactive lists/forms on three target OSes; G2 = application can keep side effects outside widgets; G3 = persistent responsive dashboard/list-detail views with no-color text. Y means no known decisive error from source review; runtime acceptance remains a separate gate. All is N if any N, otherwise ? if any ?, otherwise Y.

| Idea | All | G1 | G2 | G3 |
| --- | --- | --- | --- | --- |
| **Bubble Tea v2 + Lip Gloss v2** | Y | Y | Y | Y |
| tview + tcell | Y | Y | Y | Y |
| Prompt-only form wizard as the whole UI | N | Y | Y | N |

A form wizard alone fails G3's simultaneous responsive dashboard/list-detail behavior; it remains useful for plain/narrow mode. Both full frameworks survive; no artificial score breaks the tie. D choose Bubble Tea as a reversible routine preference for an explicit message/update/view model and custom hierarchy. These Y cells assert feasibility, not completed native validation. Keep application ports and screen state independent; reversal replaces the renderer/focus layer and UI tests, not adapters or stored profiles. Cost: moderate UI rework before visual stabilization, high after it.

## IGC: background authority and platform lifecycle

Context: optional first-run monitoring, routine foreground changes and no automatic repair/update requirement. G1 = foreground remains fully usable with daemon declined; G2 = background startup needs neither credential access nor mutation/elevation authority; G3 = identify actual user/session backend and support lifecycle controls.

| Idea | All | G1 | G2 | G3 |
| --- | --- | --- | --- | --- |
| **Read-only observer + native user lifecycle adapters** | Y | Y | Y | Y |
| Required privileged gateway/maintenance daemon | N | N | N | Y |
| Generic service wrapper assumed to supply user-session semantics | ? | Y | ? | ? |

The required privileged daemon fails G1/G2 by its proposed authority/lifetime; routing is a separate adoption decision. A wrapper is not refuted but its exact user-session contract is unknown; do not rely on it until tested. D choose explicit per-platform user adapters, with observer absent/unavailable states. Cost: three small native integration surfaces and logged-in lifetime limits. Reversal to a system daemon is expensive because identity/credential/IPC authority changes; investigate only with a demonstrated need.

## IGC: secret distribution and packaging

Context: direct gateways, no bundled private credentials, static-distribution intent. G1 = ordinary setup persists no plaintext secret by default; G2 = binary needs no non-OS language runtime; G3 = archive integrity/authenticity checked before executing downloaded code.

| Idea | All | G1 | G2 | G3 |
| --- | --- | --- | --- | --- |
| **No-cgo Go binary + vault/session launcher + authenticated release manifest** | Y | Y | Y | Y |
| Binary + persistent user-environment secret as default | N | N | Y | Y |
| Runtime-fetched unsigned executable from a mutable latest URL | N | Y | Y | N |

Persistent environment storage fails G1 (the existing Windows MVP explicitly documents that limit); keep it only as an explicit compatibility choice, preserving existing references. An unsigned mutable download fails G3 even if it has a checksum hosted beside it; TLS/checksums alone do not define an independent release-signing trust anchor. D GoReleaser supplies packaging, not a substitute for key custody or bootstrap trust. Y for the chosen *design* requires its authenticated-manifest component; no current signer availability is asserted. Pin trust identity/version at T0, then test rejection paths.

GoReleaser and hand-maintained Go build matrices can both satisfy G2/G3; use GoReleaser as a reversible routine tooling choice. Reversal changes CI/manifests, not app code. Vault/session versus persistent-environment choice changes launch ergonomics and migration, so reversal costs more; show that cost before offering bare-client compatibility. Literal “all-static macOS” is a separate unresolved acceptance interpretation because Go still uses Apple system libraries.

This default also requires the production plan's T3a client-install and T4a launch gates: verified upstream/version and prerequisite ownership, then exact vault/session credential delivery to the selected executable/config/working directory, plus terminal suspend/handoff/return. Package-manager installs do not inherit gateway credentials. Synthetic child and native PTY checks establish these contracts without inference; source-reviewed vault support or valid configuration alone does not establish a usable authenticated launch.

## Client and utility identity check

| Name in brief | Primary-source result on baseline date | Scope decision |
| --- | --- | --- |
| Pi / PyAgent | V [old repository API](https://api.github.com/repos/badlogic/pi-mono) redirects to [earendil-works/pi](https://github.com/earendil-works/pi); [coding-agent package](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/package.json) identifies `@earendil-works/pi-coding-agent` and executable `pi` | Label Pi; do not invent a separate PyAgent adapter. Verify selected installed release and older package compatibility before import |
| Hermes | V [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent) documents Hermes Agent/custom providers | Candidate identity established; local customizations/schema ownership remain U, read-only until inspected with authorization |
| Grok Build | V official [x.ai/cli](https://x.ai/cli) redirects to [x.ai/build](https://x.ai/build) and identifies Grok Build | Product identity established; config/auth/custom-endpoint contract U. Guessed `/docs/grok-build` and developer paths returned 404, so no adapter API inferred |
| herdr | V [herdrdev/herdr](https://github.com/herdrdev/herdr), API README on **master**, describes an agent terminal runtime/background server | Candidate exact identity; operator-intended match and native platform/install/license verification still required before install. A guessed main-branch raw README returned 404; corrected via repository API |
| tmux | V [tmux/tmux](https://github.com/tmux/tmux) documents multiplexer and libevent/build dependencies | Established identity; do not promise native Windows availability or bundle its dependencies into the static Go binary |
| t3 | V [pingdotgg/t3code](https://github.com/pingdotgg/t3code) README identifies T3 Code, desktop/web/mobile agent control and Windows/macOS/Linux setup paths | Plausible exact identity, not proof of operator intent. Distinct product with its own lifecycle; no bundled install by default |
| paseo | V [getpaseo/paseo](https://github.com/getpaseo/paseo) README identifies an agent development environment across desktop/mobile/web/CLI | Candidate identity, not proof of platform release/install trust or intended match; gate manifest accordingly |
| orca | U short name does not identify a unique coding utility in the retained brief | Keep unavailable; request exact repository during later wayfinding, not a guessed installation |

Codex, Claude Code and OpenCode remain retained client requirements. Codex configuration/version/protocol evidence is in repository research and Windows verification; OpenCode identity is already established as anomalyco/opencode in the existing survey. No detailed contracts for the other adapters were newly verified here.

## Evidence boundary and next checks

Source retrieval succeeded for the linked framework/module/platform/credential/release/metadata and named identity documents. No need to repeat the prior proxy inventory. Failed guessed Grok/herdr paths were corrected or left explicitly unknown; their absence is not evidence a feature is unavailable. GitHub search supplied a herdr candidate only; its repository metadata/README supplied the primary identity evidence.

Before implementation: pin released dependency versions and minimum OS floors; run the chosen UI on native PTYs; test locked/headless vault cases and Darwin subprocess cleanup; prove account-scoped background lifecycle and IPC ACLs; verify downloaded artifact trust and actual executable dependencies. Before enabling a client/utility: verify exact product/version/schema, install/remove ownership, runtime requirements and license. Before claiming gateway compatibility: obtain separately authorized protocol/capacity/tier evidence; inventory and models.dev alone cannot establish it.
