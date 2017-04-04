require 'spec_helper'
require 'tempfile'

describe 'Compile' do
  def run(cmd, env: {})
    if RUBY_PLATFORM =~ /darwin/i
      env_flags = env.map{|k,v| "-e #{k}=#{v}"}.join(' ')
      `docker run --rm #{env_flags} -v #{Dir.pwd}:/buildpack:ro -w /buildpack cloudfoundry/cflinuxfs2 #{cmd}`
    else
      `env #{env.map{|k,v| "#{k}=#{v}"}.join(' ')} #{cmd}`
    end
  end

  context 'when running in an unsupported stack' do
    it 'fails with a helpful error message' do
      output = run('./bin/compile arg1 arg2 arg3 2>&1', env: {CF_STACK: 'unsupported'})
      expect(output).to include('not supported by this buildpack')
    end
  end

  describe 'python version selecting' do
    let(:manifest) { "cf_spec/fixtures/version-manifest.yml" }

    context 'runtime.txt contains "python" prefix' do

      it 'fully specified line passes through' do
        output = run("./bin/steps/libs/version.rb #{manifest} python-2.7.12")
        expect(output.chomp).to eq('python-2.7.12')
      end

      it 'finds latest of a line' do
        output = run("./bin/steps/libs/version.rb #{manifest} python-2.7.x")
        expect(output.chomp).to eq('python-2.7.14')
      end
    end

    context 'runtime.txt contains just the version' do
      it 'fully specified line passes through' do
        output = run("./bin/steps/libs/version.rb #{manifest} 2.7.12")
        expect(output.chomp).to eq('python-2.7.12')
      end

      it 'finds latest of a line' do
        output = run("./bin/steps/libs/version.rb #{manifest} 2.7.x")
        expect(output.chomp).to eq('python-2.7.14')
      end
    end

    context 'runtime.txt contains "python" prefix and ucs2 suffix' do
      it 'fully specified line passes through' do
        output = run("./bin/steps/libs/version.rb #{manifest} python-2.7.12-ucs2")
        expect(output.chomp).to eq('python-2.7.12-ucs2')
      end

      it 'finds latest of a line' do
        output = run("./bin/steps/libs/version.rb #{manifest} python-2.7.x-ucs2")
        expect(output.chomp).to eq('python-2.7.14-ucs2')
      end
    end

    context 'runtime.txt contains the version and usc2 suffix' do
      it 'fully specified line passes through' do
        output = run("./bin/steps/libs/version.rb #{manifest} 2.7.12-ucs2")
        expect(output.chomp).to eq('python-2.7.12-ucs2')
      end

      it 'finds latest of a line' do
        output = run("./bin/steps/libs/version.rb #{manifest} 2.7.x-ucs2")
        expect(output.chomp).to eq('python-2.7.14-ucs2')
      end
    end
  end
end
