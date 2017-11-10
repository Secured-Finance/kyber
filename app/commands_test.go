package app

import (
	"testing"

	"os"

	"bytes"

	"io"

	"github.com/stretchr/testify/require"
	"github.com/dedis/kyber/group/edwards25519"
	"github.com/dedis/onet/log"
)

var suite = edwards25519.NewAES128SHA256Ed25519()

func TestCothority(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	log.ErrFatal(err)
	os.Stdout = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		log.ErrFatal(err)
		outC <- buf.String()
	}()

	os.Args = []string{os.Args[0], "help"}
	Server(suite)
	// back to normal state
	log.ErrFatal(w.Close())
	require.Contains(t, <-outC, "Serve a cothority")

	os.Stdout = origStdout
}
