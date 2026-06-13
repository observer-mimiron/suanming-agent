// Local-only knowledge base — no Clerk auth needed.
// Route handlers use service token (KNOWLEDGE_SERVICE_TOKEN) for write protection.
import { NextResponse, type NextRequest } from "next/server";

const WRITE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

// Browser query page sends POST /api/query — treat as read
const READ_POST_PATHS = new Set(["/api/query", "/api/query/stream"]);

export default function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  if (
    WRITE_METHODS.has(req.method) &&
    pathname.startsWith("/api/") &&
    !READ_POST_PATHS.has(pathname)
  ) {
    const auth = req.headers.get("authorization");
    if (!auth?.startsWith("Bearer ")) {
      return NextResponse.json(
        { error: "Sign in required to write to the knowledge base." },
        { status: 401 },
      );
    }
  }
  return NextResponse.next();
}

/**
 * Returns true when the given path uses in-route token authentication
 * (service token) instead of Clerk session auth.
 */
export function authenticatesInRoute(path: string): boolean {
  if (!path.startsWith("/api/")) return false;

  // Read-only POST paths skip auth entirely.
  if (READ_POST_PATHS.has(path)) return false;
  for (const exempt of READ_POST_PATHS) {
    if (path.startsWith(exempt + "/")) return false;
  }

  // Exact-match task entry points.
  if (path === "/api/tasks/run" || path === "/api/tasks/scan") return true;

  // Ingest routes (exact + sub-routes).
  if (
    path === "/api/ingest" ||
    path.startsWith("/api/ingest/")
  ) return true;

  // Agent seed and per-agent ingest.
  if (path === "/api/agents/seed") return true;
  if (/^\/api\/agents\/[^/]+\/ingest$/.test(path)) return true;

  // Admin routes.
  if (path.startsWith("/api/admin/")) return true;

  // Wiki write routes: /api/wiki, /api/wiki/:slug, /api/wiki/:slug/revisions.
  if (path === "/api/wiki") return true;
  if (/^\/api\/wiki\/[^/]+$/.test(path)) return true;
  if (/^\/api\/wiki\/[^/]+\/revisions$/.test(path)) return true;

  return false;
}

export const config = {
  matcher: [
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    "/(api|trpc)(.*)",
  ],
};
