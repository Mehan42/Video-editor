# Media Sandbox

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

ffmpeg and Whisper execute as separate bounded workers. Workers receive only the approved input path and a dedicated temporary/output root.

Required target controls:

- network disabled;
- no access to user profile, secret stores, or unrelated workspace paths;
- CPU, memory, duration, file-size, and child-process limits;
- explicit executable/model hashes;
- timeout and cancellation;
- no shell interpretation of media metadata;
- cleanup and retention policy for temporary files.

Implemented in the first MVP slice: separate subprocess invocation, no shell construction, no stdin, bounded input size, context timeout, dedicated `0700` run directory, and no network flags or network client.

Still required before treating media processing as fully sandboxed: OS-level job/container boundary, explicit child-process/resource limits, secret-store exclusion, executable/model allowlist and hash verification, and retention enforcement.

The Harness safety configuration does not replace runtime worker isolation.
