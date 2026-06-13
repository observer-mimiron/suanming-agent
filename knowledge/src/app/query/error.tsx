"use client";

import { PageError } from "@/components/ErrorBoundary";

export default function QueryError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <PageError
      title="Query error"
      description="Something went wrong while querying the wiki."
      backHref="/"
      backLabel="← Home"
      error={error}
      reset={reset}
    />
  );
}
