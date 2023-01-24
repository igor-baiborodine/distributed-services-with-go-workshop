### Prerequisites

#### Protobuf Compiler

```shell
$ PB_REL="https://github.com/protocolbuffers/protobuf/releases"
$ curl -LO $PB_REL/download/v21.12/protoc-21.12-linux-x86_64.zip
$ unzip protoc-21.12-linux-x86_64.zip -d /usr/local/protobuf
$ echo 'PATH=$PATH:/usr/local/protobuf/bin' >> ~/.profile
$ protoc --version
libprotoc 3.21.12
```

#### Protobuf Go Runtime and gRPC Plugin

```shell
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install google.golang.org/grpc@latest
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Protobuf Compilation
```shell
$ 	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
# or
$ make compile
```