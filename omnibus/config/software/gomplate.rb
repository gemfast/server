name "gomplate"
default_version "3.11.4"

license "MIT"
skip_transitive_dependency_licensing true

if arm?
  version("3.11.4") { source sha256: "e2ca261ee9644d037b0fc1c71c46a7c7339ae8c6aaba66270e479ad9575df926" }
  source url: "https://github.com/hairyhenderson/gomplate/releases/download/v#{version}/gomplate_linux-arm64-slim"
else
  version("3.11.4") { source sha256: "38640b4bd624254650fadb99de05fb13744ce84301abb45ce6bdbacea125f06c" }
  source url: "https://github.com/hairyhenderson/gomplate/releases/download/v#{version}/gomplate_linux-amd64-slim"
end

build do
  mkdir "#{install_dir}/embedded/bin"
  if arm?
    copy "#{project_dir}/gomplate_linux-arm64-slim", "#{install_dir}/embedded/bin/gomplate"
  else
    copy "#{project_dir}/gomplate_linux-amd64-slim", "#{install_dir}/embedded/bin/gomplate"
  end
  block "chmod 755 #{install_dir}/embedded/bin/gomplate" do
    FileUtils.chmod(0755, "#{install_dir}/embedded/bin/gomplate") 
  end
end