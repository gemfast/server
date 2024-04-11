# Gemfast

[Gemfast](https://gemfast.io) is a fast and secure rubygems server written in Go. That means it can be compiled into a single binary file and work on linux, darwin and windows operating systems. Gemfast can be quickly installed on a server without any other dependencies and configured using a single HashiCorp Configuration Language (HCL) file.

- [Gemfast](#gemfast)
  - [Why Gemfast](#why-gemfast)
  - [Installing](#installing)
    - [Debian Package](#debian-package)
      - [Debian Package SSL](#debian-package-ssl)
    - [Docker](#docker)
    - [Building From Source](#building-from-source)
  - [Docs](#docs)
  - [UI](#ui)
  - [License](#license)

## Why Gemfast

Gemfast was created for users who need to self-host their rubygems and want something quick to setup and easy to manage. Gemfast allows users to mirror and cache gems from rubygems.org as well as upload their own internally developed gems which can't be distributed publically. It supports both the legacy Dependency API and the newer Compact Index API.

Gemfast has the following unique benefits:

* There is no need to install/upgrade/manage a version of Ruby on the server
* Go is generally faster and requires less memory then Ruby (still :heart: Ruby though)
* There are no external server dependencies like postgres, redis or memcached
* Gemfast can allow/deny gems based on CVE severity or a regex list

## Installing

Gemfast is currently distributed in two different ways, a `.deb` package and a `docker` image.

The `.deb` package includes a web server which proxies HTTPS traffic to Gemfast and is recommended when installing Gemfast on a virtual machine or bare-metal instance.

### Debian Package

To install the .deb package, download it from the latest GitHub release and install it with dpkg.

```bash
curl -L -O gemfast_<version>_amd64.deb
sudo dpkg -i ./gemfast_<version>_amd64.deb
sudo systemctl start gemfast.service
```

#### Debian Package SSL

The Gemfast `.deb` package includes Caddy Server which will automatically generate and manage a let's encrypt SSL certificate for your server. Caddy offers a few ways to generate a valid SSL certificate, see: https://caddyserver.com/docs/automatic-https#acme-challenges. If the let's encrypt challenge fails, the server will use a self-signed certificate.

### Docker

When running Gemfast as a container, its important to mount the following directories:

* /etc/gemfast - The directory for the gemfast.hcl config file
* /var/gemfast - The directory for the Gemfast data including gems and database

```bash
docker run -d --name gemfast-server \
  -p 2020:2020 \
  -v /etc/gemfast:/etc/gemfast \
  -v /var/gemfast:/var/gemfast \
  ghcr.io/gemfast/server:latest
```

### Building From Source

Gemfast uses Make to build binaries. To build and run a static binary:

```bash
make
./bin/gemfast-server
```

## Docs

You can configure gemfast settings using the `/etc/gemfast/gemfast.hcl` file. There are many options all of which are listed in the documentation.

For more information see: https://gemfast.io/docs/configuration/

## UI

![Dashboard UI](https://github.com/gemfast/server/raw/main/SCREENSHOT.png)

Gemfast includes a basic ui which is accessible from `my.server.url/ui`. For example, running it locally you can access it at `http://localhost:2020/ui`.

The ui currently supports viewing and searching gems from both the private gems namespace and gems that have been cached from an upsteam. 

You can also disable the ui in /etc/gemfast/gemfast.hcl:

```terraform
ui_disabled = true
```

## License

Gemfast is open source software licensed under the Apache 2.0 License.

```text
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License..
```