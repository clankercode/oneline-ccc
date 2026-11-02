# Setup page design and verification

Mode: Docs. Product: a single Windows Codex setup command for users bringing their
own gateway. The page must expose the command, explain the prompts and establish
what is installed and stored. `README.md` and `scripts/setup-codex.ps1` are the
behavioral sources of truth. No deployment URL or credential is public content.

Visual thesis: a compact field guide organized around one copyable command.
Character: practical, legible, quiet, direct. The numbered setup sequence is the
signature; the command is the primary action. A sidebar would add navigation
overhead to one page; a marketing hero would displace the actual setup. Use a
single column with two explanatory columns only on wider screens.

System typography avoids downloads. Muted green carries links and focus; neutral
paper and dark forest themes follow the operating system. Semantic CSS variables
define each theme independently. No animation or dependencies. Code scrolls
horizontally rather than changing the command. The page has stable section IDs,
keyboard focus, a skip link and an announced copy result. Browser find suffices
for this single-page guide; no search system or mobile menu is needed.

State matrix: command visible without JavaScript; copy success announced; blocked
clipboard offers manual copying; narrow layouts keep command independently
scrollable; light/dark theme and keyboard focus remain legible.

Validation completed before publication: Chromium desktop 1440x1000 light and
mobile 390x844 dark screenshots were rendered and visually inspected. Browser
checks in both themes verified no document overflow, exact copied command text,
announced copy success and manual-copy fallback on denied clipboard access. No
page exceptions occurred. Screenshots are local ignored cache artifacts.
No endpoint calls occur from the site.
