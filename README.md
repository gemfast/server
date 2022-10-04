# gemfast

**This is currently experimental code. Although it works in its current state, do not use it for anything serious**

Gemfast is a rubygems server that can be compiled into a single binary file and built for linux, darwin and windows operating systems. Gemfast can be quickly installed on a server without any other dependencies and configured using YAML.

## Server Setup

A new gemfast server can be started with a single command:

```bash
./gemfast
```

The first time you start gemfast, it will initialize a database in the current directory and create an admin user. The admin user's password will be generated and printed out in the logs at first startup if you have not set the `GEMFAST_ADMIN_PASSWORD` environment variable.

eg:

```bash
{"level":"info","password":"9XhnYu43pqH0Wto5FGmf8MQE126kyl7P","time":1664910687,"message":"generated admin password"}
```

## Server Configuration

You can configure gemfast settings using environment variables

The default settings are:

```bash
GEMFAST_ADMIN_PASSWORD="<generated>" # The admin user password
GEMFAST_DIR="/var/gemfast" # The base directory to store gem index files
GEMFAST_GEM_DIR="/var/gemfast/gems" # The directory where to store gems
GEMFAST_DB_DIR="." # Where to create the database file
GEMFAST_AUTH="local" # The auth mode. Can be local or none.
```

## Usage

Usage with `bundler`: TBD

Usage with `gem` command: TBD

## Using the API

TBD

## Supported API Endpoints

* HEAD /
* GET /specs.4.8.gz
* GET /latest_specs.4.8.gz
* GET /prerelease_specs.4.8.gz
* GET /quick/Marshal.4.8/*gemspec.rz
* GET /gems/*gem
* GET /api/v1/dependencies
* GET /api/v1/dependencies.json

