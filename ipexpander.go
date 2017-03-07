package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/empijei/IPExpander/parsers"
)

func main() {
	os.Exit(nixMain(os.Args[1:]))
}

func nixMain(ipranges []string) (code int) {
	if len(ipranges) == 0 {
		fmt.Println("Please provide at least an ip range in Classless inter-domain routing (CIDR) form")
		//Signal that not enough args were provided
		return 1
	}

	for _, iprange := range ipranges {
		fmt.Fprintln(os.Stderr, "Now printing "+iprange)
		if strings.Contains(iprange, "-") {
			ips, err := parsers.ParseDashed(iprange)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				//Signal that an error happened
				code = 2
				break
			}
			for _, ip := range ips {
				fmt.Println(ip)
			}
		} else {
			ip, ipnet, err := net.ParseCIDR(iprange)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				//Signal that an error happened
				code = 2
				break
			}

			for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
				fmt.Println(ip)
			}
		}
	}
	return
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
