#!/bin/bash

set -ueo pipefail

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

if [[ "$BUILD_TYPE" == "docker" ]]; then
  docker load -i gemfast*.tar
  docker run -d --name gemfast -p 80:2020 -v /etc/gemfast:/etc/gemfast -v /var/gemfast:/var/gemfast -v /etc/machine-id:/etc/machine-id gemfast:latest
  sleep 5
  docker ps
  docker logs gemfast
else
  sudo dpkg -i gemfast*.deb
  sudo systemctl start gemfast
  sleep 10
  sudo systemctl status gemfast
  sleep 2
  sudo systemctl status caddy

  journalctl -u gemfast
  journalctl -u caddy
fi

cd ./clones

pushd rails
bundle config mirror.https://rubygems.org http://localhost
bundle

numGems=$(curl -s http://localhost/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
curl -s http://localhost/admin/api/v1/backup > gemfast.db

if [[ "$BUILD_TYPE" == "docker" ]]; then
  docker stop gemfast
  sudo rm -rf /var/gemfast/db/gemfast.db
  sudo mv ./gemfast.db /var/gemfast/db/gemfast.db
  sleep 5
  docker start gemfast
  sleep 5
  docker ps
  docker logs gemfast
else
  sudo systemctl stop gemfast
  sudo rm -rf /var/gemfast/db/gemfast.db
  sudo mv ./gemfast.db /var/gemfast/db/gemfast.db
  sudo chown gemfast: /var/gemfast/db/gemfast.db
  sleep 2
  sudo systemctl start gemfast
  sleep 5
  sudo systemctl status gemfast
  sleep 2
  sudo systemctl status caddy
fi

numGemsBackup=$(curl -s http://localhost/admin/api/v1/stats/bucket | jq -r '.gems.KeyN')
if [ "$numGems" != "$numGemsBackup" ]; then
  echo "Number of gems in backup ($numGemsBackup) does not match number of gems in original ($numGems)"
  exit 1
fi

# TODO: make this test work
# mv Gemfile Gemfile.backup
# cat << GEMFILE > Gemfile
# source "https://rubygems.org"
# gem "rake", ">= 13"
# GEMFILE

# bundle clean --force
# ls -la /var/gemfast/gems

# mv Gemfile.backup Gemfile
# sed -i -e 's/https:\/\/rubygems.org/http:\/\/localhost\/private/g' Gemfile
# rm Gemfile.lock
# bundle
popd