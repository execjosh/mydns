// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/execjosh/mydns/internal/blocklist"
	"github.com/execjosh/mydns/internal/dnsqueryhandler"
	"github.com/execjosh/mydns/internal/iplist"
	"github.com/execjosh/mydns/internal/roundrobin"
	"github.com/miekg/dns"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("mydns: ")

	flagTCP := flag.Int("tcp", 0, "TCP port")
	flagUDP := flag.Int("udp", 0, "UDP port")
	flagNameservers := iplist.New()
	flag.Var(flagNameservers, "nameservers", "comma-separated list of IPs for upstream nameservers to be queried round-robin")
	flagTLSServerName := flag.String("tls-server-name", "", "server name for TLS. if set, enables TLS for upstream queries")
	flagBlocklistPath := flag.String("blocklist", "", "/path/to/block.list")
	flagJSON := flag.Bool("json", false, "whether to output logs as JSON")
	flag.Parse()

	logger := initLogger(*flagJSON)
	defer logger.Sync()

	if *flagTCP <= 0 && *flagUDP <= 0 {
		logger.Fatal("at least one port for TCP or UDP must be specified")
	}

	uniqListOfNameservers := flagNameservers.Uniq()
	if len(uniqListOfNameservers) < 1 {
		logger.Fatal("at least one nameserver required!")
	}
	upstreamPort := ":53"
	if len(*flagTLSServerName) > 0 {
		upstreamPort = ":853"
	}
	for idx, val := range uniqListOfNameservers {
		uniqListOfNameservers[idx] = val + upstreamPort
	}
	nameservers := roundrobin.New(uniqListOfNameservers)
	logger.Info("upstream servers", zap.Strings("nameservers", uniqListOfNameservers))

	blocklist, blockCnt, err := loadBlocklist(*flagBlocklistPath)
	if err != nil {
		logger.Error("failed to load blocklist", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("Blocking %d domains from %q", blockCnt, *flagBlocklistPath))

	dnsCli := &dns.Client{
		DialTimeout:    2 * time.Second,
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
		SingleInflight: true,
	}
	if len(*flagTLSServerName) > 0 {
		dnsCli.Net = "tcp-tls"
		dnsCli.TLSConfig = &tls.Config{
			ServerName: *flagTLSServerName,
			MinVersion: tls.VersionTLS13,
		}
	}

	srv := dnsqueryhandler.New(
		logger,
		dnsCli,
		nameservers,
		blocklist,
	)
	dns.HandleFunc(".", srv.HandleAandAAAA)

	if *flagUDP > 0 {
		go listenAndServe(logger, *flagUDP, "udp")
	}
	if *flagTCP > 0 {
		go listenAndServe(logger, *flagTCP, "tcp")
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.Info("memory stats",
		zap.Uint64("Alloc", m.Alloc),
		zap.Uint64("Sys", m.Sys),
	)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func initLogger(useJSON bool) *zap.Logger {
	pec := zap.NewProductionEncoderConfig()
	pec.EncodeTime = zapcore.ISO8601TimeEncoder
	pec.EncodeLevel = zapcore.CapitalLevelEncoder

	newEnc := zapcore.NewConsoleEncoder
	if useJSON {
		newEnc = zapcore.NewJSONEncoder
	}

	return zap.New(zapcore.NewCore(newEnc(pec), zapcore.AddSync(os.Stdout), zap.InfoLevel))
}

func listenAndServe(logger *zap.Logger, port int, network string) {
	srv := &dns.Server{Addr: fmt.Sprintf(":%d", port), Net: network}
	logger.Info(fmt.Sprintf("listening at %s (%s)", srv.Addr, srv.Net))
	if err := srv.ListenAndServe(); err != nil {
		logger.Fatal("listenAndServe failed", zap.Error(err))
	}
}

func loadBlocklist(filepath string) (*blocklist.Blocklist, uint, error) {
	if len(filepath) < 1 {
		return blocklist.Empty(), 0, nil
	}

	f, err := os.Open(filepath)
	if err != nil {
		return blocklist.Empty(), 0, fmt.Errorf("opening blocklist: %w", err)
	}
	defer f.Close()

	return blocklist.Load(f)
}
