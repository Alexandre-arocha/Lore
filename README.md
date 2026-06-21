# Lore

> Hub que reúne a documentação de várias linguagens e frameworks num só lugar, com
> busca unificada, navegação rápida e leitura limpa. Interface, busca e curadoria
> pensadas para o dev brasileiro — a documentação em si permanece no idioma original.

## Stack

- **Backend:** Go 1.23+, Gin, River (jobs), pgx + sqlc, golang-migrate, PostgreSQL.
- **Busca:** Postgres full-text search (`tsvector` + GIN).
- **Render:** goldmark (GFM) + chroma — Markdown vira HTML com highlight já na ingestão.
- **Frontend:** Next.js 16 (App Router), TypeScript, Tailwind v4, shadcn/ui.

## Pré-requisitos

- Go 1.23+
- Node 20+ e npm
- Docker (para o Postgres local) — ou um Postgres gerenciado (Neon/Supabase)

## Setup rápido

### 1. Banco de dados

```bash
docker compose up -d postgres
```

Isso sobe um Postgres em `localhost:5432` (user/senha/db = `lore`).

### 2. Backend

```bash
cd api
cp .env.example .env        # ajuste DATABASE_URL / ADMIN_TOKEN / GITHUB_TOKEN se preciso
go run ./cmd/server
```

A API sobe em `http://localhost:8080`. Teste:

```bash
curl http://localhost:8080/api/health
# {"db":"ok","status":"ok"}
```

### 3. Frontend

```bash
cd web
cp .env.example .env.local
npm install
npm run dev                 # http://localhost:3000
```

## Estrutura

```
/api          # backend Go (Gin + River)
/web          # frontend Next.js 16
/migrations   # golang-migrate (dentro de /api)
docker-compose.yml
```

## Popular dados

```bash
cd api
go run ./cmd/migrate up    # cria o schema
go run ./cmd/seed          # cadastra as sources (config-driven, seed/sources.json)

# dispara a ingestão de uma source (precisa do servidor rodando)
curl -X POST -H "X-Admin-Token: $ADMIN_TOKEN" \
  http://localhost:8080/api/admin/sources/rust/sync
```

## Desenvolvimento

| Tarefa | Comando |
|---|---|
| Testes backend | `cd api && go test ./...` |
| Build backend | `cd api && go build ./...` |
| Regenerar sqlc | `cd api && sqlc generate` |
| Migrations up | `cd api && go run ./cmd/migrate up` |
| Seed das sources | `cd api && go run ./cmd/seed` |
| Regenerar CSS do chroma | `cd api && go run ./cmd/gen-chroma-css` |
| Checagem de tipos | `cd web && npx tsc --noEmit` |
| Build frontend | `cd web && npm run build` |

## Roadmap (fases)

0. **Setup** — estrutura, health endpoint. ✅
1. **Schema & migrations** — `sources`, `documents`, `sync_runs`. ✅
2. **Ingestão** — `SyncSourceJob` (tarball → goldmark+chroma → toc → upsert → nav). ✅
3. **API de leitura** — sources, source detail, doc. ✅
4. **Busca** — `websearch_to_tsquery`, `ts_rank`, `ts_headline`. ✅
5. **Frontend** — home, leitor (sidebar + ToC), command palette (Ctrl/Cmd+K), dark mode. ✅
6. **Expandir & agendar** — mais sources, atribuição, SEO. Re-sync periódico fica como evolução.

## Licença e atribuição

Lore apenas indexa e reexibe documentação pública mantendo o idioma original. Cada
source registra sua `license` e `official_url`, e toda página exibe atribuição visível
à fonte com link para o original.
