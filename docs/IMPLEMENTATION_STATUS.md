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

## 2026-08-08 (T4) — URL input via yt-dlp behind --allow-download

- New package `internal/download`: `Fetch(ctx, ytdlpPath, ffmpegPath, rawURL, dstDir, maxBytes)` shells out to `yt-dlp` with `--no-call-home --no-playlist --no-warnings --quiet -f bv*+ba/b --ffmpeg-location <dir-of-pipeline-ffmpeg> --merge-output-format mp4 --no-part -o <dstDir>/download.%(ext)s`. Result file must match `download.*` exactly once, live under `dstDir` and respect `max-input-bytes`; oversized downloads are deleted before returning an error.
- Operator gate: URL input is rejected by `Config.Validate` unless `--allow-download` is passed **and** a yt-dlp binary is present (`--ytdlp`, default `tools/yt-dlp.exe`). Without the gate no network egress happens at all from the pipeline.
- `config.AbsolutePaths` no longer mangles URLs into bogus filesystem paths; URL input bypasses local-file validation (size/extension checks apply to the *downloaded* file inside `Fetch`).
- Pipeline flow: on URL input the run downloads into `OutputRoot/.staging/<unixnano>/`, replaces `cfg.Input` with the local staging path, then continues through the regular ffmpeg/whisper/chunk path. `run.SourceURL` records the original URL in `manifest.json`. **Cache is skipped for URL runs** (remote bytes may change between requests, an input-hash match does not imply identity of the remote resource); `--no-cache` has no additional effect.
- Live-verified 2026-08-08: `https://www.youtube.com/watch?v=jNQXAC9IVRw` ("Me at the zoo", 19s) → transcript "Alright, so here we are, one of the elephants..." produced under `.pagevideo/url-smoke/<run>/`, manifest carries `source_url`. yt-dlp merged DASH tracks into mp4 via bundled ffmpeg (`--ffmpeg-location`).
- Tests (`internal/download/download_test.go`, 6 cases, no network): scheme whitelist, missing binary, non-positive maxBytes, pre-cancelled context, directory-as-binary rejection, supported matrix.

## 2026-08-08 (T3) — study.md / faq.md / glossary.md behind `--enable-summary`

- New package `internal/download`: `Fetch(ctx, ytdlpPath, ffmpegPath, rawURL, dstDir, maxBytes)` shells out to `yt-dlp` with `--no-call-home --no-playlist --no-warnings --quiet -f bv*+ba/b --ffmpeg-location <dir-of-pipeline-ffmpeg> --merge-output-format mp4 --no-part -o <dstDir>/download.%(ext)s`. Result file must match `download.*` exactly once, live under `dstDir` and respect `max-input-bytes`; oversized downloads are deleted before returning an error.
- Operator gate: URL input is rejected by `Config.Validate` unless `--allow-download` is passed **and** a yt-dlp binary is present (`--ytdlp`, default `tools/yt-dlp.exe`). Without the gate no network egress happens at all from the pipeline.
- `config.AbsolutePaths` no longer mangles URLs into bogus filesystem paths; URL input bypasses local-file validation (size/extension checks apply to the *downloaded* file inside `Fetch`).
- Pipeline flow: on URL input the run downloads into `OutputRoot/.staging/<unixnano>/`, replaces `cfg.Input` with the local staging path, then continues through the regular ffmpeg/whisper/chunk path. `run.SourceURL` records the original URL in `manifest.json`. **Cache is skipped for URL runs** (remote bytes may change between requests, an input-hash match does not imply identity of the remote resource); `--no-cache` has no additional effect.
- Live-verified 2026-08-08: `https://www.youtube.com/watch?v=jNQXAC9IVRw` ("Me at the zoo", 19s) → transcript "Alright, so here we are, one of the elephants..." produced under `.pagevideo/url-smoke/<run>/`, manifest carries `source_url`. yt-dlp merged DASH tracks into mp4 via bundled ffmpeg (`--ffmpeg-location`).
- Tests (`internal/download/download_test.go`, 6 cases, no network): scheme whitelist, missing binary, non-positive maxBytes, pre-cancelled context, directory-as-binary rejection, supported matrix.

