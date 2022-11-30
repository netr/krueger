package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

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

		originalIP := getMyIP().String()

		var cfgProcs []string
		if len(viper.GetString("processes")) > 0 {
			cfgProcs = strings.Split(viper.GetString("processes"), ",")
		}

		if len(cfgProcs) == 0 {
			pterm.Println("No config file found at: " + pterm.Yellow(`$HOME/.config/.krueger.yaml`))
			pterm.Println("Please type the processes you want to monitor below.")
			pterm.Println("Typical use: " + pterm.Cyan("brave,firefox,chrome,safari,signal,keybase"))
			pterm.Println()

			result, _ := pterm.DefaultInteractiveTextInput.
				WithMultiLine(false).
				WithDefaultText("Type process names").Show()
			pterm.Println()

			cfgProcs = strings.Split(result, ",")
			if len(cfgProcs) == 0 {
				return
			}

		}

		procs, err := process.Processes()
		if err != nil {
			panic(err)
		}

		tableData := pterm.TableData{{"PID", "Process Name"}}
		count := 0
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

		if err := pterm.DefaultBigText.WithLetters(
			putils.LettersFromStringWithRGB("Krueger", pterm.NewRGB(255, 150, 40))).
			Render(); err != nil {
			panic(err)
		}

		pterm.Println(pterm.Normal("Watching: ") + pterm.Blue(strings.Join(cfgProcs, ", ")))
		pterm.Println(pterm.Normal("Protecting ") + pterm.Green(count) + pterm.Normal("/") + pterm.Gray(len(procs)) + pterm.Normal(" Processes"))
		pterm.Println("Current IP: " + pterm.Magenta(originalIP) + "\n")

		if debug {
			if err := pterm.DefaultTable.WithHasHeader().WithData(tableData).WithRightAlignment().Render(); err != nil {
				panic(err)
			}
		}

		start, err := pterm.DefaultSpinner.Start("Monitoring connection. You are currently " + pterm.BgGreen.Sprintf(" SAFE ") + ".")
		if err != nil {
			panic(err)
		}

	out:
		for {
			if getMyIP().String() != originalIP {
				for _, proc := range cfgProcs {
					for {
						if err := kill(proc); err != nil {
							break
						}
					}
				}

				pterm.Println(pterm.BgRed.Sprintf(" ATTENTION ") + " Your IP has changed from: " + pterm.Magenta(originalIP) + " to: " + pterm.Red(getMyIP().String()))
				pterm.Println(pterm.BgDarkGray.Sprintf(" GOODNIGHT ") + " Terminating Processes and Krueger...")
				break out
			}
			time.Sleep(time.Millisecond * 100)
		}

		_ = start.Stop()
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
	conn, err := net.Dial("udp", "0.0.0.1:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

// kill iterates through all processors and kills the first matching name
func kill(name string) error {
	processes, err := process.Processes()
	if err != nil {
		return err
	}
	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			return err
		}
		if n == name {
			return p.Kill()
		}
	}
	return fmt.Errorf("process not found")
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/.krueger.yaml)")
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
		viper.AddConfigPath(home + "/.config")
		viper.SetConfigName(".krueger")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		pterm.Println("Using config file:", pterm.Yellow(viper.ConfigFileUsed())+"\n")
	}
}
