#!/bin/bash

set -e

$(dirname $0)/circle-deps.sh
$(dirname $0)/circle-test.sh
