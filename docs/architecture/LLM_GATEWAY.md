# LLM Gateway

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Boundary

Application code calls `LLMGateway`, never a concrete provider. The gateway performs capability discovery, request validation, egress policy checks, bounded execution, response schema validation, and evidence recording.

```text
Pipeline
  -> LLMGateway
  -> NATS transport
  -> MCP connector
  -> Provider adapter
```

## Provider adapters

The registry may contain LM Studio, OpenRouter aliases, FreeLLM-compatible providers, Ollama, and other OpenAI-compatible endpoints. No provider is mandatory by architecture.

Each adapter must expose health, model identity, supported operations, context limit, structured-output support, streaming support, and an explicit egress classification.

## Security rules

- A provider is not `READY` based only on a configured URL.
- External egress is disabled by default.
- Prompt content cannot select a provider or enable a capability.
- API keys never enter transcript text, prompts, raw evidence, or event payloads.
- Provider output is untrusted until schema validation and provenance attachment complete.
