## Let's Go

### Test Your API

Start the server:
```shell
$ go run cmd/server/main.go
```

Add records:
```shell
$ curl -X POST localhost:8080 -d '{"record": { "value": "TGV0J3MgR28gIzMK" }}'
$ curl -X POST localhost:8080 -d '{"record": { "value": "TGV0J3MgR28gIzEK" }}'
$ curl -X POST localhost:8080 -d '{"record": { "value": "TGV0J3MgR28gIzIK" }}'
```

Retrieve records:
```shell
$ curl -X GET localhost:8080 -d '{"offset": 0}'
$ curl -X GET localhost:8080 -d '{"offset": 1}'
$ curl -X GET localhost:8080 -d '{"offset": 2}'
```
