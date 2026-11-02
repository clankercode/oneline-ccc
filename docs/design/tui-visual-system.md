# TUI navigation and color pass

Status: approved visual direction and browser specimen. A native Go development
build now exists, but the operator rejected its visual fidelity and interaction
quality on 2026-11-03. Native acceptance is reopened; the
[P0 remediation backlog](../plans/remaining-items-so-far.md#p0-native-tui-visual-quality-and-interaction-remediation)
requires comparisons of actual native renders against this reference across
every screen and overlay, with visual/source review and fix loops.
The [screen flows](tui-screen-flows.md) remain the behavior and layout plan.
The [interactive specimen](../../site/tui-preview.html) uses illustrative data.

The first goal is easier navigation. A little color and emphasis should let an
operator distinguish the current section, selected client, status and next
action without reading every line. This is an incremental pass, not a new app
architecture or a claim of production readiness.

## Visual hierarchy

Use the same monospace text size throughout the terminal. Hierarchy comes from
weight, spacing and background bands rather than oversized headings or icons.

| Element | Treatment | Meaning without color |
| --- | --- | --- |
| Application/scope bar | Bold name, contrasting shallow background band | Name and current scope remain explicit |
| Current section | Accent text and a filled navigation row | Selected/current label and bold text |
| Section heading | Bold text, one blank line above, optional rule below | Weight and spacing |
| Selected client/model | Subtle full-row fill | Persistent `Selected:` text or selected-row state |
| Keyboard focus | Strong outline/reverse emphasis, distinct from selection | One `>` marker where a terminal action has focus |
| Main value | Normal foreground; model IDs and budgets align | Stable label/value columns |
| Supporting evidence | Muted foreground, source and age in words | `Local`, `Unverified`, `Stale`, timestamps |
| Status | Restrained semantic color, always accompanied by text | `Configured`, `Needs setup`, `Drift`, `Failed` |
| Next action/footer | Bold action name, compact context-specific keys | Action label and literal keyboard hints |

Texture means a small number of flat terminal background layers, separator
rhythm and table bands. It does not require grain, gradients, shadows, blinking,
font downloads, emoji or a box around every field. Keep the screen dense enough
to compare clients while leaving space between unrelated groups.

## Color roles

These extend the existing screen-flow palette. They describe explicit dark and
light themes; the eventual native automatic mode respects terminal defaults.

| Role | Dark | Light |
| --- | --- | --- |
| Surface / primary text | `#111820` / `#E6EAF0` | `#F5F6F8` / `#17212B` |
| Secondary text | `#B3BDC9` | `#465463` |
| Navigation/action accent | `#91C7FF` | `#164F91` |
| Configured/completed | `#91D6A5` | `#23613A` |
| Attention/drift/stale | `#F0CC81` | `#704700` |
| Failed/blocked | `#F5A3A3` | `#9B2727` |

Do not use green as a general promise that a client or gateway works. Configured,
authenticated, available and inference-verified remain different facts. Color
supports those labels. It never supplies evidence or hides a required action.
Tier availability and context capacity still need the evidence described in the
production plan; selected budgets and fixture model names prove neither.

Show the selected default request tier beside the exact model ID, for example
`gpt-6.1-sol · Standard`, `gpt-6.1-sol · Fast` or
`gpt-6.1-sol · Ultrafast`. This is a display label, not a model-ID suffix.
The separate **Tiers** row describes available choices, such as **Up to
Ultrafast**, independently of the default. The model table uses the same split:
**Model / default** and **Available tiers**. Unknown eligibility stays **Not
verified**. Keep evidence/provenance detail out of the tier value itself.

Monochrome keeps the bands, bold headings, labels and focus/selection distinction.
ANSI-16 and 256-color mappings, `NO_COLOR` and native-terminal contrast remain
implementation checks. A browser rendering cannot establish those behaviors.

## Specimen and implementation handoff

The browser specimen lets the operator navigate representative screens, select
items and compare layouts. Its actions do not inspect or change local client
settings, install anything, contact a gateway or control services. Browser
controls and size presets are presentation aids, not implemented Go options or
proof of exact terminal-cell dimensions.

The [usability follow-up](tui-usability.md) adds draft management checkboxes,
per-client settings, exact review, simulated receipts, `?` help and `Ctrl+P`
command search. It preserves this visual contract and its native-app boundary.

Carry this pass into T1/T4 of the [production plan](../plans/tui-production-plan.md):

1. Preserve this hierarchy in wide list/detail and compact single-pane layouts.
2. Keep current section, persistent selection and keyboard focus distinct.
3. Keep labels, actions and focus usable when color is unavailable.
4. Preserve a selected item's identity and draft on resize/navigation.
5. Review real terminal captures for clipping, contrast and focus after the
   renderer exists. Native PTY and broader release gates remain unchanged.

This pass needs a useful, rendered preview and a focused independent review.
It does not require implementing the daemon, adapters or the complete theme
system before the operator can judge whether the screen is easier to read.
