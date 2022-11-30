package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "krueger",
	Short: "vpn companion to kill processes if your IP changes",
	Long: `Krueger is designed to run alongside your VPN connection.
If for any reason your IP changes, while you are connected to a VPN,
all the processes that you've set will be shutdown immediately.
It uses a UDP connection to get your hostname IP every second.`,
	Run: func(cmd *cobra.Command, args []string) {
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			panic(err)
		}

		cfgProcs := strings.Split(viper.GetString("processes"), ",")
		count := 0

		procs, err := process.Processes()
		if err != nil {
			panic(err)
		}

		tableData := pterm.TableData{
			{"PID", "Process Name"},
		}

		for _, p := range procs {
			procName, err := p.Name()
			if err != nil {
				panic(err)
			}
			if includes(cfgProcs, procName) {
				tableData = append(tableData, []string{fmt.Sprintf("%d", p.Pid), procName})
				count++
			}
		}

		pterm.DefaultBigText.WithLetters(
			putils.LettersFromStringWithRGB("Krueger", pterm.NewRGB(255, 150, 40))).
			Render()

		pterm.Println(pterm.Normal("Watching: ") + pterm.Blue(viper.GetString("processes")))

		pterm.Println(pterm.Normal("Protecting ") + pterm.Green(count) + pterm.Normal("/") + pterm.Gray(len(procs)) + pterm.Normal(" Processes"))

		pterm.Println("Current IP: " + pterm.Magenta(getMyIP().String()) + "\n")

		if debug {
			pterm.DefaultTable.WithHasHeader().WithData(tableData).WithRightAlignment().Render()
		}

		ch := make(chan struct{})
		pterm.DefaultSpinner.Start("Monitoring connection. You are currently " + pterm.BgGreen.Sprintf(" SAFE ") + ".")
		<-ch
		return
	},
}

func includes(src []string, name string) bool {
	for _, s := range src {
		if strings.Contains(strings.ToLower(s), strings.ToLower(name)) {
			return true
		}
	}
	return false
}

// Get preferred outbound ip of this machine
func getMyIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.krueger.yaml)")
	rootCmd.Flags().BoolP("debug", "d", false, "Debug mode")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".krueger" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".krueger")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		pterm.Println("Using config file:", pterm.Yellow(viper.ConfigFileUsed())+"\n")
	}
}
