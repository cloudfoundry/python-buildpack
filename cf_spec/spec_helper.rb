require 'bundler/setup'
require 'machete'
require 'machete/matchers'
require 'timeout'

`mkdir -p log`
Machete.logger = Machete::Logger.new("log/integration.log")

RSpec.configure do |config|
  config.color = true
  config.tty = true

  config.filter_run_excluding :cached => ENV['BUILDPACK_MODE'] == 'uncached'
  config.filter_run_excluding :uncached => ENV['BUILDPACK_MODE'] == 'cached'

  config.around(:each) do |example|
    allowed_time = 20 * 60 # 20 minutes per attempt
    begin
      Timeout::timeout(allowed_time) do
        example.run
      end
    rescue Timeout::Error
      RSpec.configuration.reporter.message("Retry try #{example.location}")

      Timeout::timeout(allowed_time) do
        example.run
      end
    end
  end
end
