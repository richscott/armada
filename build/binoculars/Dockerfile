ARG BASE_IMAGE=alpine:3.21.3
FROM ${BASE_IMAGE}
LABEL org.opencontainers.image.title=binoculars
LABEL org.opencontainers.image.description="binoculars"
LABEL org.opencontainers.image.url=https://hub.docker.com/r/gresearch/binoculars

RUN addgroup -S -g 2000 armada && adduser -S -u 1000 armada -G armada
USER armada

COPY binoculars /app/
COPY config/binoculars/config.yaml /app/config/binoculars/config.yaml
COPY config/logging.yaml /app/config/logging.yaml

WORKDIR /app

ENTRYPOINT ["./binoculars"]
