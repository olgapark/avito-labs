# syntax=docker/dockerfile:1

FROM golang:1.27.1-trixie@sha256:433790e515d27dc6003e847e644cc0af956985cf315c1c58a3b73ee2dd305183 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY --parents **/*.go migrations Makefile ./
RUN make build

FROM gcr.io/distroless/base-debian13@sha256:0ebad3510af52aefe45045cc01b07564570be4feecf8d9f93d3a05d1b5f2f93b AS app

USER 1000:1000
COPY --from=builder --chmod=+x /build/bin/tripgo /tripgo

ENV HTTP_ADDR=':8080'
EXPOSE 8080

ENTRYPOINT ["/tripgo"]
