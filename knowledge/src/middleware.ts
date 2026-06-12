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

export const config = {
  matcher: [
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    "/(api|trpc)(.*)",
  ],
};
