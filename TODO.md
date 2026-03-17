# Sortie Roadmap

High-level milestones and tasks for building Sortie from zero to a self-hosting orchestration
service. Each task is atomic, independently verifiable, and sized for a single agent session.

## Milestone 0: Project Scaffold

Establish the Go project structure, tooling, and development conventions before writing any
business logic. Every subsequent task assumes this foundation exists.

- [ ] 0.1 Research Go project layout conventions (standard-layout, cmd/internal/pkg patterns)
      and select the structure for Sortie. Document the decision in a short comment in go.mod
      or a dedicated section in CONTRIBUTING.md.
      **Verify:** `go build ./...` succeeds with an empty main package.

- [ ] 0.2 Initialize Go module (`go mod init`), create `cmd/sortie/main.go` with a minimal
      `main()` that prints version and exits. Set up the directory skeleton per the chosen
      layout.
      **Verify:** `go run ./cmd/sortie` prints version string and exits 0.

- [ ] 0.3 Configure linting and formatting: add `golangci-lint` config (`.golangci.yml`),
      create a `Makefile` with targets `fmt`, `lint`, `test`, `build`. Ensure `make lint`
      passes on the empty project.
      **Verify:** `make lint` and `make fmt` exit 0 with no warnings.

- [ ] 0.4 Set up CI: create `.github/workflows/ci.yml` that runs `make lint` and `make test`
      on push and PR. Use a Go matrix build (latest stable Go version).
      **Verify:** push to GitHub triggers CI and all jobs pass.

- [ ] 0.5 Add a `CLAUDE.md` (or `AGENTS.md`) context file for coding agents. Include: build
      commands, test commands, project structure overview, naming conventions, and architectural
      boundaries that agents must not violate.
      **Verify:** an agent reading the file can answer "how do I build and test this project"
      without additional context.

## Milestone 1: Configuration Layer

Parse `WORKFLOW.md`, expose typed config, and support dynamic reload. No orchestration logic
yet - just the ability to read, validate, and watch the workflow file.

- [ ] 1.1 Research YAML parsing libraries for Go (`gopkg.in/yaml.v3`, `github.com/goccy/go-yaml`)
      and Go template engine behavior (`text/template` strict mode). Select libraries and add
      them to `go.mod`.
      **Verify:** `go mod tidy` succeeds, dependencies resolve.

- [ ] 1.2 Implement the workflow loader: read a file, split YAML front matter from Markdown
      body, parse front matter into a map, return `{config, prompt_template}`. Handle error
      cases: missing file, invalid YAML, non-map front matter.
      **Verify:** unit tests cover happy path, missing file, bad YAML, non-map YAML.

- [ ] 1.3 Implement the typed config layer: define Go structs for all config sections
      (`tracker`, `polling`, `workspace`, `hooks`, `agent`). Apply defaults. Resolve `$VAR`
      environment indirection and `~` path expansion. Validate required fields.
      **Verify:** unit tests cover defaults, env resolution, path expansion, validation errors.

- [ ] 1.4 Implement prompt template rendering using `text/template` with strict mode (no
      undefined variables). Accept `issue`, `attempt`, and `run` as template inputs. Test
      with a sample template that exercises all variables.
      **Verify:** unit tests cover successful render, unknown variable error, nested field
      access (labels, blockers).

- [ ] 1.5 Implement filesystem watcher for `WORKFLOW.md`. On change, re-read and re-apply
      config. On invalid reload, keep last known good config and log an error. Expose a
      method to get the current effective config.
      **Verify:** integration test modifies a temp WORKFLOW.md file, confirms new config is
      picked up. A second test introduces invalid YAML and confirms the old config is retained.

- [ ] 1.6 Implement CLI entry point: accept an optional positional argument for workflow file
      path, default to `./WORKFLOW.md`. Add `--port` flag (stored for later). On missing file,
      print a clear error and exit nonzero.
      **Verify:** `go run ./cmd/sortie /tmp/test-workflow.md` loads the file.
      `go run ./cmd/sortie` without a file in cwd exits with an error message.

## Milestone 2: Persistence Layer

SQLite database for retry queues, run history, session metadata, and aggregate metrics.
No orchestration logic yet - just the storage primitives.

- [ ] 2.1 Research SQLite libraries for Go (`modernc.org/sqlite` vs `mattn/go-sqlite3`).
      Select the library and add to `go.mod`. Create a minimal integration test that opens
      an in-memory SQLite database.
      **Verify:** test opens DB, creates a table, inserts a row, reads it back.

- [ ] 2.2 Implement schema migration runner: numbered migrations applied in order, tracked in
      a `schema_migrations` table. Implement the initial migration that creates the four core
      tables from the architecture doc (Section 19.2): `retry_entries`, `run_history`,
      `session_metadata`, `aggregate_metrics`.
      **Verify:** unit test applies migrations to a fresh DB, confirms all tables exist with
      correct columns.

