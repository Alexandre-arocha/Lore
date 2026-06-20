-- Enums
CREATE TYPE source_kind AS ENUM ('language', 'framework', 'library', 'tool');
CREATE TYPE source_ingest_type AS ENUM ('github_markdown');
CREATE TYPE source_status AS ENUM ('active', 'syncing', 'error', 'disabled');
CREATE TYPE sync_status AS ENUM ('running', 'success', 'error');

-- sources: a documentation set (e.g. "Prisma", "Tailwind CSS", "Go").
CREATE TABLE sources (
    id             uuid PRIMARY KEY,
    slug           text NOT NULL UNIQUE,
    name           text NOT NULL,
    kind           source_kind NOT NULL,
    category       text NOT NULL,
    description    text NOT NULL DEFAULT '',
    logo_url       text,
    official_url   text NOT NULL,
    license        text,
    version        text,
    ingest_type    source_ingest_type NOT NULL DEFAULT 'github_markdown',
    ingest_config  jsonb NOT NULL DEFAULT '{}'::jsonb,
    nav            jsonb NOT NULL DEFAULT '[]'::jsonb,
    status         source_status NOT NULL DEFAULT 'active',
    last_synced_at timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_sources_kind ON sources (kind);
CREATE INDEX idx_sources_category ON sources (category);

-- documents: a single documentation page, pre-rendered to HTML at ingestion.
-- content_text holds plain text extracted from the page, used only to build
-- search_vector (title is weighted higher than body for ranking).
CREATE TABLE documents (
    id            uuid PRIMARY KEY,
    source_id     uuid NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    slug          text NOT NULL,
    path          text NOT NULL,
    title         text NOT NULL,
    content_html  text NOT NULL DEFAULT '',
    content_text  text NOT NULL DEFAULT '',
    toc           jsonb NOT NULL DEFAULT '[]'::jsonb,
    position      integer NOT NULL DEFAULT 0,
    word_count    integer NOT NULL DEFAULT 0,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(content_text, '')), 'B')
    ) STORED,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, slug)
);

CREATE INDEX idx_documents_search_vector ON documents USING gin (search_vector);
CREATE INDEX idx_documents_source_position ON documents (source_id, position);

-- sync_runs: a log entry per ingestion run.
CREATE TABLE sync_runs (
    id                  uuid PRIMARY KEY,
    source_id           uuid NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    status              sync_status NOT NULL DEFAULT 'running',
    documents_processed integer NOT NULL DEFAULT 0,
    error_message       text,
    started_at          timestamptz NOT NULL DEFAULT now(),
    finished_at         timestamptz
);

CREATE INDEX idx_sync_runs_source ON sync_runs (source_id, started_at DESC);
