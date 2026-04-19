# HomeChef vendor-portal SPA Dockerfile.
# Build stage uses tesserix base-node-builder-22, runtime uses
# tesserix base-nginx-spa.
#
# NOTE: Build from monorepo root with: docker build -f apps/vendor-portal/Dockerfile .

# ─── builder ───────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-node-builder-22:latest AS builder
USER 0:0
WORKDIR /app

COPY pnpm-lock.yaml pnpm-workspace.yaml package.json .npmrc ./
COPY apps/vendor-portal/package.json ./apps/vendor-portal/
COPY apps/vendor-portal/.npmrc ./apps/vendor-portal/

RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) && \
    echo "//npm.pkg.github.com/:_authToken=${NPM_TOKEN}" >> .npmrc && \
    pnpm install --frozen-lockfile --filter @homechef/vendor-portal... && \
    sed -i '/_authToken/d' .npmrc

COPY apps/vendor-portal ./apps/vendor-portal
WORKDIR /app/apps/vendor-portal

ARG VITE_API_URL
ARG VITE_BFF_URL=
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG BUILD_DATE
ARG VERSION
ENV VITE_API_URL=$VITE_API_URL \
    VITE_BFF_URL=$VITE_BFF_URL \
    VITE_GIT_COMMIT=$GIT_COMMIT \
    VITE_BUILD_DATE=$BUILD_DATE \
    VITE_VERSION=$VERSION

RUN pnpm build

# ─── runtime ───────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-nginx-spa:latest AS runtime

COPY apps/vendor-portal/nginx.conf /etc/nginx/templates/default.conf.template
ENV API_URL=http://localhost:8080

COPY --from=builder --chown=10001:10001 /app/apps/vendor-portal/dist /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1
