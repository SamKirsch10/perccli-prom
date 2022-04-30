package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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

	disksOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "esxi_physical_disks_online_total",
		Help: "The total number of disks with status Online",
	})
	diskRebuildPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "esxi_disk_rebuild_percent",
		Help: "The percent progress for disk build",
	}, []string{"deviceID"})
	datastoreFree = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "esxi_datastore_bytes_free",
		Help: "The datastore bytes free",
	}, []string{"datastore"})
	datastoreSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "esxi_datastore_total_size_bytes",
		Help: "The datastore total size in bytes",
	}, []string{"datastore"})
)

func init() {
	flag.StringVar(&esxiHost, "host", "", "ESXi host")
	flag.StringVar(&esxiUser, "user", "root", "SSH user for ESXi host")
	flag.StringVar(&esxiPasswd, "passwd", "", "SSH password for ESXi host")
	flag.Parse()

	if (esxiHost == "") || (esxiPasswd == "") {
		flag.Usage()
		log.Fatal("Error, some arguments are missing/blank!")
	}
}

func main() {

	go server_disk_metrics()
	go server_esxi_metrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}

func server_disk_metrics() {
	for {
		var (
			sleepTime         time.Duration
			rebuildInProgress bool
			diskIDs           []string
			diskMap           ControllerDriveResponse
			rebuildInfo       ControllerRebuildResponse
		)
		sleepTime = 600

		out := runCMD("cd /opt/lsi/perccli/ && ./perccli /c0/e32/sall show J")
		if err := json.Unmarshal([]byte(out), &diskMap); err != nil {
			log.Println("ERROR", err.Error())
		} else {
			disksOnline.Set(0)
			for _, disk := range diskMap.Controllers[0].ResponseData.DriveInformation {
				diskIDs = append(diskIDs, fmt.Sprintf("%d", disk.DID))
				if disk.State == "Onln" {
					log.Println(disk)
					disksOnline.Inc()
				} else if disk.State == "Rbld" {
					rebuildInProgress = true
					log.Println(disk)
					cmd := fmt.Sprintf("cd /opt/lsi/perccli/ && ./perccli /c0/e32/s%d show rebuild J", disk.DID)
					out = runCMD(cmd)
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
			if !rebuildInProgress {
				diskRebuildPercent.Reset()
			}
		}

		time.Sleep(sleepTime * time.Second)
	}
}

func server_esxi_metrics() {
	for {
		var (
			sleepTime  time.Duration
			datastores []EsxiDatastoreInfo
		)
		sleepTime = 600

		out := runCMD("esxcli --debug --formatter=json storage filesystem list")
		if err := json.Unmarshal([]byte(out), &datastores); err != nil {
			log.Println("ERROR", err.Error())
		} else {
			for _, datastore := range datastores {
				if !(strings.Contains(datastore.VolumeName, "BOOT")) && !(strings.Contains(datastore.VolumeName, "OSDATA")) {
					datastoreFree.WithLabelValues(datastore.VolumeName).Set(datastore.Free)
					datastoreSize.WithLabelValues(datastore.VolumeName).Set(datastore.Size)
				}
			}

		}

		time.Sleep(sleepTime * time.Second)
	}
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
