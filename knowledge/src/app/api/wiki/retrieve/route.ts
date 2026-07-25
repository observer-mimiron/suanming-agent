import { NextResponse } from "next/server";
import { getPrincipal } from "@/lib/auth";
import { getErrorMessage } from "@/lib/errors";
import { retrievePassages } from "@/lib/query-search";
import { resolveScopeSlugs } from "@/lib/search";
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

    const results = await withPageCache(() =>
      retrievePassages(q, entries, limit),
    );
    return NextResponse.json({ results });
  } catch (err) {
    const message = getErrorMessage(err);
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
