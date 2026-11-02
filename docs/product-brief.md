# Project brief

Recorded from the operator's brief on 2026-11-02. This records intent, not an
approved architecture for the broader utility.

## Immediate MVP

One-line Windows setup for native Codex using an LLMP proxy base URL, one API key,
and exact model IDs. Generate valid configuration, expose supported service tiers,
and configure practical context budgets rather than accepting an undersized client
fallback. Make this useful on a new Windows machine immediately.
Also offer optional Windows SSH server installation and public-key access, with
the local-subnet/Private defaults and explicit choices recorded below.

## Broader direction — design pending

- One-line initial setup on Windows, macOS and Linux.
- A statically distributed Go executable, a useful TUI, and an optional background
  service offered during first run.
- Configure multiple coding clients against a custom endpoint and API key,
  including gateways, pass-throughs, OpenCodex and CLIProxyAPI deployments.
- Candidate clients: Codex, Claude Code, Pi Agent (called PyAgent in the brief),
  OpenCode, Grok Build and Hermes. Verify exact products and their configuration
  contracts before implementing adapters; Hermes may have user customizations.
- Discover proxy models and enrich reliable matches with models.dev metadata,
  including context limits. A model inventory is not proof of its capabilities.
- Manage per-model client context budgets with shared presets for easier switching.
  The operator commonly caps million-token models at 500,000 and also uses budgets
  around 272,000 and 353,000. Verify model-specific limits instead of treating these
  recollections as authoritative provider specifications.
- TUI controls for the service, client selection/installation and current status.
- Optional utilities mentioned: herdr, tmux, t3, orca and paseo. Research their exact
  identities, supported platforms and installation contracts before choosing them.
- Offer Windows OpenSSH server setup in both the MVP and full app: install the
  capability, manage its service, configure explicit firewall scope, and set up
  users/public-key authentication. The MVP uses local-subnet reach only, defaults
  to the Private network profile, and can explicitly allow Public. Warn when the
  current network is Public without changing it automatically. Broader scope may
  support sibling LAN subnets, Tailscale and WireGuard networks later.
- Potential maintenance: client updates and reacting to proxy configuration/model
  changes. Whether this needs a daemon remains a design question.
- User-triggered repair: use a functioning coding agent to diagnose and fix another
  client. Automatic operation may become opt-in later; do not enable it in the MVP.

## Next conversation

Research existing projects before implementing the broader utility. Use those
findings to inform a grilling/wayfinding session about scope, reuse, service
responsibilities, update policy, context management and repair authority.
