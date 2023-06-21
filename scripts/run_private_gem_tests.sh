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
CONFIG

sudo dpkg -i gemfast*.deb

sudo systemctl start gemfast
sleep 10
sudo systemctl status gemfast
sleep 2
sudo systemctl status caddy

journalctl -u gemfast
journalctl -u caddy

mkdir ./test-vendor
pushd test-vendor

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "rails"
CONFIG

bundle package

mkdir ~/.gem
cat << GEMCREDS > ~/.gem/credentials
:gemfast: admin:foobar
GEMCREDS
chmod 0600 ~/.gem/credentials

pushd vendor/cache
for gem in *.gem; do
  [ -f "$gem" ] || break
  echo "Uploading $gem"
  gem push "$gem" --host http://localhost:80/private -k gemfast
done
sleep 5

sudo ls -la /var/gemfast/gems
sudo rm -f Gemfile Gemfile.lock
cat << CONFIG > Gemfile
source "https://rubygems.org"
CONFIG
bundle clean --force

popd # vendor/cache
popd # test-vendor

mkdir ./test-private-gems
cd test-private-gems

cat << CONFIG > Gemfile
source "http://localhost:80/private"
gem "rails"
CONFIG
bundle