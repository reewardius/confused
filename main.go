/*
Package main implements an automated Dependency Confusion scanner.

Original research provided by Alex Birsan.

Original blog post detailing Dependency Confusion : https://medium.com/@alex.birsan/dependency-confusion-4a5d60fec610 .
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"io/ioutil"
	"net/http"
	"crypto/tls"
)

func main() {
	var resolver PackageResolver
	lang := ""
	verbose := false
	local_file := ""
	url_file := ""
	safespaces := ""
	flag.StringVar(&lang, "l", "npm", "Package repository system. Possible values: \"pip\", \"pipenv\" \"npm\", \"composer\", \"composer-installed\", \"mvn\", \"rubygems\"")
	flag.StringVar(&safespaces, "s", "", "Comma-separated list of known-secure namespaces. Supports wildcards")
	flag.StringVar(&local_file, "f", "", "Local input file")
	flag.StringVar(&url_file, "u", "", "URL input file")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	if (url_file == "" && local_file == "") || (url_file != "" && local_file != "") {
		Help()
		flag.Usage()
		os.Exit(1)
	}

	switch lang {
	case "pip":
		resolver = NewPythonLookup(verbose, lang)
	case "pipenv":
		resolver = NewPythonLookup(verbose, lang)
	case "npm":
		resolver = NewNPMLookup(verbose)
	case "composer":
		resolver = NewComposerLookup(verbose)
	case "mvn":
		resolver = NewMVNLookup(verbose)
	case "rubygems":
		resolver = NewRubyGemsLookup(verbose)
	case "composer-installed":
		resolver = NewComposerInstalledLookup(verbose)
	default:
		fmt.Printf("Unknown package repository system: %s\n", lang)
		os.Exit(1)
	}

	var rawfile []byte
	var err error
	if local_file != "" {
		rawfile, err = ioutil.ReadFile(local_file)
		if err != nil {
			fmt.Printf("Encountered an error while trying to read packages from file: %s\n", err)
			os.Exit(1)
		}
	}
	
	if url_file != "" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

        req, err := http.NewRequest("GET", url_file, nil)
        if err != nil {
			fmt.Printf(" [W] Error when trying to request %s: %s\n", url_file, err)
			os.Exit(1)
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.2535.79")
        resp, err := client.Do(req)
        // if err != nil {
        //         log.Fatalln(err)
        // }

		// resp, err := http.Get(url_file)
		if err != nil {
			fmt.Printf(" [W] Error when trying to request %s: %s\n", url_file, err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			rawfile, err = ioutil.ReadAll(resp.Body)
		}
	}

	err = resolver.ReadPackagesFromFile(rawfile)
	if err != nil {
		fmt.Printf("Encountered an error while trying to read packages from file: %s\n", err)
		os.Exit(1)
	}
	outputPackages := removeSafe(resolver.PackagesNotInPublic(), safespaces)
	PrintResult(outputPackages)
}

// Help outputs tool usage and help
func Help() {
	fmt.Printf("Usage:\n %s [-l LANGUAGENAME] depfilename.ext\n", os.Args[0])
}

// PrintResult outputs the result of the scanner
func PrintResult(notavail []string) {
	if len(notavail) == 0 {
		fmt.Printf("[*] All packages seem to be available in the public repositories. \n\n" +
			"In case your application uses private repositories please make sure that those namespaces in \n" +
			"public repositories are controlled by a trusted party.\n\n")
		return
	}
	fmt.Printf("Issues found, the following packages are not available in public package repositories:\n")
	for _, n := range notavail {
		fmt.Printf(" [!] %s\n", n)
	}
	os.Exit(1)
}

// removeSafe removes known-safe package names from the slice
func removeSafe(packages []string, safespaces string) []string {
	retSlice := []string{}
	safeNamespaces := []string{}
	var ignored bool
	safeTmp := strings.Split(safespaces, ",")
	for _, s := range safeTmp {
		safeNamespaces = append(safeNamespaces, strings.TrimSpace(s))
	}
	for _, p := range packages {
		ignored = false
		for _, s := range safeNamespaces {
			ok, err := filepath.Match(s, p)
			if err != nil {
				fmt.Printf(" [W] Encountered an error while trying to match a known-safe namespace %s : %s\n", s, err)
				continue
			}
			if ok {
				ignored = true
			}
		}
		if !ignored {
			retSlice = append(retSlice, p)
		}
	}
	return retSlice
}
