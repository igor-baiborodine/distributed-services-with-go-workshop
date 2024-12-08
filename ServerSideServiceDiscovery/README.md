## Server-to-server Service Discovery

### Prerequisites

#### Serf

```shell
go install github.com/hashicorp/serf@latest
```

### Tests

```shell
$ make clean init compile
$ make gencert
$ make test
```
