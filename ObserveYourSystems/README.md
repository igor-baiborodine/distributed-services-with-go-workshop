## Observe Your Services

### Prerequisites

#### CloudFlare CLI

```shell
$ go get go.uber.org/zap@latest
$ go get go.opencensus.io@latest
```

### Tests

```shell
$ make clean init compile
$ make gencerts
$ make test
```
