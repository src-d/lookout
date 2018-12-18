// satisfies go-bindata interface
// this file is replaced by auto generated code for production usage
// we use it instead of go-bindata -dev because it's easier to write wrapper
// than change webpack configuration in create-react-app _O.O_

package assets

import (
	"fmt"
	"io/ioutil"
	"os"
)

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	return ioutil.ReadFile(name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	b, err := Asset(name)
	if err != nil {
		panic(fmt.Errorf("can't read file %s: %s", name, err))
	}
	return b
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}
