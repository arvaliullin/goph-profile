FROM node:22-alpine AS webbuild
WORKDIR /src
COPY web/avatars/package.json web/avatars/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/avatars/ ./
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=webbuild /src/dist/ /usr/share/nginx/html/
COPY configs/nginx/default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
