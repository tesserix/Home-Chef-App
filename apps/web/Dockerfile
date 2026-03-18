# Build stage
# NOTE: Build from monorepo root with: docker build -f apps/web/Dockerfile .
FROM node:22-alpine AS builder

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@9 --activate

# Copy monorepo root files needed for pnpm workspace
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./

# Copy root .npmrc (pnpm reads from workspace root for registry config)
COPY .npmrc ./

# Copy web-specific .npmrc if it exists
COPY apps/web/.npmrc* ./apps/web/

# Copy web package.json
COPY apps/web/package.json ./apps/web/

# Install dependencies for web workspace only (with GitHub Packages auth)
RUN --mount=type=secret,id=npm_token \
    NPM_TOKEN=$(cat /run/secrets/npm_token) && \
    echo "//npm.pkg.github.com/:_authToken=${NPM_TOKEN}" >> .npmrc && \
    pnpm install --frozen-lockfile --filter @homechef/web... && \
    sed -i '/_authToken/d' .npmrc

# Copy web source code
COPY apps/web ./apps/web

WORKDIR /app/apps/web

# Build arguments
ARG VITE_API_URL
ARG VITE_BFF_URL=https://identity.fe3dr.com
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG BUILD_DATE
ARG VERSION

# Set environment variables for build
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_BFF_URL=$VITE_BFF_URL
ENV VITE_GIT_COMMIT=$GIT_COMMIT
ENV VITE_BUILD_DATE=$BUILD_DATE
ENV VITE_VERSION=$VERSION

# Build the application
RUN pnpm build

# Production stage
FROM nginx:alpine

# Copy nginx config as template (envsubst resolves ${API_URL} at runtime)
COPY apps/web/nginx.conf /etc/nginx/templates/default.conf.template

# Default API_URL if not set at runtime
ENV API_URL=http://localhost:8080

# Copy built assets from builder
COPY --from=builder /app/apps/web/dist /usr/share/nginx/html

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
