MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))
BIN      = $(CURDIR)/.bin

GOLANGCI_VERSION = v1.47.2

GO           = go
TIMEOUT_UNIT = 5m
TIMEOUT_E2E  = 20m
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

export GO111MODULE=on

COMMANDS=$(patsubst cmd/%,%,$(wildcard cmd/*))
BINARIES=$(addprefix bin/,$(COMMANDS))

.PHONY: all
all: fmt $(BINARIES) | $(BIN) ; $(info $(M) building executable…) @ ## Build program binary

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); cd $$tmp; \
		env GO111MODULE=on GOPATH=$$tmp GOBIN=$(BIN) $(GO) install $(PACKAGE) \
		|| ret=$$?; \
		env GO111MODULE=on GOPATH=$$tmp GOBIN=$(BIN) $(GO) clean -modcache \
        || ret=$$?; \
		cd - ; \
	  	rm -rf $$tmp ; exit $$ret

FORCE:

bin/%: cmd/% FORCE
	$Q $(GO) build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

KO = $(or ${KO_BIN},${KO_BIN},$(BIN)/ko)
$(BIN)/ko: PACKAGE=github.com/google/ko@latest

.PHONY: apply
apply: | $(KO) ; $(info $(M) ko apply core manifests (excluding optional/)) @ ## Apply core config to the current cluster (excludes optional/)
	$Q $(KO) apply -f config/200-clusterrole.yaml \
		-f config/200-role.yaml \
		-f config/200-serviceaccount.yaml \
		-f config/201-clusterrolebinding.yaml \
		-f config/201-rolebinding.yaml \
		-f config/600-tekton-pruner-default-spec.yaml \
		-f config/config-info.yaml \
		-f config/config-logging.yaml \
		-f config/config-observability.yaml \
		-f config/controller.yaml \
		-f config/metrics-service.yaml

.PHONY: apply-all
apply-all: | $(KO) ; $(info $(M) ko apply all manifests (including optional/)) @ ## Apply all config to the current cluster (includes optional/)
	$Q $(KO) apply -R -f config

.PHONY: apply-optional
apply-optional: | $(KO) ; $(info $(M) ko apply optional manifests only) @ ## Apply only optional manifests (requires additional components)
	$Q $(KO) apply -R -f config/optional

.PHONY: deploy-monitoring
deploy-monitoring: ; $(info $(M) deploying monitoring stack (Prometheus + Grafana)) @ ## Deploy monitoring stack for development/testing
	$Q kubectl apply -f examples/monitoring/kind-setup.yaml
	@echo "$(M) Waiting for monitoring pods to be ready..."
	$Q kubectl wait --for=condition=Ready pod -l app=prometheus -n monitoring --timeout=300s
	$Q kubectl wait --for=condition=Ready pod -l app=grafana -n monitoring --timeout=300s
	@echo "$(M) Monitoring stack deployed successfully!"
	@echo "$(M) Prometheus: kubectl port-forward svc/prometheus 9090:9090 -n monitoring"
	@echo "$(M) Grafana: kubectl port-forward svc/grafana 3000:3000 -n monitoring (admin/admin)"

# Tekton deployment targets
.PHONY: deploy-tekton
deploy-tekton: ; $(info $(M) deploying Tekton Pipelines) @ ## Deploy Tekton Pipelines to the current cluster
	$Q kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
	@echo "$(M) Waiting for Tekton Pipelines to be ready..."
	$Q kubectl wait --for=condition=Ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=300s
	$Q kubectl wait --for=condition=Ready pod -l app=tekton-pipelines-webhook -n tekton-pipelines --timeout=300s
	@echo "$(M) Tekton Pipelines deployed successfully!"

.PHONY: clean-tekton
clean-tekton: ; $(info $(M) removing Tekton Pipelines) @ ## Remove Tekton Pipelines from the current cluster
	$Q kubectl delete --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml --ignore-not-found=true
	@echo "$(M) Tekton Pipelines removed"

.PHONY: status-tekton
status-tekton: ; $(info $(M) checking Tekton Pipelines status) @ ## Check Tekton Pipelines status
	@echo "$(M) Tekton Pipelines namespace status:"
	$Q kubectl get namespace tekton-pipelines 2>/dev/null || echo "  Tekton Pipelines namespace not found"
	@echo "$(M) Tekton Pipelines pods:"
	$Q kubectl get pods -n tekton-pipelines 2>/dev/null || echo "  No Tekton Pipelines pods found"
	@echo "$(M) Tekton Pipelines services:"
	$Q kubectl get svc -n tekton-pipelines 2>/dev/null || echo "  No Tekton Pipelines services found"

.PHONY: logs-tekton
logs-tekton: ; $(info $(M) showing Tekton Pipelines logs) @ ## Show Tekton Pipelines logs
	@echo "$(M) Tekton Pipelines Controller logs:"
	$Q kubectl logs -l app=tekton-pipelines-controller -n tekton-pipelines --tail=20 2>/dev/null || echo "  Tekton Pipelines Controller not running"
	@echo "$(M) Tekton Pipelines Webhook logs:"
	$Q kubectl logs -l app=tekton-pipelines-webhook -n tekton-pipelines --tail=20 2>/dev/null || echo "  Tekton Pipelines Webhook not running"

.PHONY: deploy-all-with-monitoring
deploy-all-with-monitoring: deploy-tekton apply deploy-monitoring ; $(info $(M) deploying Tekton + tektoncd-pruner + monitoring stack) @ ## Deploy Tekton Pipelines + tektoncd-pruner + monitoring stack
	@echo "$(M) Complete deployment finished!"
	@echo "$(M) Tekton Pipelines: kubectl get pods -n tekton-pipelines"
	@echo "$(M) tektoncd-pruner: kubectl get pods -n tekton-pipelines -l app=controller"
	@echo "$(M) Monitoring: kubectl get pods -n monitoring"

.PHONY: deploy-tekton-with-pruner
deploy-tekton-with-pruner: deploy-tekton apply ; $(info $(M) deploying Tekton + tektoncd-pruner) @ ## Deploy Tekton Pipelines + tektoncd-pruner (no monitoring)
	@echo "$(M) Tekton + tektoncd-pruner deployment finished!"
	@echo "$(M) Tekton Pipelines: kubectl get pods -n tekton-pipelines"
	@echo "$(M) tektoncd-pruner: kubectl get pods -n tekton-pipelines -l app=controller"

.PHONY: clean-monitoring
clean-monitoring: ; $(info $(M) removing monitoring stack) @ ## Remove monitoring stack
	$Q kubectl delete -f examples/monitoring/kind-setup.yaml --ignore-not-found=true
	@echo "$(M) Monitoring stack removed"

.PHONY: status-monitoring
status-monitoring: ; $(info $(M) checking monitoring stack status) @ ## Check monitoring stack status
	@echo "$(M) Monitoring namespace status:"
	$Q kubectl get namespace monitoring 2>/dev/null || echo "  Monitoring namespace not found"
	@echo "$(M) Monitoring pods:"
	$Q kubectl get pods -n monitoring 2>/dev/null || echo "  No monitoring pods found"
	@echo "$(M) Monitoring services:"
	$Q kubectl get svc -n monitoring 2>/dev/null || echo "  No monitoring services found"
	@echo "$(M) tektoncd-pruner metrics service:"
	$Q kubectl get svc tekton-pruner-controller-metrics -n tekton-pipelines 2>/dev/null || echo "  tektoncd-pruner metrics service not found"

.PHONY: logs-monitoring
logs-monitoring: ; $(info $(M) showing monitoring logs) @ ## Show monitoring stack logs
	@echo "$(M) Prometheus logs:"
	$Q kubectl logs -l app=prometheus -n monitoring --tail=20 2>/dev/null || echo "  Prometheus not running"
	@echo "$(M) Grafana logs:"
	$Q kubectl logs -l app=grafana -n monitoring --tail=20 2>/dev/null || echo "  Grafana not running"

.PHONY: status-all
status-all: status-tekton status-monitoring ; $(info $(M) checking all components status) @ ## Check status of all components (Tekton + tektoncd-pruner + monitoring)
	@echo "$(M) Overall system status check completed"

.PHONY: resolve
resolve: | $(KO) ; $(info $(M) ko resolve -R -f config/) @ ## Resolve config to the current cluster
	$Q $(KO) resolve --push=false --oci-layout-path=$(BIN)/oci -R -f config

.PHONY: generated
generated: | vendor ; $(info $(M) update generated files) ## Update generated files
	$Q ./hack/update-codegen.sh

.PHONY: vendor
vendor:
	$Q ./hack/update-deps.sh

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: clean-all
clean-all: clean clean-tekton clean-monitoring ; $(info $(M) cleaning everything including Tekton and monitoring) @ ## Cleanup everything including Tekton Pipelines and monitoring stack

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:

	@echo $(VERSION)

.PHONY: dev-setup
dev-setup: # setup kind with local registry for local development
	@cd ./hack/dev/kind/;./install.sh

.PHONY: dev-setup-with-monitoring
dev-setup-with-monitoring: dev-setup deploy-all-with-monitoring ; $(info $(M) complete dev setup with Tekton + tektoncd-pruner + monitoring) @ ## Setup kind cluster + Tekton Pipelines + tektoncd-pruner + monitoring
	@echo "$(M) Development environment ready!"
	@echo "$(M) Tekton Pipelines: kubectl get pods -n tekton-pipelines"
	@echo "$(M) tektoncd-pruner: kubectl get pods -n tekton-pipelines -l app=controller"
	@echo "$(M) Prometheus: kubectl port-forward svc/prometheus 9090:9090 -n monitoring"
	@echo "$(M) Grafana: kubectl port-forward svc/grafana 3000:3000 -n monitoring"

.PHONY: dev-setup-minimal
dev-setup-minimal: dev-setup deploy-tekton-with-pruner ; $(info $(M) minimal dev setup with Tekton + tektoncd-pruner) @ ## Setup kind cluster + Tekton Pipelines + tektoncd-pruner (no monitoring)
	@echo "$(M) Minimal development environment ready!"
	@echo "$(M) Tekton Pipelines: kubectl get pods -n tekton-pipelines"
	@echo "$(M) tektoncd-pruner: kubectl get pods -n tekton-pipelines -l app=controller"

#Release
RELEASE_VERSION=v0.0.0
RELEASE_DIR ?= /tmp/tektoncd-pruner-${RELEASE_VERSION}

.PHONY: github-release
github-release:
	./hack/release.sh ${RELEASE_VERSION}

