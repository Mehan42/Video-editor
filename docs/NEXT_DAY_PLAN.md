# Next-Day Plan

**Status:** READY FOR NEXT SESSION
**Current state:** development paused after the local media/transcription MVP

## Starting evidence

- Local Git repository exists on `main`; no remote and no commit yet.
- `go test ./...`, `go vet ./...`, build, synthetic MP4 smoke, manifest read-back, and input-size negative validation passed.
- `go test -race ./...` is blocked by missing GCC/cgo.
- `govulncheck` is not installed.
- CodeGraph is initialized and the latest successful status is `Index is up to date`.
- Harness session is local, ready, network-disabled, and external-execution-disabled.
- Bionic `1.0.5` exists at `E:\LM Studio Bionic\Bionic.exe`; GUI was not launched and no API endpoint was confirmed.

## Work order

1. **Start Bionic and verify runtime readiness.** The bundle statically points to a likely loopback OpenAI-compatible API at `127.0.0.1:1234`; the current CLI evidence is `BLOCKED_PROVIDER` because no listener is running. Confirm `/v1/models`, model identity, chat, and structured-output support without sending video/transcript data.
2. **Review the provider contract.** The first loopback-only Bionic adapter now exists in `internal/llm`; review health, capabilities, response limits, failure statuses, and the explicit chat egress flag before enabling it.
3. **Add tests without network.** Extend `httptest` fixtures for malformed response, timeout, oversized response, provider error, and schema validation. Do not test against a live provider by default.
4. **Add secret and prompt boundaries.** Separate system policy, task, source transcript, retrieved context, and output schema. Add tests proving transcript content cannot select a provider or capability, and provider responses cannot grant authority.
5. **Re-run verification.** Run `gofmt`, `go test ./...`, package-level and full `go vet ./...`, build, `git diff --check`, Bionic adapter unit tests, and CodeGraph status. Keep race/govulncheck blockers explicit.

## Explicitly deferred

- Real NATS deployment.
- MCP connector execution.
- OpenRouter/FreeLLM adapters.
- Autonomous agent orchestration.
- External publication and automatic knowledge-base mutation.
- OS-level media sandbox/job-object hardening.

## Stop conditions

Stop and report `BLOCKED` if the Bionic endpoint cannot be identified, if provider egress is ambiguous, if secrets would enter request/evidence payloads, or if a model response attempts to grant a capability.
