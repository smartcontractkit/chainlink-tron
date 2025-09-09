ARG BASE_IMAGE=local_chainlink

FROM golang:1.24.5-bullseye AS buildplugins
RUN go version

WORKDIR /build
COPY relayer .
RUN go install ./cmd/chainlink-tron

FROM ${BASE_IMAGE}
COPY --from=buildplugins /go/bin/chainlink-tron /usr/local/bin/
ENV CL_TRON_CMD chainlink-tron
