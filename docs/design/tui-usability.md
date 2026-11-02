# TUI usability contract

Baseline: 2026-11-02. This interaction pass keeps the approved
[visual system](tui-visual-system.md). Try the
[browser specimen](https://clankercode.github.io/oneline-ccc/tui-preview.html).
It is an in-memory simulation, not a native Go app. Reload resets the specimen.

## A small complete path

1. Select a client to inspect its applied preview values. Selection does not
   enable management.
2. Check **Manage** to stage enablement. Installed, management-enabled and
   selected are distinct states. Missing or unsupported adapters explain why
   management is unavailable; inspection and Back remain available.
3. Open **Configure**, keep the suggested defaults or choose an exact model ID,
   context budget and default request tier. Custom budgets live in Advanced.
4. **Review [client] draft** shows that client's actual before/after values.
   **Review all drafts** collects the pending clients. Back keeps the drafts.
5. **Simulate apply** updates only the reviewed preview clients and produces a
   receipt naming those clients and changed fields. The dashboard then shows
   those applied values. No-change review has an empty state and disabled Apply.

There is no simulated native update/install/authentication success. Pi stays
Not installed; its installation action is unavailable. Grok Build and Hermes
remain inspectable with management disabled. Actual install, launch, discovery,
configuration, updates and monitor control belong to the native build tasks.

## Drafts and review

- Each client owns a separate safe draft. Navigation, filter, appearance,
  layout, help, palette and illustrative refresh preserve it.
- Dashboard values are applied values; staged enablement and unsaved changes
  have explicit text. Checkboxes reflect draft management preference.
- Disabling management preserves installed software, applied configuration and
  draft settings. Re-enabling restores that draft for review. It does not
  uninstall, erase configuration or disable authentication.
- Discard is explicit and scoped to one client. The dialog initially focuses
  Back; Escape keeps the draft. Other clients' drafts remain intact.
- Apply validates the complete review before any mutation. A bad budget or tier
  blocks the batch. Apply is bound to the rendered reviewed snapshot; a stale
  snapshot is rejected before mutation.
- Preset budgets are 500000, 353000 and 272000 tokens. Custom budgets must be
  positive whole numbers. These illustrate selected budgets, not verified model
  capacity. Actual capacity needs evidence in T5.
- Default request tier is separate from availability. For example, the selected
  model can read `gpt-6.1-sol · Standard` while availability reads **Up to
  Ultrafast**. The stored model ID never includes a tier suffix. Changing to a
  model without the chosen accelerated tier resets the default to Standard and
  announces why. Unknown tier capability is not inferred from a model name.

## Keyboard and focus

| Input | Browser specimen behavior |
| --- | --- |
| `?` | Open shortcut help |
| `Ctrl+P` | Search commands while focus is inside the console |
| `/` | Open/focus the client filter |
| `r` | Refresh the illustrative snapshot; keep drafts |
| Up / Down on a client row | Select the previous / next matching client |
| Space on a client row or checkbox | Stage management on/off once |
| Enter on a client row | Open that client's settings |
| Enter on another control | Activate the focused control |
| Escape | Close the top panel or return one screen, keeping drafts |
| Tab / Shift+Tab | Reach every available control in reading order |

Typing, native selects and composition keep normal keyboard behavior. Ordinary
console shortcuts do not run from an input, textarea, select or editable field.
Ctrl+P outside the console retains browser Print. Help and Commands also have
visible buttons, so intercepted shortcuts do not strand the user.

The palette supports search, zero results, disabled-action reasons, Up/Down and
Enter. It can stage choices or navigate to Review; it cannot Apply. Help and
palette labels come from the same command catalog as visible actions.

Only one modal is open at a time. Modal focus stays inside and returns to its
invoking control on dismissal; background clients cannot toggle behind it.
Review initially focuses Back. Returning to settings preserves the safe draft.
Filters use stable client IDs, and zero results offers Clear filter. Narrow
layout keeps the selected row plus Previous/Next traversal of matching clients.
Monochrome retains labels, selection borders, focus outlines and checkbox state.

## Native carry-forward and evidence

T1 must implement this help/palette/focus contract using native terminal input;
T4 must implement selection, enablement, safe drafts and complete review/receipt
paths against real application commands. Browser behavior does not satisfy the
native PTY, credential, transaction or adapter gates. Native quit/cancellation
semantics remain in the [production plan](../plans/tui-production-plan.md).

State regressions: `node --test tests/tui-preview-state.test.mjs`.
Browser journeys: `tests/tui-preview-browser.mjs`, using Playwright, with
`TUI_PREVIEW_URL` pointing at a locally served `site/` and optionally
`PLAYWRIGHT_MODULE` pointing at the installed Playwright module. The harness
checks state transitions, typing guards, overlay focus and responsive themes;
it does not call a gateway or native installer.
The separate `tests/tui-preview-ime.mjs` regressions use Chromium composition
events to preserve exact custom model/filter text and prevent candidate-selection
keys from activating commands. The main browser harness includes these checks.
