FROM golang:1.10-stretch as build

WORKDIR /go/src/github.com/hellolijj/k8s-device-plugin
COPY . .

RUN export CGO_LDFLAGS_ALLOW='-Wl,--unresolved-symbols=ignore-in-object-files' && \
go build -ldflags="-s -w" -o /go/bin/gputopology-device-plugin main.go

FROM debian:stretch-slim

ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=utility

COPY --from=build /go/bin/gputopology-device-plugin /usr/bin/gputopology-device-plugin

CMD ["gputopology-device-plugin","-logtostderr"]