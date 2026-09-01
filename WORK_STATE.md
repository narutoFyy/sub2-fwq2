# Multi-proxy account routing

Status: in_progress
Current task: T-001
Execution mode: state-main
Plan topology: linear

## Rules

- Execute one task at a time.
- Preserve existing user changes (worktree was clean at start).
- Keep legacy `accounts.proxy_id` and `accounts.concurrency` behavior when no relation rows exist.
- Verify each task before activating the next task.

## State Machine

pending -> ready -> implementing -> self_check -> main_verify -> done
main_verify -> needs_fix -> implementing
done -> ready(next task)

## Tasks

| ID | Status | Depends on |
| --- | --- | --- |
| T-001 | implementing | none |
| T-002 | pending | T-001 |
| T-003 | pending | T-001, T-002 |
| T-004 | pending | T-003 |

## Active Task

T-001: add account-proxy persistence and legacy compatibility.

## File Access Requests

None.

## Transition Log

- 2026-09-02: work started; T-001 -> implementing.
