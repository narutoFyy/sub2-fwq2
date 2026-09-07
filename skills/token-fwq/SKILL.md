---
name: token-fwq
description: Runtime map and deployment notes for the Token FWQ servers, including SSH hosts, internal addresses, ports, systemd services, Sub2API authentication, the shared PostgreSQL 15 database, and the standalone token draw site. Use before inspecting, deploying, migrating, routing, or troubleshooting these services.
---

# Token FWQ Runtime Map

Use this skill as the first source of truth for the current Token FWQ deployment. Confirm service state and port ownership before changing anything. Keep the existing services online unless the user explicitly requests a planned stop.

## Server Map

| Host | Role | Important addresses |
| --- | --- | --- |
| `23.148.228.159` | First application server | Internal `10.0.0.3` |
| `23.148.228.189` | Second application server | Internal `10.0.0.2` |
| `23.148.228.128` | Dedicated database server | Internal `10.0.0.4` |
| `47.85.33.153` | Deprecated legacy server | Former `vuwlmail.com` source; services migrated away |
| `47.252.50.237` | Separate application/database server | Do not merge or modify unless explicitly requested |

The `.159`, `.189`, and `.128` machines share private network `10.0.0.0/24`. Public traffic normally arrives through Cloudflare. All current services run on `.128`, `.159`, and `.189`; `.153` is deprecated and should not receive new deployments or production traffic.

## SSH Credentials

Local private operational record. Do not commit this file to Git or include these credentials in external messages.

| Host | User | Password |
| --- | --- | --- |
| `23.148.228.128` | `root` | `khLeU8yhp0fX` |
| `23.148.228.159` | `root` | `tb5IZlJ01dFw` |
| `23.148.228.189` | `root` | `tb5IZlJ01dFw` |

## Port Ownership

| Host | Port | Service | Notes |
| --- | ---: | --- | --- |
| `.159`, `.189` | `18080` | `sub2api-153.service` | Sub2API replicas copied from `.153`; keep running |
| `.189` | `18083` | `shitou-academic-figures.service` | Shitou Academic scientific-figure workflow; Docker production image, shared Sub2API auth, and editable figure storage |
| `.159` | `18081` | `token-draw-preview.service` | Standalone token draw site and same-origin auth proxy |
| `.128` | `5432` bound to `10.0.0.4` | `fwq-postgres` / PostgreSQL 15 | Shared Sub2API database |
| `.128` | `16379` bound to `10.0.0.4` | `fwq-sub2api-redis` | Shared Sub2API Redis |
| `.128` | `3306`, `6379` bound to `10.0.0.4` | Existing gift-chat MySQL/Redis | Preserve; unrelated to Sub2API PostgreSQL/Redis |
| `.159`, `.189` | `80`, `443` | Existing Docker services | `gift-chat-frontend`; host routing also proxies `yf-mail.com` to Sub2API |
| `.159`, `.189` | `8081` container-side | `gift-chat-backend` | Existing Docker service; do not reuse |
| `.153` | `8080` | `sub2api.service` | Deprecated legacy Sub2API; may remain active for rollback/reference only |

Never deploy the draw site on `18080`, `80`, `443`, or `8081`. The draw service uses `18081` and is intentionally not behind the existing load balancer yet.

## Deployment Summary

| Server | Port | Project | Database |
| --- | ---: | --- | --- |
| `23.148.228.159` | `18081` | Token draw site (`ck.yf-mail.com`) | PostgreSQL `sub2api` on `23.148.228.128:5432` / `10.0.0.4` |
| `23.148.228.159` | `18080` | Sub2API relay | PostgreSQL `sub2api` on `23.148.228.128:5432` / `10.0.0.4` |
| `23.148.228.189` | `18080` | Sub2API relay | PostgreSQL `sub2api` on `23.148.228.128:5432` / `10.0.0.4` |
| `23.148.228.189` | `18083` | Shitou Academic scientific figures | Existing Sub2API auth; `/srv/xcard-uploads/scientific-figures` storage; `/opt/shitou-academic/my-image-sci` runtime mount |
| `23.148.228.128` | `5432` | PostgreSQL 15 (`fwq-postgres`) | Database host for Sub2API and token draw data |

## Release Retention Policy

Apply this rule to every Token FWQ project release unless a project-specific note explicitly requires longer retention:

