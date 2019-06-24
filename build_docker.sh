docker build -t ulikl/oapi-exporter:v0.1.0 -f Dockerfile .
docker tag  ulikl/oapi-exporter:v0.1.0  ulikl/oapi-exporter:latest
docker push  ulikl/oapi-exporter:latest
docker push ulikl/oapi-exporter:v0.1.0
