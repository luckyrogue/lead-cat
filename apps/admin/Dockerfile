FROM node:22-alpine AS build
WORKDIR /repo
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter admin build

FROM nginx:alpine AS runtime
ENV API_UPSTREAM=http://backend:8080
COPY apps/admin/nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /repo/apps/admin/build/client /usr/share/nginx/html
EXPOSE 80
