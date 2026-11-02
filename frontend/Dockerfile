FROM node:20.19-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build -- --configuration production

FROM nginx:1.31.8-alpine
RUN printf "@nginx https://nginx.org/packages/mainline/alpine/v%s/main\n" \
      "$(cut -d. -f1,2 /etc/alpine-release)" >> /etc/apk/repositories \
    && apk add --no-cache nginx-module-otel@nginx
COPY --from=builder /app/dist/frontend/browser/ /usr/share/nginx/html/
COPY nginx-main.conf /etc/nginx/nginx.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY security-headers.conf /etc/nginx/security-headers.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
