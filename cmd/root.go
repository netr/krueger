package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	cfgProcs   []string
	originalIP string
)

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

		originalIP = getMyIP().String()

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

		tableData := buildProcessTableData(cfgProcs)

		if err := pterm.DefaultBigText.WithLetters(
			putils.LettersFromStringWithRGB("Krueger", pterm.NewRGB(255, 150, 40))).
			Render(); err != nil {
			panic(err)
		}

		// Time, Watching, Protecting, IP
		setupStatisticsAreaAndTimer()

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
				freddy(cfgProcs)

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

// setupStatisticsAreaAndTimer will create an updatable area printer and update stats once a minute. (Time, Watching, Protecting, Current IP)
func setupStatisticsAreaAndTimer() {
	area, _ := pterm.DefaultArea.Start()
	currentTimeStr := pterm.Sprintln("Time: " + pterm.Yellow(time.Now().Format(time.RFC850)) + "\n")
	watchingStr := pterm.Sprintln(pterm.Normal("Watching: ") + pterm.Blue(strings.Join(cfgProcs, ", ")))

	cntProtected, cntTotal := getProtectedProcessCounts(cfgProcs)
	protectStr := pterm.Sprintln(pterm.Normal("Protecting ") + pterm.Green(cntProtected) + pterm.Normal("/") + pterm.Gray(cntTotal) + pterm.Normal(" Processes"))

	currentIpStr := pterm.Sprintln("Current IP: " + pterm.Magenta(originalIP))
	area.Update(currentTimeStr, watchingStr, protectStr, currentIpStr)

	go func() {
		for _ = range time.Tick(time.Minute * 1) {
			currentTimeStr = pterm.Sprintln("Time: " + pterm.Yellow(time.Now().Format(time.RFC850)) + "\n")

			cntProtected, cntTotal = getProtectedProcessCounts(cfgProcs)
			protectStr = pterm.Sprintln(pterm.Normal("Protecting ") + pterm.Green(cntProtected) + pterm.Normal("/") + pterm.Gray(cntTotal) + pterm.Normal(" Processes"))

			currentIpStr = pterm.Sprintln("Current IP: " + pterm.Magenta(originalIP))
			area.Update(currentTimeStr, watchingStr, protectStr, currentIpStr)
		}
	}()
}

func buildProcessTableData(cfgProcs []string) [][]string {
	procNames, procIds := getProcessData()
	tableData := pterm.TableData{{"PID", "Process Name"}}
	for i, procName := range procNames {
		if includes(cfgProcs, procName) {
			tableData = append(tableData, []string{fmt.Sprintf("%d", procIds[i]), procName})
		}
	}
	return tableData
}

// getProtectedProcessCounts will get every process running on the machine and count how many the user is 'protecting'.
func getProtectedProcessCounts(cfgProcs []string) (protected int, total int) {
	procNames, _ := getProcessData()
	for _, procName := range procNames {
		if includes(cfgProcs, procName) {
			protected++
		}
		total++
	}
	return
}

// getProcessData get all process names and their corresponding IDs and returns them as paired arrays
func getProcessData() (names []string, procIds []int32) {
	procs, err := process.Processes()
	if err != nil {
		panic(err)
	}
	names = make([]string, len(procs))
	procIds = make([]int32, len(procs))
	for _, p := range procs {
		procName, err := p.Name()
		if err != nil {
			panic(err)
		}
		names = append(names, procName)
		procIds = append(procIds, p.Pid)
	}
	return
}

// freddy kill kill kill
func freddy(cfgProcs []string) {
	procs, err := process.Processes()
	if err != nil {
		panic(err)
	}
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if includes(cfgProcs, name) {
			for {
				if err := kill(name); err != nil {
					break
				}
			}
		}
	}
}

// includes some needle in a haystack
func includes(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.Contains(strings.ToLower(needle), strings.ToLower(s)) {
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
	c, b := exec.Command("git", "describe", "--tag"), new(strings.Builder)
	c.Stdout = b
	c.Run()
	s := strings.TrimRight(b.String(), "\n")
	rootCmd.Version = s

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
