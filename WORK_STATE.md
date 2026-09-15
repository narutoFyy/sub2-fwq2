# Model Radar implementation

Status: complete
Current task: none
Execution mode: state-main
Plan topology: linear

## Rules

- Execute one radar task at a time and verify it before activating the next task.
- Preserve all pre-existing uncommitted work; radar writes stay within the new model-radar files plus integration points and this state file.
- Support only active OpenAI groups; never expose account IDs, names, credentials, or upstream response secrets to regular users.
- Run locally only; do not deploy, migrate production, commit, or push.
- Keep IQ efficiency and station recommendations as explicit unimplemented placeholders; do not fabricate metrics.

## State Machine

pending -> ready -> implementing -> main_verify -> done
main_verify -> needs_fix -> implementing
done -> ready(next task)

## Tasks

| ID | Status | Depends on | Scope |
| --- | --- | --- | --- |
| T-RADAR-001 | done | none | Backend schema, OpenAI group probe service, APIs, and scheduler integration |
| T-RADAR-002 | done | T-RADAR-001 | User radar page, navigation entry, and admin configuration/review UI |
| T-RADAR-003 | done | T-RADAR-002 | Focused tests, local build, browser verification, and diff review |

## Active Task

None. Local verification completed; no deployment, migration execution, commit, or push performed.

Reuse source: existing Sub2API OpenAI account test service and group/account repositories.

Custom-code boundary: radar persistence, aggregation, classification, route handlers, and scheduler glue only; upstream request/auth behavior remains supplied by AccountTestService.

## File Access Requests

None; state-main execution does not delegate file access.

## Transition Log

- 2026-09-16: user explicitly invoked `$work`; confirmed a linear state-main implementation for the OpenAI-only model radar.
- 2026-09-16: T-RADAR-001 ready -> implementing.
- 2026-09-16: T-RADAR-001 implementing -> done (schema, repository, OpenAI group probe, APIs, scheduler, cleanup).
- 2026-09-16: T-RADAR-002 pending -> done (user/admin pages, routes, navigation, API clients, i18n).
- 2026-09-16: T-RADAR-003 pending -> done (focused Go tests, full compile, frontend typecheck/build, local route smoke test, diff check).

---

# Local Sub2API account importer

Status: complete
Current task: none
Execution mode: state-main
Plan topology: linear

## Rules

- Execute one importer task at a time and verify it before activating the next task.
- Preserve all pre-existing uncommitted work; importer writes stay under `backend/cmd/sub2api-importer/` plus this state file.
- Reuse the native Sub2API Admin API for account creation, updates, identity matching, expiry logic, and group binding.
- Do not connect to a real Hub, deploy, migrate, commit, or push.
- Never log or persist Admin Keys, access tokens, or refresh tokens.
- Final verification: focused Go tests, integration tests against an `httptest` Hub, browser workflow checks, screenshots at 1440x900 and 390x844, and diff review.

## State Machine

pending -> ready -> implementing -> main_verify -> done
main_verify -> needs_fix -> implementing
done -> ready(next task)

## Tasks

| ID | Status | Depends on | Scope |
| --- | --- | --- | --- |
| T-IMPORT-001 | done | none | Loopback server, local request guard, Hub URL/client, groups/import transport |
| T-IMPORT-002 | done | T-IMPORT-001 | Bounded credential parsing, wrapper normalization, safe preview, import model and idempotency |
| T-IMPORT-003 | done | T-IMPORT-002 | Embedded Swiss-style operational UI and browser flow |
| T-IMPORT-004 | done | T-IMPORT-003 | Integration verification, README, local server handoff |

## Active Task

No active task. All importer tasks passed their configured verification gates. The built application is running at `http://127.0.0.1:8765` from `output/sub2api-importer/sub2api-importer.exe`.

Reuse source: current fork's `GET /api/v1/admin/groups/all?platform=openai` and `POST /api/v1/admin/accounts/import/codex-session` contracts.

Custom-code boundary: local loopback hosting, transport validation, request guarding, normalization, UI, and tests only. Existing Sub2API import services remain untouched.

## File Access Requests

None; state-main execution does not delegate file access.

## Transition Log

