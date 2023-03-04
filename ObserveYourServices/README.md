## Observe Your Services

### Prerequisites

#### Zap & OpenCensus

```shell
$ go install go.uber.org/zap@latest
$ go isntall go.opencensus.io@latest
```

### Tests

```shell
$ make clean init compile
$ make gencerts
$ make test-log
$ make test-server
```