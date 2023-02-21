## Write a Booking Log Package

### Glossary

* **Record**: A booking data stored in the log.
* **Record Offset**: An integer indicating the distance between the
  beginning of the log and a given record.
* **Store**: A file where the records are stored.
* **Store offset**: An integer indicating the distance in bytes between the
  beginning of the file and a given point within the same file where a record is
  stored.
* **Offset Index**: A file storing index entries, where an index entry
  consists of the record offset mapped to a store offset.
* **Memory-mapped File**: a part of virtual memory assigned a direct
  byte-to-byte correlation with some portion of the file.
* **Segment**: An abstraction that ties a store and an offset index together.
* **Log**: An abstraction that ties all segments together.

### Tests

```shell
$ make compile test
```
