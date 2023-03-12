name "gemfast"
default_version "local_source"

license "BSD-3-Clause"
skip_transitive_dependency_licensing true

version("local_source") do
  source path: "/home"
end

build do
  env = with_standard_compiler_flags(with_embedded_path)
  
  make env: env

  mkdir "#{install_dir}/bin"
  mkdir "#{install_dir}/embedded/bin"
  mkdir "#{install_dir}/etc/#{name}"
  
  copy "#{project_dir}/omnibus/files/#{name}/gemfast.env", "#{install_dir}/etc/#{name}"
  copy "#{project_dir}/bin/#{name}-ctl",                   "#{install_dir}/embedded/bin"
  copy "#{project_dir}/bin/#{name}",                       "#{install_dir}/embedded/bin"

  link "#{install_dir}/embedded/bin/#{name}",     "#{install_dir}/bin/#{name}"
  link "#{install_dir}/embedded/bin/#{name}-ctl", "#{install_dir}/bin/#{name}-ctl"
end