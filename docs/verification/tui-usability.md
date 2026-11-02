# TUI preview usability verification

Local baseline: 2026-11-02. Application snapshot: `8e6fb58`.
Scope: the browser design specimen, not a native Go release.

## Delivered behavior

Client selection and draft management enablement are independent. Per-client
model, context budget and default tier lead to an exact review, explicit simulated
apply and matching receipt. Back/navigation/refresh preserve safe drafts; discard
is scoped and explicit. Missing or unsupported adapters remain inspectable with
explained unavailable actions. Help and searchable commands use the same catalog.

## Observed gates

| Gate | Result |
| --- | --- |
| `node --test tests/tui-preview-state.test.mjs` | 9 passed |
| `tests/tui-preview-browser.mjs` with Playwright 1.61.1 / Chromium | 30 passed, 0 failed |
| Module syntax and diff whitespace | Passed |
| Fresh independent state review and fixes | Passed scoped re-review |
| Fresh whole-branch review and composition/focus fixes | All accepted findings resolved in scoped re-reviews |

Browser checks cover selection/enablement, default setup, independent drafts,
client-scoped review/apply, exact receipts, tier capability versus default,
invalid/custom budgets, discard/cancel, filter zero/reset, refresh/focus, palette
search/navigation/disabled actions, modal Tab/Escape/background isolation and
normal typing. Nine viewport/theme cases exercise 1440, 620 and 390 pixels,
dark/light/monochrome, and all three explicit layout modes through each screen.
No document overflow, page errors or external requests occurred in the final run.

Composition regressions first failed against the prior implementation. Chromium
CDP then verified exact committed model/filter text and protected candidate keys.
Three additional regressions first failed when an active composition consumed the
first pointer click on Review/Back/Manage; all now pass, including focus return.
The reviewer independently repeated those three pointer transitions.

Rendered dashboard, palette, review, narrow editor and narrow light-theme help
were inspected. The approved visual tokens and selection/focus hierarchy remain.
Initial failures and intermediate experiments were excluded from passing results.

## Publication and native boundary

The Pages workflow includes state, syntax and browser journeys as prerequisites
alongside the existing Windows installer/SSH checks. Hosted runner results and
public asset bytes must also be checked during publication; the local results
above do not establish deployment by themselves.

The specimen uses memory only and resets on reload. No credential, endpoint,
client file, installation, service or real model request is accessed. Browser
tests do not satisfy native terminal, adapter, credential or transaction gates.
The [usability contract](../design/tui-usability.md) carries these paths into
T1/T4 of the [native plan](../plans/tui-production-plan.md).
