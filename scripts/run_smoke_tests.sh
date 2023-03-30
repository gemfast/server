#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

sudo dpkg -i gemfast-pkg.deb
sudo systemctl start gemfast

cd ./clones

pushd rails
bundle config mirror.https://rubygems.org http://localhost:2020
bundle
popd