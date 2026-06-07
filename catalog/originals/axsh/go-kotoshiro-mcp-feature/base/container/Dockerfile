# syntax=docker/dockerfile:1
# dev: Go + bash（run.sh 対話用）。prod: scripts ビルドの bin/function.exe を alpine で実行。

FROM golang:1.26-alpine AS dev
RUN apk add --no-cache bash

FROM alpine:3.21 AS prod
RUN apk add --no-cache ca-certificates
COPY bin/function.exe /usr/local/bin/function.exe
ENTRYPOINT ["/usr/local/bin/function.exe"]
