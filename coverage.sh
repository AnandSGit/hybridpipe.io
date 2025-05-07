#!/bin/bash
# Script to run code coverage analysis for the hybridpipe.io project

set -e

# Run tarpaulin with coverage report
cargo tarpaulin --out Html --output-dir coverage

echo "Coverage report generated in the coverage directory"
