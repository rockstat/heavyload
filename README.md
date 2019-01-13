# Heavyload

**Draft state**

File upload handler to Rockstat platform
Receives files and send webhook notification to Kernel component

### Running in Docker

Firstr build image using Dockerfile

    docker build -t rst/heavyload .

Start container with port mapping

    docker run -d \
        --name=heavyload --hostname=heavyload \
        --restart=unless-stopped \
        --network=custom \
        -p 127.0.0.1:10010:8080 \
        -e WEBHOOK=http://host.docker.internal:10001/wh/upload/mysrv/actionname \
        rst/heavyload


### Response struct 

Contains webhook notifiction payload and response
Test using `httpie`

```shell
http --form http://127.0.0.1:18080/upload/mysrv/actionname upload@test.jpg
```


will return 

```
HTTP/1.1 200 OK
Content-Length: 188
Content-Type: application/json
Date: Tue, 08 May 2018 23:41:01 GMT
```

```json
{
    "files": [
        {
            "name": "100stripusers.log",
            "param": "log",
            "size": 43762756,
            "tempName": "6b708b18-3e86-437f-bcdf-321a9f771e45"
        }
    ],
    "name": "callme",
    "query": {},
    "service": "srv"
}
```

### Deps

```
govendor fetch github.com/gin-gonic/gin
```