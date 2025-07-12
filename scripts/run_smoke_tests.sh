#!/bin/bash

set -ueo pipefail

source ./scripts/_functions.sh

ruby --version
bundle --version

gem update --system

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << CONFIG > /etc/gemfast/gemfast.hcl
auth "none" {}
CONFIG

start_server "$BUILD_TYPE"

pushd ./clones/rails

bundle config mirror.https://rubygems.org http://localhost:2020
bundle
popd

numGems=$(curl -s http://localhost:2020/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
curl -s http://localhost:2020/admin/api/v1/backup > gemfast.db

sudo rm -f ./data/db/gemfast.db
sudo mv ./gemfast.db ./data/db/gemfast.db

if [ "$BUILD_TYPE" != "docker" ]; then
  sudo chown gemfast: ./data/db/gemfast.db
fi

restart_server "$BUILD_TYPE"

numGemsBackup=$(curl -s http://localhost:2020/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
if [ "$numGems" != "$numGemsBackup" ]; then
  echo "Number of gems in backup ($numGemsBackup) does not match number of gems in original ($numGems)"
  exit 1
fi

metrics=$(curl -s http://localhost:2020/metrics | grep "gemfast")
if [[ $metrics == "" ]] ; then
  echo "Metrics endpoint is not working"
  exit 1
fi
