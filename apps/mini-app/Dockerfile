# Build context: repository root. Build: docker build -f apps/mini-app/Dockerfile .
# Mini App is a SPA (ssr: false) served inside the Telegram WebView. nginx
# serves the static build and proxies /api/* to the backend (same-origin).
# Leave VITE_API_URL empty so the client calls relative /api paths.
FROM node:22-alpine AS build
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
ARG VITE_API_URL=""
ARG VITE_BOT_USERNAME=lead_cat_bot
ARG VITE_TMA_DEV_TG_ID=""
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_BOT_USERNAME=${VITE_BOT_USERNAME}
ENV VITE_TMA_DEV_TG_ID=${VITE_TMA_DEV_TG_ID}
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter mini-app build

FROM nginx:alpine AS runtime
ENV API_UPSTREAM=http://backend:8080
COPY apps/mini-app/nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/mini-app/build/client /usr/share/nginx/html
EXPOSE 80
