import { NextResponse } from "next/server";
import { getPrincipal } from "@/lib/auth";
import { getErrorMessage } from "@/lib/errors";
import {
  retrievePassages,
  type RetrievalPassage,
} from "@/lib/query-search";
import {
  resolveScopeSlugs,
  searchWikiContent,
  type ContentSearchResult,
  type SearchScope,
} from "@/lib/search";
import {
  isAgentScopedType,
  isArtifactType,
  listReadableWikiPages,
  withPageCache,
} from "@/lib/wiki";

/**
 * GET /api/wiki/retrieve?q=search+terms&limit=3&scope=agent:yoyo
 *
 * Runtime-facing hybrid retrieval route.
 * Returns short wiki passages ranked by BM25 + vector similarity fusion,
 * without running answer synthesis.
 */
export async function GET(req: Request) {
  try {
    const url = new URL(req.url);
    const q = url.searchParams.get("q")?.trim();
    if (!q) {
      return NextResponse.json(
        { error: "q parameter is required" },
        { status: 400 },
      );
    }

    const limitRaw = Number(url.searchParams.get("limit") || "3");
    const limit = Number.isFinite(limitRaw)
      ? Math.min(10, Math.max(1, Math.trunc(limitRaw)))
      : 3;

    const principal = await getPrincipal();
    const { scopeSlugs, error: scopeError } = await resolveScopeSlugs(
      url.searchParams.get("scope"),
      principal,
    );
    if (scopeError) {
      return NextResponse.json({ error: scopeError }, { status: 400 });
    }

    const readable = await listReadableWikiPages(principal);
    const entries = scopeSlugs
      ? readable.filter((entry) => scopeSlugs.includes(entry.slug))
      : readable.filter(
          (entry) => !isAgentScopedType(entry.type) && !isArtifactType(entry.type),
        );

    const retrieveResults = await withPageCache(() =>
      retrievePassages(q, entries, limit),
    );
    const contentScope: SearchScope | undefined = scopeSlugs
      ? { agentId: "scope", slugs: scopeSlugs }
      : undefined;
    const contentResults = await searchWikiContent(
      q,
      Math.max(12, limit * 4),
      contentScope,
      principal,
    );
    const results = mergeRetrievalResults(q, retrieveResults, contentResults, limit);
    return NextResponse.json({ results });
  } catch (err) {
    const message = getErrorMessage(err);
    return NextResponse.json({ error: message }, { status: 500 });
  }
}

function mergeRetrievalResults(
  query: string,
  retrieveResults: RetrievalPassage[],
  contentResults: ContentSearchResult[],
  limit: number,
): RetrievalPassage[] {
  const ranked = new Map<string, { item: RetrievalPassage; score: number }>();
  for (const result of retrieveResults) {
    ranked.set(result.slug, { item: result, score: result.score });
  }
  for (const result of contentResults) {
    const item: RetrievalPassage = {
      slug: result.slug,
      title: result.title,
      summary: result.summary,
      snippet: result.snippet,
      score: result.score,
    };
    const score = 100 + contentExactScore(query, result);
    const existing = ranked.get(result.slug);
    if (!existing || score > existing.score) {
      ranked.set(result.slug, { item, score });
    }
  }
  return Array.from(ranked.values())
    .sort((a, b) => b.score - a.score || a.item.title.localeCompare(b.item.title))
    .slice(0, limit)
    .map((entry, index) => ({ ...entry.item, score: limit - index }));
}

function contentExactScore(query: string, result: ContentSearchResult): number {
  const haystack = (result.title + " " + result.summary + " " + result.snippet).toLowerCase();
  const terms = query
    .toLowerCase()
    .split(/\s+/)
    .map((term) => term.trim())
    .filter(Boolean);
  let score = result.score;
  score += sourceHintScore(query, result.slug);
  for (const term of terms) {
    if (!haystack.includes(term)) continue;
    score += term.length >= 4 ? 20 : 3;
  }
  return score;
}

function sourceHintScore(query: string, slug: string): number {
  const sourceHints = [
    { term: "穷通宝鉴", slugPrefix: "ref-bazi-qiongtong" },
    { term: "子平真诠", slugPrefix: "ref-bazi-ziping" },
    { term: "渊海子平", slugPrefix: "ref-bazi-yuanhai" },
    { term: "滴天髓", slugPrefix: "ref-bazi-ditiansui" },
    { term: "三命通会", slugPrefix: "ref-bazi-sanming" },
    { term: "格局论命", slugPrefix: "ref-bazi-gelulunming" },
  ];
  for (const hint of sourceHints) {
    if (query.includes(hint.term) && slug.startsWith(hint.slugPrefix)) {
      return 80;
    }
  }
  return 0;
}
