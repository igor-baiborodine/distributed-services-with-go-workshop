## Write a Booking Log Package

#### Protobuf Compiler

```shell
$ PB_REL="https://github.com/protocolbuffers/protobuf/releases"
$ curl -LO $PB_REL/download/v21.12/protoc-21.12-linux-x86_64.zip
$ unzip protoc-21.12-linux-x86_64.zip -d /usr/local/protobuf
$ echo 'PATH=$PATH:/usr/local/protobuf/bin' >> ~/.profile
$ protoc --version
libprotoc 3.21.12
```

#### Protobuf Go Runtime

```shell
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Glossary

* **Record**: A booking data stored in the log.
* **Store**: A file where the records are stored.
* **Segment**: An abstraction that ties a store and an offset index together.
* **Log**: An abstraction that ties all segments together.
* **Memory-mapped File**: a part of virtual memory assigned a direct
  byte-to-byte correlation with some portion of the file.
* **Record Offset**: An integer indicating the distance between the
  beginning of the log and a given record.
* **Record Position**: An integer indicating the distance in bytes between the
  beginning of the file and a given point within the same file where a record is
  stored.
* **Index**: A file storing index entries, where the index entry
  consists of the record offset mapped to the record position.

### Tests

```shell
$ make compile test
```
