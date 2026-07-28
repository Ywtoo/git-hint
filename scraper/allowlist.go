package scraper

// commonCommands is a curated set of CLIs worth crawling eagerly on
// install/rebuild. Everything else discovered on $PATH is still indexed
// (name + path in index.json) but only gets its help tree crawled later,
// on demand, the first time the user actually types it. This keeps the
// initial rebuild fast (seconds, not the hour it takes to exec -h against
// every binary on the system) while covering what people actually use
// day to day.
var commonCommands = map[string]bool{
	// vcs
	"git": true, "hg": true, "svn": true,
	// containers / infra
	"docker": true, "docker-compose": true, "kubectl": true, "terraform": true,
	"helm": true, "podman": true, "vagrant": true, "ansible": true,
	// package managers
	"npm": true, "yarn": true, "pnpm": true, "pip": true, "pip3": true,
	"brew": true, "apt": true, "apt-get": true, "cargo": true, "go": true,
	"gem": true, "composer": true,
	// coreutils / shell essentials
	"ls": true, "cp": true, "mv": true, "rm": true, "grep": true, "find": true,
	"sed": true, "awk": true, "tar": true, "chmod": true, "chown": true,
	"curl": true, "wget": true, "ssh": true, "scp": true, "rsync": true,
	// dev tooling
	"make": true, "cmake": true, "gcc": true, "python3": true, "node": true,
	"code": true, "vim": true, "tmux": true,
	// misc very common
	"gh": true, "aws": true, "az": true, "gcloud": true, "jq": true,
}

// IsCommon reports whether name is in the eager-crawl allowlist.
func IsCommon(name string) bool {
	return commonCommands[name]
}
