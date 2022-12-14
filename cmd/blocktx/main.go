package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	"github.com/TAAL-GmbH/arc/blocktx"
	"github.com/TAAL-GmbH/arc/blocktx/store/memory"
	"github.com/ordishs/gocore"

	_ "github.com/lib/pq"
)

// Name used by build script for the binaries. (Please keep on single line)
const progname = "block-tx"

// // Version & commit strings injected at build with -ldflags -X...
var version string
var commit string

var logger = gocore.Log(progname)

var (
	dbHost, _     = gocore.Config().Get("dbHost", "localhost")
	dbPort, _     = gocore.Config().GetInt("dbPort", 5432)
	dbName, _     = gocore.Config().Get("dbName", "blocktx")
	dbUser, _     = gocore.Config().Get("dbUser", "blocktx")
	dbPassword, _ = gocore.Config().Get("dbPassword", "blocktx")
)

func init() {
	gocore.SetInfo(progname, version, commit)
}

func main() {
	stats := gocore.Config().Stats()
	logger.Infof("STATS\n%s\nVERSION\n-------\n%s (%s)\n\n", stats, version, commit)

	go func() {
		profilerAddr, ok := gocore.Config().Get("profilerAddr.blocktx")
		if ok {
			logger.Infof("Starting profile on http://%s/debug/pprof", profilerAddr)
			logger.Fatalf("%v", http.ListenAndServe(profilerAddr, nil))
		}
	}()

	// setup signal catching
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt)

	go func() {
		<-signalChan

		appCleanup()
		os.Exit(1)
	}()

	start()
}

func appCleanup() {
	logger.Infof("Shutting down...")
}

func start() {
	/*
		dbConn, err := sql.NewProcessor("postgres", dbHost, dbUser, dbPassword, dbName, dbPort)
		if err != nil {
			panic("Could not connect to fn: " + err.Error())
		}
	*/
	dbConn, err := memory.New()
	if err != nil {
		logger.Fatal(err)
	}

	mtb := blocktx.NewHandler(logger)

	var p *blocktx.Processor
	p, err = blocktx.NewProcessor(dbConn, mtb)
	if err != nil {
		logger.Fatal(err)
	}

	zmqHost, _ := gocore.Config().Get("peer_1_host", "localhost")
	zmqPort, _ := gocore.Config().GetInt("peer_1_zmqPort", 28332)
	z := blocktx.NewZMQ(p, zmqHost, zmqPort)

	go p.Catchup()

	z.Start()

	srv := blocktx.NewServer(dbConn, p, logger)
	err = srv.StartGRPCServer()
	if err != nil {
		logger.Fatal(err)
	}
}
