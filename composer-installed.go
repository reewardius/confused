package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"strings"
)

type ComposerInstalledJSON []struct {
	Name string `json:"name"`
	Require map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

type ComposerInstalledJSONPkg struct {
	Packages ComposerInstalledJSON `json:"packages"`
	DevPackages []string `json:"dev-package-names"`
}

type ComposerInstalledLookup struct {
	Packages []string
	Verbose bool
}

func NewComposerInstalledLookup(verbose bool) PackageResolver {
	return &ComposerInstalledLookup{Packages: []string{}, Verbose: verbose}
}

func (c *ComposerInstalledLookup) ReadPackagesFromFile(rawfile []byte) error {

	var vendorflag bool = true
	data := ComposerInstalledJSON{}
	err := json.Unmarshal([]byte(rawfile), &data)
	if err != nil {
		fmt.Printf("Error reading file as Vendor file\n")
		vendorflag = false
		// return err
	}

	data2 := ComposerInstalledJSONPkg{}
	if !(vendorflag) {
		err2 := json.Unmarshal([]byte(rawfile), &data2)
		if err2 != nil {
			fmt.Printf("Error reading file as generic Composer file\n")
			return err2
		}
	}

	d := ComposerInstalledJSON{}

	if vendorflag {
		d = data
	} else {
		d = data2.Packages
		for i := 0; i < len(data2.DevPackages); i++ {
			c.Packages = append(c.Packages, data2.DevPackages[i])
		}
	}
	fmt.Printf("Successfully read file as generic Composer file\n")

	for i := 0; i < len(d); i++ {

		c.Packages = append(c.Packages, d[i].Name)

		for pkgname := range d[i].Require {
			c.Packages = append(c.Packages, pkgname)
		}

		for pkgname := range d[i].RequireDev {
			c.Packages = append(c.Packages, pkgname)
		}

	}

	return nil
}

func (c *ComposerInstalledLookup) PackagesNotInPublic() []string {
	notavail := []string{}
	for _, pkg := range c.Packages {
		if pkg == "php" {
			continue
		}

		if !c.isAvailableInPublic(pkg, 0) {
			notavail = append(notavail, pkg)
		}
	}

	return notavail
}

func (c *ComposerInstalledLookup) isAvailableInPublic(pkgname string, retry int) bool {
	if retry > 3 {
		fmt.Printf(" [W] Maximum number of retries exhausted for package %s\n", pkgname)

		return false
	}

	// check if the package is specifically a platform package https://getcomposer.org/doc/01-basic-usage.md#platform-packages
	if (strings.HasPrefix(pkgname, "ext-")) {
		return true
	}

	if c.Verbose {
		fmt.Printf("Checking: https://packagist.org/packages/%s : ", pkgname)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("https://packagist.org/packages/" + pkgname)
	if err != nil {
		fmt.Printf(" [W] Error when trying to request https://packagist.org/packages/%s : %s\n", pkgname, err)

		return false
	}
	defer resp.Body.Close()

	if c.Verbose {
		fmt.Printf("%s\n", resp.Status)
	}

	if resp.StatusCode == http.StatusOK {
		return true
	}

	if resp.StatusCode == 429 {
		fmt.Printf(" [!] Server responded with 429 (Too many requests), throttling and retrying..\n")
		time.Sleep(10 * time.Second)
		retry = retry + 1

		c.isAvailableInPublic(pkgname, retry)
	}

	return false
}
