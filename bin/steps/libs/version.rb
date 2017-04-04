#!/usr/bin/env ruby
require 'yaml'
require 'rubygems'
require 'pathname'

manifest = ARGV[0]
suffix = ''

if ARGV[1].split('-').last == "ucs2"
  suffix = '-ucs2'
end

version = ARGV[1].gsub('python-', '')

if version.match(/\.x#{suffix}$/)
  v = version.gsub(/\.x#{suffix}$/, '.')
  hash = YAML.load_file(manifest)['dependencies']
  entries = hash.select do |e|
    if suffix == '-ucs2'
      e['name'] == 'python' && e['version'].start_with?(v) && e['version'].end_with?('-ucs2')
    else
      e['name'] == 'python' && e['version'].start_with?(v) && !e['version'].end_with?('-ucs2')
    end
  end.sort_by do |e|
    Gem::Version.new(e['version'].gsub(suffix, ''))
  end
  full_version = "python-#{entries.last['version']}" if entries.last
else
  full_version = "python-#{version}"
end

puts full_version
