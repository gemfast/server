#!/bin/bash

set -ueo pipefail

source ./scripts/_functions.sh

ruby --version
bundle --version

gem update --system

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << CONFIG > /etc/gemfast/gemfast.hcl
caddy {
  port = 80
  host = "http://localhost"
}
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
CONFIG

start_server "$BUILD_TYPE"

cd ./clones

pushd rails
bundle config mirror.https://rubygems.org http://localhost
bundle

numGems=$(curl -s http://localhost/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
curl -s http://localhost/admin/api/v1/backup > gemfast.db

sudo rm -rf /var/gemfast/db/gemfast.db
sudo mv ./gemfast.db /var/gemfast/db/gemfast.db

if [ "$BUILD_TYPE" != "docker" ]; then
  sudo chown gemfast: /var/gemfast/db/gemfast.db
fi

restart_server "$BUILD_TYPE"

numGemsBackup=$(curl -s http://localhost/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
if [ "$numGems" != "$numGemsBackup" ]; then
  echo "Number of gems in backup ($numGemsBackup) does not match number of gems in original ($numGems)"
  exit 1
fi