package node

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/cli/internal/test/builders"
	"github.com/stretchr/testify/assert"
)

func TestNodeListErrorOnAPIFailure(t *testing.T) {
	testCases := []struct {
		nodeListFunc  func() ([]swarm.Node, error)
		infoFunc      func() (types.Info, error)
		expectedError string
	}{
		{
			nodeListFunc: func() ([]swarm.Node, error) {
				return []swarm.Node{}, errors.Errorf("error listing nodes")
			},
			expectedError: "error listing nodes",
		},
		{
			nodeListFunc: func() ([]swarm.Node, error) {
				return []swarm.Node{
					{
						ID: "nodeID",
					},
				}, nil
			},
			infoFunc: func() (types.Info, error) {
				return types.Info{}, errors.Errorf("error asking for node info")
			},
			expectedError: "error asking for node info",
		},
	}
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{
			nodeListFunc: tc.nodeListFunc,
			infoFunc:     tc.infoFunc,
		})
		cmd := newListCommand(cli)
		cmd.SetOutput(ioutil.Discard)
		assert.EqualError(t, cmd.Execute(), tc.expectedError)
	}
}

func TestNodeList(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(NodeID("nodeID1"), Hostname("nodeHostname1"), Manager(Leader())),
				*Node(NodeID("nodeID2"), Hostname("nodeHostname2"), Manager()),
				*Node(NodeID("nodeID3"), Hostname("nodeHostname3")),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	})

	cmd := newListCommand(cli)
	assert.NoError(t, cmd.Execute())
	out := cli.OutBuffer().String()
	assert.Contains(t, out, `nodeID1 *           nodeHostname1       Ready               Active              Leader`)
	assert.Contains(t, out, `nodeID2             nodeHostname2       Ready               Active              Reachable`)
	assert.Contains(t, out, `nodeID3             nodeHostname3       Ready               Active`)
}

func TestNodeListQuietShouldOnlyPrintIDs(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(),
			}, nil
		},
	}, buf)
	cmd := newListCommand(cli)
	cmd.Flags().Set("quiet", "true")
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), "nodeID")
}

// Test case for #24090
func TestNodeListContainsHostname(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{}, buf)
	cmd := newListCommand(cli)
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), "HOSTNAME")
}

func TestNodeListDefaultFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(NodeID("nodeID1"), Hostname("nodeHostname1"), Manager(Leader())),
				*Node(NodeID("nodeID2"), Hostname("nodeHostname2"), Manager()),
				*Node(NodeID("nodeID3"), Hostname("nodeHostname3")),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	}, buf)
	cli.SetConfigFile(&configfile.ConfigFile{
		NodesFormat: "{{.ID}}: {{.Hostname}} {{.Status}}/{{.ManagerStatus}}",
	})
	cmd := newListCommand(cli)
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), `nodeID1: nodeHostname1 Ready/Leader`)
	assert.Contains(t, buf.String(), `nodeID2: nodeHostname2 Ready/Reachable`)
	assert.Contains(t, buf.String(), `nodeID3: nodeHostname3 Ready`)
}

func TestNodeListFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCliWithOutput(&fakeClient{
		nodeListFunc: func() ([]swarm.Node, error) {
			return []swarm.Node{
				*Node(NodeID("nodeID1"), Hostname("nodeHostname1"), Manager(Leader())),
				*Node(NodeID("nodeID2"), Hostname("nodeHostname2"), Manager()),
			}, nil
		},
		infoFunc: func() (types.Info, error) {
			return types.Info{
				Swarm: swarm.Info{
					NodeID: "nodeID1",
				},
			}, nil
		},
	}, buf)
	cli.SetConfigFile(&configfile.ConfigFile{
		NodesFormat: "{{.ID}}: {{.Hostname}} {{.Status}}/{{.ManagerStatus}}",
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{.Hostname}}: {{.ManagerStatus}}")
	assert.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), `nodeHostname1: Leader`)
	assert.Contains(t, buf.String(), `nodeHostname2: Reachable`)
}
