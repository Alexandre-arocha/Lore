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

/** Consulta /api/health. Retorna null se a API estiver inacessível. */
export async function getHealth(): Promise<HealthStatus | null> {
  try {
    const res = await fetch(apiUrl("/api/health"), { cache: "no-store" });
    return (await res.json()) as HealthStatus;
  } catch {
    return null;
  }
}
