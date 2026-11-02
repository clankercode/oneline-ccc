# Protected configuration transactions

`New(root)`, `Read(path)`, `Apply(ctx, writes)` and `Recover(ctx)` implement a
bounded local-filesystem transaction engine. Use one stable private account state
root across all application invocations. Never locate that root in a configuration
file's replacement path. Target parent directories must already exist.

`Read` rejects symlinks in any component, nonregular files, special permission
bits and snapshots over 5 MiB. `Before` and the initially unchanged `Data` are
independent copies. `Apply` refuses duplicate normalized paths and at most accepts
128 writes, each with both input and output bounded to 5 MiB. Output is supplied
by the caller; native schema validation and preservation of foreign syntax belong
to that caller. No-op destinations retain their inode and modification time.

`PrivateDir(path)` creates missing directories as 0700 or verifies existing
owner-only directories. It refuses to adopt or chmod insecure existing directories.
`PrivateFile(path, data)` uses the same rules for private parents and creates a
bounded 0600 file exclusively. Existing files, including empty files and symlinks,
are never overwritten. A filesystem failure can leave an incomplete new private
file; the helper returns an error and does not promise transaction semantics.

Apply holds a stable `coordinator.lock` in the state root and a stable
`.ccc-configtxn.lock` in each affected target parent, ordered by path. Lockfiles
are never removed. These parent locks also protect first creation, including
cooperating invocations configured with different coordinator roots. Existing
lockfiles must be owned private regular files with a single hard link.

Each operation has a private `txn-<id>` directory, durable JSON intent without
config content, and exact private `<index>.before` backups for existing targets.
Same-directory stages start at 0600. Their final POSIX mode is synced before
replacement. Stages and originals are rechecked after hooks and immediately before
replacement; the target's parent directory is synced after rename. Backups remain
available after success. Mode bits survive replacement and rollback; ownership,
ACLs, extended attributes and filesystem-specific metadata are not copied.

Return outcomes are `Noop`, `Succeeded`, `RolledBack`, `Conflict`, `Cancelled`,
`NeedsAttention`, or `Unknown`; inspect the accompanying error as well. Only
operation IDs, target paths and backup paths appear in receipts. Exported error
sentinels contain no filesystem error strings, hook error strings or content.
A checkpoint error models an interruption and returns `Unknown`/`ErrInterrupted`,
leaving journal recovery to a fresh Store. Ordinary precommit failures attempt
rollback while locks remain held. Cancellation before any replacement leaves
all target files unchanged; cancellation after partial replacement attempts safe
rollback. A cancellation after the final commit may complete successfully.

`Recover` runs before new mutations, returns recorded terminal receipts as well
as recovered operations, and rolls unfinished operations back in reverse order.
It only restores/removes a target matching its own recorded output hash and mode,
or recognizes an already restored original. Foreign bytes, deletion, permission
changes, unsafe parents, or tampered stages produce `Conflict` and preserve the
foreign state. Unsafe parent locks conservatively leave the entire operation for
reconciliation. Corrupt journals/backups produce an error without blindly
restoring. A recorded Conflict blocks later Apply; reconciliation/retention UI is
outside this API. Journals/backups are retained and must be treated as sensitive.

The unexported `Store.checkpoint` fault hook uses these logical durable boundaries
(`i` is the zero-based changed-file index):

- `journal-created`
- `backup:i`, `stage:i`, `ready:i`
- `before-replace:i`, `stage-permissions:i`, `replaced:i`, `applied:i`
- `succeeded`
- `rollback-intent`, `rollback-staged:i`, `rollback-permissions:i`
- `before-rollback:i`, `rollback-replaced:i`, `restored:i`, `rolled-back`

Linux has native unit/race verification. macOS shares the POSIX implementation
and has a compilation check only. Windows and other platforms return
`ErrUnsupported` before creating state, locks, stages or targets. Windows
owner-only ACL, reparse and sharing behavior are not implemented or claimed.

Use local trusted POSIX filesystems only. Advisory locks cannot exclude foreign
editors. Hash check plus rename is not a universal filesystem compare-and-swap;
a foreign writer or directory renamer can still race the final system calls.
Close active config editors before applying. This engine does not defend against
arbitrary malicious code running as the same account. Multi-file operations are
recoverable, not simultaneously atomic. These tests do not establish release
readiness, power-loss durability, or native Windows/macOS runtime behavior.
