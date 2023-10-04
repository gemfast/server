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
private_gems_namespace = "foobar"
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
  gem push "$gem" --host http://localhost:80/foobar -k gemfast
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
source "http://localhost:80/foobar"
gem "rails"
CONFIG
bundle
