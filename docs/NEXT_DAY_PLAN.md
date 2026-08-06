# Next-Day Plan

**Status:** NEXT-DAY ITEMS DONE 2026-08-06 (1–4 and verification) / REMAINING: deferred roadmap features
**Current state:** local media/transcription MVP + verified Bionic readiness + hardened LLM boundary

## Starting evidence (updated 2026-08-06)

- Local Git repository on `main`; remote `origin` configured (github.com/Mehan42/Video-editor); commits pushed through `a7c556d`.
- `go test ./...`, `go vet ./...`, build, synthetic MP4 smoke, manifest read-back, input-size negative validation passed.
- `go test -race ./...` still blocked by missing GCC/cgo.
- `govulncheck` still not installed.
- CodeGraph initialized and refreshed after llm changes; status `Index is up to date`.
- Harness session is local, ready, network-disabled, external-execution-disabled.
- Bionic `1.0.5` at `E:\LM Studio Bionic\Bionic.exe`: **READY** on `127.0.0.1:1234`, 6 models, capabilities `chat/completions` + `models`. Chat egress still blocked by default.

## Work order — DONE 2026-08-06

1. **Bionic runtime readiness verified.** Listener active on `127.0.0.1:1234`; `provider check` returns READY with 6 models (qwen/qwen3-vl-8b, llama-3.2-3b-instruct, qwen/qwen3.6-27b, google/gemma-4-12b-qat, prism-ml/bonsai-27b, text-embedding-nomic-embed-text-v1.5). No transcript data sent.
2. **Provider contract reviewed.** `internal/llm` loopback-only adapter: `/v1/models` readiness, response-size limit, HTTP error/status mapping, explicit chat egress flag, default chat block.
3. **Offline tests added.** httptest fixtures in `internal/llm/client_test.go` cover malformed response, empty data, HTTP error, oversized response, timeout, empty choices, malformed chat. No live provider used.
4. **Secret and prompt boundaries added.** Message roles restricted to `system|user|assistant`; untrusted content sent only as user via `NewUserMessage`; provider/endpoint/capability fixed in Config at construction. `TestChatRequestConfigIsolation` proves transcript text cannot select a provider or grant authority; `TestRejectsUnknownMessageRole` rejects unknown roles.
5. **Verification re-run.** gofmt/go test/go vet/build/git diff --check/CodeGraph all pass; CodeGraph refreshed. Race/govulncheck blockers remain as noted.

## Explicitly deferred

- Real NATS deployment.
- MCP connector execution.
- OpenRouter/FreeLLM adapters.
- Autonomous agent orchestration.
- External publication and automatic knowledge-base mutation.
- OS-level media sandbox/job-object hardening.
- Markdown/summary/chapter generation, quiz/flashcards/glossary/FAQ, caching/resume: next-roadmap items, need separate approval.

## Stop conditions

Stop and report `BLOCKED` if the Bionic endpoint cannot be identified, if provider egress is ambiguous, if secrets would enter request/evidence payloads, or if a model response attempts to grant a capability.
