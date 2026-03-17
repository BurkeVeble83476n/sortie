---
status: accepted
date: 2026-03-17
decision-makers: Serghei Iakovlev
---

# Technology Decisions

## Context and Problem Statement

Sortie is a long-running orchestration service that polls an issue tracker, creates isolated
per-issue workspaces, dispatches coding agent sessions, and monitors their execution through
retries, reconciliation, and observability. The architecture is informed by
[OpenAI Symphony](https://github.com/openai/symphony) and adapted for multi-agent,
multi-tracker extensibility.

This document records the technology choices that are architecturally significant: choices
where a different decision would produce a fundamentally different system. Implementation
details that follow naturally from these choices are not documented here.

## Decision Drivers

1. **Runtime fitness.** The service is a daemon that manages concurrent subprocesses,
   enforces timeouts and stall detection, and reconciles state on every polling tick.
   Concurrency primitives, subprocess lifecycle management, and memory predictability
   under sustained load are non-negotiable requirements.

2. **Deployment simplicity.** The orchestrator must run on developer machines, CI
   environments, and remote SSH hosts. Minimizing runtime dependencies on target hosts
   directly reduces operational burden.

3. **Agent-assisted development.** The codebase will be primarily written and maintained
   by AI coding agents with human oversight on architecture, review, and critical
   debugging. The stack must produce correct, idiomatic output from current-generation
   models with minimal iteration cycles.

4. **Extensibility.** The system must support pluggable issue trackers and pluggable
   coding agents without rewriting core orchestration logic.

5. **Long-term maintainability.** The project targets public release. Technology choices
   must favor a large contributor pool, stable tooling, and predictable upgrade paths
   over novel or niche ecosystems.

## Core Runtime: Go

Go is selected for the orchestrator daemon, CLI, and all core infrastructure code.

**Concurrency model.** Goroutines and channels are the native abstraction for the
orchestrator's workload: spawn a goroutine per agent session, use `context.Context` for
cancellation propagation, coordinate through typed channels. Go's scheduler distributes
work across OS threads without application-level coordination.

**Subprocess management.** `os/exec.CommandContext` integrates process lifecycle with
context cancellation: if a ticket moves to a terminal state, `cancel()` propagates through
the process tree. Signal handling and process cleanup on all target platforms (Linux, macOS,
Windows via SSH) are well-tested in the standard library.

**Deployment.** `go build` produces a single statically-linked binary with zero runtime
dependencies. Cross-compilation is built in. For SSH worker hosts, deployment reduces to
copying one file.

**Memory predictability.** The orchestrator is a 24/7 daemon. Go's garbage collector has
tunable controls (`GOGC`, `GOMEMLIMIT`) and goroutine stacks start at 2-8 KB.

**Agent generation quality.** Current LLMs produce higher pass@1 rates on TypeScript than
Go. This gap is real but transient: it narrows with each model generation, and Go's
uniformity (`gofmt`, single error handling idiom, minimal stylistic variation) partially
compensates by reducing the space for inconsistent output. The runtime characteristics
above are permanent architectural properties; the generation quality difference is a
snapshot of the current moment.

### Considered Alternatives

**Node.js/TypeScript.** Strongest AI generation quality today. Best ecosystem for GraphQL
clients and template engines. However, the single-threaded event loop serializes all
orchestration logic; heavy JSON parsing or token accounting blocks stall detection and
reconciliation. Deployment requires a runtime on every target host. Long-running daemon
reliability demands active memory management. These are properties of the execution model,
not risks that agents can mitigate through better code.

**Elixir/OTP.** OpenAI's Symphony reference implementation uses Elixir, and BEAM's supervision
trees are an ideal fit for this workload. However, the ecosystem is small, the contributor
pool is narrow, and LLM generation quality for Elixir is significantly below Go or
TypeScript. Elixir is the right choice for a team of BEAM experts; it is the wrong default
for a public project targeting broad adoption.

**Rust.** Superior safety guarantees and performance. However, the borrow checker creates
long iteration cycles for agent-written code. For a project where development speed and
contributor accessibility matter, Rust's costs outweigh its benefits at this stage.

## Persistence: SQLite (Embedded)

[The Symphony spec](https://github.com/openai/symphony/blob/main/SPEC.md) uses in-memory state
with no persistent database, accepting that retry queues and session metadata are lost on restart.
Sortie corrects this.

SQLite in WAL mode provides concurrent reads with a single writer, matching the
orchestrator's single-authority pattern. The entire state (retry queue, session metadata,
workspace registry, token accounting, run history) lives in a single file alongside the
binary. There is no external database to provision and no network dependency.

Go has mature SQLite bindings: `modernc.org/sqlite` (pure Go, no CGo) provides full
SQLite functionality without a C toolchain on the build host.

### Considered Alternatives

**In-memory only (Symphony approach).** Simpler, but a process restart during active work
loses all retry state and requires a full cold start from the issue tracker.

**PostgreSQL/MySQL.** Adds an external dependency, connection management, and migration
tooling. Unnecessary for a single-process orchestrator that serializes all writes through
one authority.

**Embedded key-value stores (BoltDB, BadgerDB).** Viable, but SQLite provides relational
queries, schema migrations, and standard tooling (`sqlite3` CLI) that simplify debugging
and operational inspection.

## Integration Extensibility

The orchestrator defines adapter interfaces for issue tracker access and coding agent
communication. Each tracker (Jira, Linear, GitHub Issues, File System) and each agent
runtime (Claude Code, Codex, generic HTTP) is implemented as a separate package behind
its respective interface. Issue and event data is normalized into common types at the
adapter boundary.

The initial implementation targets Jira as the primary tracker and Claude Code as the
primary agent runtime. The agent adapter communicates over stdio, allowing straightforward
substitution of alternative runtimes (Codex, Copilot, or any agent exposing a compatible
CLI interface) without changes to orchestration logic.

Linear support and additional agent adapters are planned as subsequent implementations.

## Summary

| Component    | Choice             | Key reason                                                        |
| ------------ | ------------------ | ----------------------------------------------------------------- |
| Core runtime | Go                 | Goroutines, context cancellation, single binary                   |
| Persistence  | SQLite (embedded)  | Zero-dependency state survival across restarts                    |
| Integration  | Adapter interfaces | Jira + Claude Code first, extensible to other trackers and agents |
