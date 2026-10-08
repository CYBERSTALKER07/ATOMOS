FROM nginx:1.27-alpine

LABEL org.opencontainers.image.title="PegasusX Supplier Portal" \
      org.opencontainers.image.description="Static web assets served via NGINX for supplier portal" \
      org.opencontainers.image.source="https://github.com/pegasusx/pegasusx"

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY out/ /usr/share/nginx/html/

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:8080/ || exit 1

