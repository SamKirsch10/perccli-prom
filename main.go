package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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

	disksOnline = promauto.NewGauge(prometheus.GaugeOpts{
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

			out := runCMD("cd /opt/lsi/perccli/ && ./perccli /c0/e32/sall show J")
			var diskMap ControllerResponse
			if err := json.Unmarshal([]byte(out), &diskMap); err != nil {
				log.Println("ERROR", err.Error())
			}

			// diskMap := parseOutput(out)
			disksOnline.Set(0)
			for _, disk := range diskMap.Controllers[0].ResponseData.DriveInformation {
				if disk.State == "Onln" {
					log.Println(disk)
					disksOnline.Inc()
				} else if disk.State == "Rbld" {
					log.Println(disk)
					cmd := fmt.Sprintf("cd /opt/lsi/perccli/ && ./perccli /c0/e32/s%d show rebuild J", disk.DID)
					out = runCMD(cmd)
					var rebuildInfo ControllerRebuildResponse
					if err := json.Unmarshal([]byte(out), &rebuildInfo); err != nil {
						log.Println("ERROR", err.Error())
					} else {
						percent := rebuildInfo.Controllers[0].ResponseData[0].Progress
						log.Println("Got rebuild percent for disk:", percent)
						diskRebuildPercent.WithLabelValues(fmt.Sprintf("%d", disk.DID)).Set(float64(percent))
						// Allows next run to happen sooner to check next percentage
						sleepTime = 30
					}

				} else {
					log.Println("Disk in unknown state!")
					log.Println(disk)
				}
			}

			time.Sleep(sleepTime * time.Second)
		}

	}()
}

// func parseOutput(cmdOut string) []map[string]string {
// 	var diskOutput []map[string]string

// 	var re = regexp.MustCompile(`32:(?P<slotID>[0-9]+)\s+(?P<deviceID>[0-9]+)\s(?P<status>[a-zA-Z]+)\s+(?P<driveGroup>[0-9]+)\s+(?P<size>[0-9.]+\s[GMT][B])\sSATA\s[SHD]{3}\s[YN]\s+[YN]\s+[0-9B]+\s(?P<model>[a-zA-Z0-9-\s]+\s[UD])(.*)`)

// 	for _, disk := range strings.Split(cmdOut, "\n") {
// 		if strings.Contains(disk, "-----------------") {
// 			break
// 		}
// 		match := re.FindStringSubmatch(disk)
// 		if len(match) > 0 {
// 			result := make(map[string]string)
// 			for i, cgName := range re.SubexpNames() {
// 				if i != 0 && cgName != "" {
// 					if cgName == "model" && (strings.HasSuffix(match[i], "U") || strings.HasSuffix(match[i], "D")) {
// 						result[cgName] = strings.TrimSpace(match[i][:len(match[i])-2])
// 					} else {
// 						result[cgName] = match[i]
// 					}
// 				}
// 			}
// 			if result["status"] != "Onln" && result["status"] != "Offln" {
// 				cmd := fmt.Sprintf("cd /opt/lsi/perccli/ && ./perccli /c0/e32/s%s show rebuild | tail -n +11 | head -1 | awk '{print $2}'", result["deviceID"])
// 				result["rebuild_percent"] = runCMD(cmd)
// 			} else {
// 				result["rebuild_percent"] = "-"
// 			}
// 			diskOutput = append(diskOutput, result)
// 		}
// 	}
// 	return diskOutput
// }

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
