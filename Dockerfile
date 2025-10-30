# Development container for WorldCup match service
# Uses Go 1.25 as requested

FROM golang:1.25

ENV DEBIAN_FRONTEND=noninteractive

# Install some common tooling useful in development
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    curl \
    make \
    build-essential \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Keep the container running by default; devcontainer will override the command
CMD ["sleep", "infinity"]
