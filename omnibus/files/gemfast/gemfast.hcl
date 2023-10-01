# This is the configuration file for Gemfast Server.
# For documentation of all options, please visit https://gemfast.io/docs/configuration/

##### Server Settings #####
# port                   = 2020
# log_level              = "info"
# dir                    = "/var/gemfast"
# gem_dir                = "/var/gemfast/gems"
# db_dir                 = "/var/gemfast/db"
# private_gems_namespace = "private"
# ui_disabled            = false
# trial_mode             = false
# license_key            = "foobar"

##### Caddy Settings #####
# caddy {
#     admin_api_enabled = false
#     metrics_disabled  = false
#     host              = "localhost"
#     port              = 443
# }

##### Mirror Settings #####
# mirror "https://rubygems.org" {
#     enabled  = true
#     hostname = "rubygems.org"
# }

##### Gem Filter Settings #####
# filter {
#     enabled = true
#     action  = "deny"
#     regex   = ["rails-.*"]
# }

##### Gem CVE Settings #####
# cve {
#     enabled                = true
#     max_severity           = "high"
#     ruby_advisory_db_dir   = "/opt/gemfast/share/ruby-advisory-db"
# }

##### Authentication Settings #####
# None Auth
# auth "none" {}
#
# Local Auth
# auth "local"  {
#    allow_anonymous_read = false
#    admin_password       = "password"
#    user {
#      username = "user1"
#      password = "mypassword1"
#    }
#    user {
#      username = "user2"
#      password = "mypassword2"
#    }
# }
# 
# GitHub Auth
# auth "github" {
#    allow_anonymous_read = false
#    github_client_id     = "foobar"
#    github_client_secret = "foobar"
#    github_user_orgs     = ["myorg"]
# }