FROM python:3.12-slim

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir git-cliff || true

WORKDIR /app
COPY pyproject.toml .
COPY src/ src/
RUN pip install --no-cache-dir .

ENTRYPOINT ["teamwork-mcp-changelog"]
