# fe3dr.com landing Dockerfile.
# Builds the Next.js static export (out/) and serves it with the hardened
# tesserix base images — same as the apps/web SPA — so the homechef-web image
# clears the Trivy CRITICAL/HIGH gate (base-nginx-spa is non-root + Trivy-clean,
# unlike the raw nginx:alpine base which lagged the security feed).
#
# NOTE: Build from monorepo root with:
#   docker build -f apps/web-landing/Dockerfile .

# ─── builder ───────────────────────────────────────────────────────────
# base-node-builder-22: corepack + pnpm + libc6-compat pre-baked.
FROM ghcr.io/tesserix/base-node-builder-22:latest AS builder
USER 0:0
WORKDIR /app

# Monorepo root files required for pnpm workspace resolution.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json .npmrc ./
# The root package.json declares pnpm.patchedDependencies (@tesserix/native);
# pnpm reads it on every install, so the patch file must be present even though
# the landing doesn't use that package.
COPY patches ./patches
COPY apps/web-landing/package.json ./apps/web-landing/

# BuildKit secret so the GitHub Packages token never lands in a layer.
# @tesserix/otto-widget is published only to GitHub Packages, and pnpm's
# hoisted layout fetches it even though the landing doesn't import it, so the
# install needs GH Packages auth.
RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) && \
    echo "//npm.pkg.github.com/:_authToken=${NPM_TOKEN}" >> .npmrc && \
    pnpm install --frozen-lockfile --filter @homechef/web-landing... && \
    sed -i '/_authToken/d' .npmrc

COPY apps/web-landing ./apps/web-landing
WORKDIR /app/apps/web-landing

RUN pnpm build

# ─── runtime ───────────────────────────────────────────────────────────
# base-nginx-spa: non-root uid 10001, envsubst on /etc/nginx/templates,
# Trivy-clean. The apps/web SPA uses the same base + passes the gate. The
# landing's nginx.conf has no ${VAR}s, so envsubst leaves it unchanged.
FROM ghcr.io/tesserix/base-nginx-spa:latest AS runtime

COPY apps/web-landing/nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=builder --chown=10001:10001 /app/apps/web-landing/out /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1
