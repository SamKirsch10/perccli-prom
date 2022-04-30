package main

type DriveInfo struct {
	Slot  string `json:"EID:Slt"`
	DID   int
	State string
	DG    int
	Size  string
	Intf  string
	Med   string
	SED   string
	PI    string
	SeSz  string
	Model string
	Sp    string
	Type  string
}

type RebuildInfo struct {
	ID       string `json:"Drive-ID"`
	Progress int    `json:"Progress%"`
	Status   string
	ETA      string `json:"Estimated Time Left"`
}

type CmdStatus struct {
	Version     string `json:"CLI Version"`
	OS          string `json:"Operating system"`
	Controller  int
	Status      string
	Description string
}

type ControllerDriveResponse struct {
	Controllers []struct {
		CommandStatus CmdStatus `json:"Command Status"`
		ResponseData  struct {
			DriveInformation []DriveInfo `json:"Drive Information"`
		} `json:"Response Data"`
	}
}

type ControllerRebuildResponse struct {
	Controllers []struct {
		CommandStatus CmdStatus     `json:"Command Status"`
		ResponseData  []RebuildInfo `json:"Response Data"`
	}
}

type EsxiDatastoreInfo struct {
	Free       float64
	MountPoint string
	Mounted    bool
	Size       float64
	Type       string
	UUID       string
	VolumeName string
}
