/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
	logf "sigs.k8s.io/cluster-api/cmd/clusterctl/log"
)

const (
	// ConfigFolder defines the name of the config folder under $home
	ConfigFolder = ".cluster-api"
	// ConfigName defines the name of the config file under ConfigFolder
	ConfigName = "clusterctl"
)

// viperReader implements Reader using viper as backend for reading from environment variables
// and from a clusterctl config file.
type viperReader struct {
	configPaths []string
}

type viperReaderOption func(*viperReader)

func InjectConfigPaths(configPaths []string) viperReaderOption {
	return func(vr *viperReader) {
		vr.configPaths = configPaths
	}
}

// newViperReader returns a viperReader.
func newViperReader(opts ...viperReaderOption) Reader {
	vr := &viperReader{
		configPaths: []string{filepath.Join(homedir.HomeDir(), ConfigFolder)},
	}
	for _, o := range opts {
		o(vr)
	}
	return vr
}

// Init initialize the viperReader.
func (v *viperReader) Init(path string) error {
	log := logf.Log

	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return err
		}
		// Use path file from the flag.
		viper.SetConfigFile(path)
	} else {
		// Configure for searching .cluster-api/clusterctl{.extension} in home directory
		viper.SetConfigName(ConfigName)
		for _, p := range v.configPaths {
			viper.AddConfigPath(p)
		}
		if !v.checkDefaultConfig() {
			// since there is no default config to read from, just skip
			// reading in config
			log.V(5).Info("No default config file available")
			return nil
		}
	}

	// Configure for reading environment variables as well, and more specifically:
	// AutomaticEnv force viper to check for an environment variable any time a viper.Get request is made.
	// It will check for a environment variable with a name matching the key uppercased; in case name use the - delimiter,
	// the SetEnvKeyReplacer forces matching to name use the _ delimiter instead (- is not allowed in linux env variable names).
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AllowEmptyEnv(true)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	log.V(5).Info("Reading configuration", "File", viper.ConfigFileUsed())
	return nil
}

func (v *viperReader) Get(key string) (string, error) {
	if viper.Get(key) == nil {
		return "", errors.Errorf("Failed to get value for variable %q. Please set the variable value using os env variables or using the .clusterctl config file", key)
	}
	return viper.GetString(key), nil
}

func (v *viperReader) Set(key, value string) {
	viper.Set(key, value)
}

func (v *viperReader) UnmarshalKey(key string, rawval interface{}) error {
	return viper.UnmarshalKey(key, rawval)
}

// checkDefaultConfig checks the existence of the default config.
// Returns true if it finds a supported config file in the available config
// folders.
func (v *viperReader) checkDefaultConfig() bool {
	for _, path := range v.configPaths {
		for _, ext := range viper.SupportedExts {
			f := fmt.Sprintf("%s%s.%s", path, ConfigName, ext)
			_, err := os.Stat(f)
			if err == nil {
				return true
			}
		}
	}
	return false
}
