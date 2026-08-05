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

1. **Verify Bionic readiness locally.** Determine the supported local API base URL, port, health/models endpoint, chat endpoint, model identity, structured-output support, and whether the service is already running. Do not guess a port and do not send video/transcript data until the endpoint is verified.
2. **Define the provider contract.** Add provider-neutral types for health, capabilities, request, response, provider identity, egress classification, and failure statuses. Keep LM Studio Bionic behind an adapter.
3. **Implement a report-only Bionic adapter.** Use a configurable base URL, explicit allowlist, timeout, response-size limit, schema validation, and redacted evidence. Default to `REPORT_ONLY` until a Human Gate enables egress.
4. **Add tests without network.** Use `httptest` fixtures for health, model discovery, valid chat response, malformed response, timeout, oversized response, and blocked egress. Do not test against a live provider by default.
5. **Add secret and prompt boundaries.** Separate system policy, task, source transcript, retrieved context, and output schema. Add tests proving transcript content cannot select a provider or capability, and provider responses cannot grant authority.
6. **Re-run verification.** Run `gofmt`, `go test ./...`, `go vet ./...`, build, `git diff --check`, Bionic adapter unit tests, and CodeGraph status. Keep race/govulncheck blockers explicit.

## Explicitly deferred

- Real NATS deployment.
- MCP connector execution.
- OpenRouter/FreeLLM adapters.
- Autonomous agent orchestration.
- External publication and automatic knowledge-base mutation.
- OS-level media sandbox/job-object hardening.

## Stop conditions

Stop and report `BLOCKED` if the Bionic endpoint cannot be identified, if provider egress is ambiguous, if secrets would enter request/evidence payloads, or if a model response attempts to grant a capability.
