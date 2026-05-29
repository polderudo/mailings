package conf

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"app/internals"

	"github.com/nakami-lounge-GmbH/tools/helpers"
	"github.com/spf13/viper"
)

var C *Config

// LoadConfig reads the configuration and also merges in the coreConfigFile and local custom file
// The system uses 3 config files:
// config.yaml
// config_cust.yaml
// local.yaml
// they are merged in in this order
func LoadConfig(cfgDir string) error {
	var err error
	viper.Reset()

	if cfgDir != "" {
		viper.AddConfigPath(cfgDir)
	}
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./data")

	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		return err
	}

	localDir := filepath.Dir(viper.ConfigFileUsed())
	custFile := filepath.Join(localDir, "config_cust.yaml")
	if _, err := os.Stat(custFile); err == nil {
		viper.SetConfigFile(custFile)
		if err := viper.MergeInConfig(); err != nil {
			panic(err)
		}
	}

	localFile := filepath.Join(localDir, "local.yaml")
	if _, err := os.Stat(localFile); err == nil {
		viper.SetConfigFile(localFile)
		if err := viper.MergeInConfig(); err != nil {
			panic(err)
		}
	}

	C = new(Config)
	if err := viper.Unmarshal(C); err != nil {
		log.Fatalf("Unable to unmarshal config: %s \n", err)
	}

	C.ConfigDir = filepath.Dir(viper.ConfigFileUsed())
	C.TestFilesDir = path.Join(C.ConfigDir, "testdata")

	C.ApplicationDir, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	if C.SystemLangISOCode == "" {
		C.SystemLangISOCode = "en"
	}

	C.TmpDir = filepath.Join(C.ApplicationDir, "tmp")
	if err := helpers.CreateDirIfNotExists(C.TmpDir); err != nil {
		return err
	}

	C.MediaDir = filepath.Join(C.ApplicationDir, "media")
	if err := helpers.CreateDirIfNotExists(C.MediaDir); err != nil {
		return err
	}

	if C.Logging.LogDir == "" {
		C.Logging.LogDir = filepath.Join(C.TmpDir, "log")
	}
	if err := helpers.CreateDirIfNotExists(C.Logging.LogDir); err != nil {
		return err
	}

	if C.Logging.ErrorLogging != nil {
		if C.Logging.ErrorLogging.LogDir == "" {
			C.Logging.ErrorLogging.LogDir = C.Logging.LogDir
		}
		if err := helpers.CreateDirIfNotExists(C.Logging.ErrorLogging.LogDir); err != nil {
			return err
		}
	}

	if C.Logging.WebLogging != nil {
		if C.Logging.WebLogging.LogDir == "" {
			C.Logging.WebLogging.LogDir = C.Logging.LogDir
		}
		if err := helpers.CreateDirIfNotExists(C.Logging.WebLogging.LogDir); err != nil {
			return err
		}
	}

	if C.Logging.DebugLogging != nil {
		if C.Logging.DebugLogging.LogDir == "" {
			C.Logging.DebugLogging.LogDir = C.Logging.LogDir
		}
		if err := helpers.CreateDirIfNotExists(C.Logging.DebugLogging.LogDir); err != nil {
			return err
		}
	}

	if !C.SensibleDataNotCrypt {
		//user passwords are crypted -> decrypt them
		C.DB.User, err = internals.Decrypt(C.DB.User)
		if err != nil {
			return fmt.Errorf("decrypt db user: %v", err)
		}

		C.DB.Password, err = internals.Decrypt(C.DB.Password)
		if err != nil {
			return fmt.Errorf("decrypt db passowrd: %v", err)
		}
	}

	//replace home dir
	usr, err := user.Current()
	if usr != nil {
		C.HTTPSCertFile = strings.ReplaceAll(C.HTTPSCertFile, "~", usr.HomeDir)
		C.HTTPSKeyFile = strings.ReplaceAll(C.HTTPSKeyFile, "~", usr.HomeDir)
	}

	return nil
}
