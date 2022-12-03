#########################
# Build
#########################

FROM golang:1.17 as builder

WORKDIR /go/src/github.com/dtrumpfheller/toronto-hydro-exporter

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify

COPY *.go .
COPY helpers/*.go ./helpers/
COPY torontohydro/*.go ./torontohydro/
COPY influxdb/*.go ./influxdb/

RUN CGO_ENABLED=0 go build -o /go/bin/app .


#########################
# Deploy
#########################

FROM gcr.io/distroless/static

USER nobody:nobody

COPY --from=builder --chown=nobody:nobody /go/bin/app /toronto-hydro-exporter/

ENTRYPOINT ["/toronto-hydro-exporter/app"]