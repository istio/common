// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func AddConfigFlag(rootCmd *cobra.Command, viper *viper.Viper) {
	var cfgFile string
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file containing args")

	cobra.OnInitialize(func() {
		if len(cfgFile) > 0 {
			viper.SetConfigFile(cfgFile)
			err := viper.ReadInConfig() // Find and read the config file
			if err != nil {             // Handle errors reading the config file
				_, _ = os.Stderr.WriteString(fmt.Errorf("fatal error in config file: %s", err).Error())
				os.Exit(1)
			}
		}
	})
}

// ProcessViperConfig retrieves Viper values for each Cobra Val Flag
func ProcessViperConfig(cmd *cobra.Command, viper *viper.Viper) {
	viper.SetTypeByDefaultValue(true)
	fmt.Println("processing viper config")
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if reflect.TypeOf(viper.Get(f.Name)).Kind() == reflect.Slice {
			// Viper cannot convert slices to strings, so this is our workaround.
			_ = f.Value.Set(strings.Join(viper.GetStringSlice(f.Name), ","))
		} else {
			_ = f.Value.Set(viper.GetString(f.Name))
		}
	})
	fmt.Println(viper.GetString("mixerIdentity"))
}