- Keep the live production release and exactly one rollback release only.
- Before promoting a verified candidate, preserve the current production artifact as the new rollback.
- After the new production release is verified, delete the older rollback artifact or release directory. Do not accumulate historical rollback images or release directories on the servers.
- Never delete the live production artifact, the newly retained rollback, external source backups, persistent application storage, `.env` files, or shared Sub2API, database, Redis, and card-gift services as part of release cleanup.

## Current `.189` Application State

Observed and verified on `2026-08-22` after the figure-type-driven editable SCI scene deployment:

- `sub2api-153.service` is `active` and `enabled`; it listens on `*:18080`. `GET /api/v1/auth/me` without credentials returns `401`, and `/healthz` returns `200`.
- `shitou-academic-figures.service` is `active` and `enabled`; it listens on `0.0.0.0:18083` through Docker container `shitou-academic-figures`.
- The active image is `shitou-academic:production`, built as image ID prefix `e0b9f659feae` from clean Linux `.next` output after the 2026-08-22 reference-image and prompt-inspection release.
- The single retained rollback image is `shitou-academic:rollback-20260822-before-reference-image`, preserving the prior production image digest `sha256:79353919a6a8798118709a162c58673d550fcf14f7d1794faabd3b49dd2185d4`.
- The active figure workflow is figure-type-driven: paper purpose and specific figure type produce a persisted plan containing layout, 3-6 modules, and connections. The SVG uses only local labels, leader lines, anchors, and directional arrows; it no longer renders fixed three-column opaque cards.
- Server-side source of record is `/opt/shitou-academic/server-source-20260821-153705`; the pre-change bundle is `/opt/shitou-academic/backups/figure-type-20260821-173827/source.tar.gz`.
- The shared `scientific_figure_job` table now has additive `figure_plan jsonb` storage. Legacy `stage_labels` rows remain supported.
- Future figure releases should edit the source directly over SSH on `.189`, move any rollback snapshot outside the source tree, run a clean Linux Docker build with the source `.next` directory removed first, build a temporary candidate image/container on `18084`, run route and authorization probes, then tag the current production image as the single rollback before restarting `shitou-academic-figures.service`. Do not copy Windows `.next` output into the Linux image.
- The service mounts `/opt/shitou-academic/my-image-sci` read-only and `/srv/xcard-uploads/scientific-figures` for scientific-figure assets and generated output. The container resolves the Linux `sharp` module.
- `my-image-sci/scripts/build-scene.mjs` requires the top-level `scene.json.version` to be numeric `1`; do not change it to a renderer-specific version. The application scene renderer uses metadata or additive fields for future evolution instead. The pre-fix source file is backed up at `/opt/shitou-academic/backups/scene-version-20260822-0250/scene.ts`.
- The figure editor now accepts an optional PNG/JPEG/WebP reference image as multipart input. With a reference image it invokes the existing `/opt/shitou-academic/my-image-sci/scripts/edit.mjs` path (`/images/edits`) and includes the normalized reference image in the downloaded bundle. The editor also exposes a no-generation prompt preview through `POST /api/figures/prompt` with `previewOnly: true`.
- The `/figures` recent-task sidebar is collapsed per task; expanding a task shows its derived status history, stored bottom-layer description, and editable asset links.
- Anonymous `GET /figures` returns `307` to `/sign-in?redirect=%2Ffigures`; anonymous `POST /api/figures/prompt` returns `401`.
- The card-gift application is the Docker Compose project `deploy` from `/opt/tg-message/deploy/docker-compose.app.yml`. On `.189`, `gift-chat-frontend` (`deploy-frontend`) and `gift-chat-backend` (`deploy-backend`) are both `running`; the frontend owns ports `80/443`, and the backend uses container port `8081`.
- The candidate process and its `18084` proxy listeners were stopped after verification. Its candidate tag shares the active production image and can be removed when the local Docker control plane responds; do not restart Docker merely to clear this metadata because it also hosts card-gift and shared services. Keep the production image, the single rollback image, `/opt/shitou-academic/.env`, `my-image-sci`, `/opt/shitou-academic/server-source-20260821-153705`, its external backups, and scientific-figure storage. Do not remove the card-gift containers or the shared Sub2API/database services during future cleanup.

## Deployed Token Draw Site

