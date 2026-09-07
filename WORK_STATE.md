# In-site chat (browser-local, text-only)

Status: complete
Current task: none
Execution mode: state-main
Plan topology: linear

## Outcome

Logged-in users can daily-chat with their own Sub2API key and a model from `/v1/models`. History stays in this browser, namespaced by user id. V1 is text-only. Dedicated `/chat` shell, not the admin table chrome.

## Tasks

| ID | Status | Purpose |
| --- | --- | --- |
| T-001 | done | IndexedDB/memory store, SSE parser, gateway client, markdown sanitize |
| T-002 | done | Dedicated chat shell, fonts, paper/ink theme |
| T-003 | done | Conversation list, message stream, composer |
| T-004 | done | Key/model pickers and empty/error states |
| T-005 | done | Route `/chat`, sidebar, i18n |
| T-006 | done | Component tests + focused verification |

## Verification

- `pnpm exec vitest run src/features/chat/__tests__` — 7 files, 28 tests passed (after live-object persist test).
- `pnpm exec vue-tsc --noEmit` — passed.
- Live model/billing against production keys: not verified here.

## Transition Log

- 2026-09-07: `$work` started. T-001 implementing.
- 2026-09-07: T-001 done (17 data-layer tests).
- 2026-09-07: T-002–T-005 implemented as `/chat` shell + session engine + nav/i18n.
- 2026-09-07: T-006 ChatView/session tests + typecheck. Fixed persist replacing live message objects during stream.
- 2026-09-07: complete.
