ARG GOLANG_VERSION
FROM --platform=$BUILDPLATFORM golang:${GOLANG_VERSION} AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -o /${SERVICE_NAME}-init ./cmd/init
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -o /${SERVICE_NAME} ./cmd/main

RUN mkdir /etc/vdp
RUN mkdir /vdp
RUN mkdir /airbyte

# Download vdp protocol YAML file
ADD https://raw.githubusercontent.com/instill-ai/vdp/main/protocol/vdp_protocol.yaml /etc/vdp/vdp_protocol.yaml

FROM gcr.io/distroless/base:nonroot

USER nonroot:nonroot

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=docker:dind-rootless --chown=nonroot:nonroot /usr/local/bin/docker /usr/local/bin/

COPY --from=build --chown=nonroot:nonroot /src/config ./config
COPY --from=build --chown=nonroot:nonroot /src/release-please ./release-please
COPY --from=build --chown=nonroot:nonroot /src/pkg/db/migration ./pkg/db/migration

COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-migrate ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME}-init ./
COPY --from=build --chown=nonroot:nonroot /${SERVICE_NAME} ./

COPY --from=build --chown=nonroot:nonroot /vdp /vdp
COPY --from=build --chown=nonroot:nonroot /airbyte /airbyte

COPY --from=build --chown=nonroot:nonroot /etc/vdp/vdp_protocol.yaml /etc/vdp/vdp_protocol.yaml
