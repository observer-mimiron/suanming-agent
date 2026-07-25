import { NextResponse, type NextRequest } from "next/server";
import {
  listReadableWikiPages,
  isAgentScopedType,
} from "@/lib/wiki";
import { ownerToTenant } from "@/lib/links";
import { listCommonsPages } from "@/lib/commons";
import { expandMineScope, resolveScope } from "@/lib/search";
import { getPrincipal } from "@/lib/auth";
import { getBacklinkIndex } from "@/lib/backlink-index";
import { getErrorMessage } from "@/lib/errors";
import { logger } from "@/lib/logger";

interface GraphNode {
  id: string;
  label: string;
  /** Canonical tenant for the node, so clicks navigate to `/u/<tenant>/<slug>`. */
  tenant: string;
  linkCount: number;
  tags: string[];
}

interface GraphEdge {
  source: string;
  target: string;
}

export async function GET(req: NextRequest) {
  try {
    const scopeParam =
      new URL(req.url).searchParams.get("scope") || undefined;
    const principal = await getPrincipal();
    const expanded = expandMineScope(scopeParam, principal);

    let pages;
    if (expanded) {
      const resolved = await resolveScope(expanded);
      const scopeSet = new Set(resolved?.slugs ?? []);
      pages = (await listReadableWikiPages(principal)).filter(
        (p) => scopeSet.has(p.slug) && !isAgentScopedType(p.type),
      );
    } else {
      pages = await listCommonsPages();
    }
    const slugSet = new Set(pages.map((p) => p.slug));

    // ponytail: build nodes from IndexEntry (no content read).
    const nodes: GraphNode[] = pages.map((p) => ({
      id: p.slug,
      label: p.title,
      tenant: ownerToTenant(p.owner),
      linkCount: 0, // computed below
      tags: p.tags ?? [],
    }));

    // ponytail: use pre-built backlink index for edges.
    // Fallback: when the index is absent, only read 目录-tagged pages (~6)
    // instead of all 317 pages.
    const edges: GraphEdge[] = [];
    const backlinks = await getBacklinkIndex();
    if (backlinks) {
      for (const [target, sources] of Object.entries(backlinks)) {
        if (!slugSet.has(target)) continue;
        for (const src of sources) {
          if (slugSet.has(src) && src !== target) {
            edges.push({ source: src, target });
          }
        }
      }
    } else {
      // Only scan directory pages for outbound wiki links.
      const dirPages = pages.filter((p) => p.tags?.includes("目录"));
      const { readWikiPageWithFrontmatter } = await import("@/lib/wiki");
      for (const p of dirPages) {
        const wp = await readWikiPageWithFrontmatter(p.slug);
        if (!wp) continue;
        const linkRe = /\[([^\]]*)\]\(([^)]+)\.md\)/g;
        let match: RegExpExecArray | null;
        while ((match = linkRe.exec(wp.body)) !== null) {
          const target = match[2];
          if (slugSet.has(target) && target !== p.slug) {
            edges.push({ source: p.slug, target });
          }
        }
      }
    }

    // Compute linkCount from edges
    const countMap = new Map<string, number>();
    for (const edge of edges) {
      countMap.set(edge.source, (countMap.get(edge.source) ?? 0) + 1);
      countMap.set(edge.target, (countMap.get(edge.target) ?? 0) + 1);
    }
    for (const node of nodes) {
      node.linkCount = countMap.get(node.id) ?? 0;
    }

    return NextResponse.json({ nodes, edges });
  } catch (error) {
    logger.error("wiki", "Graph API error", error);
    return NextResponse.json(
      {
        error: getErrorMessage(error),
      },
      { status: 500 },
    );
  }
}
