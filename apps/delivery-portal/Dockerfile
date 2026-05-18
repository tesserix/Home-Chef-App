# HomeChef delivery-portal SPA Dockerfile.
# Build stage uses tesserix base-node-builder-22, runtime uses
# tesserix base-nginx-spa.
#
# NOTE: Build from monorepo root with: docker build -f apps/delivery-portal/Dockerfile .

# ─── builder ───────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-node-builder-22:latest AS builder
USER 0:0
WORKDIR /app

COPY pnpm-lock.yaml pnpm-workspace.yaml package.json .npmrc ./
COPY apps/delivery-portal/package.json ./apps/delivery-portal/
COPY apps/delivery-portal/.npmrc ./apps/delivery-portal/

RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) && \
    echo "//npm.pkg.github.com/:_authToken=${NPM_TOKEN}" >> .npmrc && \
    pnpm install --frozen-lockfile --filter @homechef/delivery-portal... && \
    sed -i '/_authToken/d' .npmrc

COPY apps/delivery-portal ./apps/delivery-portal
WORKDIR /app/apps/delivery-portal

ARG VITE_API_URL
ARG VITE_BFF_URL
ARG VITE_GIP_API_KEY
ARG VITE_GIP_PROJECT_ID
ARG VITE_GIP_AUTH_DOMAIN
ARG VITE_GIP_TENANT_ID
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
    VITE_GIT_COMMIT=$GIT_COMMIT \
    VITE_BUILD_DATE=$BUILD_DATE \
    VITE_VERSION=$VERSION

RUN pnpm build

# ─── runtime ───────────────────────────────────────────────────────────
FROM ghcr.io/tesserix/base-nginx-spa:latest AS runtime

COPY apps/delivery-portal/nginx.conf /etc/nginx/templates/default.conf.template
ENV API_URL=http://localhost:8080

COPY --from=builder --chown=10001:10001 /app/apps/delivery-portal/dist /usr/share/nginx/html

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1
