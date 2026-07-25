## Page conventions

- Use kebab-case slugs for all wiki pages.
- Every page must have YAML frontmatter with title, created, updated, confidence, and tags.
- Cite sources inline with `[ref]` markers and include a Sources section at the bottom.

## Operations

- Run `pnpm dev` to start the development server.
- Pages are stored as Markdown files in the wiki/ directory.
- Rebuild embeddings after bulk imports.

## Cross-reference policy

- Link related pages using `[[slug]]` syntax.
- Avoid orphan pages — every page should have at least one incoming link.

## Lint checks

- Run lint checks to validate page structure and metadata compliance.
- Fix lint issues before publishing pages.
