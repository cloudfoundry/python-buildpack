require 'spec_helper'

describe 'deploying a flask web app' do
  let(:browser)  { Machete::Browser.new(app) }
  let(:app_name) { 'flask_python_3_pipenv' }

  subject(:app)  { Machete.deploy_app(app_name) }

  after { Machete::CF::DeleteApp.new.execute(app) }

  context 'app has Pipfile.lock and no requirements.txt or runtime.txt' do
    it 'it gets the python version from pipfile.lock' do
      expect(app).to have_logged /Installing python-3.6./
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World with pipenv!')
    end

    it 'uses pipenv to generate a requirements.txt' do
      expect(app).to have_logged /Generating 'requirements.txt' with pipenv/
      expect(app).to be_running

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World with pipenv!')
    end
  end

  context 'buildpack is cached', :cached do
    let (:app_name) { 'flask_python_3_pipenv_vendored'}

    it 'deploys without hitting the internet' do
      expect(app).to have_logged /Generating 'requirements.txt' with pipenv/
      expect(app).to be_running
      expect(app).not_to have_internet_traffic

      browser.visit_path('/')
      expect(browser).to have_body('Hello, World with pipenv!')
    end
  end
end
