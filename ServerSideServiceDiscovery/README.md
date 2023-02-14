## Server-to-Server Service Discovery

### Prerequisites

#### Serf

```shell
$ go get github.com/hashicorp/serf@latest
```

### Tests

```shell
$ make clean init compile
$ make gencerts
$ make test
```
