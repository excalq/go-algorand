// Copyright (C) 2019-2022 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package logging

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/util/uuid"
)

// Note on Levels: Config files use a different level numbering scheme the codebase
// Due to refactoring from Logrus to Zerolog. logLevels.go provides translation when
// reading/saving the config. The codebase uses zerolog's native scheme for performance.

// TelemetryConfigFilename default file name for telemetry config "logging.config"
var TelemetryConfigFilename = "logging.config"

var defaultTelemetryUsername = "telemetry-v9"
var defaultTelemetryPassword = "oq%$FA1TOJ!yYeMEcJ7D688eEOE#MGCu"

const hostnameLength = 255

// TelemetryOverride Determines whether an override value is set and what it's value is.
// The first return value is whether an override variable is found, if it is, the second is the override value.
func TelemetryOverride(env string, telemetryConfig *TelemetryConfig) bool {
	env = strings.ToLower(env)

	if env == "1" || env == "true" {
		telemetryConfig.Enable = true
	}

	if env == "0" || env == "false" {
		telemetryConfig.Enable = false
	}

	return telemetryConfig.Enable
}

// createTelemetryConfig creates a new TelemetryConfig structure with a generated GUID and the appropriate Telemetry endpoint.
// Note: This should only be used/persisted when initially creating 'TelemetryConfigFilename'. Because the methods are called
//       from various tools and goal commands and affect the future default settings for telemetry, we need to inject
//       a "dev" branch check.
func createTelemetryConfig() TelemetryConfig {
	enable := false

	return TelemetryConfig{
		Enable:             enable,
		GUID:               uuid.New(),
		URI:                "",
		MinLogLevel:        Warn,
		ReportHistoryLevel: Warn,
		// These credentials are here intentionally. Not a bug.
		UserName: defaultTelemetryUsername,
		Password: defaultTelemetryPassword,
	}
}

// LoadTelemetryConfig loads the TelemetryConfig from the config file
func LoadTelemetryConfig(configPath string) (TelemetryConfig, error) {
	return loadTelemetryConfig(configPath)
}

// Save saves the TelemetryConfig to the config file
func (cfg TelemetryConfig) Save(configPath string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var marshaledConfig MarshalingTelemetryConfig
	marshaledConfig.TelemetryConfig = cfg
	marshaledConfig.TelemetryConfig.FilePath = ""
	marshaledConfig.MinLogLevel = LevelToConfigValue(cfg.MinLogLevel)
	marshaledConfig.ReportHistoryLevel = LevelToConfigValue(cfg.ReportHistoryLevel)

	// If the configuration contains both default username and password for the telemetry
	// server then we just want to substitute a blank string
	if marshaledConfig.TelemetryConfig.UserName == defaultTelemetryUsername &&
		marshaledConfig.TelemetryConfig.Password == defaultTelemetryPassword {
		marshaledConfig.TelemetryConfig.UserName = ""
		marshaledConfig.TelemetryConfig.Password = ""
	}

	enc := json.NewEncoder(f)
	err = enc.Encode(marshaledConfig)
	return err
}

// getHostGUID returns the Host GUID for telemetry (GUID:Name -- :Name is optional if blank)
func (cfg TelemetryConfig) getHostGUID() string {
	ret := cfg.GUID
	if cfg.Enable && len(cfg.Name) > 0 {
		ret += ":" + cfg.Name
	}
	return ret
}

// getInstanceName allows us to distinguish between multiple instances running on the same node.
func (cfg TelemetryConfig) getInstanceName() string {
	p := config.GetCurrentVersion().DataDirectory
	hash := sha256.New()
	hash.Write([]byte(cfg.GUID))
	hash.Write([]byte(p))
	pathHash := sha256.Sum256(hash.Sum(nil))
	pathHashStr := base64.StdEncoding.EncodeToString(pathHash[:])

	// NOTE: We used to report HASH:DataDir but DataDir is Personally Identifiable Information (PII)
	// So we're removing it entirely to avoid GDPR issues.
	return fmt.Sprintf("%s", pathHashStr[:16])
}

// SanitizeTelemetryString applies sanitization rules and returns the sanitized string.
func SanitizeTelemetryString(input string, maxParts int) string {
	// Truncate to a reasonable size, allowing some undefined separator.
	maxReasonableSize := maxParts*hostnameLength + maxParts - 1
	if len(input) > maxReasonableSize {
		input = input[:maxReasonableSize]
	}
	return input
}

// Returns err if os.Open fails or if config is mal-formed
func loadTelemetryConfig(path string) (TelemetryConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return createTelemetryConfig(), err
	}
	defer f.Close()
	var cfg TelemetryConfig
	var marshaledConfig MarshalingTelemetryConfig
	marshaledConfig.TelemetryConfig = createTelemetryConfig()
	dec := json.NewDecoder(f)
	err = dec.Decode(&marshaledConfig)
	cfg = marshaledConfig.TelemetryConfig
	// Errors will also return Debug
	cfg.MinLogLevel, err = LevelFromConfigValue(int32(marshaledConfig.MinLogLevel))
	cfg.ReportHistoryLevel, err = LevelFromConfigValue(int32(marshaledConfig.ReportHistoryLevel))
	cfg.FilePath = path

	if cfg.UserName == "" && cfg.Password == "" {
		cfg.UserName = defaultTelemetryUsername
		cfg.Password = defaultTelemetryPassword
	}

	// Sanitize user-defined name.
	if len(cfg.Name) > 0 {
		cfg.Name = SanitizeTelemetryString(cfg.Name, 1)
	}

	return cfg, err
}


