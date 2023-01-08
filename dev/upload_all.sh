#!/bin/bash
set -euo pipefail

gems="$HOME/.rvm/src/ruby-3.1.2/gems"
for gem in *.gem; do
  [ -f "$gem" ] || break
  echo "Uploading $gem"
  gem push "$gem" --host http://localhost:8080 -k gemfast
done