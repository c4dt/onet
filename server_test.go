package onet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	bbolt "go.etcd.io/bbolt"
	uuid "gopkg.in/satori/go.uuid.v1"
)

func TestServer_ProtocolRegisterName(t *testing.T) {
	c := NewLocalServer(tSuite, 0)
	defer c.Close()
	plen := len(c.protocols.instantiators)
	require.True(t, plen > 0)
	id, err := c.ProtocolRegister("ServerProtocol", NewServerProtocol)
	log.ErrFatal(err)
	require.NotNil(t, id)
	require.True(t, plen < len(c.protocols.instantiators))
	_, err = c.protocolInstantiate(ProtocolID(uuid.Nil), nil)
	require.NotNil(t, err)
	// Test for not overwriting
	_, err = c.ProtocolRegister("ServerProtocol", NewServerProtocol2)
	require.NotNil(t, err)
}

func TestServer_GetService(t *testing.T) {
	c := NewLocalServer(tSuite, 0)
	defer c.Close()
	s := c.Service("nil")
	require.Nil(t, s)
}

func TestServer_Database(t *testing.T) {
	c := NewLocalServer(tSuite, 0)
	require.NotNil(t, c.serviceManager.db)

	for _, s := range c.serviceManager.availableServices() {
		c.serviceManager.db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(s))
			require.NotNil(t, b)
			return nil
		})
	}
	c.Close()
}

func TestServer_FilterConnectionsOutgoing(t *testing.T) {
	local := NewTCPTest(tSuite)
	defer local.CloseAll()

	srv := local.GenServers(3)
	msg := &SimpleMessage{42}

	// Initially, messages can be sent freely as there is no restriction
	_, err := srv[0].Send(srv[1].ServerIdentity, msg)
	require.NoError(t, err)
	_, err = srv[1].Send(srv[2].ServerIdentity, msg)
	require.NoError(t, err)
	_, err = srv[2].Send(srv[0].ServerIdentity, msg)
	require.NoError(t, err)

	// Set the valid peers of Srv0 to Srv1 only
	validPeers := []*network.ServerIdentity{srv[1].ServerIdentity}
	srv[0].SetValidPeers(validPeers)
	// Now Srv0 can send to Srv1, but not to Srv2
	_, err = srv[0].Send(srv[1].ServerIdentity, msg)
	require.NoError(t, err)
	_, err = srv[0].Send(srv[2].ServerIdentity, msg)
	require.Regexp(t, "rejecting.*invalid peer", err.Error())

	// Add Srv2 to the valid peers of Srv0
	srv[0].SetValidPeers(append(validPeers, srv[2].ServerIdentity))
	// Now Srv0 can send to both Srv1 and Srv2
	_, err = srv[0].Send(srv[1].ServerIdentity, msg)
	require.NoError(t, err)
	_, err = srv[0].Send(srv[2].ServerIdentity, msg)
	require.NoError(t, err)
}

func TestServer_FilterConnectionsIncoming(t *testing.T) {
	local := NewTCPTest(tSuite)
	defer local.CloseAll()

	srv := local.GenServers(3)
	msg := &SimpleMessage{42}

	// Set the valid peers of Srv0 to Srv1
	validPeers0 := []*network.ServerIdentity{srv[1].ServerIdentity}
	srv[0].SetValidPeers(validPeers0)
	// Set the valid peers of Srv1 to Srv2
	validPeers1 := []*network.ServerIdentity{srv[2].ServerIdentity}
	srv[1].SetValidPeers(validPeers1)

	// Now Srv0 can send to Srv1, but Srv1 cannot receive from Srv0
	log.OutputToBuf()
	_, err := srv[0].Send(srv[1].ServerIdentity, msg)
	require.Error(t, err)
	time.Sleep(500 * time.Millisecond)
	log.OutputToOs()

	require.Regexp(t, "rejecting incoming connection.*invalid peer", log.GetStdErr())

	// Set the valid peers of Srv1 to Srv0
	validPeers1 = []*network.ServerIdentity{srv[0].ServerIdentity}
	srv[1].SetValidPeers(validPeers1)

	// Now Srv1 can receive from Srv0
	log.OutputToBuf()
	_, err = srv[0].Send(srv[1].ServerIdentity, msg)
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	log.OutputToOs()

	require.Empty(t, log.GetStdErr())
}

type ServerProtocol struct {
	*TreeNodeInstance
}

// NewExampleHandlers initialises the structure for use in one round
func NewServerProtocol(n *TreeNodeInstance) (ProtocolInstance, error) {
	return &ServerProtocol{n}, nil
}

// NewExampleHandlers initialises the structure for use in one round
func NewServerProtocol2(n *TreeNodeInstance) (ProtocolInstance, error) {
	return &ServerProtocol{n}, nil
}

func (cp *ServerProtocol) Start() error {
	return nil
}
