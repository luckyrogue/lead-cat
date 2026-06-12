FROM node:22-alpine AS build
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter landing build

FROM node:22-alpine AS runtime
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=build /repo .
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/ || exit 1
CMD ["pnpm", "--filter", "landing", "start"]
