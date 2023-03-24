# connector-backend

[![Integration Test](https://github.com/instill-ai/connector-backend/actions/workflows/integration-test.yml/badge.svg)](https://github.com/instill-ai/connector-backend/actions/workflows/integration-test.yml)

`connector-backend` manages all connector resources and communicates with [Versatile Data Pipeline (VDP)](https://github.com/instill-ai/vdp) to connect a pipeline to flow the source unstructured data to the destination.

## Local dev

On the local machine, clone `vdp` repository in your workspace, move to the repository folder, and launch all dependent microservices:
```bash
$ cd <your-workspace>
$ git clone https://github.com/instill-ai/vdp.git
$ cd vdp
$ make dev PROFILE=connector ITMODE=true
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

- **pull_request** to the `main` branch will trigger the **`Integration Test`** workflow running the integration test using the image built on the PR head branch.
- **push** to the `main` branch will trigger
  - the **`Integration Test`** workflow building and pushing the `:latest` image on the `main` branch, following by running the integration test, and
  - the **`Release Please`** workflow, which will create and update a PR with respect to the up-to-date `main` branch using [release-please-action](https://github.com/google-github-actions/release-please-action).

Once the release PR is merged to the `main` branch, the [release-please-action](https://github.com/google-github-actions/release-please-action) will tag and release a version correspondingly.

The images are pushed to Docker Hub [repository](https://hub.docker.com/r/instill/connector-backend).

## License

See the [LICENSE](./LICENSE) file for licensing information.
