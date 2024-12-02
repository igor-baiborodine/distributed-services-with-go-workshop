## Secure Your Services

### Prerequisites

#### CloudFlare CLI

```shell
$ go get google.golang.org/grpc@latest
$ go install github.com/cloudflare/cfssl/cmd/cfssl@latest
$ go install github.com/cloudflare/cfssl/cmd/cfssljson@latest
```

### Tests

```shell
$ make clean init compile
$ make gencert
$ make test
```
