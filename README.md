# Catalog API

## Overview
This is a small Microservice example of simple Shopping Catalog API endpoint to be used with [Shipped](http://shipped-cisco.com).

## Getting Started
1. Simple clone the repo or download the repo and cd into the repo and do `go run catalog.go` to start the server on localhost.
2. Then run this simple curl command to make sure the service is up and running `curl -i http://localhost:8000` or view it on your web browser

```
Expected Result

HTTP/1.1 200 OK
Date: Tue, 19 Jan 2016 15:52:02 GMT
Content-Length: 160
Content-Type: text/html; charset=utf-8

<html>
  <head>
    <title>An example layout</title>
  </head>
  <body>

<p>Catalog is up and running.</p>
<p>Try This command</p>
<p></p>

  </body>
</html>

```

## Endpoints

- **GET /v1/catalog/1?mock=true**
```
curl -i -X GET -H "Content-Type: application/json" http://localhost:8000/v1/catalog/1?mock=true
Expected Result:
HTTP/1.1 202 Accepted
Content-Type: application/json
{
  "item_id": 1,
  "name": "BB8 Toy",
  "price": 149.50,
  "image": "images/bb8.png"
}
```
- **GET /v1/catalog?mock=true**
```
curl -i -X GET -H "Content-Type: application/json" http://localhost:8000/v1/catalog/?mock=true
Expected Result:
HTTP/1.1 202 Accepted
Content-Type: application/json
{
  "items": [{
    "item_id": 1,
    "name": "BB8 Toy",
    "price": 149.50,
    "image": "images/bb8.png"
  }, {
    "item_id": 2,
    "name": "Light Saber",
    "price": 14.50,
    "image": "images/lightsaber.jpg"
  }, {
    "item_id": 1,
    "name": "Blaster",
    "price": 20,
    "image": "images/blaster.jpg"
  }]
}
```

## Requirements
* [Go](https://github.com/golang/example)

## Credits
- [Nick Hayward](https://github.com/nehayward)
