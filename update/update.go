package update

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const version = "0.0.3"
const repository = "vladutt/sshm"
const (
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

func DoSelfUpdate() {
	v := semver.MustParse(version)
	latest, err := selfupdate.UpdateSelf(v, repository)
	if err != nil {
		fmt.Println("Binary update failed:", err)
		return
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		fmt.Println("Current binary is the latest version", version)
	} else {
		fmt.Println("Successfully updated to version", latest.Version)
		fmt.Println("Release note:\n", latest.ReleaseNotes)
	}
}

func DisplayCurrentVersion() {

	latest, found, err := selfupdate.DetectLatest(repository)

	if err != nil {
		fmt.Println("Error occurred while detecting version:", err)
		return
	}

	v := semver.MustParse(version)
	if !found || latest.Version.LTE(v) {
		fmt.Printf("%s✔ You are up to date!%s (current: %s%s%s)\n",
			green, reset, yellow, version, reset)
		return
	} else {
		fmt.Printf("%s▲ Update available!%s \nLatest: %s%s%s, Current: %s%s%s\n",
			red, reset,
			green, latest.Version.String(), reset,
			yellow, version, reset)
	}

}
