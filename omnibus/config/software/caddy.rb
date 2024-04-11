name "caddy"
default_version "2.7.6"

license "Apache-2.0"
skip_transitive_dependency_licensing true

if arm?
  version("2.7.6") { source sha512: "6513d40365c0570ff72c751db2d5f898d4ee9abe9241e73c3ad1062e21128745071b4efd3cc3443fc04fae2da49b69f06f70aadbe79d6a5327cc677fb86fb982" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_arm64.tar.gz"
else
  version("2.7.6") { source sha512: "b74311ec8263f30f6d36e5c8be151e8bc092b377789a55300d5671238b9043de5bd6db2bcefae32aa1e6fe94c47bbf02982c44a7871e5777b2596fdb20907cbf" }
  source url: "https://github.com/caddyserver/caddy/releases/download/v#{version}/caddy_#{version}_linux_amd64.tar.gz"
end

build do
  mkdir "#{install_dir}/embedded/bin"
  mkdir "#{install_dir}/etc/#{name}"
  mkdir "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/../gemfast/omnibus/files/#{name}/caddy.service", "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/caddy", "#{install_dir}/embedded/bin"
end