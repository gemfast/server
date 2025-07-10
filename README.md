# Gemfast

[Gemfast](https://gemfast.io) is a fast and secure rubygems server written in Go. That means it can be compiled into a single binary file and work on linux, darwin and windows operating systems. Gemfast can be quickly installed on a server without any other dependencies and configured using a single HashiCorp Configuration Language (HCL) file.

- [Gemfast](#gemfast)
  - [Why Gemfast](#why-gemfast)
  - [Installing](#installing)
    - [Docker](#docker)
    - [Prebuilt Binaries](#prebuilt-binaries)
    - [Building From Source](#building-from-source)
  - [Docs](#docs)
  - [UI](#ui)
  - [License](#license)

## Why Gemfast

Gemfast was created for users who need to self-host their rubygems and want something quick to setup and easy to manage. Gemfast allows users to mirror and cache gems from rubygems.org as well as upload their own internally developed gems which can't be distributed publically. It supports both the legacy Dependency API and the newer Compact Index API.

Gemfast has the following unique benefits:

* Two installation methods: [Docker Image](#docker) or precompiled binaries
* No need to install/upgrade/manage a version of Ruby on the server
* No external server dependencies like postgres, redis or memcached
* User login via GitHub oAuth
* Allow/Deny gems based on CVE severity or a regex list using the [ruby-advisory-db](https://github.com/rubysec/ruby-advisory-db)
* Performance benefits of Go

## Installing

Gemfast is currently distributed in two different ways, a `docker` image and precompiled binaries.

### Docker

When running Gemfast as a container, its important to mount the following directories:

* /var/gemfast - The directory for the Gemfast data including gems and database
* /etc/gemfast - The directory for the gemfast.hcl config file. It is possible to configure the config file path using `env GEMFAST_CONFIG_FILE=/path/to/my/file.hcl`

```bash
docker run -d --name gemfast-server \
  -p 2020:2020 \
  -v /etc/gemfast:/etc/gemfast \
  -v /var/gemfast:/var/gemfast \
  ghcr.io/gemfast/server:latest
```

### Prebuilt Binaries

Currently Prebuild binaries are available on each GitHub release for the following platforms.

- linux_amd64
- linux_arm64
- darwin_arm64

Simply download and extract the tarball to run the server.

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

🚧 **The UI feature is currently under construction and is considered experimental** 🚧

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