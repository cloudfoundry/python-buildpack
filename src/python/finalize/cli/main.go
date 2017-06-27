package main

import (
	"os"
	"python/finalize"
	_ "python/hooks"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.SetStagingEnvironment(); err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(11)
	}

	f := finalize.Finalizer{
		Stager:   stager,
		Manifest: manifest,
		Command:  &libbuildpack.Command{},
		Log:      logger,
	}

	if err := finalize.Run(&f); err != nil {
		os.Exit(12)
	}

	if err := libbuildpack.RunAfterCompile(stager); err != nil {
		logger.Error("After Compile: %s", err.Error())
		os.Exit(13)
	}

	if err := stager.SetLaunchEnvironment(); err != nil {
		logger.Error("Unable to setup launch environment: %s", err.Error())
		os.Exit(14)
	}

	stager.StagingComplete()
}
