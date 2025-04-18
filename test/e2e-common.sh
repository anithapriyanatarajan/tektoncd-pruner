#!/usr/bin/env bash

# Copyright 2025 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

# Helper functions for E2E tests.

# Check if we have a specific RELEASE_YAML global environment variable to use
# instead of detecting the latest released one from tektoncd/pipeline releases
RELEASE_YAML=${RELEASE_YAML:-}

source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/e2e-tests.sh

function install_tkn() {
  echo ">> Installing tkn"
  TKN_VERSION=0.40.0
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v$TKN_VERSION/tkn_$TKN_VERSION_Linux_x86_64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  tar xvzf tkn_$TKN_VERSION_Linux_x86_64.tar.gz -C /usr/local/bin/ tkn
}

function install_pipeline_crd() {
  local latestreleaseyaml
  echo ">> Deploying Tekton Pipelines"
  if [[ -n ${RELEASE_YAML} ]];then
	  latestreleaseyaml=${RELEASE_YAML}
  else
    latestreleaseyaml=$(curl -s https://api.github.com/repos/tektoncd/pipeline/releases|python -c "import sys, json;x=json.load(sys.stdin);ass=x[0]['assets'];print([ x['browser_download_url'] for x in ass if x['name'] == 'release.yaml'][0])")
  fi
  [[ -z ${latestreleaseyaml} ]] && fail_test "Could not get latest released release.yaml"
  kubectl apply -f ${latestreleaseyaml} ||
    fail_test "Tekton pipeline installation failed"

  # Make sure that eveything is cleaned up in the current namespace.
  for res in pipelineresources tasks pipelines taskruns pipelineruns; do
    kubectl delete --ignore-not-found=true ${res}.tekton.dev --all
  done

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-pipelines || fail_test "Tekton Pipeline did not come up"
}

function install_pruner() {
  echo ">> Deploying Tekton Pruner"
  ko apply -f config/ || fail_test "Tekton Pruner installation failed"

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running ${namespace} || fail_test "Tekton Pruner did not come up"
}