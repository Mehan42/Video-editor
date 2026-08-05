# Prompt Injection Boundary

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

Requests sent to an LLM have separate logical fields:

1. system policy;
2. task request;
3. untrusted source content;
4. retrieved context;
5. output schema.

Source content and retrieved context cannot modify policy or invoke tools. Tool calls, provider selection, network access, and filesystem access are controlled outside the model response and require explicit capability checks.
