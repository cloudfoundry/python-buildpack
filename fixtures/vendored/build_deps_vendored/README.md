# App with build dependencies vendored

This is a vendored app that installs sdist of a package (oss2) that follows PEP 517.
Its build-time dependencies (setuptools and wheel) are added to `requirements.txt` before [vendoring](https://docs.cloudfoundry.org/buildpacks/python/#vendoring).
Then, the app is pushed with the env var setting of `BP_ENABLE_BUILD_ISOLATION_VENDORED: true`.
