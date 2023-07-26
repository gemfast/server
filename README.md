# gemfast

Gemfast is a fast and secure rubygems server that can be compiled into a single binary file and built for linux, darwin and windows operating systems. Gemfast can be quickly installed on a server without any other dependencies and configured using a single HCL file.

## Server Setup

Gemfast is distributed as a `.deb` package that can be installed on ubunutu or debian operating systems.

```bash
dpkg -i gemfast-<VERSION>-<ARCH>.deb
```

The first time you start gemfast, it will initialize a database in the current directory and create an admin user. The admin user's password will be generated and printed out in the logs at first startup if you have not set the `admin_password` setting in the `/etc/gemfast/gemfast.hcl` file.

eg:

```bash
{"level":"info","detail":"9XhnYu43pqH0Wto5FGmf8MQE126kyl7P","time":1664910687,"message":"generated admin password"}
```

## Server Configuration

You can configure gemfast settings using the `/etc/gemfast/gemfast.hcl` file.

For more information see: https://gemfast.io/docs

## License

Apache 2.0
