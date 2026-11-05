# Power loss, for real

*The crash simulator (decision 61) models what a power cut does to unsynced state. This note
records runs against a real machine crashed without syncing, the closest software gets to a
power cut: `echo b > /proc/sysrq-trigger` reboots the kernel at once, discarding every dirty
page and every unjournalled directory operation. The program is `hack/powertest`.*

## Method

A writer process appends records in rounds: twenty records, `Flush`, then an acknowledgement
line appended to a separate file and fsynced, then seven more records left unflushed. Segments
are 1 MiB, so a rotation (which fsyncs the directory) happens every few thousand records
while ordinary flushes leave the marker's rename unjournalled. After some time the machine is
crashed. After the reboot, `verify` reads the acknowledgement file, opens the log read-only to
see what the marker said at the crash, then opens it writable (recovery), scans every record,
and runs `Check`. It fails if `Open` fails, if the sequence is not contiguous from the first
record, if any record's payload is not the one written for its sequence, if there is a hole,
if any acknowledged sequence is missing, or if `Check` is unhealthy.

Machine: a Debian 13 server (Linux 6.12.113, 4 vCPUs, 8 GiB), ext4 with default mount
options (`data=ordered`, a journal commit every 5 s) on a virtual disk; whether the disk's
own cache honours `fsync` is not known, which is the usual position for a VM.

## Runs, 2026-10-28

| Cycle | Ran for | Acknowledged | Marker at the crash | Recovered | Result |
|---|---|---|---|---|---|
| 1 | about 40 min | 44,484,329 (2.5 GB, 2,531 segments) | 44,484,329, stale by 27 records | 44,484,356 contiguous, no holes, `Check` healthy in 60 s | OK |
| 2 | about 30 min | 33,861,341 (1.9 GB, 1,923 segments) | 33,861,341, fresh | 33,861,341 contiguous, no holes, `Check` healthy in 44 s | OK |
| 3 | 90 s | 1,758,989 (97 segments) | 1,759,016, fresh | 1,759,016 contiguous, no holes, `Check` healthy | OK |
| 4 (2026-11-05, the `v0.2.0` tag source; see "Re-run at the tag" in [live-integration.md](../passes/live-integration.md)) | 90 s | 1,742,195 (96 segments) | 1,742,195, stale by 27 records | 1,742,222 contiguous, no holes, `Check` healthy | OK |

Cycle 3 shows the opposite order: the crash lost the last acknowledgement line (written and
fsynced *after* the flush) while the flush's marker survived, so the marker was 27 records
ahead of the last acknowledgement; every acknowledged record was present, as required, and
the extra 27 were durable records the writer had not yet acknowledged.

Cycle 1 is the case decision 39 exists for: the last flush's marker rename was lost with the
crash while its fsynced data survived, so the marker was 27 records behind the durable end of
the segment. Parse-forward recovery took those 27 records (never acknowledged, so their
absence would also have been correct) and republished the marker. Every acknowledged record
was present.

## What this shows and does not show

- It shows that on a journalling filesystem with ordered data, an immediate kernel reboot
  loses exactly the kind of state the simulator models (an unjournalled rename) and nothing
  the model does not cover, in these runs.
- It does not show what the disk's write cache does on a real power cut: a VM's virtual disk
  may acknowledge `fsync` before the bytes reach anything persistent. That remains in
  `untestable-without-x.md`.
- It does not show behaviour under a filesystem with `data=writeback`, where a rename can
  reach the journal before the data it names; quill's atomic writes fsync the data first,
  which is the defence, but the run did not exercise that mode.
