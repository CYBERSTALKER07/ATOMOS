#!/bin/bash
echo "Initiating Chaos Test: Redis Termination..."
docker kill pegasus-redis 2>/dev/null || echo "pegasus-redis not running or already killed"
echo "Redis is DEAD. Monitoring backend-go logs for Panics..."
# Observe if outbox/telemetry fail-open logic catches the error
# and falls back to local-only broadcast.
