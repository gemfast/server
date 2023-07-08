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
auth "local"  {
  allow_anonymous_read = false
  admin_password = "foobar"
  secret_key = "secretkey"
  user {
    username = "bobvance"
    password = "mypassword"
  }
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

jwt=$(curl -s -X POST -H "Content-Type: application/json" http://localhost:80/admin/api/v1/login -d '{"username": "admin", "password":"foobar"}' | jq -r .token)
token=$(curl -s -X POST -H "Authorization: Bearer $jwt" -H "Content-Type: application/json" http://localhost:80/admin/api/v1/token | jq -r .token)
bvjwt=$(curl -s -X POST -H "Content-Type: application/json" http://localhost:80/admin/api/v1/login -d '{"username": "bobvance", "password":"mypassword"}' | jq -r .token)
bvtoken=$(curl -s -X POST -H "Authorization: Bearer $bvjwt" -H "Content-Type: application/json" http://localhost:80/admin/api/v1/token | jq -r .token)



mkdir ./test-vendor
pushd test-vendor

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "rails"
CONFIG

bundle package

mkdir ~/.gem
cat << GEMCREDS > ~/.gem/credentials
:gemfast: admin:$token
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

# admin user
cat << CONFIG > Gemfile
source "http://localhost:80/private"
gem "rails"
CONFIG

bundle config http://localhost:80/private/ "admin:$token"
bundle

# unauthorized user
sudo rm -f Gemfile Gemfile.lock
cat << CONFIG > Gemfile
source "https://rubygems.org"
CONFIG
bundle clean --force

sudo rm -f Gemfile Gemfile.lock
cat << CONFIG > Gemfile
source "http://localhost:80/private"
gem "rails"
CONFIG

bundle config http://localhost:80/private/ "noauth:faketoken"
if [[ $(bundle 2>&1 | grep "Bad username or password") ]]; then
  echo "gemfast is blocking unauthenticated access"
else
  echo "gemfast is not blocking unauthenticated access"
  exit 1
fi

# read-only user
sudo rm -f Gemfile Gemfile.lock
cat << CONFIG > Gemfile
source "https://rubygems.org"
CONFIG
bundle clean --force

sudo rm -f Gemfile Gemfile.lock
cat << CONFIG > Gemfile
source "http://localhost:80/private"
gem "rails"
CONFIG

bundle config http://localhost:80/private/ "bobvance:$bvtoken"
bundle