- [ ] 2.3 Implement CRUD operations for `retry_entries`: save, load all, delete by issue_id.
      **Verify:** unit tests for save, load, delete, and idempotent save (upsert).

- [ ] 2.4 Implement CRUD operations for `run_history`: append a completed run, query by
      issue_id, query recent runs with pagination.
      **Verify:** unit tests for append, query by issue, and pagination.

- [ ] 2.5 Implement CRUD operations for `session_metadata` and `aggregate_metrics`: upsert
      session metadata, read/write aggregate metrics.
      **Verify:** unit tests for each operation.

- [ ] 2.6 Implement startup recovery: load persisted retry entries, reconstruct timers from
      `due_at_ms` timestamps, return a list of entries with computed remaining delays.
      **Verify:** unit test creates retry entries with past and future `due_at_ms`, confirms
      the loader returns correct remaining delays (past entries get delay 0).

## Milestone 3: Domain Model and Tracker Adapter Interface

Define the normalized issue model, the tracker adapter interface, and implement the first
adapter (Jira). No orchestration logic yet - just the ability to talk to a tracker.

- [ ] 3.1 Define the normalized `Issue` struct with all fields from architecture Section 4.1.1.
      Define the `TrackerAdapter` interface with the five required operations. Place these in
      `internal/domain/` or equivalent.
      **Verify:** code compiles, interfaces are importable from other packages.

- [ ] 3.2 Implement a file-based tracker adapter for development and testing. Reads issues
      from a JSON or YAML file on disk. Supports all five adapter operations against the file
      contents.
      **Verify:** unit tests with a fixture file containing sample issues. Tests cover
      candidate fetch, state refresh, terminal fetch, single issue fetch, comments.

- [ ] 3.3 Research Jira REST API: authentication methods (API token, OAuth, PAT), relevant
      endpoints (search, issue, comments, transitions), pagination model, rate limits.
      Document findings in a short `docs/jira-adapter-notes.md`.
      **Verify:** document exists with endpoint references and auth requirements.

- [ ] 3.4 Implement Jira tracker adapter: candidate issue fetch using JQL, issue state refresh
      by ID batch, terminal state fetch, single issue fetch with comments. Normalize Jira
      responses to the `Issue` model.
      **Verify:** unit tests with HTTP response fixtures (recorded or hand-crafted JSON).
      Tests cover normalization, pagination, error mapping to generic categories.

- [ ] 3.5 Implement real Jira integration test (guarded by env var `SORTIE_JIRA_TEST=1` and
      credentials). Fetch real issues from a test project, confirm normalization produces valid
      Issue structs.
      **Verify:** `SORTIE_JIRA_TEST=1 go test ./internal/tracker/jira/... -run Integration`
      passes against a real Jira instance. Skipped cleanly when env var is absent.

## Milestone 4: Agent Adapter Interface and Claude Code Adapter

Define the agent adapter interface and implement the first adapter (Claude Code). No
orchestration logic yet - just the ability to launch an agent, run a turn, and receive events.

- [ ] 4.1 Define the `AgentAdapter` interface with `StartSession`, `RunTurn`, `StopSession`.
      Define the normalized event types from architecture Section 10.3. Place these in
      `internal/domain/` or equivalent.
      **Verify:** code compiles, interfaces are importable.

- [ ] 4.2 Research Claude Code CLI: available flags, subprocess behavior, stdio output format,
      session lifecycle, how to detect turn completion and failures. Document findings in
      `docs/claude-code-adapter-notes.md`.
      **Verify:** document exists with CLI reference and observed behavior.

- [ ] 4.3 Implement a mock agent adapter for testing. Simulates session start, emits canned
      events on `RunTurn`, supports configurable success/failure outcomes.
      **Verify:** unit tests demonstrate the mock adapter satisfying the interface contract.

- [ ] 4.4 Implement Claude Code agent adapter: subprocess launch, stdio reading, event parsing,
      session lifecycle (start, turn, stop). Normalize Claude Code output to the standard event
      types.
      **Verify:** unit tests with captured Claude Code output fixtures. Tests cover event
      parsing, timeout handling, subprocess cleanup.

- [ ] 4.5 Implement real Claude Code integration test (guarded by env var
      `SORTIE_CLAUDE_TEST=1`). Launch Claude Code with a trivial prompt in a temp workspace,
      confirm session starts, a turn completes, and events are received.
      **Verify:** `SORTIE_CLAUDE_TEST=1 go test ./internal/agent/claude/... -run Integration`
      passes. Skipped cleanly when env var is absent.

## Milestone 5: Workspace Manager

