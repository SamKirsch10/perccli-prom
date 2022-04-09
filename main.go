package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	esxiUser   string
	esxiPasswd string
	esxiHost   string

	disksOnline = promauto.NewCounter(prometheus.CounterOpts{
		Name: "esxi_physical_disks_online_total",
		Help: "The total number of disks with status Online",
	})
	diskRebuildPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "esxi_disk_rebuild_percent",
		Help: "The percent progress for disk build",
	}, []string{"deviceID"})
)

func init() {
	flag.StringVar(&esxiHost, "host", "", "ESXi host")
	flag.StringVar(&esxiUser, "user", "root", "SSH user for ESXi host")
	flag.StringVar(&esxiPasswd, "passwd", "", "SSH password for ESXi host")
	flag.Parse()
}

func main() {

	metrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}

func metrics() {
	go func() {
		for {
			var sleepTime time.Duration
			sleepTime = 600

			out := runCMD("cd /opt/lsi/perccli/ && ./perccli /c0/e32/sall show | tail -n +14")
			diskMap := parseOutput(out)

			for _, disk := range diskMap {
				if disk["status"] == "Onln" {
					log.Println(disk)
					disksOnline.Inc()
				}
				if disk["rebuild_percent"] != "-" {
					// Allows next run to happen sooner to check next percentage
					sleepTime = 5

					perc, err := strconv.ParseFloat(disk["rebuild_percent"], 10)
					if err != nil {
						log.Println("Could not convert string rebuild_percent to float64!")
						log.Println(err)
					} else {
						diskRebuildPercent.WithLabelValues(disk["deviceID"]).Set(perc)
					}

				}
			}

			time.Sleep(sleepTime * time.Second)
		}

	}()
}

func parseOutput(cmdOut string) []map[string]string {
	var diskOutput []map[string]string

	var re = regexp.MustCompile(`32:(?P<slotID>[0-9]+)\s+(?P<deviceID>[0-9]+)\s(?P<status>[a-zA-Z]+)\s+(?P<driveGroup>[0-9]+)\s+(?P<size>[0-9.]+\s[GMT][B])\sSATA\s[SHD]{3}\s[YN]\s+[YN]\s+[0-9B]+\s(?P<model>[a-zA-Z0-9-\s]+\s[UD])(.*)`)

	for _, disk := range strings.Split(cmdOut, "\n") {
		if strings.Contains(disk, "-----------------") {
			break
		}
		match := re.FindStringSubmatch(disk)
		if len(match) > 0 {
			result := make(map[string]string)
			for i, cgName := range re.SubexpNames() {
				if i != 0 && cgName != "" {
					if cgName == "model" && (strings.HasSuffix(match[i], "U") || strings.HasSuffix(match[i], "D")) {
						result[cgName] = strings.TrimSpace(match[i][:len(match[i])-2])
					} else {
						result[cgName] = match[i]
					}
				}
			}
			if result["status"] != "Onln" && result["status"] != "Offln" {
				cmd := fmt.Sprintf("cd /opt/lsi/perccli/ && ./perccli /c0/e32/s%s show rebuild | tail -n +11 | head -1 | awk '{print $2}'", result["deviceID"])
				result["rebuild_percent"] = runCMD(cmd)
			} else {
				result["rebuild_percent"] = "-"
			}
			diskOutput = append(diskOutput, result)
		}
	}
	return diskOutput
}

func runCMD(cmd string) string {
	var b bytes.Buffer
	client, session := connectViaSsh(esxiUser, esxiHost+":22", "")
	defer client.Close()

	session.Stdout = &b
	session.Run(cmd)

	return b.String()
}

func connectViaSsh(user, host string, password string) (*ssh.Client, *ssh.Session) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(SshInteractive),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var client *ssh.Client
	var err error

	if client, err = ssh.Dial("tcp", host, config); err != nil {
		log.Fatal(err)
	}
	session, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	return client, session
}

func SshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	answers = make([]string, len(questions))
	// The second parameter is unused
	for n, _ := range questions {
		answers[n] = esxiPasswd
	}

	return answers, nil
}
