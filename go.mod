module github.com/cloudfoundry/python-buildpack

go 1.23.4

require (
	github.com/Dynatrace/libbuildpack-dynatrace v1.8.0
	github.com/blang/semver v3.5.1+incompatible
	github.com/cloudfoundry/libbuildpack v0.0.0-20240717165421-f2ae8069fcba
	github.com/cloudfoundry/switchblade v0.7.0
	github.com/golang/mock v1.6.0
	github.com/kr/text v0.2.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.36.2
	github.com/sclevine/spec v1.4.0
	golang.org/x/crypto v0.32.0
	gopkg.in/ini.v1 v1.67.0
)

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v27.4.1+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elazarl/goproxy v1.2.8 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/paketo-buildpacks/packit v1.3.1 // indirect
	github.com/paketo-buildpacks/packit/v2 v2.16.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/teris-io/shortid v0.0.0-20220617161101-71ec9f2aa569 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.29.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/docker/distribution => github.com/docker/distribution v2.8.2+incompatible

replace github.com/docker/docker => github.com/docker/docker v24.0.2+incompatible
