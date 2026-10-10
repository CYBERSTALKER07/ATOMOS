FROM python:3.12-slim-bookworm

LABEL org.opencontainers.image.title="PegasusX Dispatch Optimizer Python" \
      org.opencontainers.image.description="FastAPI sidecar for vehicle routing & pick path optimization" \
      org.opencontainers.image.source="https://github.com/pegasusx/pegasusx"

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PORT=8000

RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 10001 optimizer \
    && useradd -u 10001 -g optimizer -d /app -s /bin/false optimizer

COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r requirements.txt

COPY . .
RUN chown -R optimizer:optimizer /app

EXPOSE 8000

HEALTHCHECK --interval=20s --timeout=5s --start-period=15s --retries=3 \
  CMD curl -f http://localhost:8000/healthz || exit 1

USER optimizer:optimizer

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
