# Pinned terminal ownership correction

The root `go.mod` replaces Bubble Tea v2.1.1 and Ultraviolet
v0.0.0-20261008173134-6b8d4baf91b4 with the complete, licensed Go module
archives in `bubbletea/` and `ultraviolet/`. No module-cache files are edited.
`upstream.json` pins their public commits, module/go.mod h1 checksums and archive
SHA-256 checksums. The modules' upstream license files are retained verbatim.

The complete source correction is in `terminal-input.patch`. It also includes
the three added dependency regression files; all other archive bytes are unchanged.
The unmodified package and platform sources are kept so builds need no patch
application step and upstream API/source comparisons remain straightforward.

## Why a local correction is needed

Checked upstream on 2026-11-03: Tea `main` resolves to the pinned v2.1.1;
Ultraviolet `main` resolves to a45b7c86333f but has identical terminal reader
implementations. Neither supplies this correction. Saved lookup and source
comparison evidence is in `.superpowers/sdd/product-completion/A1-fix1-*`.

Upstream Ultraviolet sends decoded events without observing cancellation. Tea
waits at most 500 ms on graceful shutdown/release and skips that wait on forced
shutdown. Pending input can therefore strand its decoder; closing the raw reader
before its read finishes also causes the reproduced cancelreader descriptor race.

The correction makes event dispatch observe its input-loop context. Tea owns a
separate cancellation scope for each input loop, so `ReleaseTerminal` also stops
delivery while the main program context stays live. Release and shutdown cancel
delivery, cancel the underlying read, join the decoder (which joins its raw read
goroutine), and then close its reader. There is no timeout fallback. Release also
closes the joined reader before `RestoreTerminal` replaces it. Forced shutdown
uses the same join, allowing the application to pass its actual root context
directly to Tea again.

The A6 extension gives `Run` a deferred ownership boundary, including partial
terminal setup, initial-size errors, input-reader setup errors and panics.
Acquisition, release, restoration and forced shutdown share a lock; Kill cancels
before waiting for that lock. A renderer is stopped only after it has started,
and its flush loop is joined before restoration or restart. Successfully
restored mode snapshots are consumed once. Run also closes only the TTY handles
it opened itself, after input completion and mode restoration. Startup
cancellation skips further model initialization/rendering, and a simultaneous
root cancellation is joined with the original failure. RestoreTerminal rolls
back failed reacquisition and refuses to reacquire after shutdown.

Linux PTY regressions exercise size and reader-setup errors after raw mode,
concurrent cancellation, partial-setup/model-init panics, runtime read errors
and graceful rendering. An additional test-only overlay returns a setup error
after real MakeRaw, with and without concurrent cancellation; its reproduction
is retained in the A6 report/workspace. Dependency barriers cover Kill during initialization,
an unstarted renderer, pre-acquisition cancellation and panic join ordering.
The invalid-descriptor constructor probes run in reaped fixture processes:
pinned cancelreader's EpollCtl error branch leaves its private wake-pipe files
for finalization/process exit. That inherited constructor allocation leak is
outside this Tea/Ultraviolet ownership correction; no successful reader or
decoder is abandoned. Native macOS/Windows startup failure injection and console
restoration remain unproved by Linux runs or cross-compilation.

This preserves API signatures and all native platform implementations. Native
cancelable terminal readers remain supported. An arbitrary custom `io.Reader`
that cannot interrupt its blocking `Read` must be unblocked by its owner before
shutdown/release can finish; the implementation deliberately waits for actual
completion instead of abandoning that read. A nil input starts no reader.

The upstream mouse-renderer race fixture used such an `io.Pipe` but closed its
writer only after waiting for Quit to return. Its input owner now closes that
writer immediately before Quit. All mouse events, renderer assertions and
timeouts are retained; the fixture no longer depends on an abandoned read.

## Verify and maintain

From the repository root, with the pinned public archives already downloaded:

```sh
python3 third_party/verify.py /path/to/go/pkg/mod/cache/download
```

The verifier is offline and read-only. It checks the archive checksums, Go h1
hashes, matching root requirements/replacements and every mirrored file, including
additions/deletions. It reconstructs the exact correction and compares it to the
committed patch. The supplied cache must contain only the ordinary public module
archives; the verifier does not read client configuration or credentials.

To update, inspect upstream first. If it contains the equivalent dispatch and
join fixes, remove these replacements/mirrors and rerun the application and
dependency lifetime regressions. Otherwise import the complete new public
archives, update `upstream.json` from their verified sums, rebase the small source
correction and its regressions, and run the verifier with `--write-patch` to
produce the diff for review. Then rerun it without that flag. Do not regenerate
the patch to bless unexplained edits. Keep the root requirements and provenance
manifest aligned.

Dependency tests can use a disposable Go workspace listing these two module
directories, so their existing test dependencies and checksums stay in their
own modules. Run both packages' tests and race tests, the foreground/process-owner
tests and vet, the real rendered pending-input and raw-child SIGTERM regressions,
and all six static OS/architecture builds. Compilation alone is not native
macOS/Windows console proof.
