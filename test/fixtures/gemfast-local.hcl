# Local-auth config used by the OTel verification harness.
# Run: GEMFAST_CONFIG_FILE=test/fixtures/gemfast-local.hcl ./gemfast-server

port      = 2020
log_level = "info"
dir       = "/tmp/gemfast"

auth "local" {
  admin_password       = "test-admin-pw"
  default_user_role    = "write"
  allow_anonymous_read = false

  user {
    username = "alice"
    password = "alice-pw"
    role     = "write"
  }

  user {
    username = "bob"
    password = "bob-pw"
    role     = "read"
  }

  user {
    username = "carol"
    password = "carol-pw"
    role     = "write"
  }

  user {
    username = "dave"
    password = "dave-pw"
    role     = "write"
  }

  user {
    username = "erin"
    password = "erin-pw"
    role     = "write"
  }

  user {
    username = "frank"
    password = "frank-pw"
    role     = "write"
  }
}

mirror "https://rubygems.org" {
  enabled = true
}
