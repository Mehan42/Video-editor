# LLM Request and Response Contract

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

## Request

The request must separate policy, task, source content, retrieved context, output schema, provider selection, run ID, and egress classification.

## Response

The response must include provider identity, model identity, request ID, output, schema validation status, usage metadata when available, and evidence references. Invalid or incomplete responses are blocked, not silently coerced into success.
