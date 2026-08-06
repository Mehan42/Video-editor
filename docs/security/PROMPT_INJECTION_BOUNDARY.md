# Prompt Injection Boundary

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Requests sent to an LLM have separate logical fields:

1. system policy;
2. task request;
3. untrusted source content;
4. retrieved context;
5. output schema.

Source content and retrieved context cannot modify policy or invoke tools. Tool calls, provider selection, network access, and filesystem access are controlled outside the model response and require explicit capability checks.

## Enforced in code (internal/llm)

- `Message.Role` is a closed set: `system | user | assistant`. Encoding any other role fails (`invalid message role`).
- Transcript, chunks, and retrieved text are sent only via `NewUserMessage` — always as `user` content, never as system/authority.
- Provider endpoint (`Config.BaseURL`) and chat capability (`Config.AllowChat`) are fixed at client construction and cannot be reached from a chat message or response.
- Chat messages are data-only: role + content. No field can select a model/provider, alter policy, or grant a capability. `TestChatRequestConfigIsolation` proves an embedded injection string does not change the model or endpoint actually used.
- `ChatResponse` is parsed for content only; it has no authority field and is treated as untrusted output.
