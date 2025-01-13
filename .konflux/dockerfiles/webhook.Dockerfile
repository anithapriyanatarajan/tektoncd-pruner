ARG GO_BUILDER=brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.23
ARG RUNTIME=registry.access.redhat.com/ubi9/ubi-minimal@sha256:b87097994ed62fbf1de70bc75debe8dacf3ea6e00dd577d74503ef66452c59d6

FROM $GO_BUILDER AS builder

WORKDIR /go/src/github.com/openshift-pipelines/tektoncd-pruner
COPY . .

RUN go build -v -o /tmp/webhook  ./cmd/webhook

FROM $RUNTIME
ARG VERSION=tektoncd-pruner

COPY --from=builder /tmp/webhook /ko-app/webhook


LABEL \
      com.redhat.component="openshift-pipelines-tektoncd-pruner-webhook" \
      name="openshift-pipelines/pipelines-tektoncd-pruner-rhel9" \
      version=$VERSION \
      summary="Red Hat OpenShift Pipelines Tekton Pruner Webhook" \
      maintainer="pipelines-extcomm@redhat.com" \
      description="Red Hat OpenShift Pipelines Tekton Pruner Webhook" \
      io.k8s.display-name="Red Hat OpenShift Pipelines Tekton Pruner Webhook" \
      io.k8s.description="Red Hat OpenShift Pipelines Tekton Pruner Webhook" \
      io.openshift.tags="pipelines,tekton,openshift,tektoncd-pruner-webhook"

RUN microdnf install -y shadow-utils
RUN groupadd -r -g 65532 nonroot && useradd --no-log-init -r -u 65532 -g nonroot nonroot
USER 65532
