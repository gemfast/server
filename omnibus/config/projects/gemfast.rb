name "gemfast"
friendly_name "Gemfast Server"
maintainer "Greg Schofield <greg@gemfast.io>"
homepage "https://www.gemfast.io"
license "GEMFAST EULA"
license_file "GEMFAST-EULA.md"

build_iteration 1
current_file ||= __FILE__
version_file = File.expand_path("../../../../VERSION", current_file)
build_version IO.read(version_file).strip

install_dir "#{default_root}/#{name}"

dependency "gemfast"
dependency "caddy"
dependency "runit"

package :deb do
  compression_level 1
  compression_type :xz
end

runtime_dependency "cron"