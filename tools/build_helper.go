package main

import (
	"flag"
	"fmt"
	"os"

	"inspur.com/storage/instorage-k8s/pkg/utils"
)

var (
	cfg13000 = flag.Bool("sample-cfg-13000", false, "Dump as13000 configuration sample.")
	cfg18000 = flag.Bool("sample-cfg-18000", false, "Dump as18000 configuration sample.")
	version  = flag.Bool("version", false, "Show the version.")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Version: %s\n", utils.GenerateVersionStr())
		os.Exit(0)
	}

	if *cfg13000 {
		fmt.Print(utils.Dump13000SampleConfig())
		os.Exit(0)
	}

	if *cfg18000 {
		fmt.Print(utils.Dump18000SampleConfig())
		os.Exit(0)
	}
}
