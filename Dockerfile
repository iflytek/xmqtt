FROM golang:latest

COPY ./xmqtt /opt/xmqtt/
COPY ./config.yaml /opt/xmqtt/
WORKDIR /opt/xmqtt
CMD ./xmqtt