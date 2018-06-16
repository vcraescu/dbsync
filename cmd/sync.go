package cmd

import (
	"log"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vcraescu/dbsync/internal/database/mysql"
)

var syncCmd = &cobra.Command{
	Use:   "sync [MASTER_NAME] [SLAVE_NAME]",
	Short: "Sync master server with name [MASTER_NAME] from config to slave server with name [SLAVE_NAME] from config.",
	Args:  cobra.ExactArgs(2),
	Run:   runSyncCmd,
}

func runSyncCmd(_ *cobra.Command, _ []string) {
	masterCfg := config.CreateMasterConnectionConfig()
	slaveCfg := config.CreateSlaveConnectionConfig()

	if config.MasterSSHTunnelIsRequired() {
		masterTunn, err := startSSHTunnel(masterCfg, config.Master.SSHConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("SSH Tunnel for master started at %s:%d\n", masterTunn.LocalHost(), masterTunn.LocalPort())
	}

	if config.SlaveSSHTunnelIsRequired() {
		slaveTunn, err := startSSHTunnel(slaveCfg, config.Slave.SSHConfig)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("SSH Tunnel for slave started at %s:%d\n", slaveTunn.LocalHost(), slaveTunn.LocalPort())
	}

	masterConn := mysql.New(*masterCfg)
	slaveConn := mysql.New(*slaveCfg)

	log.Println("Computing differences between master and slave...")
	diff, err := mysql.GenerateDiff(masterConn, slaveConn)
	if err != nil {
		log.Fatal(err)
	}

	if diff.Empty() {
		log.Println("Nothing to sync. Exit")
		return
	}

	if len(diff.Create) > 0 {
		log.Println(fmt.Sprintf("Create tables: %s", strings.Join(diff.Create, ", ")))
	}

	if len(diff.Delete) > 0 {
		log.Println(fmt.Sprintf("Delete tables: %s", strings.Join(diff.Create, ", ")))
	}

	dumper := mysql.NewDumper(*masterCfg)
	if err != nil {
		log.Fatal(err)
	}

	dump, err := diff.GenerateSQL(dumper)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Syncing...")

	imp := mysql.NewImporter(*slaveCfg)
	if err = imp.Import(dump); err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")
}

