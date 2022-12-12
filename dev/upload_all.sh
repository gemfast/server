#!/bin/bash

# cd test/gems
# gems="$HOME/.rvm/src/ruby-3.1.2/gems"
# for gem in *.gem; do
#   [ -f "$gem" ] || break
#   gem push "$gem" --host http://localhost:8080 -k gemfast
# done

find $HOME/.rvm -name "*.gem" -type f -exec gem push {} --host http://localhost:8080 -k gemfast \;