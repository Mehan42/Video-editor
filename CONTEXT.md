# PageVideo — Context

**Status:** DEVELOPMENT PAUSED / NEXT SESSION PLANNED
**Development mode:** local-only
**Code changes allowed:** approved for MVP implementation

## Purpose

PageVideo is a local-first knowledge extraction pipeline: it accepts video, extracts audio, transcribes it, splits the transcript into traceable chunks, optionally retrieves context, and asks a configured LLM provider to produce learning artifacts.

## Current baseline

- `ffmpeg` binary is present under `ffmpeg/bin`.
- `whisper.cpp` CLI and a local model are present under `whisper.cpp`.
- Initial application Go code exists under `cmd/pagevideo` and `internal`.
- Local Git repository is initialized on branch `main`; no remote and no commit exist yet.
- CodeGraph is initialized; the latest verified status reports `Index is up to date`.
- Harness has a local ready session, with network and external execution disabled.
- LM Studio Bionic is installed at `E:\LM Studio Bionic\Bionic.exe`, version `1.0.5`; **verified 2026-08-06**: listener on `127.0.0.1:1234` owned by the Bionic process, `provider check` returns READY with 6 local models (`qwen/qwen3-vl-8b`, `llama-3.2-3b-instruct`, `qwen/qwen3.6-27b`, `google/gemma-4-12b-qat`, `prism-ml/bonsai-27b`, `text-embedding-nomic-embed-text-v1.5`). Chat egress is still blocked by default.

## Implemented MVP slice

- `pagevideo process --input FILE` accepts local MP4/MOV/MKV/AVI input.
- ffmpeg and Whisper are invoked with separate arguments through `exec.CommandContext`; no shell command is constructed.
- Input size and processing duration are bounded by CLI options.
- The run writes WAV, TXT, SRT, deterministic chunks, and an atomic JSON manifest below a `0700` run directory.
- Chunks include source hash, content hash, run ID, and `untrusted_source` trust class.
- Bionic provider adapter exists with loopback URL validation, `/v1/models` readiness, response-size limit, and chat egress blocked by default.
- Local launcher exists at `scripts/pagevideo-start.bat`; it builds the cached CLI only when missing and forwards arguments.

## Not yet implemented

- OS-level media sandbox beyond process separation, timeout, size limit, and network-free invocation.
- NATS transport, MCP connectors, provider registry, LLM Gateway, Retriever, and external egress policy enforcement.
- Obsidian/RAG export, downloader adapters, persistent cache, and agent orchestration.
- ~~Bionic runtime readiness~~ **Resolved 2026-08-06**: Bionic is READY on `127.0.0.1:1234`. Remaining not-implemented items stand as listed above.

## MVP boundary

The first implementation must support a local video file and produce a verified transcript plus a minimal knowledge artifact. The architecture must already expose boundaries for NATS, MCP, provider adapters, retrieval, evidence, security policy, and Human Gate.

The MVP does not include an autonomous agent, arbitrary tool execution, automatic publishing, unrestricted network access, or a mandatory Ollama dependency.

## Authority boundary

Video, subtitles, metadata, transcript text, retrieved chunks, LLM output, MCP responses, and external tools are untrusted data. None of them can change policy, select permissions, execute tools, read secrets, or publish artifacts.

Harness is a verification and evidence layer. It is not runtime authority. Human approval is required before moving from documentation to implementation and before enabling external egress or mutating actions.

## Required gates

1. Documentation reviewed and explicitly approved by the operator.
2. Local Git baseline recorded before the next source phase.
3. Security contracts and schemas pass review.
4. Isolated media-worker smoke test passes.
5. CodeGraph is initialized after application code appears and is refreshed after source changes.
6. External provider egress is separately enabled and verified.

## Pause state

Development is intentionally paused at the end of the first local media/transcription slice. No external provider was started and no data was sent to Bionic or any network endpoint.

The next session starts from `docs/NEXT_DAY_PLAN.md`. Do not enable external provider egress until Bionic health, capabilities, data classification, redaction, and Human Gate checks pass.
