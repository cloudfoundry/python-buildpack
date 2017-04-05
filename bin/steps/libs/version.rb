#!/usr/bin/env ruby
require 'yaml'
require 'rubygems'
require 'pathname'

manifest = ARGV[0]
suffix = ''

if ARGV[1].split('-').last == "ucs2"
  suffix = '-ucs2'
end

version = ARGV[1].gsub('python-', '').gsub('-ucs2', '')

if version.match(/\.x$/)
  v = version.gsub(/\.x$/, '.')
  hash = YAML.load_file(manifest)['dependencies']
  entries = hash.select do |e|
      e['name'] == 'python'+suffix && e['version'].start_with?(v)
  end.sort_by do |e|
    Gem::Version.new(e['version'])
  end
  full_version = "python-#{entries.last['version']}#{suffix}" if entries.last
else
  full_version = "python-#{version}#{suffix}"
end

puts full_version
