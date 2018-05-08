# Heavyload

**Draft state**

File upload handler to Rockstat platform
Receives files and send webhook notification to Kernel component

### Running in Docker

Firstr build image using Dockerfile

    docker build -t rst/heavyload .

Start container with port mapping

    docker run -d --rm \
        --name=heavyload --hostname=heavyload \
        -p 127.0.0.1:18080:8080 \
        -e WEBHOOK=http://host.docker.internal:10001/wh/upload/notify \
        rst/heavyload

