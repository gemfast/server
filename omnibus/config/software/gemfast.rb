name "gemfast"
default_version "local_source"

license "Apache-2.0"
skip_transitive_dependency_licensing true

version("local_source") do
  source path: "#{project.files_path}/../.."
end

build do
  env = with_standard_compiler_flags(with_embedded_path)
  
  make env: env

  mkdir "#{install_dir}/bin"
  mkdir "#{install_dir}/embedded/bin"
  mkdir "#{install_dir}/etc/#{name}"
  mkdir "#{install_dir}/systemd/#{name}"
  mkdir "#{install_dir}/share"
  mkdir "#{install_dir}/default"

  %w(auth_model.conf gemfast_acl.csv).each do |f|
    copy "#{project_dir}/omnibus/files/#{name}/#{f}", "#{install_dir}/etc/#{name}"
  end
  copy "#{project_dir}/omnibus/files/#{name}/gemfast.service", "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/bin/#{name}-server", "#{install_dir}/embedded/bin"
  copy "#{project_dir}/omnibus/files/gemfast/gemfast.hcl", "#{install_dir}/default"
  
  link "#{install_dir}/embedded/bin/#{name}-server", "#{install_dir}/bin/#{name}-server"
end