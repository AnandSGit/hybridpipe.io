#!/bin/bash
# Script to run code coverage analysis for the hybridpipe.io project
#
# NOTE: Rust build process has been temporarily disabled to streamline development
# and reduce computational overhead. The Python implementation remains fully functional.
#
# To re-enable Rust builds, see RUST_BUILD_DISABLED.md for instructions.

set -e

echo "⚠️  Rust build process is currently disabled"
echo "   The Rust implementation build has been temporarily disabled to reduce"
echo "   build time and computational resources during Python-focused development."
echo ""
echo "   To re-enable Rust builds, see RUST_BUILD_DISABLED.md for instructions."
echo ""
echo "   Python coverage can be run with: pytest --cov=hybridpipe --cov-report=html"

# DISABLED: Rust coverage generation
# cargo tarpaulin --out Html --output-dir coverage
# echo "Coverage report generated in the coverage directory"
