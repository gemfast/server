#!/bin/bash
set -euo pipefail

gems_dir="./data/gems"
cd $gems_dir
for gem in *.gem; do
  [ -f "$gem" ] || break
  echo "Uploading $gem"
  gem push "$gem" --host http://localhost:2020 -k gemfast
done