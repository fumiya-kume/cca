#!/bin/bash
set -e

# Run tests with necessary permissions
deno test --allow-all

# Run linter
deno lint