- 2026-09-04: user explicitly invoked `$work`; confirmed state-main, linear topology, and API wrap/integration reuse path.
- 2026-09-04: T-IMPORT-001 pending -> ready -> implementing.
- 2026-09-04: T-IMPORT-001 implementing -> main_verify -> done; focused Go tests passed against an `httptest` Hub, including redirect and redaction checks.
- 2026-09-04: T-IMPORT-002 pending -> ready -> implementing.
- 2026-09-04: T-IMPORT-002 implementing -> main_verify -> done; table tests, integration tests, `go vet`, and a 3-second fuzz run passed.
- 2026-09-04: T-IMPORT-003 pending -> ready -> implementing.
- 2026-09-04: T-IMPORT-003 implementing -> main_verify -> done; fake-Hub browser workflow, invalid JSON, upload, preview, import, idempotency replay, 1440x900/390x844 screenshots, zero-overflow check, and clean new-session console passed.
- 2026-09-04: T-IMPORT-004 pending -> ready -> implementing.
- 2026-09-04: T-IMPORT-004 implementing -> main_verify -> done; final Go tests (78.8% package coverage), `go vet`, JavaScript syntax, formatting, secret scans, binary health smoke test, final browser stale-group check, and live health check passed.
- 2026-09-04: importer plan complete; Windows binary SHA256 `76D257D18F1A5EFA32524877DCCBD489BBE2EE95BDCECC4733823DD465CA46A1` is running on loopback port 8765.

---

## Previous Completed Run

# OAuth account monitoring completion

Status: complete
Current task: none
Execution mode: state-main
Plan topology: linear

## Rules

- Execute one OAuth task at a time and review it before activating the next task.
- Preserve all existing uncommitted changes at `aca3478ba`; do not revert or rewrite unrelated work.
- Freeze all low-cost probing, OpenAI scheduling, probe migration, probe UI, and related dependency-injection changes.
- Do not run automated tests or add test files during this run.
- Do not connect to `23.148.228.159` or `23.148.228.189`, deploy, run production migrations, perform real upstream probes, commit, or push.
- Verification is limited to static code review and `git diff --check`.
- Keep the OAuth monitor's Redis/database leader lock and legacy APIs compatible.
- Never log OAuth credentials or PushPlus tokens; API responses must keep PushPlus tokens masked.

## State Machine

pending -> ready -> implementing -> main_verify -> done
main_verify -> needs_fix -> implementing
done -> ready(next task)

## Tasks

| ID | Status | Depends on | Scope |
| --- | --- | --- | --- |
| T-OAUTH-001 | done | none | Backend timeout, bounded concurrency, error propagation, notification success semantics, manual-cycle API |
| T-OAUTH-002 | done | T-OAUTH-001 | Always-visible entry, resilient right drawer, refresh/manual check, sequential authoritative saves |
| T-OAUTH-003 | done | T-OAUTH-002 | OAuth-only scope, compatibility, secret-handling and static diff review |

## Active Task

No active task. All OAuth-only tasks passed their configured static verification gates.

## File Access Requests

None; state-main execution does not delegate file access.

## Transition Log

- 2026-09-02: user narrowed the confirmed plan to OAuth monitoring only and explicitly invoked `$work`.
- 2026-09-02: execution mode confirmed as state-main, topology linear; T-OAUTH-001 -> implementing.
- 2026-09-02: low-cost probing, OpenAI scheduling, automated tests, servers, deployment, commit and push frozen by user instruction.
- 2026-09-02: T-OAUTH-001 -> main_verify -> done; static diff and route/secret review passed, `git diff --check` clean. Local Go formatter was unavailable and automated checks remain intentionally skipped.
- 2026-09-02: T-OAUTH-002 -> implementing.
- 2026-09-02: T-OAUTH-002 -> main_verify -> done; drawer flow, persistent entry, mixed selection, sequential save/reload and API export reviewed; `git diff --check` clean.
- 2026-09-02: T-OAUTH-003 -> implementing.
- 2026-09-02: T-OAUTH-003 -> main_verify -> done; duplicate settings controls, legacy endpoints, token masking/logging and frozen-scope boundaries reviewed; whole-worktree `git diff --check` clean.
- 2026-09-02: OAuth-only plan complete. Automated tests, browser checks, servers, deployment, commit and push remain intentionally untouched.
