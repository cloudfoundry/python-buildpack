#!/usr/bin/env ruby
require 'yaml'
require 'rubygems'
require 'pathname'

manifest, version = ARGV
begin
  if version.match(/\.x$/)
    name, v = version.gsub(/\.x$/, '.').split(/-/, 2)
    hash = YAML.load_file(manifest)['dependencies']
    entries = hash.select do |e|
      e['name'] == name && e['version'].start_with?(v)
    end.sort_by do |e|
      Gem::Version.new(e['version'])
    end
    # p entry || hash
    version = "#{name}-#{entries.last['version']}" if entries.last
  end
# rescue
  # ## pass through
end
puts version
