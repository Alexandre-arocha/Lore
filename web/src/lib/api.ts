// Cliente HTTP fino para a API do Atlas.

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** Monta a URL absoluta de um endpoint da API. */
export function apiUrl(path: string): string {
  return `${API_URL}${path.startsWith("/") ? path : `/${path}`}`;
}

export type HealthStatus = {
  status: string;
  db: string;
};

export type Source = {
  slug: string;
  name: string;
  kind: string;
  category: string;
  description: string;
  logo_url: string | null;
  official_url: string;
  license: string | null;
  version: string | null;
  status: string;
  nav?: NavNode[];
  last_synced_at: string | null;
};

export type NavNode = {
  title: string;
  slug?: string;
  children?: NavNode[];
};

export type DocumentPage = {
  source: {
    slug: string;
    name: string;
    official_url: string;
    license: string | null;
  };
  slug: string;
  path: string;
  title: string;
  content_html: string;
  toc: Array<{
    title: string;
    anchor: string;
    children?: Array<{ title: string; anchor: string }>;
  }>;
  position: number;
  word_count: number;
  updated_at: string | null;
};

export type SearchResult = {
  source: {
    slug: string;
    name: string;
    official_url: string;
    license: string | null;
  };
  document_url: string;
  slug: string;
  title: string;
  excerpt: string;
  rank: number;
};

type ItemsResponse<T> = {
  items: T[];
};

/** Consulta /api/health. Retorna null se a API estiver inacessível. */
export async function getHealth(): Promise<HealthStatus | null> {
  return getJson<HealthStatus>("/api/health");
}

export async function getSources(): Promise<Source[]> {
  const data = await getJson<ItemsResponse<Source>>("/api/sources");
  return data?.items ?? [];
}

export async function getSource(slug: string): Promise<Source | null> {
  return getJson<Source>(`/api/sources/${encodeURIComponent(slug)}`);
}

export async function getDocument(
  source: string,
  docSlug: string
): Promise<DocumentPage | null> {
  const encodedDocSlug = docSlug
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");
  return getJson<DocumentPage>(
    `/api/sources/${encodeURIComponent(source)}/docs/${encodedDocSlug}`
  );
}

export async function searchDocuments(q: string): Promise<SearchResult[]> {
  if (!q.trim()) {
    return [];
  }

  const params = new URLSearchParams({ q, limit: "8" });
  const data = await getJson<ItemsResponse<SearchResult>>(
    `/api/search?${params.toString()}`
  );
  return data?.items ?? [];
}

async function getJson<T>(path: string): Promise<T | null> {
  try {
    const res = await fetch(apiUrl(path), { cache: "no-store" });
    if (!res.ok) {
      return null;
    }
    return (await res.json()) as T;
  } catch {
    return null;
  }
}
