## Write a Log Package

### Glossary

* **Record**: A data stored in the log.
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
