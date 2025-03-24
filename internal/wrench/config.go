package wrench

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hexops/wrench/internal/errors"
)

type ModeType string

const (
	ModeWrench ModeType = "wrench"
	ModePkg    ModeType = "pkg"
	ModeZig    ModeType = "zig"
)

type Config struct {
	// ExternalURL where Wrench is hosted, if any.
	ExternalURL string

	// Address serve on, e.g. ":443" or ":80".
	//
	// Disabled if an empty string.
	Address string `toml:"Address,omitempty"`

	// When PkgProxy=true, disable any HTTP requests to pkg.machengine.org
	//
	// Note: setting this means your mirror may not be able to get Mach nominated
	// versions, because ziglang.org purges them after some time.
	PkgProxyDisableMachMirror bool `toml:"PkgProxyDisableMachMirror,omitempty"`

	// Where Wrench should store its data, cofiguration, etc. Defaults to the directory containing
	// this config file.
	WrenchDir string `toml:"WrenchDir,omitempty"`
}

func (c *Config) LogFilePath() string {
	return filepath.Join(c.WrenchDir, "logs")
}

func (c *Config) WriteTo(file string) error {
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return errors.Wrap(err, "MkdirAll")
	}
	f, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "Create")
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return errors.Wrap(enc.Encode(c), "Encode")
}

func LoadConfig(file string, out *Config) error {
	_, err := toml.DecodeFile(file, out)
	if err != nil {
		return err
	}
	if out.WrenchDir == "" {
		out.WrenchDir, err = filepath.Abs(filepath.Dir(file))
		if err != nil {
			return errors.Wrap(err, "Abs")
		}
	}
	return nil
}
