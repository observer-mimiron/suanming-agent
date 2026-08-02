# Stage 1: Install dependencies
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm pkg delete devDependencies.@next/swc-win32-x64-msvc && \
    npm ci --legacy-peer-deps

# Stage 2: Build the application
FROM node:22-alpine AS build
WORKDIR /app
ARG NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=""
ARG NEXT_PUBLIC_OWNER_HANDLE=""
ENV NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=$NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY
ENV NEXT_PUBLIC_OWNER_HANDLE=$NEXT_PUBLIC_OWNER_HANDLE
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# Stage 3: Production runner
FROM node:22-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy application files (standard Next.js output — no standalone mode)
COPY --from=build /app/package.json ./
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/.next ./.next
COPY --from=build /app/public ./public
COPY --from=build /app/next.config.ts ./

# Create data directories and set ownership
RUN mkdir -p /app/wiki /app/raw && \
    chown -R nextjs:nodejs /app

USER nextjs
EXPOSE 3000

CMD ["npx", "next", "start"]
