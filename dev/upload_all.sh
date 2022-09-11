#!/bin/bash

cd test/gems
for gem in *.gem; do
  [ -f "$gem" ] || break
  gem push "$gem" --host http://localhost:8080 -k gemfast
done
