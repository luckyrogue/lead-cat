ARG FRONTEND_DEPS_IMAGE=leadcat-frontend-deps:local
FROM ${FRONTEND_DEPS_IMAGE} AS build
WORKDIR /repo
COPY . .
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}
RUN pnpm --filter admin build

FROM nginx:alpine AS runtime
ENV API_UPSTREAM=http://backend:8080
COPY deploy/nginx/security-headers.conf /etc/nginx/security-headers.conf
COPY deploy/nginx/ackee-location.conf /etc/nginx/ackee-location.conf
COPY deploy/nginx/spa.conf /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/admin/build/client /usr/share/nginx/html
EXPOSE 80
