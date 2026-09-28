# Cloud Foundry Python Buildpack

[![CF Slack](https://www.google.com/s2/favicons?domain=www.slack.com) Join us on Slack](https://cloudfoundry.slack.com/messages/buildpacks/)

A Cloud Foundry [buildpack](http://docs.cloudfoundry.org/buildpacks/) for Python based apps.

This buildpack supports running Django and Flask apps.

### Buildpack User Documentation

Official buildpack documentation can be found at [python buildpack docs](http://docs.cloudfoundry.org/buildpacks/python/index.html).

### uv

Apps managed with [uv](https://docs.astral.sh/uv/) are supported by pushing the `pyproject.toml` together with a `uv.lock`:

* During staging the buildpack runs `uv export --frozen` to generate a temporary `requirements.txt` from `uv.lock`, which is then installed with pip. Keep `uv.lock` up to date (`uv lock`) before pushing, as it is used as is.
* Only the default dependencies are installed, dependency groups such as `dev` are skipped. If the project defines a `[build-system]`, the project itself is installed as well.
* The Python version is taken from `.python-version` (e.g. `3.12`, as written by `uv python pin`), unless a `runtime.txt` exists.
* Package indexes configured via `[[tool.uv.index]]` are passed on to pip. Credentials are not part of `uv.lock`, so for authenticated indexes set `PIP_INDEX_URL` / `PIP_EXTRA_INDEX_URL` instead, which then take precedence.
* If the app also contains a `requirements.txt`, the `uv.lock` is ignored.
* For vendored (offline) installs, download the wheels for the target platform into the `vendor` directory. Write the exported file outside the app, so that it does not take precedence over `uv.lock`:

  ```bash
  uv export --frozen --no-dev --no-hashes -o /tmp/requirements.txt
  pip download -r /tmp/requirements.txt -d vendor --only-binary=:all: --platform manylinux2014_x86_64 --python-version 3.12
  ```

## Adding new dependencies

If you want to add a new dependency to the buildpack, please add it to the [config.yml](https://github.com/cloudfoundry/buildpacks-ci/blob/5a63d13df09f83d5dff7c71d0a12c3e2dc798d39/pipelines/dependency-builds/config.yml#L272) file. For example, if you want to add a new version of Python, add an entry like the following:

```yaml
python:
  lines:
    - line: 3.14.X
      deprecation_date: 2030-10-07
      link: https://peps.python.org/pep-0745/
```

The new dependency will be automatically added to the buildpack [manifest.yml](manifest.yml) file.

### Building the Buildpack

To build this buildpack, run the following commands from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Install buildpack-packager

    ```bash
    go install github.com/cloudfoundry/libbuildpack/packager/buildpack-packager@latest
    ```

1. Build the buildpack

    ```bash
    buildpack-packager build [ --cached=(true|false) ]
    ```

1. Use in Cloud Foundry

   Upload the buildpack to your Cloud Foundry and optionally specify it by name

    ```bash
    cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
    cf push my_app [-b BUILDPACK_NAME]
    ```

### Testing

Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/tree/master/cutlass) framework for running integration tests.

To test this buildpack, run the following commands from the buildpack's directory:
 
1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Run unit tests

    ```bash
    ./scripts/unit.sh
    ```

1. Run integration tests

   Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/tree/master/cutlass) framework for running integration tests against Cloud Foundry. Before running the integration tests, you need to login to your Cloud Foundry using the [cf cli](https://github.com/cloudfoundry/cli):

    ```bash
    cf login -a https://api.your-cf.com -u name@example.com -p pa55woRD
    ```

   Note that your user requires permissions to run `cf create-buildpack` and `cf update-buildpack`. To run the integration tests, run the following command from the buildpack's directory:

    ```bash
    ./scripts/integration.sh
    ```
    
1. Note: To run the network tests, you must have Docker installed.   

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/) if you need any further assistance.

### Reporting Issues

Open a GitHub issue on this project [here](https://github.com/cloudfoundry/python-buildpack/issues/new).

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066).
