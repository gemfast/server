name "gemfast"
default_version "local_source"

license "BSD-3-Clause"
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

  command "git clone https://github.com/rubysec/ruby-advisory-db.git #{install_dir}/share/ruby-advisory-db"

  %w(.env auth_model.conf gemfast_acl.csv).each do |f|
    copy "#{project_dir}/omnibus/files/#{name}/#{f}", "#{install_dir}/etc/#{name}"
  end
  copy "#{project_dir}/omnibus/files/#{name}/gemfast.service", "#{install_dir}/systemd/#{name}"
  copy "#{project_dir}/bin/#{name}-server", "#{install_dir}/embedded/bin"
  
  link "#{install_dir}/embedded/bin/#{name}-server", "#{install_dir}/bin/#{name}-server"
end