- Public URL: `https://ck.yf-mail.com`
- Direct test URL: `http://23.148.228.159:18081`
- Release root: `/opt/token-draw/releases/`
- Current release symlink: `/opt/token-draw/current`
- Current release: `/opt/token-draw/releases/20260822-story-reboot-ch4-v1`
- Rollback release: `/opt/token-draw/releases/20260822-story-reboot-ch3-v1` (retained for rollback).
- Release retention audit (2026-08-22): `.159` keeps only the current release and this single rollback release under `/opt/token-draw/releases/`; older Token draw releases were removed after verification. Future cleanup should preserve at most these two directories.
- Recent story deployment: rebooted chapter 4 `锈冠醒来` on 2026-08-22; only the static frontend/story bundle changed. The service remains on port `18081`.
- Unit: `token-draw-preview.service`
- Exec: `/usr/bin/python3 /opt/token-draw/current/server.py`
- Static files: `/opt/token-draw/current/dist`
- Health endpoint: `http://127.0.0.1:18081/healthz`
- Upstream auth service: `http://127.0.0.1:18080`

`ck.yf-mail.com` is a proxied Cloudflare A record pointing only to `23.148.228.159`; it is not load balanced across `.159` and `.189`. The `gift-chat-frontend` Nginx container terminates HTTPS and proxies this host to `http://172.18.0.1:18081`. Its bind-mounted config is `/opt/tg-message/deploy/nginx.conf`, with the pre-change backup at `/opt/tg-message/deploy/nginx.conf.backup-20260818-040621`. Dedicated origin certificate files are `/opt/tg-message/deploy/certs/ck.yf-mail.com.{crt,key}`; the current self-signed certificate has SAN `ck.yf-mail.com` and expires `2028-11-20`. Replacing the config file inode requires restarting `gift-chat-frontend` so Docker remounts it; an ordinary Nginx reload may continue reading the old bind-mounted inode.

The service is a Python static server with a restricted same-origin proxy for:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/login/2fa`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`

It serves the Vue build and proxies only those auth paths to Sub2API. The browser never connects directly to PostgreSQL.

The game API is served directly by the draw service:

- `GET /api/v1/game/state`
- `GET /api/v1/game/history`
- `POST /api/v1/game/draw`
- `POST /api/v1/game/draw-ten`
- `POST /api/v1/game/free-draw`
- `POST /api/v1/game/claim-reward`

Every game route first validates the bearer token through local Sub2API `/api/v1/auth/me`; the draw site does not store passwords or maintain a second account system.

## Shared Database

Sub2API on `.159` and `.189` reads its database configuration from `/opt/sub2api-153/app.env`:

```text
DATABASE_HOST=10.0.0.4
DATABASE_PORT=5432
DATABASE_DBNAME=sub2api
DATABASE_SSLMODE=disable
REDIS_HOST=10.0.0.4
REDIS_PORT=16379
```

PostgreSQL 15 runs in Docker container `fwq-postgres` on `.128`. The primary user table is `public.users`; its live wallet field is `balance numeric`. Normal user auth responses include `email`, `username`, and `balance`.

Use the existing Sub2API service/API for authentication and current balance. Do not duplicate password hashes or create a second user account system in the draw site.

The draw migration added these isolated tables in the `sub2api` database:

- `public.token_draw_profiles`
- `public.token_draw_inventory`
- `public.token_draw_history`
- `public.token_draw_wallet_events`
- `public.token_draw_stage_rewards`

The tables are owned by database administrator `admin`; runtime role `sub2api` has `SELECT`, `INSERT`, `UPDATE`, and `DELETE` on all four. The game updates `public.users.balance` in the same PostgreSQL transaction as draw inventory/history or collection-reward writes.

Pre-migration backup verified on `2026-08-18`:

- Path: `/opt/backups/sub2api-before-token-draw-20260818-101201.dump`
- Format: PostgreSQL custom archive; `pg_restore -l` succeeded
- Size: `132,644,153` bytes
- SHA256: `6da81539eba443f038cadd6490ebb71f4b899408068830aaa7864e5305dc3825`

## Sub2API Deployment State

Observed and verified on `2026-09-07` after the in-site chat frontend release (`0.2.2`):

