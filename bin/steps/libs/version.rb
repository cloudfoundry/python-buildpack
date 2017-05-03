#!/usr/bin/env ruby
require 'yaml'
require 'rubygems'
require 'pathname'

manifest = ARGV[0]

version = ARGV[1].to_s.strip.gsub('python-', '')

if version == ''
  default_version = YAML.load_file(manifest)['default_versions']
  version = default_version.detect { |a| a['name'] == 'python' }.fetch('version')
end

if version.match(/\.x$/)
  v = version.gsub(/\.x$/, '.')
  hash = YAML.load_file(manifest)['dependencies']
  entries = hash.select do |e|
      e['name'] == 'python' && e['version'].start_with?(v)
  end.sort_by do |e|
    Gem::Version.new(e['version'])
  end
  full_version = "python-#{entries.last['version']}" if entries.last
else
  full_version = "python-#{version}"
end

STDOUT.write full_version
