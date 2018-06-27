package pb

import (
	"math/rand"
	"path"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/require"
	"gopkg.in/bblfsh/sdk.v1/uast"
)

func TestMarshallingRoundTrip(t *testing.T) {
	require := require.New(t)

	f := func(ff *FileFixture) bool {
		d, err := ff.File.Marshal()
		if err != nil {
			return false
		}

		of := &File{}
		err = of.Unmarshal(d)
		return err == nil
	}

	require.NoError(quick.Check(f, nil))
}

type FileFixture struct {
	File *File
}

var ValidGitModes = []uint32{
	100644,
	100755,
	120000,
}

func (f *FileFixture) Generate(rand *rand.Rand, size int) reflect.Value {
	ff := &FileFixture{}
	ff.File = randomFile(rand, size)
	return reflect.ValueOf(ff)
}

func randomFile(rand *rand.Rand, size int) *File {
	f := &File{}
	f.Mode = randomValidGitMode(rand)
	f.Path = randomValidPath(rand)
	f.Content = randomBytes(rand, size)
	f.UAST = &uast.Node{
		InternalType: "TEST",
	}
	return f
}

func randomValidGitMode(rand *rand.Rand) uint32 {
	n := rand.Intn(3)
	return ValidGitModes[n]
}

func randomValidPath(rand *rand.Rand) string {
	n := rand.Intn(50) + 1
	parts := make([]string, n)
	for i := range parts {
		parts[i] = randomValidPathPart(rand)
	}

	return path.Join(parts...)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_$á")

func randomValidPathPart(rand *rand.Rand) string {
	n := rand.Intn(20) + 1
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func randomBytes(rand *rand.Rand, size int) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}

	return b
}
