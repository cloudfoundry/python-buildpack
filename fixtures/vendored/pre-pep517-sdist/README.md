# Vendored sdist app pre-PEP517

This is a vendored app that installs sdist of a package (oss2) that does not
follow PEP-517.

`pip download` (i.e. vendoring) only downloads the run-time dependencies to the
output directory, and not build-time dependencies.

Before PEP-517, the python ecosystem did not have a standard to specify
build-time dependencies of packages. This app installs such a package.

oss2 is only officially supported until python 3.8. See https://pypi.org/project/oss2

Vendoring for this app is done with the following command on a bionic env:

```
apt-get install -y python3.8 && update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.8 1 && pip install pip==24.0 && pip download --no-binary=:none: -d vendor -r requirements.txt'
```
