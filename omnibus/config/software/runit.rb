name "runit"
default_version "2.1.2"

license "BSD-3-Clause"
skip_transitive_dependency_licensing true

version("2.1.2") { source sha256: "6fd0160cb0cf1207de4e66754b6d39750cff14bb0aa66ab49490992c0c47ba18" }

source url: "http://smarden.org/runit/runit-#{version}.tar.gz"

relative_path "admin/runit-#{version}/src"

build do
  env = with_standard_compiler_flags(with_embedded_path)

  # Put runit where we want it, not where they tell us to
  command 'sed -i -e "s/^char\ \*varservice\ \=\"\/service\/\";$/char\ \*varservice\ \=\"' + install_dir.gsub("/", "\\/") + '\/service\/\";/" sv.c', env: env

  # TODO: the following is not idempotent
  command "sed -i -e s:-static:: Makefile", env: env

  # Build it
  make env: env
  make "check", env: env

  # Move it
  mkdir "#{install_dir}/embedded/bin"
  copy "#{project_dir}/chpst",                 "#{install_dir}/embedded/bin"
  copy "#{project_dir}/runit",                 "#{install_dir}/embedded/bin"
  copy "#{project_dir}/runit-init",            "#{install_dir}/embedded/bin"
  copy "#{project_dir}/runsv",                 "#{install_dir}/embedded/bin"
  copy "#{project_dir}/runsvchdir",            "#{install_dir}/embedded/bin"
  copy "#{project_dir}/runsvdir",              "#{install_dir}/embedded/bin"
  copy "#{project_dir}/sv",                    "#{install_dir}/embedded/bin"
  copy "#{project_dir}/svlogd",                "#{install_dir}/embedded/bin"
  copy "#{project_dir}/utmpset",               "#{install_dir}/embedded/bin"

  # Keeps the runsvdir process running
  copy "#{project_dir}/omnibus/files/makerun", "#{install_dir}/embedded/bin"

  erb source: "runsvdir-start.erb",
      dest: "#{install_dir}/embedded/bin/runsvdir-start",
      mode: 0755,
      vars: { install_dir: install_dir }

  # Setup service directories
  touch "#{install_dir}/service/.gitkeep"
  touch "#{install_dir}/sv/.gitkeep"
  touch "#{install_dir}/init/.gitkeep"
end