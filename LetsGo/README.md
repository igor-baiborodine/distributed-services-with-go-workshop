## Let's Go

### Test Your API

Start the server:
```shell
$ go run cmd/server/main.go
```

Create a booking:
```shell
$ curl -X POST localhost:8080 -d \
  '{"booking": { "email": "john.smith@email.com", "fullName": "John Smith", "startDate": "2022-08-21", "endDate": "2022-08-23" }}'
# response
{"uuid":"0c7b89ea-150b-4fc7-a768-d0d8b7a1dff1"}  
```

Retrieve the newly created booking:
```shell
$ curl -X GET localhost:8080 -d '{"uuid": "0c7b89ea-150b-4fc7-a768-d0d8b7a1dff1"}'
# response
{"booking":{"uuid":"0c7b89ea-150b-4fc7-a768-d0d8b7a1dff1","email":"john.smith@email.com","fullName":"John Smith","startDate":"2022-08-21","endDate":"2022-08-23","active":true}}
```

Retrieve non-existing booking:
```shell
$ curl -X GET localhost:8080 -d '{"uuid": "non-existing-booking-uuid"}'
# response
booking not found
```