/**
 * File        : client.go
 * Description : High-level client interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket"
	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket/keyspace"
	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	"gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"

	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/streamstore"
	"github.com/enzoh/go-logging"
	"github.com/hashicorp/golang-lru"
)

// Config configures for the client.
type Config struct {
	AnalyticsInterval           time.Duration
	AnalyticsURL                string
	AnalyticsUserData           string
	ArtifactCacheSize           int
	ArtifactChunkSize           uint32
	ArtifactMaxBufferSize       uint32
	ArtifactQueueSize           int
	ClusterID                   int
	DisableAnalytics            bool
	DisableBroadcast            bool
	DisableNATPortMap           bool
	DisablePeerDiscovery        bool
	DisableStreamDiscovery      bool
	IP                          string
	KBucketSize                 int
	LatencyTolerance            time.Duration
	LogLevel                    string
	NATMonitorInterval          time.Duration
	NATMonitorTimeout           time.Duration
	Network                     string
	PingBufferSize              uint32
	Port                        uint16
	ProcessID                   int
	ProofBufferSize             uint32
	RandomSeed                  string
	SampleMaxBufferSize         uint32
	SampleSize                  int
	SeedNodes                   []string
	StreamstoreIncomingCapacity int
	StreamstoreOutgoingCapacity int
	StreamstoreQueueSize        int
	Timeout                     time.Duration
	VerificationBufferSize      uint32
	Version                     string
	WitnessCacheSize            int
}

// DefaultConfig -- Get the default configuration parameters.
func DefaultConfig() *Config {

	return &Config{
		AnalyticsInterval:      time.Minute,
		AnalyticsURL:           "https://analytics.dfinity.build/report",
		AnalyticsUserData:      "",
		ArtifactCacheSize:      65536,
		ArtifactChunkSize:      65536,
		ArtifactMaxBufferSize:  16777216,
		ArtifactQueueSize:      8192,
		ClusterID:              0,
		DisableAnalytics:       false,
		DisableBroadcast:       false,
		DisableNATPortMap:      false,
		DisablePeerDiscovery:   false,
		DisableStreamDiscovery: false,
		IP:                          "0.0.0.0",
		KBucketSize:                 16,
		LatencyTolerance:            time.Minute,
		LogLevel:                    "info",
		NATMonitorInterval:          time.Second,
		NATMonitorTimeout:           time.Minute,
		Network:                     "revolver",
		PingBufferSize:              32,
		Port:                        0,
		ProcessID:                   0,
		ProofBufferSize:             0,
		RandomSeed:                  "",
		SampleMaxBufferSize:         8192,
		SampleSize:                  4,
		SeedNodes:                   nil,
		StreamstoreIncomingCapacity: 64,
		StreamstoreOutgoingCapacity: 8,
		StreamstoreQueueSize:        8192,
		Timeout:                     10 * time.Second,
		VerificationBufferSize:      0,
		Version:                     "0.1.0",
		WitnessCacheSize:            65536,
	}

}

func (config *Config) validate() error {

	// The analytics interval must be a positive time duration.
	if config.AnalyticsInterval <= 0 {
		return fmt.Errorf("Invalid analytics interval: %d", config.AnalyticsInterval)
	}

	// The analytics URL must be parsable.
	_, err := url.Parse(config.AnalyticsURL)
	if err != nil {
		return fmt.Errorf("Invalid analytics URL: %s", config.AnalyticsURL)
	}

	// The artifact cache size must be a positive integer.
	if config.ArtifactCacheSize <= 0 {
		return fmt.Errorf("Invalid artifact cache size: %d", config.ArtifactCacheSize)
	}

	// The artifact chunk size must be a non-zero unsigned 32-bit integer.
	if config.ArtifactChunkSize == 0 {
		return errors.New("Invalid artifact chunk size: 0")
	}

	// The artifact max buffer size must be a non-zero unsigned 32-bit integer.
	if config.ArtifactMaxBufferSize == 0 {
		return errors.New("Invalid artifact max buffer size: 0")
	}

	// The artifact queue size must be a positive integer.
	if config.ArtifactQueueSize <= 0 {
		return fmt.Errorf("Invalid artifact queue size: %d", config.ArtifactQueueSize)
	}

	// The IP address must be parsable.
	if net.ParseIP(config.IP) == nil {
		return fmt.Errorf("Invalid IP address: %s", config.IP)
	}

	// The Kademlia bucket size must be a positive integer.
	if config.KBucketSize <= 0 {
		return fmt.Errorf("Invalid Kademlia bucket size: %d", config.KBucketSize)
	}

	// The latency tolerance must be a positive time duration.
	if config.LatencyTolerance <= 0 {
		return fmt.Errorf("Invalid latency tolerance: %d", config.LatencyTolerance)
	}

	// The log level must be recognizable.
	_, err = logging.LogLevel(config.LogLevel)
	if err != nil {
		return fmt.Errorf("Invalid log level: %s", config.LogLevel)
	}

	// The NAT monitor interval must be a positive time duration.
	if config.NATMonitorInterval <= 0 {
		return fmt.Errorf("Invalid NAT monitor interval: %d", config.NATMonitorInterval)
	}

	// The NAT monitor timeout must be a positive time duration.
	if config.NATMonitorTimeout <= 0 {
		return fmt.Errorf("Invalid NAT monitor timeout: %d", config.NATMonitorTimeout)
	}

	// The ping buffer size must be a non-zero unsigned 32-bit integer.
	if config.PingBufferSize == 0 {
		return errors.New("Invalid ping buffer size: 0")
	}

	// The random seed must be a zero or 32-byte hex-encoded string.
	_, err = hex.DecodeString(config.RandomSeed)
	if len(config.RandomSeed) != 0 && len(config.RandomSeed) != 64 || err != nil {
		return fmt.Errorf("Invalid random seed: %s", config.RandomSeed)
	}

	// The peer sample max buffer size must be a non-zero unsigned 32-bit integer.
	if config.SampleMaxBufferSize == 0 {
		return errors.New("Invalid peer sample max buffer size: 0")
	}

	// The peer sample size must be a positive integer.
	if config.SampleSize <= 0 {
		return fmt.Errorf("Invalid peer sample size: %d", config.SampleSize)
	}

	// The seed nodes must be parsable.
	for i := range config.SeedNodes {
		_, err = multiaddr.NewMultiaddr(config.SeedNodes[i])
		if err != nil {
			return fmt.Errorf("Invalid seed node: %s", config.SeedNodes[i])
		}
	}

	// The stream store incoming capacity must be a positive integer.
	if config.StreamstoreIncomingCapacity <= 0 {
		return fmt.Errorf("Invalid stream store incoming capacity: %d", config.StreamstoreIncomingCapacity)
	}

	// The stream store outgoing capacity must be a positive integer.
	if config.StreamstoreOutgoingCapacity <= 0 {
		return fmt.Errorf("Invalid stream store incoming capacity: %d", config.StreamstoreOutgoingCapacity)
	}

	// The stream store transaction queue size must be a positive integer.
	if config.StreamstoreQueueSize <= 0 {
		return fmt.Errorf("Invalid stream store transaction queue size: %d", config.StreamstoreQueueSize)
	}

	// The stream timeout must be a positive time duration.
	if config.Timeout <= 0 {
		return fmt.Errorf("Invalid stream timeout: %d", config.Timeout)
	}

	// The witness cache size must be a positive integer.
	if config.WitnessCacheSize <= 0 {
		return fmt.Errorf("Invalid witness cache size: %d", config.WitnessCacheSize)
	}

	return nil

}

// Client interacts with the network.
type Client interface {

	// List the addresses.
	Addresses() []string

	// Get the ID.
	ID() string

	// Get the peer count.
	PeerCount() int

	// Get the stream count.
	StreamCount() int

	// Send an artifact.
	Send(artifact artifact.Artifact)

	// Receive an artifact.
	Receive() artifact.Artifact

	// Request an artifact.
	Request(checksum [32]byte) (artifact.Artifact, error)

	// Register an artifact request handler.
	SetArtifactHandler(handler ArtifactHandler)

	// Register a zero-knowledge proof request handler.
	SetProofHandler(handler ProofHandler)

	// Register a verification request handler.
	SetVerificationHandler(handler VerificationHandler)
}

// ArtifactHandler -- This type represents a function that executes when
// receiving an artifact request. The function can be registered as a callback
// using SetArtifactHandler.
type ArtifactHandler func(checksum [32]byte, response chan artifact.Artifact)

// ProofHandler -- This type represents a function that executes when receiving
// a zero-knowledge proof request. The function can be registered as a callback
// using SetProofHandler.
type ProofHandler func(data []byte, response chan []byte)

// VerificationHandler -- This type represents a function that executes when
// receiving a verification request. The function can be registered as a
// callback using SetVerificationHandler.
type VerificationHandler func(data []byte, response chan bool)

// Addresses -- List the addresses.
func (client *client) Addresses() []string {
	addrs := client.host.Addrs()
	result := make([]string, len(addrs))
	for i := range result {
		result[i] = addrs[i].String()
	}
	return result
}

// ID -- Get the ID.
func (client *client) ID() string {
	return client.id.Pretty()
}

// PeerCount -- Get the peer count.
func (client *client) PeerCount() int {
	return client.table.Size()
}

// StreamCount -- Get the stream count.
func (client *client) StreamCount() int {
	return client.streamstore.IncomingSize() + client.streamstore.OutgoingSize()
}

// Send -- Send an artifact.
func (client *client) Send(artifact artifact.Artifact) {
	client.send <- artifact
}

// Receive -- Receive an artifact.
func (client *client) Receive() artifact.Artifact {
	return <-client.receive
}

// Request -- Request an artifact.
func (client *client) Request(checksum [32]byte) (artifact.Artifact, error) {
	return nil, errors.New("TODO: Implement request method.")
}

// SetArtifactHandler -- Register an artifact request handler.
func (client *client) SetArtifactHandler(handler ArtifactHandler) {

	notify := make(chan struct{})

	client.unsetArtifactHandlerLock.Lock()
	client.unsetArtifactHandler()
	client.unsetArtifactHandler = func() {
		close(notify)
	}
	client.unsetArtifactHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.artifactRequests:
				handler(request.checksum, request.response)
			}
		}
	}()

}

// SetProofHandler -- Register a zero-knowledge proof request handler.
func (client *client) SetProofHandler(handler ProofHandler) {

	notify := make(chan struct{})

	client.unsetProofHandlerLock.Lock()
	client.unsetProofHandler()
	client.unsetProofHandler = func() {
		close(notify)
	}
	client.unsetProofHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.proofRequests:
				handler(request.data, request.response)
			}
		}
	}()

}

// SetVerificationHandler -- Register a verification request handler.
func (client *client) SetVerificationHandler(handler VerificationHandler) {

	notify := make(chan struct{})

	client.unsetVerificationHandlerLock.Lock()
	client.unsetVerificationHandler()
	client.unsetVerificationHandler = func() {
		close(notify)
	}
	client.unsetVerificationHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.verificationRequests:
				handler(request.data, request.response)
			}
		}
	}()

}

// New -- Create a client.
func (config *Config) New() (Client, func(), error) {
	return config.create()
}

type client struct {
	artifactCache     *lru.Cache
	artifactCacheLock *sync.Mutex
	artifactRequests  chan struct {
		checksum [32]byte
		response chan artifact.Artifact
	}
	config        *Config
	context       context.Context
	host          *basichost.BasicHost
	id            peer.ID
	key           keyspace.Key
	logger        *logging.Logger
	peerstore     peerstore.Peerstore
	proofRequests chan struct {
		data     []byte
		response chan []byte
	}
	protocol                     protocol.ID
	receive                      chan artifact.Artifact
	send                         chan artifact.Artifact
	streamstore                  streamstore.Streamstore
	table                        *kbucket.RoutingTable
	unsetArtifactHandler         func()
	unsetArtifactHandlerLock     *sync.Mutex
	unsetProofHandler            func()
	unsetProofHandlerLock        *sync.Mutex
	unsetVerificationHandler     func()
	unsetVerificationHandlerLock *sync.Mutex
	verificationRequests         chan struct {
		data     []byte
		response chan bool
	}
	witnessCache     *lru.Cache
	witnessCacheLock *sync.Mutex
}

func (config *Config) create() (*client, func(), error) {

	// Validate the configuration.
	err := config.validate()
	if err != nil {
		return nil, nil, err
	}

	// Create a client from the configuration.
	client := &client{}
	client.config = &Config{}
	*client.config = *config

	// Create an artifact cache.
	client.artifactCache, err = lru.New(client.config.ArtifactCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.artifactCacheLock = &sync.Mutex{}

	// Create a context.
	client.context = context.Background()

	// Create or decode the random seed.
	seed := make([]byte, 32)
	if len(client.config.RandomSeed) == 0 {
		_, err = rand.Read(seed)
	} else {
		seed, err = hex.DecodeString(client.config.RandomSeed)
	}
	if err != nil {
		return nil, nil, err
	}

	// Create an Ed25519 key pair from the random seed.
	secretKey, publicKey, err := crypto.GenerateEd25519Key(
		bytes.NewReader(seed),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create an identity from the public key.
	client.id, err = peer.IDFromPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}

	// Create a key in the Kademlia key space.
	client.key = keyspace.XORKeySpace.Key(kbucket.ConvertPeerID(client.id))

	// Create a logger.
	client.logger = logging.MustGetLogger("p2p")

	// Create a peer store.
	client.peerstore = peerstore.NewPeerstore()
	client.peerstore.AddPrivKey(client.id, secretKey)
	client.peerstore.AddPubKey(client.id, publicKey)

	// Create a protocol.
	client.protocol = protocol.ID(
		fmt.Sprintf(
			"/%s/%s",
			client.config.Network,
			client.config.Version,
		),
	)

	// Create the broadcast queues.
	client.send = make(chan artifact.Artifact, client.config.ArtifactQueueSize)
	client.receive = make(chan artifact.Artifact, client.config.ArtifactQueueSize)

	// Create the request queues.
	client.artifactRequests = make(chan struct {
		checksum [32]byte
		response chan artifact.Artifact
	}, client.config.ArtifactQueueSize)
	client.proofRequests = make(chan struct {
		data     []byte
		response chan []byte
	}, 1)
	client.verificationRequests = make(chan struct {
		data     []byte
		response chan bool
	}, 1)

	// Create a stream store.
	client.streamstore = streamstore.New(
		client.config.StreamstoreIncomingCapacity,
		client.config.StreamstoreOutgoingCapacity,
		client.config.StreamstoreQueueSize,
	)

	// Create a routing table.
	client.table = kbucket.NewRoutingTable(
		client.config.KBucketSize,
		client.key.Bytes,
		client.config.LatencyTolerance,
		client.peerstore,
	)

	// Add the client to its routing table.
	client.table.Update(client.id)

	// Create a witness cache.
	client.witnessCache, err = lru.New(client.config.WitnessCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.witnessCacheLock = &sync.Mutex{}

	// Initialize the handler deregistration functions.
	client.unsetArtifactHandler = func() {}
	client.unsetArtifactHandlerLock = &sync.Mutex{}
	client.unsetProofHandler = func() {}
	client.unsetProofHandlerLock = &sync.Mutex{}
	client.unsetVerificationHandler = func() {}
	client.unsetVerificationHandlerLock = &sync.Mutex{}

	// Start the client.
	shutdown, err := client.bootstrap()
	if err != nil {
		return nil, nil, err
	}

	// Ready for action.
	return client, shutdown, nil

}
