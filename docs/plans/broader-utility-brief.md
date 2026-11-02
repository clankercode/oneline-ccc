# Broader utility: brief retained for the design session

Status: requirements captured; implementation has not started. Finish and test
Windows Codex MVP first, then use the existing-project research and a grilling /
wayfinding session to decide the broader design. Follow `ultra-code-workflows`
for development, adapted explicitly to the available harness.

## Intended outcome

Provide one-line first-run setup on Windows, macOS and Linux, followed by a
statically linked Go executable with a TUI and an optional daemon. Users point
coding clients at a custom gateway using one endpoint and API key. Support both
packaged proxies and custom gateways; avoid assumptions tied to one deployment.

Candidate clients named by the operator: Codex, Claude Code, PyAgent, OpenCode,
GrokBuild and Hermes. Hermes may have user-specific customization that must be
understood before an adapter writes configuration.

## Management capabilities to investigate

- Discover available model IDs from the configured endpoint. Query models.dev
  for metadata such as context capacity; distinguish metadata from what the
  actual gateway exposes or permits.
- Manage per-client/model context budgets, including shared budgets that allow
  context switching. The operator commonly caps million-token models at 500000
  and also uses 272000 or 353000; these are selectable budgets, not universal
  model capacity claims.
- Represent service tiers only where supported, including Codex Fast and
  Ultrafast. Keep model identity, request tier and UI catalog metadata separate.
- Offer service installation on first run. The TUI manages service installation,
  status and supported clients. Automatic updates and watching endpoint/config
  or model changes are possibilities to evaluate, not settled mechanisms.
- Offer useful utilities, with examples supplied as herdr, tmux, t3, orca and
  paseo. Determine exact projects, platforms and installation behavior during
  design rather than guessing from names.
- Let a currently working coding agent help repair another client when the user
  explicitly runs the repair. Automatic repair is a possible opt-in later;
  manual invocation is the initial requirement.
- Offer Windows OpenSSH installation, firewall, user and key authentication in
  both MVP and full application. Private profile is the default; Public is an
  explicit option. Warn when the current network is Public without silently
  changing its classification. MVP scope is LocalSubnet; broader configurable
  subnet choices may include sibling networks, Tailscale and WireGuard.
  Support GitHub username public-key import and pasted public keys.

## Constraints and next decisions

Keep private deployment details and credentials out of public files and Git
history. Preserve existing client configuration and make failures recoverable.
Do not turn speculative daemon behavior into a commitment before the design
session. Clarify credential storage, platform service models, update ownership,
client adapter contracts, model metadata precedence, repair authority and the
smallest useful TUI scope there.

[Existing-project research](../research/existing-projects.md) is already available
and must inform the design before implementing overlapping functionality.
