#!/usr/bin/env ruby

require 'json'

# Detect Python-version with Pipenv.

build_dir = ARGV[0]

Dir.chdir(build_dir) do
  if File.exist?('Pipfile.lock') && !File.exist?('runtime.txt')
    pipfile_lock = JSON.parse(File.read('Pipfile.lock'))
    python_version = pipfile_lock['_meta']['requires']['python_version']

    exit 0 if python_version.nil?

    if python_version.match /^\d\.\d+$/
      File.write('runtime.txt', "python-#{python_version}.x")
    else
      File.write('runtime.txt', "python-#{python_version}")
    end
  end
end
