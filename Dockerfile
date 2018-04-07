FROM golang:1.10 AS build

WORKDIR /go/src/github.com/danillouz/api/

RUN go get github.com/gin-gonic/gin
RUN go get github.com/mitchellh/mapstructure
RUN go get k8s.io/api/batch/v1
RUN go get k8s.io/api/core/v1
RUN go get k8s.io/apimachinery/pkg/apis/meta/v1
RUN go get k8s.io/client-go/kubernetes
RUN go get k8s.io/client-go/rest

COPY server.go .

RUN CGO_ENABLED=0 GOOS=linux go build .

FROM scratch
COPY --from=build /go/src/github.com/danillouz/api .
ENTRYPOINT [ "/api" ]