# Perccli-Prom
A simple golang binary that will run some commands on my ESXI host via [PERC-CLI](https://www.dell.com/support/kbdoc/en-us/000177280/how-to-use-the-poweredge-raid-controller-perc-command-line-interface-cli-utility-to-manage-your-raid-controller). Since it's difficult to get much to run on the specialized kernel that runs on ESXi hosts, the commands are run over ssh.