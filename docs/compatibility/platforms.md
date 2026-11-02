# Native platform and distribution contract

Checked **2026-11-02**. Build inputs: Go **1.26.1**,
`charm.land/bubbletea/v2` **v2.1.1**, `charm.land/lipgloss/v2` **v2.0.6**;
executable **`oneline-ccc`**, with `.exe` on Windows. See
[stage-zero sources and experiments](../verification/tui-stage-zero.md).
Do not install under `ccc`, which is an unrelated existing command.

## Compile targets versus runtime support

| GOOS / GOARCH | First native-build target | Compile proof in this scout | Native runtime proof in this scout | Product minimum OS |
| --- | --- | --- | --- | --- |
| `linux/amd64` | `CGO_ENABLED=0` build | Not built here; integration gate | Linux x86_64 Codex 0.166.0 configuration/profile checks plus sandboxed CC Switch CLI 5.11.3 fixture trial; no Go app/PTY/lifecycle proof | Unestablished |
| `linux/arm64` | `CGO_ENABLED=0` build | Not built here; integration gate | None | Unestablished |
| `darwin/amd64` | `CGO_ENABLED=0` build | Not built here; integration gate | None | Unestablished |
| `darwin/arm64` | `CGO_ENABLED=0` build | Not built here; integration gate | None | Unestablished |
| `windows/amd64` | `CGO_ENABLED=0` build | Not built here; integration gate | None for native Go app; prior PowerShell/SSH evidence is separate | Unestablished |
| `windows/arm64` | `CGO_ENABLED=0` build | Not built here; integration gate | None | Unestablished |

These are six intended compile targets, **not six supported runtime claims**.
The controller can append exact integrated build/CI results in its own report;
parallel build work is not certified by this scout. Compiler success establishes
neither terminal behavior nor minimum OS version. Go/toolchain dependency floors
and CC Switch's own OS requirements must not be presented as this app's tested
OS floors. Architecture-specific runtime claims require corresponding evidence.

Before advertising runtime support: native terminal input/resize/interrupt and
child handoff/return; configuration validation/backup/recovery; locked or absent
vault and session-key delivery; optional user lifecycle install/start/stop/remove;
and filesystem ownership/permission failure checks on the actual supported OS.
Native CI on all three OS families is an independent requirement, with actual
runner OS/architecture/version recorded. WSL Linux results are not native Windows
results, and emulation/cross-compilation is not native terminal proof.

`CGO_ENABLED=0` is a build choice, not a universal fully-static guarantee. Inspect
final artifact dependencies: verify the promised Linux static/no-language-runtime
surface; document macOS system-library linkage. A literal fully-static macOS
requirement would remain blocked. The existing
[stack research](../research/tui-stack.md) links the relevant Go/platform sources;
no dependency inspection or OS floor trial was performed by this scout.

## Available distribution surface and missing trust

**No release signer resources or bootstrap trust identity have been established.**
The first slice may publish reviewed source, reproducible build instructions and
truthfully labelled CI artifacts. Source installation means building a reviewed,
pinned checkout locally; it does not imply a trustworthy remote executable-fetch
bootstrap. CI artifact availability does not mean authenticated production release.
No automatic self-update, automatic bootstrap installation or signed/notarized
release capability is claimed or enabled by this contract.

Before executable-fetch bootstrap or self-update is offered, establish and test:

- Exact release publisher/location and an authenticated manifest binding version,
  OS/architecture and artifact digest to a pinned signer or verified workload
  identity, with an initial trust anchor independent of a mutable download URL.
- Signer custody/availability, verification implementation and identity rotation;
  provenance/SBOM and explicit Windows Authenticode/macOS notarization status.
- Reject wrong target/version/identity, tampering and missing signer before
  execution; package-manager ownership; interrupted replacement, rollback and
  failed restart in disposable native environments.

HTTPS plus a checksum downloaded beside a binary is useful integrity data, but
does not by itself satisfy this authenticated distribution contract. Missing
signers block automatic distribution/update, not local source work or the native
UI build. A CI artifact must display its actual unsigned/unverified status.

This shipping contract is separate from bounded evaluation trust. For the
[completed adoption trial](../verification/tui-stage-zero.md), the user authorized
a fixed v5.11.3 publisher-selected executable, verified against GitHub's release
asset digest and published checksum, running only inside a network/PID-isolated
sandbox with operator/repository roots hidden and fixture-only state. An
independent release signer was not a prerequisite for that trial. No host install
or bootstrap occurred, and this Linux third-party trial establishes no native
Go app support, minimum OS floor, other architecture or signed-release claim.

The optional observer remains separately gated: systemd **user** unit on Linux,
LaunchAgent on macOS, current-user logon Task Scheduler task on Windows; absent
backend means unavailable with foreground operation retained. These are intended
backends, not runtime tests completed here. Windows SSH remains its separately
owned existing path: [current MVP evidence](../verification/windows-mvp.md).
No SSH implementation, service configuration or existing support claim was altered.