- `internal/llm/summarize.go` renamed to `internal/llm/artifacts.go` (git mv) and generalized: shared `sharedPolicy` (system role) plus per-artifact task text (`summaryTask`, `studyTask`, `faqTask`, `glossaryTask`), all sent as `task + "\n\n--- ТРАНСКРИПТ (untrusted) ---\n" + transcript` via `NewUserMessage`. New methods: `GenerateStudy`, `GenerateFAQ`, `GenerateGlossary`; `SummarizeTranscript` is now a thin wrapper over the shared `GenerateArtifact`.
- Single size gate: `defaultArtifactMaxChars = 24000` (was `defaultSummaryMaxChars`); all four generators reject oversized transcripts with `ErrTranscriptTooLarge`.
- `internal/pipeline`: `maybeSummarize` replaced by `maybeArtifacts`, which iterates `artifactGenerators` (filename + generator + Result-field assigner). Per-artifact failures are logged and skipped, never fail the run; `Result` now includes `Study`, `FAQ`, `Glossary` alongside `Summary`. Cache save/restore does not include LLM artifacts (they are non-deterministic; re-running with `--enable-summary` always calls the LLM).
- CLI help updated to state all four files produced under `--enable-summary`.
- New tests (`artifacts_test.go`): `TestGenerateStudy_PolicySeparatedRequest`, `TestGenerateFAQ_UsesTaskSpecificHeader`, `TestGenerateGlossary_RejectsEmptyTranscript`, `TestGenerateArtifact_RejectsOversizedTranscript` (all four generators), plus existing summary/role/isolation cases all pass unchanged.
- Verified: `go test ./... -count=1` (llm 8 + cache 6 + chunk + cli cases all PASS), `go vet ./...`, `go build ./...`, `gofmt -l cmd internal` clean.

## 2026-08-08 — Run cache (--no-cache to disable)

- New package `internal/cache`: completed runs are mirrored into `OutputRoot/.cache/<inputHash16>-<paramsHash16>/` (audio.wav, transcript.txt, transcript.srt, cache.json). Key = input SHA-256 + stable hash of (ffmpeg, whisper, model binaries' SHA-256, language, chunk-chars, overlap-words) — swapping a dependency binary or chunk geometry invalidates prior entries.
- Cache hit skips ffmpeg and whisper entirely: artifacts are copied into a fresh run directory, chunks are re-split deterministically (same bytes in → same chunks out, only embedded runID changes), manifest.json is rewritten with the new run id and paths.
- Cache failures degrade to "miss" everywhere — corrupt metadata, missing files, or hash mismatches never fail the run; they cause a full recompute. `--no-cache` opts out explicitly.
- Tests (`internal/cache/cache_test.go`, 6 cases, no network): params-hash determinism and sensitivity to language/ffmpeg changes, short-hash padding, empty-cache miss, store→load round-trip, corruption rejection, restore semantics (manifest/cache.json never copied).
- Verified: `gofmt -l cmd internal scripts`, `go test ./...`, `go vet ./...`, `go build` all pass with repo-local caches.

**State:** implemented, unit-tested, live-verified 2026-08-08 on synthetic `smoke.mp4`: miss run produced transcript hash `66d91d05...`, immediate `--no-cache` re-run produced identical transcript/audio hashes, follow-up cached run logged `pagevideo: cache hit` and produced a distinct `run_id` with byte-identical artifacts in a fresh directory.

## 2026-08-07 — UX fixes: bare path/URL as input

- CLI: a bare first token that is an existing local file OR an `http(s)://` URL is accepted as the `--input` of `process` (`internal/cli.cli_test.go` covers both, plus the still-rejected "unknown command" path). Useful inside the interactive REPL of `scripts/pagevideo-start.bat`.
- `Config.Validate` now returns a clear, explicit error when `--input` is a URL: remote downloaders (YouTube/VK/RuTube/http) are not implemented. Local files remain the only supported input.
- REPL: leading whitespace typed before a command is trimmed; `version` and `provider check` verified working in both direct and interactive modes. A live full run against the user video `Свой ЛИЧНЫЙ VPN за 30 минут.mp4` (57 MB) completed: `transcript.txt` (38 KB, ~450 lines), `transcript.srt`, chunk manifest with hashes — status READY.
