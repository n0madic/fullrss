FROM golang:alpine AS builder

RUN apk add --quiet --no-cache ca-certificates git

WORKDIR /src

ADD go.* ./

RUN go mod download

ADD *.go ./

RUN go install --tags netgo,osusergo -ldflags="-s -w"


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/* /usr/bin/
COPY fullrss.yaml /

EXPOSE 8000

ENTRYPOINT [ "fullrss" ]
