# HomeChef web SPA Dockerfile.
# Build stage uses tesserix base-node-builder-22 (corepack + libc6-compat
# pre-baked), runtime uses tesserix base-nginx-spa (nginx:1.29-alpine,
# non-root uid 10001, envsubst templating, /health endpoint ready).
#
# NOTE: Build from monorepo root with: docker build -f apps/web/Dockerfile .

# ─── builder ───────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-node-builder-22:latest AS builder
USER 0:0
WORKDIR /app

# Monorepo root files required for pnpm workspace resolution.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
# The root package.json declares pnpm.patchedDependencies (@tesserix/native);
# pnpm reads it on every install and fails if the patch file is missing, even
# though this app never imports that package. Mirrors apps/web-landing.
COPY patches ./patches
COPY .npmrc ./
COPY apps/web/.npmrc* ./apps/web/
COPY apps/web/package.json ./apps/web/

# BuildKit secret so the token never lands in a layer.
RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) && \
    echo "//npm.pkg.github.com/:_authToken=${NPM_TOKEN}" >> .npmrc && \
    pnpm install --frozen-lockfile --filter @homechef/web... && \
    sed -i '/_authToken/d' .npmrc

COPY apps/web ./apps/web
# The bakery, dietary and payment rules are dependency-free TS the web app aliases
# straight from source, so the browser runs the rules the server enforces.
# Source only — the rest of mobile-shared pulls in React Native.
COPY packages/mobile-shared/src/bakery ./packages/mobile-shared/src/bakery
COPY packages/mobile-shared/src/dietary ./packages/mobile-shared/src/dietary
COPY packages/mobile-shared/src/payments ./packages/mobile-shared/src/payments
WORKDIR /app/apps/web

ARG VITE_API_URL
# VITE_BFF_URL is intentionally unset by default — when empty, the web app
# falls back to same-origin /bff which is the correct production behaviour
# (Istio rewrites /bff/* on fe3dr.com to the homechef-auth-bff service).
ARG VITE_BFF_URL=
ARG VITE_GIP_API_KEY
ARG VITE_GIP_PROJECT_ID
ARG VITE_GIP_AUTH_DOMAIN
ARG VITE_GIP_TENANT_ID
ARG VITE_GOOGLE_CLIENT_ID
# Public write key for the self-hosted OpenPanel; unset means analytics off.
ARG VITE_OPENPANEL_CLIENT_ID=
ARG VITE_OPENPANEL_API_URL=https://analytics.tesserix.app/api
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG BUILD_DATE
ARG VERSION
ENV VITE_API_URL=$VITE_API_URL \
    VITE_BFF_URL=$VITE_BFF_URL \
    VITE_GIP_API_KEY=$VITE_GIP_API_KEY \
    VITE_GIP_PROJECT_ID=$VITE_GIP_PROJECT_ID \
    VITE_GIP_AUTH_DOMAIN=$VITE_GIP_AUTH_DOMAIN \
    VITE_GIP_TENANT_ID=$VITE_GIP_TENANT_ID \
    VITE_GOOGLE_CLIENT_ID=$VITE_GOOGLE_CLIENT_ID \
    VITE_OPENPANEL_CLIENT_ID=$VITE_OPENPANEL_CLIENT_ID \
    VITE_OPENPANEL_API_URL=$VITE_OPENPANEL_API_URL \
    VITE_GIT_COMMIT=$GIT_COMMIT \
    VITE_BUILD_DATE=$BUILD_DATE \
    VITE_VERSION=$VERSION

RUN pnpm build

# ─── runtime ───────────────────────────────────────────────────────────
# base-nginx-spa: non-root uid 10001, envsubst on /etc/nginx/templates,
# listens on 8080 by default. We keep the 80 port + API_URL templating.
FROM ghcr.io/tesserix/base-nginx-spa:latest AS runtime

COPY apps/web/nginx.conf /etc/nginx/templates/default.conf.template
# OTTO_URL: shared Otto in support-platform (cross-namespace svc DNS).
# OTTO_INTERNAL_AUTH: empty by default; populated from K8s Secret at deploy
# time via the homechef-web Helm chart. envsubst on /etc/nginx/templates
# substitutes both at container start.
ENV API_URL=http://localhost:8080 \
    OTTO_URL=http://support-platform-otto.support-platform.svc.cluster.local:8089 \
    OTTO_INTERNAL_AUTH=

COPY --from=builder --chown=10001:10001 /app/apps/web/dist /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1
