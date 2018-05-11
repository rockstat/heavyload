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
        -e WEBHOOK=http://host.docker.internal:10001/wh/upload/notify \
        rst/heavyload


### Response struct 

Contains webhook notifiction payload and response
Test using `httpie`

    http --form http://127.0.0.1:18080/upload upload@test.jpg

will return 

    HTTP/1.1 200 OK
    Content-Length: 188
    Content-Type: application/json
    Date: Tue, 08 May 2018 23:41:01 GMT

    {
        "message": "OK",
        "payload": {
            "files": [
                {
                    "fn": "7e27a3d38927ea06b6979655",
                    "orig_fn": "test.jpg",
                    "size": 91141
                }
            ],
            "success": true
        },
        "resp": {
            "id": "6399764926386143233",
            "key": "in.indep.upload.notify"
        }
    }


