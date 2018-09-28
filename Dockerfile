FROM golang:alpine

RUN apk add --quiet --no-cache build-base ca-certificates git

ADD . /go/src/fullrss

RUN go get -d -v fullrss
RUN go install -ldflags "-linkmode external -extldflags -static" fullrss


FROM scratch

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /go/bin/fullrss /
COPY fullrss.yaml /

EXPOSE 8000

ENTRYPOINT [ "/fullrss" ]