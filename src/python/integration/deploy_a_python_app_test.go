package integration_test

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Python Buildpack", func() {
	var app *cutlass.App

	BeforeEach(func() {
		if isSerialTest {
			Skip("Skipping parallel tests")
		}
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
			app = nil
		}
	})

	Context("with an unsupported dependency", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("unsupported_version"))
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(Succeed())

			logs := exec.Command("cf", "logs", "--recent", app.Name)
			out, err := logs.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())

			Expect(out).To(ContainSubstring("-----> Python Buildpack version " + buildpackVersion))

			Expect(out).To(ContainSubstring("Could not install python: no match found for 99.99.99"))
			Expect(out).ToNot(ContainSubstring("-----> Installing python"))
		})
	})

	It("deploy a web app with -e in requirements.txt", func() {
		app = cutlass.New(Fixtures("flask_git_req"))
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

	It("deploy a web app with -r in requirements.txt", func() {
		app = cutlass.New(Fixtures("recursive_requirements"))
		PushAppAndConfirm(app)
		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
	})

	It("deploy a web app that uses an nltk corpus", func() {
		app = cutlass.New(Fixtures("nltk_flask"))
		app.SetEnv("BP_DEBUG", "1")
		app.Memory = "256M"
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced"))
		Expect(app.Stdout.String()).To(ContainSubstring("Downloading NLTK packages: brown"))
		Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
	})

	It("deploy a web app that uses an tkinter", func() {
		app = cutlass.New(Fixtures("tkinter"))
		app.Buildpacks = []string{"python_buildpack"}
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("tkinter was imported"))
	})

	It("should not display the allow-all-external deprecation message", func() {
		app = cutlass.New(Fixtures("flask"))
		PushAppAndConfirm(app)
		Expect(app.Stdout.String()).ToNot(ContainSubstring("DEPRECATION: --allow-all-external has been deprecated and will be removed in the future"))
	})

	It("app has pre and post scripts", func() {
		app = cutlass.New(Fixtures("with_hooks"))
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
					app = cutlass.New(Fixtures("flask_python_3"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.9"))
					Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
				})
			})

			Context("including flask and no build isolation", func() {
				BeforeEach(func() {
					app = cutlass.New(Fixtures("no_build_isolation"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.9"))
					Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
				})
			})

			Context("including django with specified python version", func() {
				BeforeEach(func() {
					app = cutlass.New(Fixtures("django_python_3"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("The install worked successfully!"))
					Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.9"))
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
					app = cutlass.New(Fixtures("flask"))
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

					re := regexp.MustCompile("Installing python (.*)[\r\n|\r|\n]")
					match := re.FindStringSubmatch(app.Stdout.String())
					foundVersion := match[1]

					versionRange := semver.MustParseRange("<=" + defaultV)
					v1 := semver.MustParse(foundVersion)
					Expect(versionRange(v1)).To(BeTrue())
				})
			})

			Context("including django but not specified Python version", func() {
				BeforeEach(func() {
					app = cutlass.New(Fixtures("django_web_app"))
				})

				It("deploys", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("The install worked successfully!"))
					Expect(app.Stdout.String()).To(ContainSubstring("collectstatic --noinput"))
					Expect(app.Stdout.String()).NotTo(ContainSubstring("Error while running"))
					Eventually(app.Stdout.String()).ShouldNot(MatchRegexp(`WARNING: You are using pip version \d+.\d+.\d+; however, version \d+.\d+.\d+ is available.`))
				})
			})

			Context("including flask without a vendor directory", func() {
				BeforeEach(func() {
					app = cutlass.New(Fixtures("flask_not_vendored"))
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
				app = cutlass.New(Fixtures("mercurial"))
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
			Context("with Python 3", func() {
				BeforeEach(func() {
					app = cutlass.New(Fixtures("flask_python_3"))
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
				app = cutlass.New(Fixtures("mercurial"))
			})

			It("deploys", func() {
				PushAppAndConfirm(app)
				Expect(app.Stdout.String()).To(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
			})
		})
	})

	It("sets gunicorn to send access logs to stdout by defualt", func() {
		app = cutlass.New(Fixtures("flask_latest_gunicorn"))
		PushAppAndConfirm(app)

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		Eventually(app.Stdout.String).Should(MatchRegexp(`\[APP/PROC/WEB/0\] .* "GET / HTTP/1.1"`))
	})

	Context("specifying pip version", func() {
		Context("default", func() {
			BeforeEach(func() {
				app = cutlass.New(Fixtures("flask"))
				app.SetEnv("BP_PIP_VERSION", "")
			})

			It("uses python's pip module", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				Expect(app.Stdout.String()).To(ContainSubstring("Using python's pip module"))
			})
		})

		Context("latest", func() {
			BeforeEach(func() {
				app = cutlass.New(Fixtures("flask"))
				app.SetEnv("BP_PIP_VERSION", "latest")
			})

			It("uses latest from manifest", func() {
				PushAppAndConfirm(app)
				Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
				Expect(app.Stdout.String()).To(ContainSubstring("Installing pip"))
				Expect(app.Stdout.String()).To(MatchRegexp(`Successfully installed pip-\d+.\d+(.\d+)?`))
			})
		})
	})
})
