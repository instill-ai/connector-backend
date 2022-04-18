FROM golang:1.17.2 AS build

WORKDIR /go/src
COPY . /go/src

RUN go get -d -v ./...

RUN --mount=type=cache,target=/root/.cache/go-build go build -o /connector-backend ./cmd/main
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /connector-backend-migrate ./cmd/migration
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /connector-backend-init ./cmd/init

FROM gcr.io/distroless/base AS runtime

ENV GIN_MODE=release
WORKDIR /connector-backend

COPY --from=build /connector-backend ./
COPY --from=build /connector-backend-migrate ./
COPY --from=build /connector-backend-init ./
COPY --from=build /go/src/configs ./configs

EXPOSE 8080/tcp
ENTRYPOINT ["./connector-backend"]
