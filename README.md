# _Fetch Rewards_' Receipt Processor Challenge
## About

A web service that implements the API described at https://github.com/fetch-rewards/receipt-processor-challenge/.
Built using [Go](https://go.dev/) 1.23.3 and [Gin](https://github.com/gin-gonic/gin). 

## How to build and run from source:

Requirements: 
- Go 1.23.3

(This project was built on Linux.)
- In one shell, change current directory to the root of the source folder, and do the following:
+ `go get .`
+ `go run .`
- In a separate shell, make your requests by `curl`, targeting `http://localhost:8080/`

Example requests:
- For the `/receipts/process` endpoint: 

  ```
  $ curl http://localhost:8080/receipts/process \
    --include \
    --header "Content-Type: application/json" \
    --request "POST" \
    --data '{ "retailer": "M&M Corner Market", "purchaseDate": "2022-03-20", "purchaseTime": "14:33", "items": [ { "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" } ], "total": "9.00" }'
  ```

  Example output (valid receipt):
  ```
  HTTP/1.1 200 OK
  Content-Type: application/json; charset=utf-8
  Date: Wed, 20 Nov 2024 06:21:43 GMT
  Content-Length: 45

  {"id":"84d2fac5-f94b-45aa-8d38-8e0cfea7d89a"}
  ```

  Example output (invalid receipt):
  ```
  HTTP/1.1 400 Bad Request
  Date: Wed, 20 Nov 2024 19:30:49 GMT
  Content-Length: 0
  ```
  
- For the `/receipts/:id/points` endpoint: 
  ```
  $ curl http://localhost:8080/receipts/e4392770-31bb-4b38-8494-aaf1560a1a48/points \
    --include \
    --header "Content-Type: application/json" \
    --request "GET"
  ```
  
  Example output (valid ID matching a previously processed receipt):
  ```
  HTTP/1.1 200 OK
  Content-Type: application/json; charset=utf-8
  Date: Wed, 20 Nov 2024 19:33:46 GMT
  Content-Length: 14

  {"points":109}
  ```

  Example output (invalid/unrecognized ID):
  ```
  HTTP/1.1 404 Not Found
  Date: Wed, 20 Nov 2024 19:36:16 GMT
  Content-Length: 0
  ```






