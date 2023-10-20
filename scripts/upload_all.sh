#!/bin/bash
set -euo pipefail

gems_dir="/var/gemfast/gems"
cd $gems_dir
for gem in *.gem; do
  [ -f "$gem" ] || break
  echo "Uploading $gem"
  gem push "$gem" --host http://localhost:8080 -k gemfast
done