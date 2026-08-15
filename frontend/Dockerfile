FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# nginx-unprivileged: resmi nginx image'ının non-root çalışan sürümü.
# Standart nginx:alpine'de master process root olarak çalışır (80 portuna
# bind edebilmek için); bu image ise zaten "nginx" kullanıcısıyla, 8080/8443
# gibi ayrıcalıksız portlarda çalışacak şekilde tasarlanmış.
FROM nginxinc/nginx-unprivileged:alpine

USER root
RUN apk add --no-cache openssl && \
    mkdir -p /etc/nginx/tls && \
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout /etc/nginx/tls/selfsigned.key \
      -out /etc/nginx/tls/selfsigned.crt \
      -subj "/C=TR/ST=Mersin/L=Mersin/O=BAUM/CN=localhost" && \
    chown -R nginx:nginx /etc/nginx/tls
USER nginx

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080 8443
