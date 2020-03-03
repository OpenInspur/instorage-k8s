package utils

import (
	"fmt"
)

const (
	majorVersion   = 2
	minorVersion   = 1
	releaseVersion = 0
)

//CommitID is the last commit ID of this build.
var CommitID string

// GenerateVersionStr generate the version string.
func GenerateVersionStr() string {
	versionStr := fmt.Sprintf("%d.%d.%d", majorVersion, minorVersion, releaseVersion)

	if CommitID != "" {
		versionStr = versionStr + "." + CommitID
	}

	return versionStr
}
