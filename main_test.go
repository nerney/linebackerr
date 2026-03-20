package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain_PanicsWhenDefaultPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":6666")
	require.NoError(t, err)
	defer ln.Close()

	require.Panics(t, func() {
		main()
	})
}
