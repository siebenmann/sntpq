// sntpq queries one or more (S)NTP servers and prints out the result,
// which includes basic information about what the server itself is
// synchronized to.
//
package main

import (
	"fmt"
	"log"
	"net"

	"github.com/beevik/ntp"
	"github.com/pborman/getopt/v2"
)

// printable returns the length of the printable prefix of a []byte.
func printable(in []byte) int {
	for i, b := range in {
		if b < ' ' || b > 126 {
			return i
		}
	}
	return len(in)
}

func refIDToBytes(refid uint32) []byte {
	n := make([]byte, 4)
	for i, sh := range []uint{24, 16, 8, 0} {
		n[i] = byte((refid >> sh) & 0xff)
	}
	return n
}

func genAddr(b []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
}

// Format a (S)NTP Reference ID as something useful. What these mean
// depends.
//
// For stratum 1 servers, the Reference ID is theoretically up to four
// ASCII characters (0-padded at the end) that describe the clock source.
// For instance, 50505300 is 'PPS'. This is conventionally printed
// as '.PPS.'. See RFC 5905, page 21, for a 2010 list (IANA has no update
// to it; perhaps no new ones have ever been registered).
//
// For NTP servers synchronized to something over IPv4, it is
// conventional for this RefID to be the encoded IP address of their source.
// If they are synchronized over IPv6, well, who knows. We attempt to
// resolve such IPv4 addresses, if possible.
//
// For fuzzy reasons, we always print the decoded nominal IPv4 address,
// even when we know it's meaningless (eg stratum 0 servers). We don't
// attempt DNS resolution on it at stratum 0, though, or if the first
// octet is >= 224 (the start of the multicast range).
//
func formatRefID(refid uint32, stratum uint8) string {
	bts := refIDToBytes(refid)
	addr := genAddr(bts)

	// We don't assume that a stratum 1 Reference ID is entirely
	// printable. Just to be cautious.
	if stratum == 1 {
		mx := printable(bts)
		if mx > 0 {
			return fmt.Sprintf("%s .%s.", addr, bts[:mx])
		}
		return addr
	}

	if bts[0] >= 224 {
		return addr
	}

	r, e := net.LookupAddr(addr)
	if e != nil || len(r) == 0 {
		return addr
	}
	return fmt.Sprintf("%s %s", addr, r[0][:len(r[0])-1])
}

// This badly named function takes what may be an IP address and turns
// it into a hostname if it is an IP address and it can be resolved to
// a hostname. If there are more than one, we pick the first.
func maybeHostname(host string) string {
	if net.ParseIP(host) == nil {
		return ""
	}
	r, e := net.LookupAddr(host)
	if e != nil || len(r) == 0 {
		return ""
	}
	return r[0][:len(r[0])-1]
}

// As a precaution against badly formed NTP server chains, we limit
// how many recursion steps we're willing to make. You might think
// that we could insist on an always-decreasing stratum, but in
// practice this is not the case. Your parent may legitimately change
// stratum (upward) after you last talked to it, leaving you and it
// at the same stratum (or it above you) until you re-contact it.
func reportOn(host string, recurs bool, rlimit int) {
	// We delegate hostname to IP resolution to ntp.Query. It's
	// not clear that this is the right answer (although as a UDP
	// based thing, ntp.Query is unlikely to query multiple IPs),
	// but it is easy.
	resp, err := ntp.Query(host)
	if err != nil {
		log.Printf("error querying '%s': %s", host, err)
		return
	}

	ph := maybeHostname(host)
	if ph != "" {
		fmt.Printf("%s %s:\n", host, ph)
	} else {
		fmt.Printf("%s:\n", host)
	}
	err = resp.Validate()
	if err != nil {
		fmt.Printf("  validity problem: %s\n", err)
	}
	if resp.KissCode != "" {
		// See https://tools.ietf.org/html/rfc5905#section-7.4
		fmt.Printf("  go-away code: '%s'\n", resp.KissCode)
	}
	// We don't print all of the fields, just the ones that Chris
	// Siebenmann considers useful.
	fmt.Printf("  Stratum:         %d\n", resp.Stratum)
	fmt.Printf("  Time Source:     %s (%08x)\n", formatRefID(resp.ReferenceID, resp.Stratum), resp.ReferenceID)
	fmt.Printf("  Time at xmit:    %s\n", resp.Time)
	fmt.Printf("  RTT:             %s\n", resp.RTT)
	fmt.Printf("  Precision:       %s\n", resp.Precision)
	fmt.Printf("  MinError:        %s\n", resp.MinError)
	fmt.Printf("  Clock updated:   %s\n", resp.ReferenceTime)
	fmt.Printf("  Root delay:      %s\n", resp.RootDelay)
	fmt.Printf("  Root dispersion: %s\n", resp.RootDispersion)
	// The local offset is relative to *us*, not the server's
	// offset to anything else, as is the root distance.
	fmt.Printf("  local root distance: %s (via this server)\n", resp.RootDistance)
	fmt.Printf("  local adjustment:    %s (based on this server's time)\n", resp.ClockOffset)
	if resp.Leap != 0 {
		fmt.Printf("  Leap second marker: %d\n", resp.Leap)
	}

	if !recurs || err != nil || rlimit == 0 || resp.Stratum <= 1 {
		return
	}
	bts := refIDToBytes(resp.ReferenceID)
	if bts[0] >= 224 {
		return
	}
	addr := genAddr(bts)
	reportOn(addr, recurs, rlimit - 1)
}

func main() {
	var help, recurs bool
	log.SetPrefix("ntpq: ")
	log.SetFlags(0)

	getopt.FlagLong(&help, "help", 'h', "Print this help")
	getopt.FlagLong(&recurs, "follow", 'f', "Attempt to follow the chain of time sources for each command line source.")
	getopt.SetParameters("SOURCE [SOURCE ...]")
	getopt.Parse()

	args := getopt.Args()
	if help {
		getopt.Usage()
		return
	}
	if len(args) == 0 {
		fmt.Printf("No arguments.\n")
		getopt.Usage()
		return
	}

	for _, host := range args {
		reportOn(host, recurs, 15)
	}
}
