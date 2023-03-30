#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

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
bundle config mirror.https://rubygems.org http://localhost:2020
bundle
popd