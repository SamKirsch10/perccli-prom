package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	disksOnline = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "server_physical_disks_online_total",
		Help: "The total number of disks with status Online",
	})
	diskRebuildPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "server_disk_rebuild_percent",
		Help: "The percent progress for disk build",
	}, []string{"deviceID"})
)

func main() {

	go server_disk_metrics()

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

		out := runCMD("/opt/MegaRAID/perccli/perccli /c0/e32/sall show J")
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
					cmd := fmt.Sprintf("/opt/MegaRAID/perccli/perccli /c0/e32/s%d show rebuild J", disk.DID)
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

func runCMD(cmd string) string {
	command := strings.Split(cmd, " ")
	out, err := exec.Command(command[0], command[1:]...).Output()
	if err != nil {
		log.Fatal(err)
	}
	return string(out)
}
