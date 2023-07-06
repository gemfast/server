name "caddy"
default_version "2.6.4"

license "Apache-2.0"
skip_transitive_dependency_licensing true

if arm?
  version("2.6.4") { source sha512: "6513d40365c0570ff72c751db2d5f898d4ee9abe9241e73c3ad1062e21128745071b4efd3cc3443fc04fae2da49b69f06f70aadbe79d6a5327cc677fb86fb982" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_arm64.tar.gz"
else
  version("2.6.4") { source sha512: "eed413b035ffacedfaf751a8431285c5d9a0a81a2a861444f4b95dd4c7508eabe2f3fcba6c5b8e6c70e30c9351dfa96ba39def47fa0879334d965dae3a869f1a" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_amd64.tar.gz"
end

build do
  mkdir "#{install_dir}/embedded/bin"
  mkdir "#{install_dir}/etc/#{name}"
  mkdir "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/../gemfast/omnibus/files/#{name}/caddy.service", "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/caddy", "#{install_dir}/embedded/bin"
end