## Coordinate Your Services with Consensus

### Prerequisites

#### Serf

```shell
go install github.com/hashicorp/serf@latest
```

### Tests

```shell
$ make clean init compile
$ make gencerts
$ make test
```