Workspace creation, reuse, path safety, and hook execution. No orchestration logic yet -
just the ability to prepare and clean workspaces.

- [ ] 5.1 Implement workspace path computation: sanitize issue identifier to workspace key,
      join with workspace root, validate containment (path must be under root, no symlink
      escape).
      **Verify:** unit tests cover sanitization, containment check, symlink rejection.

- [ ] 5.2 Implement workspace creation and reuse: create directory if missing, reuse if exists,
      replace if exists but is not a directory. Track `created_now` flag.
      **Verify:** unit tests with temp directories covering create, reuse, and replace cases.

- [ ] 5.3 Implement hook execution: run a shell script with workspace as cwd, enforce timeout,
      set environment variables (`SORTIE_ISSUE_ID`, etc.), capture and truncate output.
      **Verify:** unit tests run a trivial hook script, confirm env vars are set, confirm
      timeout kills the hook, confirm output truncation.

- [ ] 5.4 Implement workspace lifecycle orchestration: `after_create` on new, `before_run`
      before agent, `after_run` after agent, `before_remove` before cleanup. Enforce failure
      semantics (fatal vs. ignored per hook).
      **Verify:** integration test exercises the full lifecycle with a temp workspace and
      script hooks that write marker files.

- [ ] 5.5 Implement workspace cleanup for terminal issues: given a list of issue identifiers,
      remove matching workspace directories (with `before_remove` hook).
      **Verify:** unit test creates temp workspaces, marks some as terminal, confirms cleanup
      removes only terminal workspaces.

## Milestone 6: Orchestrator Core

The polling loop, dispatch, reconciliation, retry, and state machine. This is the central
component. Uses mock adapters for tracker and agent - no real external calls.

- [ ] 6.1 Implement the orchestrator state struct: running map, claimed set, retry attempts,
      agent totals. Implement slot availability calculation (global and per-state).
      **Verify:** unit tests for slot math with various running/claimed combinations.

- [ ] 6.2 Implement candidate selection and dispatch sorting: priority ascending, created_at
      oldest first, identifier tiebreaker. Implement eligibility checks (active state, not
      claimed, not running, slots available, blocker rule).
      **Verify:** unit tests with various issue sets confirm correct sort order and
      eligibility filtering.

- [ ] 6.3 Implement the dispatch function: claim issue, spawn worker (using mock agent
      adapter), add to running map. Handle spawn failure by scheduling retry.
      **Verify:** unit tests confirm issue is claimed, running entry is created, and spawn
      failure triggers retry scheduling.

- [ ] 6.4 Implement worker exit handling: normal exit schedules continuation retry (1s delay),
      abnormal exit schedules exponential backoff retry. Persist completed run to SQLite.
      **Verify:** unit tests for both exit paths, confirm correct retry delays and SQLite
      persistence.

- [ ] 6.5 Implement retry timer handling: on timer fire, re-fetch candidates, check eligibility,
      dispatch or requeue. Release claim if issue is gone or no longer active.
      **Verify:** unit tests with mock tracker returning various states on retry.

- [ ] 6.6 Implement reconciliation: stall detection (elapsed > stall_timeout), tracker state
      refresh (terminal -> stop + cleanup, active -> update, other -> stop without cleanup).
      Handle refresh failure gracefully.
      **Verify:** unit tests for each reconciliation outcome including refresh failure.

- [ ] 6.7 Implement the poll loop: tick scheduling, reconciliation before dispatch, config
      validation before dispatch, dispatch until slots exhausted. Wire everything together
      with mock adapters.
      **Verify:** integration test runs the orchestrator with mock tracker (returns 3 issues)
      and mock agent (completes after 1 turn). Confirm all 3 issues are dispatched, run, and
      completed. Confirm retry on simulated failure.

- [ ] 6.8 Implement startup recovery from SQLite: load retry entries, reconstruct timers,
      enumerate existing workspaces, reconcile with tracker state.
      **Verify:** integration test saves retry entries to SQLite, restarts the orchestrator,
      confirms retries fire at correct times.

- [ ] 6.9 Implement dynamic config reload integration: when WORKFLOW.md changes, the
      orchestrator picks up new polling interval, concurrency limits, active/terminal states.
      **Verify:** integration test modifies WORKFLOW.md while orchestrator is running, confirms
      behavior changes (e.g., new polling interval takes effect).

## Milestone 7: End-to-End with Real Adapters

Connect real Jira and real Claude Code adapters to the orchestrator. This is the first time
the system does real work.

- [ ] 7.1 Wire the Jira adapter and Claude Code adapter into the orchestrator startup. Add
      adapter registration and selection based on `tracker.kind` and `agent.kind` config.
      **Verify:** `go run ./cmd/sortie ./WORKFLOW.md` starts, connects to Jira, and polls
      for issues (with a valid WORKFLOW.md and credentials).

