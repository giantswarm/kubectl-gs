FROM gsoci.azurecr.io/giantswarm/alpine:3.20.3 AS binaries

ARG KUBECTL_VERSION=1.24.2

RUN apk add --no-cache ca-certificates curl \
    && mkdir -p /binaries \
    && curl -sSLf https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl -o /binaries/kubectl \
    && chmod +x /binaries/*

FROM gsoci.azurecr.io/giantswarm/alpine:3.20.3

COPY --from=binaries /binaries/* /usr/bin/
COPY ./kubectl-gs /usr/bin/kubectl-gs
RUN ln -s /usr/bin/kubectl-gs /usr/bin/kgs

ENTRYPOINT ["kubectl-gs"]
