FROM node:20-slim AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM deps AS build
COPY . .
RUN npm run build


FROM nginx:alpine
LABEL MAINTAINER="bypanghu@163.com"
COPY --from=build  /app/.docker-compose/nginx/conf.d/my.conf /etc/nginx/conf.d/my.conf
COPY --from=build  /app/dist /usr/share/nginx/html
RUN ls -al /usr/share/nginx/html
