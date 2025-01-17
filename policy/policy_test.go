package policy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Create(t *testing.T) {
	// check if all variables are set
	// check default variables

	policy, err := create(strings.NewReader(""))
	assert.NoError(t, err)
	assert.EqualValues(t, &Policy{
		ReserveOnchainMsat: defaultReserveOnchainMsat,
		PeerAllowlist:      defaultPeerAllowlist,
		SuspiciousPeerList: defaultSuspiciousPeerList,
		AcceptAllPeers:     defaultAcceptAllPeers,
	}, policy)

	peer1 := "123"
	peer2 := "345"
	accept := true
	var acceptInt int8
	if accept {
		acceptInt = 1
	}

	conf := fmt.Sprintf(
		"accept_all_peers=%d\n"+
			"allowlisted_peers=%s\n"+
			"allowlisted_peers=%s\n"+
			"suspicious_peers=%s\n"+
			"suspicious_peers=%s",
		acceptInt,
		peer1,
		peer2,
		peer1,
		peer2,
	)

	policy2, err := create(strings.NewReader(conf))
	assert.NoError(t, err)
	assert.EqualValues(t, &Policy{
		ReserveOnchainMsat: defaultReserveOnchainMsat,
		PeerAllowlist:      []string{peer1, peer2},
		SuspiciousPeerList: []string{peer1, peer2},
		AcceptAllPeers:     accept,
	}, policy2)
}

func Test_Reload(t *testing.T) {
	peer1 := "123"
	peer2 := "345"
	accept := true
	var acceptInt int8
	if accept {
		acceptInt = 1
	}

	conf := fmt.Sprintf("accept_all_peers=%d\nallowlisted_peers=%s\nallowlisted_peers=%s", acceptInt, peer1, peer2)

	policy, err := create(strings.NewReader(conf))
	assert.NoError(t, err)
	assert.EqualValues(t, &Policy{
		ReserveOnchainMsat: defaultReserveOnchainMsat,
		PeerAllowlist:      []string{peer1, peer2},
		SuspiciousPeerList: defaultSuspiciousPeerList,
		AcceptAllPeers:     accept,
	}, policy)

	newPeer := "new_peer"
	newConf := fmt.Sprintf("allowlisted_peers=%s\nsuspicious_peers=%s", newPeer, newPeer)

	err = policy.reload(strings.NewReader(newConf))
	assert.NoError(t, err)
	assert.EqualValues(t, &Policy{
		ReserveOnchainMsat: defaultReserveOnchainMsat,
		PeerAllowlist:      []string{newPeer},
		SuspiciousPeerList: []string{newPeer},
		AcceptAllPeers:     defaultAcceptAllPeers,
	}, policy)
}

func Test_Reload_NoOverrideOnError(t *testing.T) {
	peer1 := "123"
	peer2 := "345"
	accept := true
	var acceptInt int8
	if accept {
		acceptInt = 1
	}

	conf := fmt.Sprintf("accept_all_peers=%d\nallowlisted_peers=%s\nallowlisted_peers=%s", acceptInt, peer1, peer2)

	policy, err := create(strings.NewReader(conf))
	assert.NoError(t, err)
	assert.EqualValues(t, &Policy{
		ReserveOnchainMsat: defaultReserveOnchainMsat,
		PeerAllowlist:      []string{peer1, peer2},
		SuspiciousPeerList: defaultSuspiciousPeerList,
		AcceptAllPeers:     accept,
	}, policy)

	// copy policy
	oldPolicy := &Policy{}
	*oldPolicy = *policy

	// Malformed config string
	newConf := "this_is_unknown:3"

	err = policy.reload(strings.NewReader(newConf))
	assert.Error(t, err)

	// assert policy did not change
	assert.EqualValues(t, oldPolicy, policy)
}

func Test_AddRemovePeer_Runtime(t *testing.T) {
	policyFilePath := path.Join(t.TempDir(), "policy.conf")
	file, err := os.Create(policyFilePath)
	assert.NoError(t, err)

	err = file.Close()
	assert.NoError(t, err)

	policy, err := CreateFromFile(policyFilePath)
	assert.NoError(t, err)

	err = policy.AddToAllowlist("foo")
	assert.NoError(t, err)
	err = policy.AddToSuspiciousPeerList("bar")
	assert.NoError(t, err)

	policyFile, err := ioutil.ReadFile(policyFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "allowlisted_peers=foo\nsuspicious_peers=bar\n", string(policyFile))

	err = policy.AddToAllowlist("foo2")
	assert.NoError(t, err)
	err = policy.RemoveFromAllowlist("foo")
	assert.NoError(t, err)

	err = policy.AddToSuspiciousPeerList("bar2")
	assert.NoError(t, err)
	err = policy.RemoveFromSuspiciousPeerList("bar")
	assert.NoError(t, err)

	policyFile, err = ioutil.ReadFile(policyFilePath)
	assert.NoError(t, err)
	assert.Equal(t, "allowlisted_peers=foo2\nsuspicious_peers=bar2\n", string(policyFile))
}

func Test_AddRemovePeer_Runtime_ConcurrentWrite(t *testing.T) {
	const N_CONC_W = 500

	policyFilePath := path.Join(t.TempDir(), "policy.conf")
	file, err := os.Create(policyFilePath)
	if err != nil {
		t.Fatalf("Failed Create(): %v", err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("Failed Close(): %v", err)
	}

	policy, err := CreateFromFile(policyFilePath)
	assert.NoError(t, err)

	var expectedPeers []string
	for i := 0; i < N_CONC_W; i++ {
		expectedPeers = append(expectedPeers, fmt.Sprintf("foo%d", i))
	}

	wg := &sync.WaitGroup{}
	wg.Add(2 * N_CONC_W)
	for i := 0; i < N_CONC_W; i++ {
		go func(n int) {
			ierr := policy.AddToSuspiciousPeerList(fmt.Sprintf("foo%d", n))
			assert.NoError(t, ierr)
			wg.Done()
		}(i)
		go func(n int) {
			ierr := policy.AddToAllowlist(fmt.Sprintf("foo%d", n))
			assert.NoError(t, ierr)
			wg.Done()
		}(i)

	}

	wg.Wait()

	assert.ElementsMatch(t, expectedPeers, policy.PeerAllowlist)
	assert.ElementsMatch(t, expectedPeers, policy.SuspiciousPeerList)

}

func Test_IsPeerAllowed_Allowlist(t *testing.T) {
	// is on allowlist and not
	peer1 := "peer1"
	peer2 := "peer2"

	// peer1 is allowlisted, peer2 not
	conf := fmt.Sprintf("allowlisted_peers=%s", peer1)

	policy, err := create(strings.NewReader(conf))
	assert.NoError(t, err)
	assert.True(t, policy.IsPeerAllowed(peer1))
	assert.False(t, policy.IsPeerAllowed(peer2))

	// accept all peers

}

func Test_IsPeerAllowed_AcceptAll(t *testing.T) {
	// all incomming requests should be allowed
	conf := "accept_all_peers=1"

	policy, err := create(strings.NewReader(conf))
	assert.NoError(t, err)
	assert.True(t, policy.IsPeerAllowed("some_peer"))
	assert.True(t, policy.IsPeerAllowed("some_other_peer"))

	// accept all peers

}

func Test_CreateFile(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "peerswap.conf")

	policy, err := CreateFromFile(confPath)
	require.NoError(t, err)

	fileInfo, err := os.Stat(confPath)
	require.NoError(t, err)

	assert.Equal(t, "peerswap.conf", fileInfo.Name())

	err = policy.AddToAllowlist("test123")
	require.NoError(t, err)
}
