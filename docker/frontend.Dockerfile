# ============================================================
# Stage 1: Install dependencies
# ============================================================
FROM node:22-alpine AS deps

WORKDIR /app

# Copy dependency manifests for layer caching
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts

# ============================================================
# Stage 2: Build the Next.js application
# ============================================================
FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY frontend/ .

# Build argument for API URL (baked into client bundle)
ARG NEXT_PUBLIC_API_URL=http://localhost/api
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}

RUN npm run build

# ============================================================
# Stage 3: Production runtime
# ============================================================
FROM node:22-alpine

WORKDIR /app

# Create non-root user
RUN addgroup -S verdox && adduser -S verdox -G verdox

# Copy standalone build output
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

# Set ownership
RUN chown -R verdox:verdox /app

USER verdox

ENV NODE_ENV=production
ENV HOSTNAME="0.0.0.0"
ENV PORT=3000

EXPOSE 3000

CMD ["node", "server.js"]
