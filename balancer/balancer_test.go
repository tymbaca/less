package balancer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_needDrop(t *testing.T) {
	require.Equal(t, 0, needDrop(3, 0, 3))
	require.Equal(t, 0, needDrop(3, 1, 3))
	require.Equal(t, 0, needDrop(3, 2, 3))
	require.Equal(t, 1, needDrop(3, 3, 3))

	require.Equal(t, 0, needDrop(9, 0, 3))
	require.Equal(t, 0, needDrop(9, 1, 3))
	require.Equal(t, 0, needDrop(9, 4, 3))
	require.Equal(t, 1, needDrop(9, 5, 3))
	require.Equal(t, 5, needDrop(9, 9, 3))

	require.Equal(t, 0, needDrop(9, 0, 1))
	require.Equal(t, 0, needDrop(9, 9, 1))
}
