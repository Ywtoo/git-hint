package daemon

import (
	"fmt"
	"os"
)

func socketPath() string {
	return fmt.Sprintf("/tmp/githint-%s.sock", os.Getenv("USER"))
}

func lockFilePath() string { return socketPath() + ".lock" }
