package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Python Buildpack", func() {
	var app *cutlass.App

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	It("with an unsupported dependency", func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "unsupported_version"))
		app.Push()
		Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

		By("displays a nice error messages and gracefully fails", func() {
			Expect(app).ToNot(HaveLogged("Downloaded ["))
			Expect(app).To(HaveLogged("DEPENDENCY MISSING IN MANIFEST: python 99.99.99"))
		})
	})

	It("deploy a web app with -e in requirements.txt", func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_git_req"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

		Expect(app).ToNot(HaveLogged("Error while running"))
		Expect(app).ToNot(HaveLogged("ImportError:"))
	})

	It("deploy a web app that uses an nltk corpus", func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "nltk_flask"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced"))

		Expect(app).To(HaveLogged("Downloading NLTK packages: brown"))
	})

	It("should not display the allow-all-external deprecation message", func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask"))
		PushAppAndConfirm(app)

		Expect(app).To(HaveLogged("DEPRECATION: --allow-all-external has been deprecated and will be removed in the future"))
	})

	Context("with cached buildpack dependencies", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("but running uncached tests")
			}
		})

		Context("app has dependencies", func() {
			Context("with Python 2", func() {
				It("deploy a flask web app", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask"))
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app).To(HaveLogged("Downloaded [file://"))
				})
				AssertNoInternetTraffic("flask")
			})

			Context("with Python 3", func() {
				It("", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_python_3"))
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
				AssertNoInternetTraffic("flask_python_3")
			})
		})

		Context("Warning when pip has mercurial dependencies", func() {
			It("logs a warning that it may not work offline", func() {
				app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "mercurial"))
				PushAppAndConfirm(app)

				Expect(app).To(HaveLogged("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
			})
		})
	})

	Context("without cached buildpack dependencies", func() {
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("but running cached tests")
			}
		})
		Context("app has dependencies", func() {
			Context("with mercurial dependencies", func() {
				It("starts successfully", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "mercurial"))
					PushAppAndConfirm(app)

					Expect(app).ToNot(HaveLogged("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
			})

			Context("with Python 2", func() {
				PContext("deploy a flask web app without runtime.txt", func() {
					// let(:app_name) { 'flask' }

					// subject(:app) do
					//   Machete.deploy_app(app_name)
					// end

					// before do
					//   default_versions = YAML.load_file(File.join(File.dirname(__FILE__), '..', '..', 'manifest.yml'))['default_versions']
					//   @default = default_versions.detect { |a| a['name'] == 'python' }.fetch('version')
					// end

					It("uses the default python version", func() {
						// expect(app).to be_running(60)

						// browser.visit_path('/')
						// expect(browser).to have_body('Hello, World!')
						// expect(app).to have_logged("-----> Installing python-#{@default}")
					})
				})

				It("deploy a django web app", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "django_web_app"))
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))

					// Check that collectstatic ran
					Expect(app).ToNot(HaveLogged("Error while running"))
					Expect(app).To(HaveLogged("collectstatic --noinput"))
				})
			})

			Context("with Python 3", func() {
				It("deploy a flask web app", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_python_3"))
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})

				It("deploy a django web app", func() {
					app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "django_python_3"))
					PushAppAndConfirm(app)

					Expect(app).To(HaveLogged("-----> Installing python-3.5."))
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))

					// Check that collectstatic ran
					Expect(app).ToNot(HaveLogged("Error while running"))
					Expect(app).To(HaveLogged("collectstatic --noinput"))
				})
			})
		})

		Context("app has non-vendored dependencies", func() {
			It("", func() {
				app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_not_vendored"))
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})

			AssertUsesProxyDuringStagingIfPresent("flask_not_vendored")
		})

		Context("an app that uses miniconda and python 2", func() {
			It("", func() {
				app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "miniconda_python_2"))
				PushAppAndConfirm(app)

				body, err := app.GetBody("/")
				Expect(err).ToNot(HaveOccurred())

				Expect(body).To(ContainSubstring("numpy: 1.10.4"))
				Expect(body).To(ContainSubstring("scipy: 0.17.0"))
				Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
				Expect(body).To(ContainSubstring("pandas: 0.18.0"))
				Expect(body).To(ContainSubstring("python-version2"))
			})

			AssertUsesProxyDuringStagingIfPresent("miniconda_python_2")
		})

		PContext("an app that uses miniconda and python 3", func() {
			//       let(:app_name) { 'miniconda_python_3' }
			//       before(:each) { create_environment_yml app_name }
			//       after(:each) {create_environment_yml app_name}

			Describe("keeping track of environment.yml", func() {
				It("", func() {
					//           expect(app).to be_running(120)

					//           # Check that scipy was installed in the logs
					//           expect(app).to have_logged("scipy")

					//           browser.visit_path('/')
					//           expect(browser).to have_body('numpy: 1.10.4')
					//           expect(browser).to have_body('scipy: 0.17.0')
					//           expect(browser).to have_body('sklearn: 0.17.1')
					//           expect(browser).to have_body('pandas: 0.18.0')
					//           expect(browser).to have_body('python-version3')
				})

				It("doesn't re-download unchanged dependencies", func() {
					//           expect(app).to be_running(120)
					//           Machete.push(app)
					//           expect(app).to be_running(120)

					//           # Check that scipy was not re-installed in the logs
					//           expect(app).to_not have_logged("scipy")
				})

				It("it updates dependencies if environment.yml changes", func() {
					//           contents = <<~HERE
					//                         name: pydata_test
					//                         dependencies:
					//                         - pip
					//                         - pytest
					//                         - flask
					//                         - nose
					//                         - numpy=1.11.0
					//                         - scikit-learn=0.17.1
					//                         - pandas=0.18.0
					//                         HERE
					//           create_environment_yml app_name, contents
					//           Machete.push(app)
					//           expect(app).to be_running(120)

					//           browser.visit_path('/')
					//           expect(browser).to have_body('numpy: 1.11.0')
				})

				It("uses a proxy during staging if present", func() {
					//TODO Uncached only

					//           expect(app).to use_proxy_during_staging
				})
			})
		})

		Context("an app that uses miniconda and specifies python 2 in runtime.txt but python3 in the environment.yml", func() {
			It("", func() {
				app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "miniconda_python_2_3"))
				PushAppAndConfirm(app)

				Expect(app.GetBody("/")).To(ContainSubstring("python-version3"))
				Expect(app).To(HaveLogged("WARNING: you have specified the version of Python runtime both in 'runtime.txt' and 'environment.yml'. You should remove one of the two versions"))
			})

			AssertUsesProxyDuringStagingIfPresent("miniconda_python_2_3")
		})
	})
})
