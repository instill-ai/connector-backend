# Changelog

## [0.5.2-alpha](https://github.com/instill-ai/connector-backend/compare/v0.5.1-alpha...v0.5.2-alpha) (2022-08-18)


### Miscellaneous Chores

* release v0.5.2-alpha ([d2daad6](https://github.com/instill-ai/connector-backend/commit/d2daad608b3489a5d703d5d5266fab6d06688fb6))

## [0.5.1-alpha](https://github.com/instill-ai/connector-backend/compare/v0.5.0-alpha...v0.5.1-alpha) (2022-08-17)


### Bug Fixes

* fix http and grpc state change logic ([9e7b7eb](https://github.com/instill-ai/connector-backend/commit/9e7b7eb3cc29fa04d5880ea27dcc532c9407a0fa))
* fix worker container naming ([b0ad69e](https://github.com/instill-ai/connector-backend/commit/b0ad69e7825a880073561ef8aa066f52e6de0d64))

## [0.5.0-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.5-alpha...v0.5.0-alpha) (2022-07-29)


### Features

* add data association with pipeline ([c3236f7](https://github.com/instill-ai/connector-backend/commit/c3236f79474bcd45bffeaaa806c37635ab086308))


### Bug Fixes

* fix dst data duplicate and lost issue ([eb7ad97](https://github.com/instill-ai/connector-backend/commit/eb7ad9786f6c4ab7d42311e3e0746e713e839d4a)), closes [#25](https://github.com/instill-ai/connector-backend/issues/25)

## [0.4.5-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.4-alpha...v0.4.5-alpha) (2022-07-19)


### Bug Fixes

* fix airbyte emitted_at timestamp ([3eca9d6](https://github.com/instill-ai/connector-backend/commit/3eca9d67e5c14838145133b775c7c15de43c290c))
* fix wrong state when connector check failed ([92367e1](https://github.com/instill-ai/connector-backend/commit/92367e118fe7091e1e6b5b15a66486ab83ae6dc8))
* remove airbyte namespace ([bd7baa4](https://github.com/instill-ai/connector-backend/commit/bd7baa4300c004968d6b3601259c24fdccb4bc4f))

## [0.4.4-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.3-alpha...v0.4.4-alpha) (2022-07-10)


### Bug Fixes

* use google.protobuf.Struct for connector configuration ([6bbf8a7](https://github.com/instill-ai/connector-backend/commit/6bbf8a71c15dd93f20598218e2a525a0d54702c6))

## [0.4.3-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.2-alpha...v0.4.3-alpha) (2022-07-07)


### Bug Fixes

* fix error handler middleware ([2fc9487](https://github.com/instill-ai/connector-backend/commit/2fc9487cdf44c62b783f10772e77191177198554))
* skip JSON Schema snake_case convert ([c19cc53](https://github.com/instill-ai/connector-backend/commit/c19cc536334cbfbb1f97bc1700d234af8a66c7c5)), closes [#17](https://github.com/instill-ai/connector-backend/issues/17)

## [0.4.2-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.1-alpha...v0.4.2-alpha) (2022-06-27)


### Bug Fixes

* return 422 delete error msg ([5498494](https://github.com/instill-ai/connector-backend/commit/54984945cf8c8d2914ba014d882da651c9b43bdc))

## [0.4.1-alpha](https://github.com/instill-ai/connector-backend/compare/v0.4.0-alpha...v0.4.1-alpha) (2022-06-27)


### Miscellaneous Chores

* release v0.4.1-alpha ([e46a702](https://github.com/instill-ai/connector-backend/commit/e46a7026bb5fc501cfab4264bcbb71bce5368eaa))

## [0.4.0-alpha](https://github.com/instill-ai/connector-backend/compare/v0.3.1-alpha...v0.4.0-alpha) (2022-06-26)


### Features

* add delete guardian ([355922a](https://github.com/instill-ai/connector-backend/commit/355922a77d2fb0aca898c547a0946a1107a50812))
* add usage collection ([c06e98a](https://github.com/instill-ai/connector-backend/commit/c06e98a47e3b97b59bbc70877b895b7db006f2ab))
* add write destination connector ([5995260](https://github.com/instill-ai/connector-backend/commit/5995260a98b6264dcbece6c9dddc2a4bb8f20878))


### Bug Fixes

* fix det jsonschema for empty case ([cd4f102](https://github.com/instill-ai/connector-backend/commit/cd4f1028142c604d9123e314907878168883a9e0))
* fix duration configuration bug ([4a4111c](https://github.com/instill-ai/connector-backend/commit/4a4111c0a06cec777b29f48626ce653a3b6a25e0))
* fix usage collection ([c19cf9b](https://github.com/instill-ai/connector-backend/commit/c19cf9bda1dcbdf3ce18520d2b460c3868b6c60a))
* fix usage disbale logic ([664660d](https://github.com/instill-ai/connector-backend/commit/664660d6dfb656074a8c7822a1088a5540b55d16))
* fix usage-backend non-tls dial ([457a74d](https://github.com/instill-ai/connector-backend/commit/457a74d039d6c6adef7d182dfba8906a5240a5a6))
* remove worker debug volume ([acaa01c](https://github.com/instill-ai/connector-backend/commit/acaa01c9689628b770fca657482412fd99cf9878))

### [0.3.1-alpha](https://github.com/instill-ai/connector-backend/compare/v0.3.0-alpha...v0.3.1-alpha) (2022-05-31)


### Miscellaneous Chores

* release 0.3.1-alpha ([7e07b79](https://github.com/instill-ai/connector-backend/commit/7e07b79c68955bab51a9ec8491b9b3ce2f8ea41d))

## [0.3.0-alpha](https://github.com/instill-ai/connector-backend/compare/v0.2.0-alpha...v0.3.0-alpha) (2022-05-31)


### Features

* add check state using workflow ([0848f05](https://github.com/instill-ai/connector-backend/commit/0848f054551296afda9a96be2245614b08000852))
* add connect/disconnect custom rpc method ([fd67fa5](https://github.com/instill-ai/connector-backend/commit/fd67fa5b80d8f69a98e80f4d1fec757b31baba4b))
* add cors support ([5ac2a53](https://github.com/instill-ai/connector-backend/commit/5ac2a535161445bba987958e13ab82d20330faf7))


### Bug Fixes

* fix backend network config in docker-compose ([9697b65](https://github.com/instill-ai/connector-backend/commit/9697b6546e9361cad4e9205de6ab082afb138097))
* fix docker in docker volume mount issue ([0c404cd](https://github.com/instill-ai/connector-backend/commit/0c404cd05d4b4a58d2ad0e8bd9dac4d9226b86d7))

## [0.2.0-alpha](https://github.com/instill-ai/connector-backend/compare/v0.1.0-alpha...v0.2.0-alpha) (2022-05-21)


### Features

* add json schema validation ([bc609e2](https://github.com/instill-ai/connector-backend/commit/bc609e231029d8a2f600ae60d9d505028a6e900b))
* integrate with mgmt-backend ([02c2aa3](https://github.com/instill-ai/connector-backend/commit/02c2aa389d78ec616a82351b38f5f917b48a0f8d))


### Bug Fixes

* fix connector definition response empty name issue ([72179df](https://github.com/instill-ai/connector-backend/commit/72179df7182be4f3d26e256d57ed72f05544463f))
* fix create dup error code ([8af5ce1](https://github.com/instill-ai/connector-backend/commit/8af5ce113ad48b3b01f747cc9bf6122fc862714c))
* fix directness connector constant validation ([2deb38a](https://github.com/instill-ai/connector-backend/commit/2deb38a228ae353805936fd88cea25fb41bc627e))
* fix directness connectors null spec issue ([e192252](https://github.com/instill-ai/connector-backend/commit/e192252331346fca361967406001d18aed9dd115))
* fix empty UpdateSourceConnectorResponse ([6649e1e](https://github.com/instill-ai/connector-backend/commit/6649e1e0d1b69e38d6ccb25c98d432deb9ed67f3))
* fix list empty case ([4483ec1](https://github.com/instill-ai/connector-backend/commit/4483ec13ac24e034048cca43489c549d70b41a94))
* update JSON schema ([015f001](https://github.com/instill-ai/connector-backend/commit/015f001227579aaedc3c9ef0d2cbd4f8e0ce3280))

## [0.1.0-alpha](https://github.com/instill-ai/connector-backend/compare/v0.0.0-alpha...v0.1.0-alpha) (2022-05-13)


### Features

* implement full connector CRUD ([9fd2d47](https://github.com/instill-ai/connector-backend/commit/9fd2d475ca9cf3d55079149b56bbef2e26ff1d8f))
