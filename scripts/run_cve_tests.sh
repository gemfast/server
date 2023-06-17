#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

gem update --system

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << CONFIG > /etc/gemfast/gemfast.hcl
caddy_port = 80
url = "http://localhost"
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
cve {
    enabled = true
    max_severity = "medium"
}
CONFIG

sudo dpkg -i gemfast*.deb


sudo systemctl start gemfast
sleep 10
sudo systemctl status gemfast
sleep 2
sudo systemctl status caddy

journalctl -u gemfast
journalctl -u caddy

mkdir ./test-cve

pushd test-cve
cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "activerecord", "4.2.0"
CONFIG

bundle config mirror.https://rubygems.org http://localhost
if [[ $(bundle 2>&1 | grep "418") ]]; then
    echo "Gemfast is working"
else
    echo "Gemfast is not working"
    exit 1
fi

popd