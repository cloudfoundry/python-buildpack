require 'spec_helper'

describe 'deploying a flask web app' do
  let(:browser)  { Machete::Browser.new(app) }
  let(:app_name) { 'flask_ucs2' }

  subject(:app)  { Machete.deploy_app(app_name) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  context 'runtime.txt python version is 2.7.x-ucs2' do
    it 'uses a ucs2 python 2.7.x version' do
      expect(app).to have_logged /Installing python-2.7./
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('max unicode: 65535')
    end
  end
end
