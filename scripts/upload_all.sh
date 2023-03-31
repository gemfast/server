#!/bin/bash
set -euo pipefail

gems="/var/gemfast/gems"
cd $gems
for gem in *.gem; do
  [ -f "$gem" ] || break
  echo "Uploading $gem"
  gem push "$gem" --host http://localhost:8080 -k gemfast
done