package balancer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_needDrop(t *testing.T) {
	require.Equal(t, false, needDrop(3, 0, 3))
	require.Equal(t, false, needDrop(3, 1, 3))
	require.Equal(t, false, needDrop(3, 2, 3))
	require.Equal(t, true, needDrop(3, 3, 3))

	require.Equal(t, false, needDrop(9, 0, 3))
	require.Equal(t, false, needDrop(9, 1, 3))
	require.Equal(t, false, needDrop(9, 4, 3))
	require.Equal(t, true, needDrop(9, 5, 3))
	require.Equal(t, true, needDrop(9, 9, 3))

	require.Equal(t, false, needDrop(9, 0, 1))
	require.Equal(t, false, needDrop(9, 9, 1))
}
