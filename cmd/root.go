package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/dsrosen6/hyprdocked/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	version = "0.3.0"
)

var (
	rootCmd = &cobra.Command{
		Use: "hyprdocked",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			debug := viper.GetBool("debug")
			if debug {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}
		},
	}

	versionCmd = &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	pingCmd = &cobra.Command{
		Use: "ping",
		Run: func(cmd *cobra.Command, args []string) {
			cobra.CheckErr(app.SendPingCmd())
			fmt.Println("OK")
		},
	}

	idleCmd = &cobra.Command{
		Use:     "idle",
		Aliases: []string{"i"},
		Run: func(cmd *cobra.Command, args []string) {
			source, _ := cmd.Flags().GetString("source")
			cobra.CheckErr(app.SendIdleCmd(source))
			fmt.Println("OK")
		},
	}

	resumeCmd = &cobra.Command{
		Use:     "resume",
		Aliases: []string{"r"},
		Run: func(cmd *cobra.Command, args []string) {
			source, _ := cmd.Flags().GetString("source")
			cobra.CheckErr(app.SendResumeCmd(source))
			fmt.Println("OK")
		},
	}

	listenCmd = &cobra.Command{
		Use:     "listen",
		Aliases: []string{"l"},
		Run: func(cmd *cobra.Command, args []string) {
			var c app.Config
			cobra.CheckErr(viper.Unmarshal(&c))
			cobra.CheckErr(app.RunListener(c))
		},
	}

	lockCmd = &cobra.Command{
		Use:   "lock",
		Short: "Run the configured lock command",
		Run: func(cmd *cobra.Command, args []string) {
			lockCmd := viper.GetString("lock-cmd")
			if lockCmd == "" {
				lockCmd = app.DefaultLockCmd
			}
			cobra.CheckErr(exec.Command("sh", "-c", lockCmd).Start())
		},
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().StringP("laptop", "l", "eDP-1", "laptop monitor name")
	rootCmd.PersistentFlags().Bool("lock-on-idle", true, "run lock command when idle command is sent")
	rootCmd.PersistentFlags().String("lock-cmd", "", "command to run for locking (default: \"pidof hyprlock || hyprlock\")")
	rootCmd.PersistentFlags().Int("lock-delay", 0, "seconds to wait after lock command before continuing idle procedures (default 1)")
	rootCmd.PersistentFlags().Int("suspend-delay", 0, "seconds to wait after enabling display before suspending (default 1)")
	rootCmd.PersistentFlags().Bool("suspend-idle", true, "suspend device when idle command is sent")
	rootCmd.PersistentFlags().Bool("suspend-closed", true, "suspend device on lid closed if only laptop")
	rootCmd.PersistentFlags().Bool("sequential-hooks", false, "run post-hooks sequentially instead of concurrently")
	rootCmd.PersistentFlags().Int("settle-window", 1, "seconds to wait after an event before processing (default 1)")

	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("laptop", rootCmd.PersistentFlags().Lookup("laptop"))
	_ = viper.BindPFlag("lock-on-idle", rootCmd.PersistentFlags().Lookup("lock-on-idle"))
	_ = viper.BindPFlag("lock-cmd", rootCmd.PersistentFlags().Lookup("lock-cmd"))
	_ = viper.BindPFlag("lock-delay", rootCmd.PersistentFlags().Lookup("lock-delay"))
	_ = viper.BindPFlag("suspend-delay", rootCmd.PersistentFlags().Lookup("suspend-delay"))
	_ = viper.BindPFlag("suspend-idle", rootCmd.PersistentFlags().Lookup("suspend-idle"))
	_ = viper.BindPFlag("suspend-closed", rootCmd.PersistentFlags().Lookup("suspend-closed"))
	_ = viper.BindPFlag("sequential-hooks", rootCmd.PersistentFlags().Lookup("sequential-hooks"))
	_ = viper.BindPFlag("settle-window", rootCmd.PersistentFlags().Lookup("settle-window"))

	idleCmd.Flags().String("source", "", "source of the idle command (logged by listener)")
	resumeCmd.Flags().String("source", "", "source of the resume command (logged by listener)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(pingCmd)
	rootCmd.AddCommand(idleCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(listenCmd)
	rootCmd.AddCommand(lockCmd)
}
