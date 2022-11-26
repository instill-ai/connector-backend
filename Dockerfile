FROM --platform=$BUILDPLATFORM golang:1.18.2 AS build

ARG SERVICE_NAME

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-migrate ./cmd/migration
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-init ./cmd/init
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME}-worker ./cmd/worker
RUN --mount=target=. --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /${SERVICE_NAME} ./cmd/main

FROM gcr.io/distroless/base

ARG SERVICE_NAME

WORKDIR /${SERVICE_NAME}

COPY --from=docker:dind /usr/local/bin/docker /usr/local/bin/

COPY --from=build /src/config ./config
COPY --from=build /src/release-please ./release-please
COPY --from=build /src/internal/db/migration ./internal/db/migration

COPY --from=build /${SERVICE_NAME}-migrate ./
COPY --from=build /${SERVICE_NAME}-init ./
COPY --from=build /${SERVICE_NAME}-worker ./
COPY --from=build /${SERVICE_NAME} ./

# Download vdp protocol YAML file
RUN mkdir /usr/local/vdp && curl https://raw.githubusercontent.com/instill-ai/vdp/main/protocol/vdp_protocol.yaml -so /usr/local/vdp/vdp_protocol.yaml