- `.159` and `.189` run Sub2API replicas from `/opt/sub2api-153/bin/sub2api` on port `18080` via `sub2api-153.service`.
- Both replicas use the same PostgreSQL and Redis services on `.128`. Environment remains `/opt/sub2api-153/app.env`.
- Rolling order for binary releases: verify `.159` first, then copy the verified candidate to `.189` over private `10.0.0.0/24` and promote. Keep existing services online except the Sub2API unit restart.
- Live binary SHA256 on both application nodes: `b10195669d0056f45fdfb526009a05c2e4574f63f7c5d931c81915bc6e831e88`.
- Single rollback binary on both nodes: `/opt/sub2api-153/bin/sub2api.rollback`, SHA256 `fed60a722973b6284e4282c07213793754eb5e8d4e33c51c9e698c40e6ee0de6` (previous `2026-09-06` production).
- Candidate copy is also at `/opt/sub2api-153/stage/sub2api-0.2.2-chat` with the live digest. Future releases should replace this staged file rather than accumulate extra binaries under `bin/`.
- This release embeds the Vue chat page at `/chat`. No database migration. History is browser-local IndexedDB only.
- Unauthenticated `/api/v1/auth/me` returns `401`. `GET /chat` returns `200` SPA HTML. The chat chunk `assets/ChatView-CU8UL5RW.js` returns `200`.
- Public checks through Cloudflare: `https://yf-mail.com/chat` `200`, `https://yf-mail.com/assets/ChatView-CU8UL5RW.js` `200`, `https://yf-mail.com/api/v1/auth/me` `401`.
- `/healthz` still returns HTTP `200` through the SPA fallback, so use service state, listener state, and the auth API check together rather than treating that route alone as a process-health proof.
- The copied source tree `/opt/sub2api-153/source` on the servers was not updated in this frontend-embed binary release. Do not assume GitHub or that source tree is newer than the live binary.
- `.153:8080` remains deprecated and is not part of the current production path.

Future Sub2API binary releases: keep live `bin/sub2api` plus exactly one `bin/sub2api.rollback`. Before promoting a verified candidate, copy the current live binary to the rollback slot. After verification, delete only the previous rollback, not `app.env`, `data/`, source backups, or shared database/Redis/card-gift services.

## Recent `.153` Database Operation

Historical operation verified on `2026-08-18` for the former `vuwlmail.com` database on `47.85.33.153` only. `.153` is now deprecated; this record does not describe the current `.128` shared database.

- Pre-change PostgreSQL backup: `/opt/backups/vuwlmail-sub2api-before-clear-balances-20260818.dump`
- Backup size: `130,915,571` bytes
- Backup SHA256: `a293dfc48ec453924d03376f4bdae461052d4ecf26755333c2aa42b6b5bc22b5`
- Before update: `168` users, `81` non-zero balances, total `4626.65220475`
- Operation: set `public.users.balance = 0` where the balance was non-zero
- After update: `168` users, `0` non-zero balances, total `0`
- `public.users.frozen_balance` was left unchanged
- User accounts, API keys, announcements, and transaction/history records were left unchanged
- `sub2api.service` remained active after the update; `.128`, `.159`, and `.189` were not modified by this operation

Treat this as a historical pre-deprecation state of `.153`; verify the live shared database on `.128` before any balance migration or reconciliation.

## Cloudflare Routing

Observed in the Cloudflare dashboard and verified against both origins on `2026-08-18`:

- `stonetradex.com` and `yf-mail.com` each have a Cloudflare load balancer.
- Both load balancers share pool `token` (`c711ebe49511a413df3fd9feb7dd8ba6`).
- Pool routing is random with two enabled, equal-weight origins:
  - `app-a`: `23.148.228.159:443`, 50%.
  - `app-b`: `23.148.228.189:443`, 50%.
- `stonetradex.com` serves the existing Xcard application.
- `yf-mail.com` is routed by the origin reverse proxy to Sub2API on local port `18080`.
- Direct origin checks for `https://yf-mail.com/healthz` returned `200` on both `.159` and `.189`.

Known monitoring mismatch:

- Shared monitor `xcard-https-health` checks HTTPS port `443`, path `/api/health`, expected status `200`, every 60 seconds.
- It does not validate TLS certificates and has no Host request header.
- `yf-mail.com/api/health` returns `404`; the Sub2API health path is `/healthz`.
- Therefore a green `token` pool proves the shared endpoint/default Xcard check passed, but does not independently prove the `yf-mail.com` virtual host and Sub2API process are healthy.

Do not change this shared monitor casually because it also protects `stonetradex.com`. A correct fix needs either a dedicated `yf-mail.com` pool/monitor or a shared origin health endpoint that validates both applications.

`tokentradex.com` is not the `token` pool and was not in this Cloudflare account. Its public DNS resolved to GoDaddy-hosted addresses and served `DPS/2.0.0`; do not treat it as part of this load-balanced deployment without a fresh DNS check.

