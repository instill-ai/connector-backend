# connector-backend

connector-backend manages all connector resources and communicates with [Visual Data Preparation (VDP)](https://github.com/instill-ai/vdp) to connect a pipeline to flow the source unstructured visual data to the destination structured data.

## Development

Pre-requirements:

- Go v1.17 or later installed on your development machine

### Binary build

```bash
$ make
```

### Docker build

```bash
# Build images with BuildKit
$ DOCKER_BUILDKIT=1 docker build -t instill/connector-backend:dev .
```

The latest images will be published to Docker Hub [repository](https://hub.docker.com/r/instill/connector-backend) at release time.

## License

See the [LICENSE](./LICENSE) file for licensing information.
