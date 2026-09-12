package remote

import (
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

func InitScanners(remote *Library) {
	numverifySupplier := suppliers.NewNumverifySupplier()
	ovhSupplier := suppliers.NewOVHSupplier()

	remote.AddScanner(NewLocalScanner())
	remote.AddScanner(NewNumverifyScanner(numverifySupplier))
	remote.AddScanner(NewGoogleSearchScanner())
	remote.AddScanner(NewOVHScanner(ovhSupplier))
	remote.AddScanner(NewGoogleCSEScanner(nil))

	for _, name := range []string{"serpapi", "github", "reddit", "duckduckgo"} {
		remote.AddScanner(newNativeScanner(name))
	}

	remote.LoadPlugins()
}
