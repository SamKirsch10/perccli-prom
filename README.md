# Perccli-Prom
A simple golang binary that will run some commands on my ESXI host via [PERC-CLI](https://www.dell.com/support/kbdoc/en-us/000177280/how-to-use-the-poweredge-raid-controller-perc-command-line-interface-cli-utility-to-manage-your-raid-controller). Since it's difficult to get much to run on the specialized kernel that runs on ESXi hosts, the commands are run over ssh.


## Building
[Github Actions](https://github.com/SamKirsch10/perccli-prom/actions) builds this repo and will post the binary to the released version. See [the workflow yaml](https://github.com/SamKirsch10/perccli-prom/blob/master/.github/workflows/main.yml) for the steps that build, set a version, and set the release.

## PERC CLI
For my setup, all disks are under Controller 0 - Enclosure 32 (not sure why...). Current output of a show all for my disks looks something like:
```
Drive Information :
=================

------------------------------------------------------------------------------------
EID:Slt DID State DG      Size Intf Med SED PI SeSz Model                   Sp Type
------------------------------------------------------------------------------------
32:0      0 Onln   2 111.25 GB SATA SSD N   N  512B INTEL SSDSA2CW120G3     U  -
32:1      1 Onln   1  931.0 GB SATA SSD Y   N  512B Samsung SSD 870 EVO 1TB U  -
32:2      2 Onln   1  931.0 GB SATA SSD Y   N  512B Samsung SSD 860 EVO 1TB U  -
32:3      3 Onln   0  3.637 TB SATA HDD N   N  512B WDC WD4000F9YZ-09N20L1  U  -
32:4      4 Onln   0  3.637 TB SATA HDD N   N  512B WDC WD4000F9YZ-09N20L1  U  -
32:5      5 Onln   0  3.637 TB SATA HDD N   N  512B WDC WD4000F9YZ-09N20L1  U  -
32:6      6 Onln   0  3.637 TB SATA HDD N   N  512B WDC WD4003FRYZ-01F0DB0  U  -
32:7      7 Onln   0  3.637 TB SATA HDD N   N  512B WDC WD4003FRYZ-01F0DB0  U  -
------------------------------------------------------------------------------------
```

The code uses regex to grab each piece (since disk models unfortunately can have unkown amount of spaces, it gets weird towards the end there...). 

** UPDATE** There is actually a json output for the perc cli cmd :) . MUCH easier to parse.
