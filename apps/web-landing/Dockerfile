# fe3dr.com landing Dockerfile.
# Builds the Next.js static export (out/) and serves it with nginx.
#
# NOTE: Build from monorepo root with:
#   docker build -f apps/web-landing/Dockerfile .

# ─── builder ───────────────────────────────────────────────────────────
FROM node:22-alpine AS builder
RUN corepack enable && corepack prepare pnpm@9.15.1 --activate
WORKDIR /app

# Monorepo root files required for pnpm workspace resolution.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json .npmrc ./
# The root package.json declares pnpm.patchedDependencies (@tesserix/native);
# pnpm reads it on every install, so the patch file must be present even though
# the landing doesn't use that package.
COPY patches ./patches
COPY apps/web-landing/package.json ./apps/web-landing/

RUN pnpm install --frozen-lockfile --filter @homechef/web-landing...

COPY apps/web-landing ./apps/web-landing
WORKDIR /app/apps/web-landing

RUN pnpm build

# ─── runtime ───────────────────────────────────────────────────────────
# nginx:1.29-alpine aligns with the tesserix base-nginx-spa the other SPAs use.
# `apk upgrade` pulls the latest patched Alpine packages (openssl/libxml2/zlib/
# musl/libpng/libexpat) so the image clears the Trivy CRITICAL/HIGH gate (#15)
# — the pinned base tag lags a few package revisions behind the security feed.
FROM nginx:1.29-alpine AS runtime

RUN apk upgrade --no-cache

COPY apps/web-landing/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/apps/web-landing/out /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1
