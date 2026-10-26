package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

type record struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	CountryCode string `maxminddb:"country_code"`

	ASN    uint   `maxminddb:"autonomous_system_number"`
	ASOrg  string `maxminddb:"autonomous_system_organization"`
	Traits struct {
		ASN   uint   `maxminddb:"autonomous_system_number"`
		ASOrg string `maxminddb:"autonomous_system_organization"`
	} `maxminddb:"traits"`
}

func (r *record) country() string {
	if r.Country.ISOCode != "" {
		return r.Country.ISOCode
	}
	return r.CountryCode
}

func (r *record) asn() (uint, string) {
	if r.ASN != 0 {
		return r.ASN, r.ASOrg
	}
	return r.Traits.ASN, r.Traits.ASOrg
}

func sanitize(s string) string {
	s = strings.NewReplacer("|", " ", "\n", " ", "\r", " ", "\t", " ").Replace(s)
	return strings.TrimSpace(s)
}

func main() {
	dbPath := flag.String("db", "", "path to the source .mmdb file")
	mode := flag.String("mode", "", "country | asn")
	outPath := flag.String("out", "-", "output map path (- for stdout)")
	flag.Parse()

	if *dbPath == "" || (*mode != "country" && *mode != "asn") {
		fmt.Fprintln(os.Stderr, "usage: geoip2map -db FILE.mmdb -mode country|asn [-out FILE.map]")
		os.Exit(2)
	}

	db, err := maxminddb.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", *dbPath, err)
		os.Exit(1)
	}
	defer db.Close()

	out := os.Stdout
	if *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}
	w := bufio.NewWriter(out)
	defer w.Flush()

	var rec record
	written, skipped := 0, 0

	nets := db.Networks(maxminddb.SkipAliasedNetworks)
	for nets.Next() {
		rec = record{}
		subnet, err := nets.Network(&rec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decode: %v\n", err)
			os.Exit(1)
		}

		var value string
		switch *mode {
		case "country":
			value = rec.country()
		case "asn":
			num, org := rec.asn()
			switch {
			case num == 0:
				value = ""
			case org == "":
				value = fmt.Sprintf("%d", num)
			default:
				value = fmt.Sprintf("%d|%s", num, sanitize(org))
			}
		}

		if value == "" {
			skipped++
			continue
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", subnet.String(), value); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		written++
	}
	if err := nets.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "iterate: %v\n", err)
		os.Exit(1)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "flush: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%s map: wrote %d entries, skipped %d empty\n", *mode, written, skipped)
}
