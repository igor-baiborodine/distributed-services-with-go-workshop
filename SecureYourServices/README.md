## Secure Your Services

### Prerequisites

#### CloudFlare CLI

```shell
$ go install github.com/cloudflare/cfssl/cmd/cfssl@latest
$ go install github.com/cloudflare/cfssl/cmd/cfssljson@latest
```

### Tests

```shell
$ make clean init compile
$ make gencerts
$ make test
```