## Current Feature Boundary

Completed:

- Vue 3 single-draw experience with 10 original generated creatures, collection progress, quantities for duplicates, draw history, and completion-reward state.
- Same-origin relay login using the existing Sub2API email/password flow.
- Existing Sub2API two-factor login flow is supported.
- Each draw atomically deducts `1.00` from shared `public.users.balance`; insufficient balance is rejected before any game write.
- The active season has 30 configured cards: 20 three-star, 7 four-star, and 3 five-star. Rarity rates are 90% / 8% / 2%; the rarity distribution is configuration-driven and later cards are included dynamically.
- Every tenth draw guarantees at least four stars, including ten consecutive single draws. The pity counter guarantees a five-star within 50 draws and includes free draws.
- There is no collection-completion draw pity: a 50th draw only forces one randomly selected five-star from the five-star pool, and duplicates remain possible.
- Ten-pull responses render as a staggered 5x2 card grid with individual flip animations; single draws retain the single-card flip animation.
- One free draw is available every 48 hours. Paid draws cost 1.00 shared balance; all draw rarities and evolution stages grant 0 balance, and full configured collection grants 100.00 once.
- Duplicate creatures increment inventory quantity. At quantity `3`, `6`, and `9`, the same creature unlocks its `少年`, `成熟`, and `高阶` visual stages. Quantity is cumulative and never reset; the stage is computed server-side from `token_draw_inventory.quantity`, so no new database column or migration is needed.
- The public `game-config.json` has four `evolution_stages` per creature. Stage 1 uses `/assets/creatures/cards/*.webp`; stages 2-4 use `/assets/creatures/evolutions/<creature-id>/stage-2.webp` through `stage-4.webp`.
- Draw responses and history always return the stage-1幼崽 image. The current evolved image is only used by the collection state and the clickable four-stage evolution gallery.
- Completing all configured cards unlocks one `100.00` shared-balance reward per account for season `brightlings-02`.
- Draw request idempotency keys, profile/user row locks, and a wallet event ledger prevent duplicate charges, duplicate rewards, and concurrent overspending.
- The draw site is deployed and isolated on `.159:18081`.
- Cloudflare DNS and HTTPS reverse proxy are active at `https://ck.yf-mail.com`.
- The standalone site includes a frontend-static `篇章` sidebar view for season `奇光原野`. The rebooted chapters 1 `影子先到了`, 2 `第二个我`, 3 `门里的人没有影子`, and 4 `锈冠醒来` are published; chapters 5-9 remain locked continuations and chapters 10-12 remain planned. Story content does not add database tables or API routes.
- The current draw release is season `brightlings-02` (`奇光原野 · 星痕回路`) with 30 cards: 20 are 3-star, seven are 4-star, and three are 5-star. Rarity rates are 90% / 8% / 2%; there is no missing-card completion guarantee. The prior season `brightlings-01` data remains isolated in the same additive tables.

Not implemented: creature skills, trading, gameplay effects, equipment, time cards, multiple collections, or recurring rewards.

## Operational Checks

Run these checks before and after deployment:

```bash
# Run on both .159 and .189 for the Sub2API replica.
systemctl is-active sub2api-153.service
ss -lntp | grep ':18080'
curl -fsS http://127.0.0.1:18080/healthz
curl -i http://127.0.0.1:18080/api/v1/auth/me
curl -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18080/chat
grep -a -q ChatView /opt/sub2api-153/bin/sub2api && echo chat-embedded

# The draw preview exists only on .159.
systemctl is-active token-draw-preview.service
ss -lntp | grep ':18081'
curl -fsS http://127.0.0.1:18081/healthz
curl -i http://127.0.0.1:18081/api/v1/game/state
curl -I http://127.0.0.1:18081/game-config.json
curl -I http://127.0.0.1:18081/assets/creatures/cards/arclet.webp
```

The unauthenticated Sub2API and draw-game requests should each return `401`. Public draw health, config, and image requests should return `200`.

When releasing a new draw build, use a new directory under `/opt/token-draw/releases/`, update `/opt/token-draw/current`, run `systemctl daemon-reload`, restart only `token-draw-preview.service`, and repeat the checks above. Old draw releases are removed after verification; a rollback requires redeploying a separately preserved artifact.

The draw tables are additive. Do not restore the full database merely to replace the draw application; a full restore would overwrite unrelated production changes.
