#!/bin/bash

GOPATH="${pwd}"
GOBIN="${pwd}/bin"
export PATH="$PATH:$GOBIN"
alias test="go install cdf_score_finder.go && cdf_score_finder"
