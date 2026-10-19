FROM python:3.12-slim-bookworm

LABEL org.opencontainers.image.title="PegasusX Optimizer Core" \
      org.opencontainers.image.description="VRP optimization engine powered by Google OR-Tools" \
      org.opencontainers.image.source="https://github.com/pegasusx/pegasusx"

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    OPTIMIZER_HTTP_PORT=8082

RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 10001 optimizer \
    && useradd -u 10001 -g optimizer -d /app -s /bin/false optimizer

COPY server/requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r requirements.txt

COPY server/ .
RUN chown -R optimizer:optimizer /app

EXPOSE 8082

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8082/healthz')"

USER optimizer:optimizer

CMD ["python", "http_main.py"]

