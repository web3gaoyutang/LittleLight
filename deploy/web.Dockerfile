ARG NODE_IMAGE=node:20-alpine
ARG NGINX_IMAGE=nginx:1.27-alpine

FROM ${NODE_IMAGE} AS builder
WORKDIR /src/app
ARG NPM_REGISTRY=https://registry.npmjs.org/
COPY app/package.json app/package-lock.json ./
RUN npm ci --registry=${NPM_REGISTRY}
COPY app/ ./
RUN npm run build:h5

FROM ${NGINX_IMAGE}
RUN apk add --no-cache curl
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/app/dist /usr/share/nginx/html
EXPOSE 80
