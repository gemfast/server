#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << ENV > /etc/gemfast/.env
GEMFAST_PORT=2020
GEMFAST_DIR=/var/gemfast
GEMFAST_GEM_DIR=/var/gemfast/gems
GEMFAST_DB_DIR=/var/gemfast/db
GEMFAST_MIRROR=1
GEMFAST_FILTER_ENABLED=false
GEMFAST_FILTER_DEFAULT_DENY=false
GEMFAST_AUTH=none
ENV

sudo dpkg -i gemfast*.deb
sudo systemctl start gemfast
sleep 2
sudo systemctl status gemfast
sleep 2
sudo systemctl status caddy

journalctl -u gemfast
journalctl -u caddy

cd ./clones

pushd rails
bundle config mirror.https://rubygems.org http://localhost
bundle

mv Gemfile Gemfile.backup
touch Gemfile
bundle clean --force
mv Gemfile.backup Gemfile
sed -i -e 's/https:\/\/rubygems.org/http:\/\/localhost\/private/g' Gemfile
rm Gemfile.lock
bundle
popd