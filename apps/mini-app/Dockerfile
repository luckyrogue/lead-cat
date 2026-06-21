ARG FRONTEND_DEPS_IMAGE=leadcat-frontend-deps:local
FROM ${FRONTEND_DEPS_IMAGE} AS build
WORKDIR /repo
COPY . .
ARG VITE_API_URL=""
ARG VITE_BOT_USERNAME=lead_cat_bot
ARG VITE_TMA_DEV_TG_ID=""
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_BOT_USERNAME=${VITE_BOT_USERNAME}
ENV VITE_TMA_DEV_TG_ID=${VITE_TMA_DEV_TG_ID}
RUN if [ -n "$VITE_TMA_DEV_TG_ID" ]; then echo "VITE_TMA_DEV_TG_ID must be empty in production Docker builds" && exit 1; fi
RUN pnpm --filter mini-app build

FROM nginx:alpine AS runtime
ENV API_UPSTREAM=http://backend:8080
COPY deploy/nginx/security-headers.conf /etc/nginx/security-headers.conf
COPY deploy/nginx/security-headers-miniapp.conf /etc/nginx/security-headers-miniapp.conf
COPY deploy/nginx/ackee-location.conf /etc/nginx/ackee-location.conf
COPY deploy/nginx/spa-miniapp.conf /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/mini-app/build/client /usr/share/nginx/html
EXPOSE 80
