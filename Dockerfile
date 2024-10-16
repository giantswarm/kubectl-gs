FROM gsoci.azurecr.io/giantswarm/alpine:3.20.3 AS binaries

ARG KUBECTL_VERSION=1.24.2
ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates curl \
    && mkdir -p /binaries \
    && curl -sSLf https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${TARGETPLATFORM}/kubectl -o /binaries/kubectl \
    && chmod +x /binaries/*

FROM gsoci.azurecr.io/giantswarm/alpine:3.20.3

ARG TARGETPLATFORM

COPY --from=binaries /binaries/* /usr/bin/
COPY ./kubectl-gs/platform/${TARGETPLATFORM}/kubectl-gs /usr/bin/kubectl-gs
RUN ln -s /usr/bin/kubectl-gs /usr/bin/kgs

ENTRYPOINT ["kubectl-gs"]
