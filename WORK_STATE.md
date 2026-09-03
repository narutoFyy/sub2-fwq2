# OAuth monitor noise reduction

Status: complete
Current task: none
Execution mode: state-main
Plan topology: linear
Baseline: origin/main@55daac165

## Rules

- Execute one task at a time and verify it before activating the next task.
- Keep all unfinished low-cost probing work outside this clean worktree and release.
- Preserve legacy monitor config fields for read compatibility but normalize them off.
- Do not add active probes or query usage_logs for availability monitoring.
- Redis failures must never affect user requests.

## State Machine

pending -> ready -> implementing -> main_verify -> done
main_verify -> needs_fix -> implementing
done -> ready(next task)

## Tasks

| ID | Status | Purpose | Write scope | Acceptance / verification | Depends on |
| --- | --- | --- | --- | --- | --- |
| T-001 | done | Add migration 234, independent OAuth email config, and extended persisted state | monitor service/repository/handler/routes, migration 234, wiring/tests | config validation and migration tests pass; Ops alert email is disabled without changing report settings | none |
| T-002 | done | Record deduplicated real OpenAI/Codex account request outcomes | OpenAI gateway final-result paths, Redis helper, monitor service/tests | one outcome per account/request; success resets; classified failures increment; Redis failure is fail-open | T-001 |
| T-003 | done | Implement quota-cycle and sustained-unavailability notifications | monitor service and focused tests | third failure in 15 minutes alerts once; hourly reminder requires a new failure; quota unknown is silent; no recovery notifications | T-002 |
| T-004 | done | Simplify the OAuth drawer and separate channel settings | account API/view/drawer and frontend tests | only two event switches shown; advanced controls collapsed; independent email/PushPlus save and refresh correctly | T-003 |
| T-005 | done | Whole-change verification, GitHub push, and rolling deployment | tests/build/release artifacts, .159 then .189 | backend tests, frontend typecheck/build, smoke checks and two-node health checks pass; one rollback binary retained per host | T-004 |

## Active Task

None. All planned tasks are verified and complete.

## File Access Requests

None. Main-agent execution does not use delegated file-access requests.

## Transition Log

- 2026-09-03: clean baseline confirmed at 55daac165; T-001 -> implementing.
- 2026-09-03: T-001 -> done after migration/config/API focused tests.
- 2026-09-03: T-002 -> done after Redis dedupe, reset, window, and out-of-order tests plus handler compile checks.
- 2026-09-03: T-003 -> done after quota-transition, unknown-quota, sustained-failure, and reminder-cadence tests.
- 2026-09-03: T-004 -> done after OAuth drawer/API regression tests, TypeScript typecheck, and scoped ESLint.
- 2026-09-03: T-005 -> implementing.
- 2026-09-03: frontend production build, focused Vitest, TypeScript typecheck, and scoped ESLint passed.
- 2026-09-03: full backend `go test ./... -count=1` passed under Linux/WSL with Go 1.27.1; OAuth-focused tests also passed on Windows.
- 2026-09-03: release diff review confirmed migration 234 and no low-cost probe implementation files.
- 2026-09-03: commit `b589dfe85` pushed to `origin/codex/oauth-monitor-noise-reduction`; embedded Linux/amd64 release SHA-256 `e299409db6ae25ae5fa4243d3c4df3fe015f512fe678f0fdc4d457e4ea48f370`.
- 2026-09-03: `.159` then `.189` deployed and verified active on `18080`; both source trees point to `b589dfe85`, both keep exactly one rollback binary, and both origin HTTPS checks return health `200` and unauthenticated auth `401`.
- 2026-09-03: shared database migration 234 and all nine added monitor-state columns verified; OAuth email retained two recipients, Ops realtime alert email is disabled, and existing report state is preserved.
- 2026-09-03: T-005 -> done; workflow complete.
