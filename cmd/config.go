package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dsrosen6/hyprdocked/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var checkCfgCmd = &cobra.Command{
	Use:     "check-cfg",
	Aliases: []string{"check-config"},
	Short:   "Make sure config is valid and output values",
	Run: func(cmd *cobra.Command, args []string) {
		var cfg app.Config
		cobra.CheckErr(viper.Unmarshal(&cfg))

		sw := cfg.SettleWindow
		if sw <= 0 {
			sw = 1
		}

		lockCmd := cfg.LockCmd
		if lockCmd == "" {
			lockCmd = app.DefaultLockCmd + " (default)"
		}

		ld := cfg.LockDelay
		if ld <= 0 {
			ld = 1
		}

		fmt.Printf("%-25s %v\n", "Debug:", cfg.Debug)
		fmt.Printf("%-25s %s\n", "Laptop:", cfg.Laptop)
		fmt.Printf("%-25s %v\n", "Lock On Idle:", cfg.LockOnIdle)
		fmt.Printf("%-25s %s\n", "Lock Command:", lockCmd)
		fmt.Printf("%-25s %ds\n", "Lock Delay:", ld)

		sd := cfg.SuspendDelay
		if sd <= 0 {
			sd = 1
		}
		fmt.Printf("%-25s %ds\n", "Suspend Delay:", sd)
		fmt.Printf("%-25s %v\n", "Suspend On Idle:", cfg.SuspendIdle)
		fmt.Printf("%-25s %v\n", "Suspend On Closed:", cfg.SuspendClosed)
		fmt.Printf("%-25s %v\n", "Sequential Hooks:", cfg.SequentialHooks)
		fmt.Printf("%-25s %ds\n", "Settle Window:", sw)

		fmt.Printf("%-25s", "Post Hooks:")
		if len(cfg.PostUpdateHooks) == 0 {
			fmt.Println(" None")
		} else {
			fmt.Println()
			for _, h := range cfg.PostUpdateHooks {
				fmt.Printf("  %-23s %s\n", "Command:", h.Command)
				fmt.Printf("  %-23s %v\n", "On Status Change:", h.OnStatusChange)
			}
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/hypr/hyprdocked.yaml)")
	rootCmd.AddCommand(checkCfgCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		cfgDir := filepath.Join(home, ".config")

		viper.AddConfigPath(filepath.Join(cfgDir, "hypr"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("hyprdocked")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			cobra.CheckErr(err)
		}
	}
}
