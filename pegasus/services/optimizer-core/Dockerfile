FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY proto ./proto
COPY server ./server
COPY scripts ./scripts

RUN chmod +x ./scripts/gen_proto.sh && ./scripts/gen_proto.sh

ENV PYTHONUNBUFFERED=1
ENV OPTIMIZER_CORE_GRPC_PORT=50055

EXPOSE 50055

CMD ["python", "-m", "server.main"]
