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

	Context("with an unsupported dependency", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "unsupported_version"))
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

			Expect(app.Stdout.String()).To(ContainSubstring("Could not install python: no match found for 99.99.99"))
			Expect(app.Stdout.String()).ToNot(ContainSubstring("-----> Installing"))
		})
	})

	It("deploy a web app with -e in requirements.txt", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_git_req"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
		Expect(app.Stdout.String()).NotTo(ContainSubstring("ImportError:"))
	})

	// FIt("deploy a web app that uses an nltk corpus", func() {
	// 	app = cutlass.New(filepath.Join(bpDir, "fixtures", "nltk_flask"))
	// 	PushAppAndConfirm(app)

	// 	Expect(app.GetBody("/")).To(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced"))
	// 	Expect(app.Stdout.String()).To(ContainSubstring("Downloading NLTK packages: brown"))
	// })

	Context("uncached buildpack", func() {
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("Running uncached tests")
			}
		})
		Context("pushing a Python 3 app with a runtime.txt", func() {
			Context("including flask", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
			})

			Context("including django with specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_python_3"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.5"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
				})
			})

		})

		Context("pushing a Python app without the runtime.txt", func() {
			Context("including django but not specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_web_app"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
				})
			})

			Context("including flask without a vendor directory", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_not_vendored"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
				AssertUsesProxyDuringStagingIfPresent("flask_not_vendored")
			})

		})

		Context("with mercurial dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "mercurial"))
			})

			It("deploys", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).NotTo(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})
		})
	})
	Context("cached buildpack", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("Running cached tests")
			}
		})

		Context("when using flask", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask"))
			})

			It("deploys", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Copy [/"))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})
			AssertNoInternetTraffic("flask")
		})

		Context("with mercurial dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "mercurial"))
			})

			It("deploys", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})
		})
	})
})
