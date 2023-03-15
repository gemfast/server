name "caddy"
default_version "2.6.2"

license "BSD-3-Clause"
skip_transitive_dependency_licensing true

if arm?
  version("2.6.2") { source sha256: "5af0ee65a0220108b7b96322b0418abcda526d5f7fec5afaea029f1aebcca62a" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_amd64.tar.gz"
else
  version("2.6.2") { source sha256: "5af0ee65a0220108b7b96322b0418abcda526d5f7fec5afaea029f1aebcca62a" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_amd64.tar.gz"
end



build do
  mkdir "#{install_dir}/embedded/bin"
  mkdir "#{install_dir}/etc/#{name}"
  mkdir "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/../gemfast/omnibus/files/#{name}/Caddyfile.tmpl", "#{install_dir}/etc/#{name}"
  copy "#{project_dir}/../gemfast/omnibus/files/#{name}/caddy.service", "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/caddy", "#{install_dir}/embedded/bin"
end