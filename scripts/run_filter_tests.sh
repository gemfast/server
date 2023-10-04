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
filter {
    enabled = true
    action = "allow"
    regex = ["stringio"]
}
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

mkdir ./test-filter-allow
pushd test-filter-allow

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "stringio", "~> 3.0", ">= 3.0.7"
CONFIG

bundle config mirror.https://rubygems.org http://localhost
bundle

popd

sudo rm -rf /etc/gemfast/gemfast.hcl
sudo tee /etc/gemfast/gemfast.hcl > /dev/null <<'CONFIG'
caddy {
  port = 80
  host = "http://localhost"
}
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
filter {
    enabled = true
    action = "deny"
    regex = ["active.*"]
}
CONFIG
if [[ "$BUILD_TYPE" == "docker" ]]; then
  docker restart gemfast
  sleep 10
  docker ps
  docker logs gemfast
else
  sudo chown -R $USER: /etc/gemfast
  sudo systemctl restart gemfast
  sleep 5
  sudo systemctl status gemfast
  sleep 2
  sudo systemctl status caddy
fi

mkdir ./test-filter-deny
pushd test-filter-deny

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "activesupport", "~> 7.0", ">= 7.0.5"
CONFIG

bundle config mirror.https://rubygems.org http://localhost
if [[ $(bundle 2>&1 | grep "405") ]]; then
    echo "gemfast is filtering activesupport 7.0.5"
else
    echo "gemfast is not filtering activesupport 7.0.5"
    exit 1
fi

