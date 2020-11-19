// Copyright (C) 2021  execjosh
// SPDX-License-Identifier: AGPL-3.0-or-later

package dnsqueryhandler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

type chooser interface {
	Next() string
}

type set interface {
	Contains(string) bool
}

type exchanger interface {
	Exchange(m *dns.Msg, address string) (r *dns.Msg, rtt time.Duration, err error)
}

// DNSQueryHandler represents a DNS query handler.
type DNSQueryHandler struct {
	logger      *zap.Logger
	exchanger   exchanger
	nameservers chooser
	blocklist   set
}

// New returns a new instance of DNSQueryHandler.
func New(
	logger *zap.Logger,
	exchanger exchanger,
	nameservers chooser,
	blocklist set,
) *DNSQueryHandler {
	return &DNSQueryHandler{
		logger:      logger,
		exchanger:   exchanger,
		nameservers: nameservers,
		blocklist:   blocklist,
	}
}

// HandleAandAAAA handles DNS queries for class INET and types A and AAAA. If the
// requested domain name is blocked, it responds with `0.0.0.0` for A (or `::`
// for AAAA). Otherwise, it forwards the request to an upstream server.
func (s *DNSQueryHandler) HandleAandAAAA(w dns.ResponseWriter, r *dns.Msg) {
	logger := s.logger

	reqID, err := generateRequestID()
	if err != nil {
		s.logger.Error("failed to generate request ID",
			zap.Error(err),
		)
		writeErr(w, r, dns.RcodeServerFailure)
		return
	}
	logger = logger.With(zap.String("request.ID", reqID))

	if len(r.Question) < 1 {
		logger.Info("refusing to answer because there are no questions")
		writeErr(w, r, dns.RcodeRefused)
		return
	}

	// we only care about the first question
	q := r.Question[0]

	fqdn := dns.Fqdn(q.Name)
	logger = logger.With(zap.String("query.fqdn", fqdn))

	remoteAddr, err := addrToIP(w.RemoteAddr())
	if err != nil {
		logger.Error("failed to convert remote addr to IP",
			zap.Error(err),
		)
		remoteAddr = net.IPv4zero
	}
	logger = logger.With(zap.Stringer("remoteAddr", remoteAddr))

	if q.Qclass != dns.ClassINET {
		logger.Info("refusing to answer non-INET class question",
			zap.String("Qclass", qclassToString(q.Qclass)),
		)
		writeErr(w, r, dns.RcodeRefused)
		return
	}

	if !isValidQtype(q.Qtype) {
		logger.Info("refusing to answer non-A/AAAA type question",
			zap.String("Qtype", qtypeToString(q.Qtype)),
		)
		writeErr(w, r, dns.RcodeRefused)
		return
	}

	if s.blocklist.Contains(fqdn) {
		ans := generateBlockedAnswer(fqdn, q.Qtype, q.Qclass)
		logger.Info("block",
			zap.String("response.answer", ans.String()),
		)
		writeAnswer(w, r, ans)
		return
	}

	nameserver := s.nameservers.Next()
	logger = logger.With(zap.String("nameserver", nameserver))

	uquery := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: r.RecursionDesired,
			Opcode:           dns.OpcodeQuery,
		},
		Question: []dns.Question{
			{
				Name:   fqdn,
				Qtype:  q.Qtype,
				Qclass: q.Qclass,
			},
		},
	}
	ures, _, err := s.exchanger.Exchange(uquery, nameserver)
	if err != nil {
		logger.Error("upstream DNS query failed",
			zap.Error(err),
		)
		writeErr(w, r, dns.RcodeServerFailure)
		return
	}

	if uquery.Id != ures.Id {
		logger.Info("query response ID mismatch",
			zap.Uint16("upstreamQuery.ID", uquery.Id),
			zap.Uint16("upstreamResponse.ID", ures.Id),
		)
		writeErr(w, r, dns.RcodeServerFailure)
		return
	}

	if len(ures.Answer) < 1 {
		// TODO check ures.Rcode and behave accordingly
		logger.Info("no answer in query response")
		writeErr(w, r, dns.RcodeNameError)
		return
	}

	// TODO maybe cache upstream responses
	var answers []dns.RR
	for _, ans := range ures.Answer {
		logger.Info("answer",
			zap.String("response.answer", ans.String()),
		)
		answers = append(answers, ans)
	}

	writeAnswer(w, r, answers...)
}

func generateRequestID() (string, error) {
	const size = 16
	var buf [size]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		return strings.Repeat("00", size), fmt.Errorf("reading random: %w", err)
	}

	return hex.EncodeToString(buf[:]), nil
}

func isValidQtype(qtype uint16) bool {
	switch qtype {
	case dns.TypeA, dns.TypeAAAA:
		return true
	}
	return false
}

func writeAnswer(w dns.ResponseWriter, r *dns.Msg, ans ...dns.RR) error {
	res := &dns.Msg{
		Answer: ans,
	}
	res.SetReply(r)
	return w.WriteMsg(res)
}

func writeErr(w dns.ResponseWriter, r *dns.Msg, code int) error {
	res := &dns.Msg{}
	res.SetRcode(r, dns.RcodeNameError)
	return w.WriteMsg(res)
}

func generateBlockedAnswer(fqdn string, qclass uint16, qtype uint16) dns.RR {
	hdr := dns.RR_Header{
		Name:   fqdn,
		Rrtype: qtype,
		Class:  qclass,
	}

	switch qtype {
	case dns.TypeA:
		return &dns.A{
			Hdr: hdr,
			A:   net.IPv4zero,
		}
	case dns.TypeAAAA:
		return &dns.AAAA{
			Hdr:  hdr,
			AAAA: net.IPv6zero,
		}
	}

	// TODO should this be server error?
	return &dns.A{
		Hdr: hdr,
		A:   net.IPv4zero,
	}
}

func addrToIP(addr net.Addr) (net.IP, error) {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return a.IP, nil
	case *net.TCPAddr:
		return a.IP, nil
	default:
		return nil, fmt.Errorf("unknown addr type: %T", a)
	}
}

func qclassToString(qclass uint16) string {
	qclassStr, ok := dns.ClassToString[qclass]
	if !ok {
		return fmt.Sprintf("unknown<%d>", qclass)
	}
	return qclassStr
}

func qtypeToString(qtype uint16) string {
	qtypeStr, ok := dns.TypeToString[qtype]
	if !ok {
		return fmt.Sprintf("unknown<%d>", qtype)
	}
	return qtypeStr
}
