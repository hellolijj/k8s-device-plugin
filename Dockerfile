FROM golang:1.10-stretch as build

WORKDIR /go/src/gsoc-device-plugin
COPY . .

RUN export CGO_LDFLAGS_ALLOW='-Wl,--unresolved-symbols=ignore-in-object-files' && \
go install -ldflags="-s -w" -v gsoc-device-plugin

FROM debian:stretch-slim

ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=utility

COPY --from=build /go/bin/gsoc-device-plugin /usr/bin/gsoc-device-plugin

CMD ["gsoc-device-plugin","-logtostderr"]