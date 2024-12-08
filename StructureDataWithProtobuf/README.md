## Structure Data With Protobuf

### Prerequisites

#### Protobuf Compiler

```shell
$ PB_REL="https://github.com/protocolbuffers/protobuf/releases"
$ curl -LO $PB_REL/download/v28.3/protoc-28.3-linux-x86_64.zip
$ sudo unzip protoc-28.3-linux-x86_64.zip -d /usr/local/protobuf
$ echo 'PATH=$PATH:/usr/local/protobuf/bin' >> ~/.profile
$ protoc --version
libprotoc 3.28.3
```

#### Protobuf Go Runtime

```shell
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Protobuf Compilation
```shell
$ make compile
```
