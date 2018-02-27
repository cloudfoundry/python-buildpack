package integration_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/cloudfoundry/libbuildpack"
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
		app.SetEnv("BP_DEBUG", "1")
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
		Expect(app.Stdout.String()).NotTo(ContainSubstring("ImportError:"))
		Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))

		By("Caching pip files", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Size of pip cache dir: "))
			matches := regexp.MustCompile("(?m)Size of pip cache dir: (\\d+)\\s.*pip_cache$").FindStringSubmatch(app.Stdout.String())
			Expect(matches).To(HaveLen(2))
			size, err := strconv.Atoi(matches[1])
			Expect(err).ToNot(HaveOccurred())

			Expect(size).To(BeNumerically(">", 5000))
		})
	})

	It("deploy a web app that uses an nltk corpus", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "nltk_flask"))
		app.SetEnv("BP_DEBUG", "1")
		app.Memory = "256M"
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced"))
		Expect(app.Stdout.String()).To(ContainSubstring("Downloading NLTK packages: brown"))
		Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
	})

	It("deploy a web app that uses an tkinter", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "tkinter"))
		app.Buildpacks = []string{"python_buildpack"}
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("tkinter was imported"))
	})

	It("should not display the allow-all-external deprecation message", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask"))
		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).ToNot(ContainSubstring("DEPRECATION: --allow-all-external has been deprecated and will be removed in the future"))
	})

	It("app has pre and post scripts", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_hooks"))
		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).To(ContainSubstring("Echo from app pre compile"))
		Expect(app.Stdout.String()).To(ContainSubstring("Echo from app post compile"))
	})

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
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
				})
			})

			Context("including django with specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_python_3"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("It worked!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.5"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Copying "))
					Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
				})
			})
		})

		Context("pushing a Python app without the runtime.txt", func() {
			Context("", func() {
				var defaultV string

				type manifestContent struct {
					DefaultVersions []struct {
						Name    string `yaml:"name"`
						Version string `yaml:"version"`
					} `yaml:"default_versions"`
				}

				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask"))
					mc := manifestContent{}
					err := libbuildpack.NewYAML().Load(filepath.Join(bpDir, "manifest.yml"), &mc)
					Expect(err).To(BeNil())
					for _, defaultDep := range mc.DefaultVersions {
						if defaultDep.Name == "python" {
							defaultV = defaultDep.Version
						}
					}
				})

				It("deploys with default Python version", func() {
					PushAppAndConfirm(app)

					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app.Stdout.String()).To(ContainSubstring(fmt.Sprintf("-----> Installing python %s", defaultV)))
				})
			})

			Context("including django but not specified Python version", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "django_web_app"))
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
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
				})
				AssertUsesProxyDuringStagingIfPresent("flask_not_vendored")
			})

		})

		Context("with mercurial dependencies", func() {
			BeforeEach(func() {
				app = cutlass.New(filepath.Join(bpDir, "fixtures", "mercurial"))
				app.SetEnv("BP_DEBUG", "1")
			})

			It("deploys", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).NotTo(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
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
			Context("with Python 2", func() {
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

			Context("with Python 3", func() {
				BeforeEach(func() {
					app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.Stdout.String()).To(ContainSubstring("Copy [/"))
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				})
				AssertNoInternetTraffic("flask")
			})
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

	It("sets gunicorn to send access logs to stdout by defualt", func() {
		app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_latest_gunicorn"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		Eventually(app.Stdout.String).Should(MatchRegexp(`\[APP/PROC/WEB/0\] .* "GET / HTTP/1.1"`))
	})
})
