// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package charm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/juju/loggo"
	"os/exec"
)

var logger = loggo.GetLogger("juju.charm")

// The Charm interface is implemented by any type that
// may be handled as a charm.
type Charm interface {
	Meta() *Meta
	Config() *Config
	Metrics() *Metrics
	Actions() *Actions
	Revision() int
}

// ReadCharm reads a Charm from path, which can point to either a charm archive or a
// charm directory.
func ReadCharm(path string) (charm Charm, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		charm, err = ReadCharmDir(path)
	} else {
		charm, err = ReadCharmArchive(path)
	}
	if err != nil {
		return nil, err
	}
	return charm, nil
}

// SeriesForCharm takes a requested series and a list of series supported by a
// charm and returns the series which is relevant.
// If the requested series is empty, then the first supported series is used,
// otherwise the requested series is validated against the supported series.
func SeriesForCharm(requestedSeries string, supportedSeries []string) (string, error) {
	// Old charm with no supported series.
	if len(supportedSeries) == 0 {
		if requestedSeries == "" {
			return "", missingSeriesError
		}
		return requestedSeries, nil
	}
	// Use the charm default.
	if requestedSeries == "" {
		return supportedSeries[0], nil
	}
	for _, s := range supportedSeries {
		if s == requestedSeries {
			return requestedSeries, nil
		}
	}
	return "", &unsupportedSeriesError{requestedSeries, supportedSeries}
}

// missingSeriesError is used to denote that SeriesForCharm could not determine
// a series because a legacy charm did not declare any.
var missingSeriesError = fmt.Errorf("series not specified and charm does not define any")

// IsMissingSeriesError returns true if err is an missingSeriesError.
func IsMissingSeriesError(err error) bool {
	return err == missingSeriesError
}

// UnsupportedSeriesError represents an error indicating that the requested series
// is not supported by the charm.
type unsupportedSeriesError struct {
	requestedSeries string
	supportedSeries []string
}

func (e *unsupportedSeriesError) Error() string {
	return fmt.Sprintf(
		"series %q not supported by charm, supported series are: %s",
		e.requestedSeries, strings.Join(e.supportedSeries, ","),
	)
}

// NewUnsupportedSeriesError returns an error indicating that the requested series
// is not supported by a charm.
func NewUnsupportedSeriesError(requestedSeries string, supportedSeries []string) error {
	return &unsupportedSeriesError{requestedSeries, supportedSeries}
}

// IsUnsupportedSeriesError returns true if err is an UnsupportedSeriesError.
func IsUnsupportedSeriesError(err error) bool {
	_, ok := err.(*unsupportedSeriesError)
	return ok
}

// MaybeCreateVersionFile creates/overwrite charm version file.
func MaybeCreateVersionFile(path string) error {
	var charmVersion string
	var cmdArgs []string
	var err error
	// Verify that it is revision control directory.
	if _, err = os.Stat(filepath.Join(path, ".git")); err == nil {
		// It is git version control.
		cmdArgs = []string{"git", "describe", "--dirty"}
	} else if _, err = os.Stat(filepath.Join(path, ".bzr")); err == nil {
		// It is baazar.
		cmdArgs = []string{"bzr", "revision-info"}
	} else if _, err = os.Stat(filepath.Join(path, ".hg")); err == nil {
		cmdArgs = []string{"hg", "id", "--id"}
	} else {
		logger.Infof("Charm is not in revision control directory")
		return nil
	}

	var args []string
	for pos, arg := range cmdArgs {
		if pos != 0 {
			args = append(args, arg)
		}
	}
	cmd := exec.Command(cmdArgs[0], args...)
	outStr, err := cmd.CombinedOutput()
	if err != nil {
		logger.Infof("Command output: %v", outStr)
		return err
	}
	charmVersion = string(outStr)

	versionPath := filepath.Join(path, "version")
	// Overwrite the existing version file.
	file, err := os.OpenFile(versionPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(charmVersion)
	if err != nil {
		return err
	}

	return nil
}
