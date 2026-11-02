# Protected configuration transactions

`New(root)`, `Read(path)`, `ReadUncreated(path)`, `Apply(ctx, writes)` and
`Recover(ctx)` implement a
bounded local-filesystem transaction engine. Use one stable private account state
root across all application invocations. Never locate that root in a configuration
file's replacement path. Target parent directories must already exist.

`Read` rejects symlinks in any component, nonregular files, special permission
bits and snapshots over 5 MiB. `Before` and the initially unchanged `Data` are
independent copies. `ReadUncreated` additionally allows missing ancestor
directories without creating them. It verifies every existing ancestor: on POSIX
these must belong to the current account or root and lack group/other write bits;
on Windows every existing component is checked through a non-following native
handle for reparse points. Existing targets still use the strict Read rules.

`Apply` refuses duplicate normalized paths and at most accepts
128 writes, each with both input and output bounded to 5 MiB. Output is supplied
by the caller; native schema validation and preservation of foreign syntax belong
to that caller. No-op destinations retain their inode and modification time. All requested writes,
including no-ops, remain locked and checked after staging, before each replacement
and before success. Only committed own outputs advance their expected snapshots.

`PrivateDir(path)` creates missing directories as 0700 on POSIX (a protected
restricted DACL on Windows) or verifies existing private directories. It refuses to adopt or chmod insecure existing directories.
`PrivateFile(path, data)` uses the same rules for private parents and creates a
bounded 0600 file exclusively (the same restricted file DACL on Windows). Existing files, including empty files and symlinks,
are never overwritten. A filesystem failure can leave an incomplete new private
file; the helper returns an error and does not promise transaction semantics.

Apply holds a stable `coordinator.lock` in the state root and a stable
`.ccc-configtxn.lock` in each affected target parent, ordered by path. Lockfiles
are never removed. These parent locks also protect first creation, including
cooperating invocations configured with different coordinator roots. Existing
lockfiles must be owned private regular files with a single hard link.

Each operation has a private `txn-<id>` directory, durable JSON intent without
config content, and exact private `<index>.before` backups for existing targets.
Same-directory stages start private. Their final POSIX mode is synced before
replacement. Stages and originals are rechecked after hooks and immediately before
replacement; the target's parent directory is synced after rename. Backups remain
available after success. Supported POSIX mode bits, owner and effective group survive replacement and
rollback. POSIX mutation refuses foreign owner/group, special permission bits,
any xattrs (including access/default ACLs), and unsupported native metadata on
both target and direct parent. Unsupported existing metadata is refused before
creating state/locks/stages. Private helpers likewise refuse unsupported parent
metadata before creating a child; they never inherit an ACL and then chmod its
mask into broader access. Metadata is checked again at replacement/recovery, so
foreign ACL/group changes become Conflict without overwriting them. Inspection
Read itself remains read-only and can report a snapshot later refused by Apply.

On macOS, listxattr alone is insufficient: an fd-based native fgetattrlist query
checks ATTR_CMN_EXTENDED_SECURITY, accepts only absent ACL or explicit NOACL, and
refuses empty/meaningful/ambiguous ACLs, errors and native flags. Parser fixtures
run here; the native probe/test is cross-compiled and still needs macOS execution.
The layout follows Apple's xnu bsd/sys/kauth.h and bsd/vfs/vfs_attrlist.c. No custom
ACL/xattr migration or silent metadata dropping is implemented.

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
- `sharing-retry:i` (Windows only, after a sharing failure)
- `succeeded`
- `rollback-intent`, `rollback-staged:i`, `rollback-permissions:i`
- `before-rollback:i`, `rollback-sharing-retry:i` (Windows only), `rollback-replaced:i`, `restored:i`, `rolled-back`

Linux has native unit/race verification. macOS shares the POSIX implementation
and has compilation checks only. Windows uses x/sys/windows with a deliberately
narrow native policy, compiled here with platform-specific tests for parent CI:

- New directories/files, journals, backups and stable locks are created with the
  current account as owner, its token primary group, and a protected DACL granting full access only to that
  account, SYSTEM and Administrators. Directories have object/container inheritance
  flags; files have no inheritance flags. Existing broad or custom ACLs are refused,
  never silently narrowed. The accepted owner/group/DACL policy is regenerated exactly on replace;
  no new field in the public Write or journal is necessary for that fixed policy.
- Windows transaction target parents must already satisfy this private directory
  policy. System ancestors are checked for reparses, without requiring a private
  DACL on system roots such as C:\Users. Existing ordinary inherited configurations
  require explicit future migration; the engine does not adopt them implicitly.
- Every existing path component is opened with OPEN_REPARSE_POINT, and inspected
  for all reparse attributes, including junctions/cloud links. Opens used for data
  reads additionally verify owner, protected DACL, file type, attributes, link count
  and streams on that very handle. Paths must be local drive-qualified paths;
  UNC/device namespaces, ADS syntax, DOS device names and trailing-dot/space aliases
  are refused. Duplicate/reserved paths and parent locks use Windows case folding.
- Snapshot Mode is 0600 as an API marker for the accepted native security policy;
  POSIX chmod bits are not used as Windows security evidence. Custom ACLs, readonly,
  compressed, encrypted, sparse, hidden and other nonordinary attributes, extra
  hard links, named streams and unsupported stream queries fail closed. The ordinary
  data stream is checked through NtQueryInformationFile on the open file handle.
- File FlushFileBuffers and same-volume write-through MoveFileEx replacements are
  used. Sharing/lock errors receive at most eight attempts separated by 20 ms, with
  destination drift and cancellation checked on every attempt. Every attempt also
  checks the complete apply precondition set and revalidates source bytes against
  the caller's immutable reviewed hash (journal output hash, rollback input hash,
  or generated journal bytes), source mode, owner/group/DACL, attributes, link
  count and streams. A tampered source is never accepted as a fresh baseline.
  Windows stages require the ordinary ARCHIVE attribute and no additional flags.
  Tests exercise the bounded algorithm with injected syscalls and add native
  held-sharing/tamper cases for bytes, ACLs, attributes, links, streams and rollback.
  There is no fallback
  to truncate-in-place. Windows directory sync verifies the handle; it cannot offer
  an unprivileged POSIX directory-fsync guarantee. Power-loss persistence is unproven.
- This slice preserves the supported owner/DACL policy, not arbitrary SACL/audit
  metadata, extended attributes or timestamps. It targets ordinary private NTFS
  configuration files; native CI must validate the supported contract before any
  Windows release claim. Other unsupported OS targets still return ErrUnsupported.

Windows descriptor/path policy tests run under an isolated Wine prefix here.
Wine's filesystem translation drops SE_DACL_PROTECTED and changes ACEs, so native
filesystem tests intentionally fail shut there. Wine and cross-compilation do not
establish native Windows ACL, junction, sharing or recovery behavior.

Use local trusted filesystems only. Advisory locks cannot exclude foreign
editors. Hash check plus rename is not a universal filesystem compare-and-swap;
a foreign writer or directory renamer can still race the final system calls.
Close active config editors before applying. This engine does not defend against
arbitrary malicious code running as the same account. Multi-file operations are
recoverable, not simultaneously atomic. These tests do not establish release
readiness, power-loss durability, or native Windows/macOS runtime behavior.
