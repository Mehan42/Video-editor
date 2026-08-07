# Implementation Status

**Status:** MVP CODE PRESENT / DEVELOPMENT PAUSED / SECURITY GATES PARTIAL

## Verified

- Local Go CLI builds with Go 1.25.3.
- `go test ./...` passes with repository-local Go caches.
- `go vet ./...` passes.
- End-to-end synthetic MP4 smoke passes without network: ffmpeg extraction, Whisper transcription, TXT/SRT output, deterministic chunk, and manifest.
- CLI uses explicit subprocess arguments and does not invoke a shell.
- Input extension, regular-file, size, dependency-path, timeout, and chunk parameter validation exists.

## Not verified or not implemented

- `go test -race ./...`: blocked because the environment has no GCC for cgo.
- `govulncheck`: not installed.
- CodeGraph: initialized; the latest verified status is `Index is up to date` with the built-in node:sqlite backend.
- LM Studio Bionic is locally installed at `E:\LM Studio Bionic\Bionic.exe`, version `1.0.5`.
- Bionic bundle statically exposes LM Studio identity, `127.0.0.1:1234`, `/v1/models`, and `/v1/chat/completions` candidates.
- `internal/llm` now provides a loopback-only Bionic adapter; `provider check` returns structured readiness and blocks chat unless explicitly enabled.
- `internal/llm` message roles are a closed set (`system|user|assistant`); untrusted content is sent only as `user` via `NewUserMessage`, and injected text cannot select a provider/model/endpoint (`TestChatRequestConfigIsolation`, `TestRejectsUnknownMessageRole`).
- `internal/llm` tests (httptest only, no network) cover: loopback validation, default chat block, malformed JSON, empty `data`, provider HTTP error, `MaxResponseBytes` limit, timeout, empty `choices`, malformed chat response.
- Live Bionic readiness verified 2026-08-06: listener on `127.0.0.1:1234` owned by `Bionic.exe`; `provider check` returned **READY** with models `qwen/qwen3-vl-8b`, `llama-3.2-3b-instruct`, `qwen/qwen3.6-27b`, `google/gemma-4-12b-qat`, `prism-ml/bonsai-27b`, `text-embedding-nomic-embed-text-v1.5` and capabilities `chat/completions`, `models`. The readiness probe sends no transcript data. Chat egress remains blocked by default.
- `scripts/pagevideo-start.bat` provides the supported Windows local launch path and does not start external services.
- OS-level sandbox and resource/job limits.
- NATS, MCP, LLM provider adapters, external egress, Retriever, and agent orchestration.

This status is evidence for the local MVP slice only. It is not a production-ready verdict.

## 2026-08-06 — Bionic READY + opt-in local summary generation

- Bionic listener on `127.0.0.1:1234` verified; `provider check` returns READY with 6 local models (capabilities: `chat/completions`, `models`).
- New opt-in flag `process --enable-summary` (plus `--llm-base-url`, `--llm-timeout`, `--llm-max-response-bytes`, `--summary-max-chars`): when enabled, the transcript is sent to the local Bionic chat endpoint and `summary.md` (Markdown, SHA-256-hashed) is written into the run directory and recorded in `manifest.json`.
- Summary is opt-in only; without `--enable-summary` no network/LLM activity occurs. Callers of `pipeline.Run` never crash on a summary failure — it degrades to READY without summary (failure is logged). Config validation requires `--llm-base-url` when summary is enabled.
- "untrusted content" boundary is enforced: system policy and task text are fixed in code, transcript goes only as a `user` message via `NewUserMessage`, the model ID is read from readiness (not from content), provider endpoint/capability are set in `Config` at construction. Tests: `TestSummarizeTranscript_PolicySeparatedRequest`, `_TooLarge`, `_ChatBlockedByDefault`, `_NoModelLoaded`, plus existing role/provider isolation tests.
- Live smoke run against real Bionic (input `smoke.mp4`, transcript "[музыка]") produced a policy-compliant `summary.md` and a matching manifest entry.

## 2026-08-07 — UX fixes: bare path/URL as input

- CLI: a bare first token that is an existing local file OR an `http(s)://` URL is accepted as the `--input` of `process` (`internal/cli.cli_test.go` covers both, plus the still-rejected "unknown command" path). Useful inside the interactive REPL of `scripts/pagevideo-start.bat`.
- `Config.Validate` now returns a clear, explicit error when `--input` is a URL: remote downloaders (YouTube/VK/RuTube/http) are not implemented. Local files remain the only supported input.
- REPL: leading whitespace typed before a command is trimmed; `version` and `provider check` verified working in both direct and interactive modes. A live full run against the user video `Свой ЛИЧНЫЙ VPN за 30 минут.mp4` (57 MB) completed: `transcript.txt` (38 KB, ~450 lines), `transcript.srt`, chunk manifest with hashes — status READY.
