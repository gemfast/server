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

* /var/lib/gemfast/data - The directory for the Gemfast data including gems and database
* /etc/gemfast - The directory for the gemfast.hcl config file

```bash
docker run -d --name gemfast-server \
  -p 2020:2020 \
  -v ./gemfast.hcl:/etc/gemfast/gemfast.hcl:ro \
  -v ./data:/var/lib/gemfast/data \
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

## Configuration

### Configuration File

Gemfast is configured using an HCL file. You can customize the location of this file using the `$GEMFAST_CONFIG_FILE` environment variable or by passing the `--config` argument when starting the server.

The places Gemfast automatically checks for configuration files are: `["/etc/gemfast/gemfast.hcl", "~/.config/gemfast/gemfast.hcl"]`

### Configuration Options

```terraform
# ========== gemfast.hcl ==========

# Port to bind the HTTP server to. Defaults to 2020.
port = 2020

# Log level (trace, debug, info, warn, error, fatal, panic)
log_level = "info"

# Base data directory for gemfast. If not set, defaults to platform-specific user data dir.
dir = "/var/lib/gemfast/data"

# Directory to store downloaded gem files.
gem_dir = "/var/lib/gemfast/data/gems"

# Directory to store SQLite database files.
db_dir = "/var/lib/gemfast/data/db"

# Optional path to an ACL file (Casbin policy).
acl_path = "/var/lib/gemfast/data/acl.csv"

# Optional path to an authorization model file (Casbin model).
auth_model_path = "/var/lib/gemfast/data/model.conf"

# Namespace prefix for private gems (default is "private").
private_gems_namespace = "private"

# Disable the web UI if true.
ui_disabled = false

# Disable Prometheus metrics endpoint if true.
metrics_disabled = false

# ==== Mirror block ====
# Define external sources to mirror gems from.
mirror "https://rubygems.org" {
  enabled = true  # Set to false to disable this mirror
  # hostname is auto-derived from upstream, but can be overridden
  # hostname = "rubygems.org"
}

# ==== Filter block ====
# Configure regex-based gem allow/deny logic.
filter {
  enabled = true        # Enable filtering
  action  = "deny"       # Action can be "allow" or "deny"
  regex   = ["^evil-.*", "^bad-gem$"]  # Regex list for filtering gem names
}

# ==== CVE block ====
# Control Ruby CVE integration.
cve {
  enabled = true               # Enable CVE scanning
  max_severity = "high"        # Only block gems above this severity
  ruby_advisory_db_dir = "/var/lib/gemfast/data/ruby-advisory-db"  # Directory to store the CVE DB
}

# ==== Auth block ====
# Configure authentication settings. You can only specify a single auth block.
auth "local" {                  # can be local, github, or none
  bcrypt_cost = 10              # bcrypt cost for hashing passwords
  allow_anonymous_read = false  # Allow unauthenticated read access
  default_user_role = "read"    # Default role for newly created users

  # If no users are specified, a default admin user will be created and the password written to the logs
  user {                        # repeat this block to add more users
    username = "admin"
    password = "changeme"
    role     = "admin"
  }

  # JWT secret used to sign access tokens (you can provide one or generate it automatically)
  secret_key_path = "/var/lib/gemfast/data/.jwt_secret_key"
}

# GitHub OAuth integration
# auth "github" {
  
  # github_client_id     = ""
  # github_client_secret = ""
  # github_user_orgs     = ["my-org"]  # Restrict access to users in these GitHub orgs
# }

# No auth
# auth "none" {}
```

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