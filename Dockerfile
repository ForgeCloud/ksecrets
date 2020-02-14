FROM golang:1.13.7-alpine3.11 AS build
RUN apk add --no-cache curl git openssh build-base
ENV GO111MODULE=on
COPY . /go/src/github.com/ForgeCloud/ksecrets/
RUN cd /go/src/github.com/ForgeCloud/ksecrets/kustomize/plugin/crd.forgecloud.com/v1/encryptedsecret && \
    go get -gcflags="all=-N -l" sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 && \
    go build -gcflags="all=-N -l" -buildmode plugin -o EncryptedSecret.so encryptedsecret.go

FROM alpine:3.11
RUN apk add --no-cache ca-certificates
ENV XDG_CONFIG_HOME=/
COPY --from=build /go/src/github.com/ForgeCloud/ksecrets/kustomize/plugin/crd.forgecloud.com/v1/encryptedsecret/EncryptedSecret.so /kustomize/plugin/crd.forgecloud.com/v1/encryptedsecret/
COPY --from=build /go/bin/kustomize /usr/local/bin/kustomize
ENTRYPOINT ["sh", "-c"]

