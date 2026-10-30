# Security

Quill is an embeddable append-only log: a library and a command-line tool that read and write
files in one directory. It opens no sockets and runs no service.

## Reporting

Report a vulnerability privately, through the repository's Security tab: choose "Report a
vulnerability" (GitHub private vulnerability reporting), rather than opening a public issue. If that
option is not shown, the repository has not enabled it; open an issue that asks for a private contact
and says nothing about the problem itself. Expect an acknowledgement within a week.

Only the latest release is supported; quill versions are `v0.y.z`, and a fix ships as a new release.

## What quill defends against

- **Hostile or damaged files on disk.** A declared frame length over 64 MiB, a segment file over
  1.5 GiB, an oversize payload or batch is refused before it is allocated. Every state file the log
  depends on (`quill.tail`, `quill.format`, consumer files) is parsed strictly; unknown shapes are
  refused, not guessed. `quill.policy` is informational and parsed tolerantly: unknown fields are
  ignored, and a file that does not parse never blocks `Open`.
- **Symlink substitution.** Every file in a log directory is opened without following symbolic
  links, and segments are accepted only as regular files.
- **Silent corruption.** Every frame carries a CRC; damage is reported as a gap and never served.

## What it does not defend against

- A process with write access to the log directory. File permissions (0700 directories, 0600
  files by default) are the boundary.
- Encryption at rest; payloads are stored as given.
- Network filesystems, where locks and `fsync` may not mean what quill assumes.

The on-disk format is specified in `docs/format.md`; the threat model is refined by the audit
sequence described in `docs/design/decisions.md`.
