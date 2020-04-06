FROM golang:1-alpine as builder

ARG GOLANG_NAMESPACE="github.com/mohemohe/switchfeed"
ENV GOLANG_NAMESPACE="$GOLANG_NAMESPACE"

RUN apk --no-cache add alpine-sdk coreutils git tzdata upx util-linux zsh
SHELL ["/bin/zsh", "-c"]
RUN cp -f /usr/share/zoneinfo/Asia/Tokyo /etc/localtime
RUN go get -u -v github.com/pwaller/goupx
WORKDIR /go/src/$GOLANG_NAMESPACE
ADD ./go.mod /go/src/$GOLANG_NAMESPACE/
ADD ./go.sum /go/src/$GOLANG_NAMESPACE/
ENV GO111MODULE=on
RUN go mod download
ADD . /go/src/$GOLANG_NAMESPACE/
RUN go build -ldflags "\
      -w \
      -s \
    " -o /build/app
RUN goupx /build/app

# ====================================================================================

FROM alpine

RUN apk --no-cache add ca-certificates
COPY --from=builder /etc/localtime /etc/localtime
COPY --from=builder /build/app /switchfeed/app

EXPOSE 8080
WORKDIR /switchfeed
CMD ["/switchfeed/app"]