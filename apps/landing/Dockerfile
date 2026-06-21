FROM node:22-alpine AS build
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
ARG VITE_SITE_URL=https://lead-cat.space
ARG VITE_BOT_USERNAME=lead_cat_bot
ARG VITE_ADMIN_URL=
ENV VITE_SITE_URL=${VITE_SITE_URL}
ENV VITE_BOT_USERNAME=${VITE_BOT_USERNAME}
ENV VITE_ADMIN_URL=${VITE_ADMIN_URL}
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter @leadcat/brand generate && pnpm --filter @leadcat/brand sync
RUN pnpm --filter landing build

FROM node:22-alpine AS runtime
RUN apk add --no-cache nginx wget
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
ENV NODE_ENV=production
ENV PORT=3001
COPY --from=build /repo .
COPY deploy/nginx/ackee-location.conf /etc/nginx/ackee-location.conf
COPY deploy/nginx/landing-ssr.conf /etc/nginx/http.d/default.conf
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/ || exit 1
CMD ["sh", "-c", "pnpm --filter landing start & exec nginx -g 'daemon off;'"]
