package main

//go:generate make -C .. -f lamp/build/protobuf.mk
//go:generate make storedeps

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"

	"github.com/lampdb/lamp/cli"
	"github.com/lampdb/lamp/util/log"
	"github.com/lampdb/lamp/util/randutil"
)

func main() {
	// Instruct Go to use all CPU cores.
	// TODO(spencer): this may be excessive and result in worse
	// performance. We should keep an eye on this as we move to
	// production workloads.
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	rand.Seed(randutil.NewPseudoSeed())
	if log.V(1) {
		log.Infof("running using %d processor cores", numCPU)
	}

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "help")
	}
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Failed running command %q: %v\n", os.Args[1:], err)
		os.Exit(1)
	}
}
