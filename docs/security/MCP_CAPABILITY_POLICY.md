# MCP Capability Policy

**Status:** DRAFT / WAITING FOR HUMAN APPROVAL

MCP connectors are untrusted adapters with narrow capabilities. The default capability set is read-only and report-only.

Forbidden by default:

- arbitrary shell execution;
- broad filesystem access;
- secret retrieval;
- scheduler or hook installation;
- network access to arbitrary destinations;
- automatic publication or mutation.

Each connector call must record connector identity, capability, arguments hash, result hash, and evidence references.
