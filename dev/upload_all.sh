#!/bin/bash

for gem in *.gem; do
  [ -f "$gem" ] || break
  gem push "$gem" --host http://localhost:8080 -k someother_api_key
done