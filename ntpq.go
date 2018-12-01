//
//
package main

import (
	"fmt"
	"log"

	"github.com/beevik/ntp"
	"github.com/pborman/getopt/v2"
)

func asIP(rawip uint32) string {
	a := (rawip >> 24) & 0xff
	b := (rawip >> 16) & 0xff
	c := (rawip >> 8) & 0xff
	d := rawip & 0xff
	return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
}

func main() {
	log.SetPrefix("ntpq: ")
	log.SetFlags(0)

	getopt.SetParameters("SOURCE [SOURCE ...]")
	getopt.Parse()
	args := getopt.Args()
	if len(args) == 0 {
		fmt.Printf("No arguments.\n")
		getopt.Usage()
		return
	}

	for _, host := range args {
		resp, err := ntp.Query(host)
		if err != nil {
			log.Printf("error querying '%s': %s", host, err)
			continue
		}
		err = resp.Validate()
		fmt.Printf("%s:\n", host)
		if err != nil {
			fmt.Printf("  validity problem: %s\n", err)
		}
		fmt.Printf("  Stratum:       %d\n", resp.Stratum)
		fmt.Printf("  Time Source:   %s (%08x)\n", asIP(resp.ReferenceID), resp.ReferenceID)
		fmt.Printf("  Time at xmit:  %s\n", resp.Time)
		fmt.Printf("  Offset:        %s\n", resp.ClockOffset)
		fmt.Printf("  RTT:           %s\n", resp.RTT)
		fmt.Printf("  Precision:     %s\n", resp.Precision)
		fmt.Printf("  MinError:      %s\n", resp.MinError)
		fmt.Printf("  Clock updated: %s\n", resp.ReferenceTime)
		fmt.Printf("  Root distance:   %s\n", resp.RootDistance)
		fmt.Printf("  Root delay:      %s\n", resp.RootDelay)
		fmt.Printf("  Root dispersion: %s\n", resp.RootDispersion)
		if resp.Leap != 0 {
			fmt.Printf("  Leap second marker: %d\n", resp.Leap)
		}
	}
}
