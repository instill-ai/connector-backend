# connector-backend

`connector-backend` manages all connector resources and communicates with [Visual Data Preparation (VDP)](https://github.com/instill-ai/vdp) to connect a pipeline to flow the source unstructured visual data to the destination structured data.

## Local dev

On the local machine, clone `vdp` repository in your workspace, move to the repository folder, and launch all dependent microservices:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/vdp.git
$ cd vdp
$ make dev PROFILE=connector
```

Clone `connector-backend` repository in your workspace and move to the repository folder:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/connector-backend.git
$ cd connector-backend
```

### Build the dev image

```bash
$ make build
```

### Run the dev container

```bash
$ make dev
```

Now, you have the Go project set up in the container, in which you can compile and run the binaries together with the integration test in each container shell.

### Run the server

```bash
$ docker exec -it connector-backend /bin/bash
$ go run ./cmd/migration
$ go run ./cmd/init
$ go run ./cmd/main
```

### Run the Temporal worker

```bash
$ docker exec -it connector-backend /bin/bash
$ go run ./cmd/worker
```

### Run the integration test

``` bash
$ docker exec -it connector-backend /bin/bash
$ make integration-test
```

### Stop the dev container

```bash
$ make stop
```

### CI/CD

The latest images will be published to Docker Hub [repository](https://hub.docker.com/r/instill/connector-backend) at release.

## License

See the [LICENSE](./LICENSE) file for licensing information.
