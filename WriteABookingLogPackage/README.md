## Write a Booking Log Package

### Glossary

* Record - a booking data stored in the log.
* Store - a file where the records are stored.
* Index - a file where the index entries are stored.
* Segment - an abstraction that ties a store and an index together.
* Log - an abstraction that ties all segments together.
* Offset - an integer indicating the distance in bytes between the beginning of
  the file and a given point within the same file.
* Memory-mapped file - a part of virtual memory that has been assigned a 
  direct byte-to-byte correlation with some portion of the file. 

### Tests

```shell
$ make compile
$ make test
```
