/*
Copyright 2018 The pdfcpu Authors.

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

package pdfcpu

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/pdfcpu/pdfcpu/internal/config"
	"github.com/pdfcpu/pdfcpu/pkg/font"
)

const (
	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict int = iota

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed

	// ValidationNone bypasses validation.
	ValidationNone
)

const (

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"

	// PermissionsAll enables all user access permission bits.
	PermissionsAll int16 = -1 // 0xFFFF

	// PermissionsNone disables all user access permissions bits.
	PermissionsNone int16 = -3901 // 0xF0C3

)

// CommandMode specifies the operation being executed.
type CommandMode int

// The available commands.
const (
	VALIDATE CommandMode = iota
	OPTIMIZE
	SPLIT
	MERGECREATE
	MERGEAPPEND
	EXTRACTIMAGES
	EXTRACTFONTS
	EXTRACTPAGES
	EXTRACTCONTENT
	EXTRACTMETADATA
	TRIM
	ADDATTACHMENTS
	ADDATTACHMENTSPORTFOLIO
	REMOVEATTACHMENTS
	EXTRACTATTACHMENTS
	LISTATTACHMENTS
	SETPERMISSIONS
	LISTPERMISSIONS
	ENCRYPT
	DECRYPT
	CHANGEUPW
	CHANGEOPW
	ADDWATERMARKS
	REMOVEWATERMARKS
	IMPORTIMAGES
	INSERTPAGESBEFORE
	INSERTPAGESAFTER
	REMOVEPAGES
	ROTATE
	NUP
	INFO
	INSTALLFONTS
	LISTFONTS
	LISTKEYWORDS
	ADDKEYWORDS
	REMOVEKEYWORDS
	LISTPROPERTIES
	ADDPROPERTIES
	REMOVEPROPERTIES
	COLLECT
)

type configuration struct {
	Reader15          bool
	DecodeAllStreams  bool   `yaml:"decodeAllStreams"`
	ValidationMode    string `yaml:"validationMode"`
	Eol               string `yaml:"eol"`
	WriteObjectStream bool   `yaml:"writeObjectStream"`
	WriteXRefStream   bool   `yaml:"writeXRefStream"`
	EncryptUsingAES   bool   `yaml:"encryptUsingAES"`
	EncryptKeyLength  int    `yaml:"encryptKeyLength"`
	Permissions       int    `yaml:"permissions"`
	Units             string `yaml:"units"`
}

// Configuration of a Context.
type Configuration struct {
	Path string

	// Enables PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enables decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed
	ValidationMode int

	// End of line char sequence for writing.
	Eol string

	// Turns on object stream generation.
	// A signal for compressing any new non-stream-object into an object stream.
	// true enforces WriteXRefStream to true.
	// false does not prevent xRefStream generation.
	WriteObjectStream bool

	// Switches between xRefSection (<=V1.4) and objectStream/xRefStream (>=V1.5) writing.
	WriteXRefStream bool

	// Turns on stats collection.
	// TODO Decision - unused.
	CollectStats bool

	// A CSV-filename holding the statistics.
	StatsFileName string

	// Supplied user password
	UserPW    string
	UserPWNew *string

	// Supplied owner password
	OwnerPW    string
	OwnerPWNew *string

	// EncryptUsingAES ensures AES encryption.
	// true: AES encryption
	// false: RC4 encryption.
	EncryptUsingAES bool

	// AES:40,128,256 RC4:40,128
	EncryptKeyLength int

	// Supplied user access permissions, see Table 22
	Permissions int16

	// Command being executed.
	Cmd CommandMode

	// Chosen units for outputting paper sizes.
	Units DisplayUnit
}

// ConfigPath defines the location of pdfcpu's configuration directory.
// If set to a file path, pdfcpu will ensure the config dir at this location.
// Other possible values:
// 	default:	Ensure config dir at default location
// 	disable:	Disable config dir usage
var ConfigPath string = "default"

var loadedDefaultConfig *Configuration

func loadedConfig(c configuration, configPath string) *Configuration {
	var conf Configuration
	conf.Reader15 = c.Reader15
	conf.DecodeAllStreams = c.DecodeAllStreams
	conf.WriteObjectStream = c.WriteObjectStream
	conf.WriteXRefStream = c.WriteXRefStream
	conf.EncryptUsingAES = c.EncryptUsingAES
	conf.EncryptKeyLength = c.EncryptKeyLength
	conf.Permissions = int16(c.Permissions)

	switch c.ValidationMode {
	case "ValidationStrict":
		conf.ValidationMode = ValidationStrict
	case "ValidationRelaxed":
		conf.ValidationMode = ValidationRelaxed
	case "ValidationNone":
		conf.ValidationMode = ValidationNone
	}

	switch c.Eol {
	case "EolLF":
		conf.Eol = EolLF
	case "EolCR":
		conf.Eol = EolCR
	case "EolCRLF":
		conf.Eol = EolCRLF
	}

	switch c.Units {
	case "points":
		conf.Units = POINTS
	case "inches":
		conf.Units = INCHES
	case "cm":
		conf.Units = CENTIMETRES
	case "mm":
		conf.Units = MILLIMETRES
	}

	conf.Path = configPath

	return &conf
}

func parseConfigFile(bb []byte, configPath string) error {
	var c configuration
	if err := yaml.Unmarshal(bb, &c); err != nil {
		return err
	}
	if !MemberOf(c.ValidationMode, []string{"ValidationStrict", "ValidationRelaxed", "ValidationNone"}) {
		return errors.Errorf("parseConfigFile: invalid validationMode: %s", c.ValidationMode)
	}
	if !MemberOf(c.Eol, []string{"EolLF", "EolCR", "EolCRLF"}) {
		return errors.Errorf("parseConfigFile: invalid eol: %s", c.Eol)
	}
	if !MemberOf(c.Units, []string{"points", "inches", "cm", "mm"}) {
		return errors.Errorf("parseConfigFile: invalid units: %s", c.Units)
	}
	loadedDefaultConfig = loadedConfig(c, configPath)
	//fmt.Println(loadedDefaultConfig)
	return nil
}

func generateConfigFile(fileName string) error {
	if err := ioutil.WriteFile(fileName, config.ConfigFileBytes, os.ModePerm); err != nil {
		return err
	}
	loadedDefaultConfig = newDefaultConfiguration()
	loadedDefaultConfig.Path = fileName
	return nil
}

func ensureConfigFileAt(path string) error {
	bb, err := ioutil.ReadFile(path)
	if err != nil {
		// Create path/pdfcpu/config.yml
		//fmt.Printf("writing %s ..\n", path)
		return generateConfigFile(path)
	}
	// Load configuration into loadedDefaultConfig.
	//fmt.Printf("loading %s ...\n", path)
	return parseConfigFile(bb, path)
}

// EnsureDefaultConfigAt tries to load the default configuration from path.
// If path/pdfcpu/config.yaml is not found, it will be created.
func EnsureDefaultConfigAt(path string) error {
	configDir := filepath.Join(path, "pdfcpu")
	font.UserFontDir = filepath.Join(configDir, "fonts")
	if err := os.MkdirAll(font.UserFontDir, os.ModePerm); err != nil {
		return err
	}
	if err := ensureConfigFileAt(filepath.Join(configDir, "config.yml")); err != nil {
		return err
	}
	return font.LoadUserFonts()
}

func newDefaultConfiguration() *Configuration {
	// NOTE: pdfcpu/internal/config/config.yml must be updated whenever the default configuration changes.
	return &Configuration{
		Reader15:          true,
		DecodeAllStreams:  false,
		ValidationMode:    ValidationRelaxed,
		Eol:               EolLF,
		WriteObjectStream: true,
		WriteXRefStream:   true,
		EncryptUsingAES:   true,
		EncryptKeyLength:  256,
		Permissions:       PermissionsNone,
	}
}

// NewDefaultConfiguration returns the default pdfcpu configuration.
func NewDefaultConfiguration() *Configuration {
	if loadedDefaultConfig != nil {
		c := *loadedDefaultConfig
		return &c
	}
	if ConfigPath != "disable" {
		path, err := os.UserConfigDir()
		if err != nil {
			path = os.TempDir()
		}
		if err := EnsureDefaultConfigAt(path); err == nil {
			c := *loadedDefaultConfig
			return &c
		}
	}
	return newDefaultConfiguration()
}

// NewAESConfiguration returns a default configuration for AES encryption.
func NewAESConfiguration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = true
	c.EncryptKeyLength = keyLength
	return c
}

// NewRC4Configuration returns a default configuration for RC4 encryption.
func NewRC4Configuration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = false
	c.EncryptKeyLength = keyLength
	return c
}

func (c Configuration) String() string {
	path := "default"
	if len(c.Path) > 0 {
		path = c.Path
	}
	return fmt.Sprintf("pdfcpu configuration:\n"+
		"Path:              %s\n"+
		"Reader15:          %t\n"+
		"DecodeAllStreams:  %t\n"+
		"ValidationMode:    %s\n"+
		"Eol:               %s\n"+
		"WriteObjectStream: %t\n"+
		"WriteXrefStream:   %t\n"+
		"EncryptUsingAES:   %t\n"+
		"EncryptKeyLength:  %d\n"+
		"Permissions:       %d\n"+
		"Units:             %s\n",
		path,
		c.Reader15,
		c.DecodeAllStreams,
		c.ValidationModeString(),
		c.EolString(),
		c.WriteObjectStream,
		c.WriteXRefStream,
		c.EncryptUsingAES,
		c.EncryptKeyLength,
		c.Permissions,
		c.UnitsString())
}

// EolString returns a string rep for the eol in effect.
func (c *Configuration) EolString() string {
	var s string
	switch c.Eol {
	case EolLF:
		s = "EolLF"
	case EolCR:
		s = "EolCR"
	case EolCRLF:
		s = "EolCRLF"
	}
	return s
}

// ValidationModeString returns a string rep for the validation mode in effect.
func (c *Configuration) ValidationModeString() string {
	if c.ValidationMode == ValidationStrict {
		return "strict"
	}
	if c.ValidationMode == ValidationRelaxed {
		return "relaxed"
	}
	return "none"
}

// UnitsString returns a string rep for the display unit in effect.
func (c *Configuration) UnitsString() string {
	var s string
	switch c.Units {
	case POINTS:
		s = "points"
	case INCHES:
		s = "inches"
	case CENTIMETRES:
		s = "cm"
	case MILLIMETRES:
		s = "mm"
	}
	return s
}

// ApplyReducedFeatureSet returns true if complex entries like annotations shall not be written.
func (c *Configuration) ApplyReducedFeatureSet() bool {
	switch c.Cmd {
	case SPLIT, TRIM, EXTRACTPAGES, MERGECREATE, MERGEAPPEND, IMPORTIMAGES:
		return true
	}
	return false
}
