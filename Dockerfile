FROM centos:7.4
#alpine:3.9
MAINTAINER Klusik, Ulrike; ulrike.klusik@consol.de

COPY oapi-exporter /bin/oapi-exporter

CMD ["/bin/oapi-exporter"]

