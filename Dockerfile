FROM gsoci.azurecr.io/giantswarm/alpine:3.22.1 AS binaries

ARG KUBECTL_VERSION=1.24.2
ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates curl \
    && mkdir -p /binaries \
    && curl -sSLf https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${TARGETPLATFORM}/kubectl -o /binaries/kubectl \
    && chmod +x /binaries/*

FROM gsoci.azurecr.io/giantswarm/alpine:3.22.1

ARG TARGETARCH

COPY --from=binaries /binaries/* /usr/bin/
COPY ./kubectl-gs-${TARGETARCH} /usr/bin/kubectl-gs
RUN ln -s /usr/bin/kubectl-gs /usr/bin/kgs

ENTRYPOINT ["kubectl-gs"]