- [ ] 7.2 Create a sample `WORKFLOW.md` for testing: configure Jira project, workspace root,
      a simple after_create hook (e.g., `git clone`), and a minimal prompt template.
      **Verify:** the sample file passes config validation when loaded by Sortie.

- [ ] 7.3 Run the first real end-to-end test: create a test issue in Jira, start Sortie,
      confirm it dispatches the issue, Claude Code runs a turn, and the run completes.
      **Verify:** Jira issue shows evidence of agent activity (comment or state change).
      Run history is persisted in SQLite.

- [ ] 7.4 Test failure and retry: create an issue that will cause Claude Code to fail (e.g.,
      invalid workspace), confirm Sortie retries with exponential backoff.
      **Verify:** SQLite run_history shows multiple attempts with increasing delays.

- [ ] 7.5 Test reconciliation: start Sortie with a running issue, move the issue to Done in
      Jira, confirm Sortie stops the agent and cleans the workspace.
      **Verify:** workspace directory is removed after reconciliation.

## Milestone 8: Observability

HTTP dashboard, JSON API, and structured logging. The system should be monitorable by
operators after this milestone.

- [ ] 8.1 Implement structured logging with `slog`: add issue_id, issue_identifier, and
      session_id context to all relevant log lines. Use key=value format.
      **Verify:** run the orchestrator, grep logs for structured fields, confirm they are
      present and consistent.

- [ ] 8.2 Implement the runtime snapshot function: return running sessions, retry queue,
      agent totals, rate limits.
      **Verify:** unit test populates orchestrator state, calls snapshot, confirms all
      fields are populated.

- [ ] 8.3 Implement the JSON API server: `GET /api/v1/state`, `GET /api/v1/<identifier>`,
      `POST /api/v1/refresh`. Use Go `net/http` and `encoding/json`.
      **Verify:** integration test starts the HTTP server, calls each endpoint, validates
      response shapes against the architecture doc.

- [ ] 8.4 Implement the HTML dashboard: server-rendered page at `/` showing running sessions,
      retry queue, token totals. Use Go `html/template`. Add auto-refresh via SSE or
      meta-refresh.
      **Verify:** start Sortie with `--port 8080`, open `http://localhost:8080` in a browser,
      confirm the dashboard renders current state.

## Milestone 9: Self-Hosting (Sortie Builds Sortie)

At this point, Sortie has enough functionality to orchestrate its own development. Create
Jira issues for remaining work, point Sortie at its own repository, and let agents implement
features.

- [ ] 9.1 Create a production `WORKFLOW.md` for the Sortie repository itself. Define the
      prompt template, hooks (git clone, go mod download, make lint), and agent config.
      **Verify:** Sortie starts and polls the Sortie Jira project.

- [ ] 9.2 Create 3-5 small Jira issues for real improvements (e.g., "add graceful shutdown",
      "add request logging middleware", "add --version flag"). Start Sortie and observe it
      dispatching agents to work on these issues.
      **Verify:** at least one issue results in a working PR or code change.

- [ ] 9.3 Iterate on the WORKFLOW.md prompt based on observed agent behavior. Improve
      instructions for continuation turns, error recovery, and coding conventions.
      **Verify:** subsequent agent runs produce higher quality output than initial runs.

## Milestone 10: Hardening and Production Readiness

Polish for public release. Security, documentation, graceful shutdown, and operational
tooling.

- [ ] 10.1 Implement graceful shutdown: on SIGTERM/SIGINT, stop accepting new dispatches,
      wait for running agents to complete (with timeout), close SQLite cleanly.
      **Verify:** send SIGTERM to running Sortie, confirm it shuts down without data loss.

- [ ] 10.2 Implement the `tracker_api` client-side tool: expose tracker API access to agents
      during sessions, scoped to the configured project.
      **Verify:** agent can query Jira issues through the tool during a session.

- [ ] 10.3 Write CONTRIBUTING.md: build instructions, test instructions, code conventions,
      PR process, architecture overview reference.
      **Verify:** a new contributor can follow the guide to build, test, and submit a change.

- [ ] 10.4 Write SECURITY.md: trust model, secret handling, workspace isolation guarantees,
      prompt injection risks.
      **Verify:** document covers all items from architecture Section 15.

- [ ] 10.5 Add release automation: GoReleaser config for building cross-platform binaries,
      GitHub Actions release workflow triggered by tags.
      **Verify:** `git tag v0.1.0 && git push --tags` produces release artifacts on GitHub.

- [ ] 10.6 Review and finalize README.md: add installation instructions, quick start guide,
      and configuration reference now that the software exists.
      **Verify:** a new user can follow the README to install and run Sortie against their
      own Jira project.
