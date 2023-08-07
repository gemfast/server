# Gemfast

[Gemfast](https://gemfast.io) is a fast and secure rubygems server written in Go. That means it can be compiled into a single binary file and work on linux, darwin and windows operating systems. Gemfast can be quickly installed on a server without any other dependencies and configured using a single HashiCorp Configuration Language (HCL) file.

Gemfast is also distributed as a `.deb` package that can be installed on ubunutu or debian operating systems. The `.deb` package also includes a caddy web server which proxies HTTPS traffic to Gemfast.

## Why Gemfast

Gemfast was created for users who need to self-host their rubygems and want something quick to setup and easy to manage. Gemfast allows users to mirror and cache gems from rubygems.org as well as upload their own internally developed gems which can't be distributed publically. It supports both the legacy Dependency API and the newer Compact Index API.

Gemfast has the following unique benefits:

* There is no need to install/upgrade/manage a version of Ruby on the server
* Go is generally faster and requires less memory then Ruby (still :heart: Ruby though)
* There are no external server dependencies like postgres, redis or memcached
* Gemfast can allow/deny gems based on CVE severity or a regex list (license version only)

## Docs

You can configure gemfast settings using the `/etc/gemfast/gemfast.hcl` file. There are many options all of which are listed in the documentation.

For more information see: https://gemfast.io/docs

## License

Gemfast is source available software licensed under the Elastic License 2.0 (ELv2) License. The license restricts users from:

* Providing the software to third parties as a hosted or managed service
* Circumventing the license key functionality

Users who purchase a license key will get access to all Gemfast features, support from the Gemfast creator and a commercial friendly license.