// ReadTelemetryConfigOrDefault reads telemetry config from file or defaults if no config file found.
func ReadTelemetryConfigOrDefault(dataDir string, genesisID string) (cfg TelemetryConfig, err error) {
	err = nil
	dataDirProvided := dataDir != ""
	var configPath string

	// If we have a data directory, then load the config
	if dataDirProvided {
		configPath = filepath.Join(dataDir, TelemetryConfigFilename)
		// Load the config, if the GUID is there then we are all set
		// However if it isn't there then we must create it, save the file and load it.
		cfg, err = LoadTelemetryConfig(configPath)
	}

	// We couldn't load the telemetry config for some reason
	// If the reason is because the directory doesn't exist or we didn't provide a data directory then...
	if (err != nil && os.IsNotExist(err)) || !dataDirProvided {

		configPath, err = config.GetConfigFilePath(TelemetryConfigFilename)
		if err != nil {
			// If the path could not be opened do nothing, the IsNotExist error
			// is handled below.
		} else {
			// Load the telemetry from the default config path
			cfg, err = LoadTelemetryConfig(configPath)
		}
	}

	// If there was some error loading the configuration from the config path...
	if err != nil {
		// Create an ephemeral config
		cfg = createTelemetryConfig()

		// If the error was that the the config wasn't there then it wasn't really an error
		if os.IsNotExist(err) {
			err = nil
		} else {
			// The error was actually due to a malformed config file...just return
			return
		}
	}
	ver := config.GetCurrentVersion()
	ch := ver.Channel
	// Should not happen, but default to "dev" if channel is unspecified.
	if ch == "" {
		ch = "dev"
	}
	cfg.ChainID = fmt.Sprintf("%s-%s", ch, genesisID)
	cfg.Version = ver.String()
	return cfg, err
}

// EnsureTelemetryConfig creates a new TelemetryConfig structure with a generated GUID and the appropriate Telemetry endpoint
// Err will be non-nil if the file doesn't exist, or if error loading.
// Cfg will always be valid.
func EnsureTelemetryConfig(dataDir *string, genesisID string) (TelemetryConfig, error) {
	cfg, _, err := EnsureTelemetryConfigCreated(dataDir, genesisID)
	return cfg, err
}

// EnsureTelemetryConfigCreated is the same as EnsureTelemetryConfig but it also returns a bool indicating
// whether EnsureTelemetryConfig had to create the config.
func EnsureTelemetryConfigCreated(dataDir *string, genesisID string) (TelemetryConfig, bool, error) {
	/*
		Our logic should be as follows:
			- We first look inside the provided data-directory.  If a config file is there, load it
			  and return it
			- Otherwise, look in the global directory.  If a config file is there, load it and return it.
			- Otherwise, if a data-directory was provided then save the config file there.
			- Otherwise, save the config file in the global directory
	*/

	configPath := ""
	var cfg TelemetryConfig
	var err error

	if dataDir != nil && *dataDir != "" {
		configPath = filepath.Join(*dataDir, TelemetryConfigFilename)
		cfg, err = LoadTelemetryConfig(configPath)
		if err != nil && os.IsNotExist(err) {
			// if it just didn't exist, try again at the other path
			configPath = ""
		}
	}
	if configPath == "" {
		configPath, err = config.GetConfigFilePath(TelemetryConfigFilename)
		if err != nil {
			cfg := createTelemetryConfig()
			// Since GetConfigFilePath failed, there is no chance that we
			// can save the next config files
			return cfg, true, err
		}
		cfg, err = LoadTelemetryConfig(configPath)
	}
	created := false
	if err != nil {
		created = true
		cfg = createTelemetryConfig()

		if dataDir != nil && *dataDir != "" {

			/*
				There could be a scenario where a data directory was supplied that doesn't exist.
				In that case, we don't want to create the directory, just save in the global one
			*/

			// If the directory exists...
			if _, err := os.Stat(*dataDir); err == nil {

				// Remember, if we had a data directory supplied we want to save the config there
				configPath = filepath.Join(*dataDir, TelemetryConfigFilename)
			}

		}

		cfg.FilePath = configPath // Initialize our desired cfg.FilePath

		// There was no config file, create it.
		err = cfg.Save(configPath)
	}

	ver := config.GetCurrentVersion()
	ch := ver.Channel
	// Should not happen, but default to "dev" if channel is unspecified.
	if ch == "" {
		ch = "dev"
	}
	cfg.ChainID = fmt.Sprintf("%s-%s", ch, genesisID)
	cfg.Version = ver.String()

	return cfg, created, err
}
