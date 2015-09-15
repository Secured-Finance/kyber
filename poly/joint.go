package poly

import (
	"errors"
	"fmt"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/config"
)

// This package provides  a dealer-less distributed verifiable secret sharing using Pedersen VSS scheme
// as explained in "Provably Secure Distributed Schnorr Signatures and a (t, n) Threshold Scheme for Implicit Certificates"
// This file is only responsible for the setup of a shared secret among n peers.
// The output is a global public polynomial (pubPoly) and a secret share for each peers.

// PolyInfo describe the information needed to construct (and verify) a matrixShare
type PolyInfo struct {
	// Group used for working on the polynomials
	Suite abstract.Suite
	// How many peer do we need to reconstruct a secret
	T int
	// How many peers do we need to verify
	R int
	// How many peers are collaborating into constructing the shared secret ( i.e. MatrixShare is of size NxN)
	N int
}

// Represent the output of a VSS Pedersen scheme : a global public polynomial and a share of its related priv poly
// for a peer
type SharedSecret struct {

	// The shared public polynomial
	Pub *PubPoly

	// The share of the shared secret
	Share *abstract.Secret

	// The index of this share regarding the secret private poly / pub poly
	// i.e. it is the same as the receiver's index
	Index int
}

// Dealer is a peer that will create a promise and distribute it to each receivers needed
type Dealer struct {

	// Info about the polynomials config used
	info PolyInfo

	// Promise is the promise of peer j
	Promise *Promise

	// State related to peer j 's promise
	State *State
}

// Receiver Part : Receiver struct is basically the underlying structure of the general matrix.
// If a peer is a receiver, it will receive all promises and compute all of its share and then he will
// be able to generate the SharedSecret
type Receiver struct {
	// info is just the info about the polynomials we're gonna use
	info PolyInfo

	// This index is the index used by the dealers to make the share for this receiver
	// For a given receiver, It should be the same for every dealers /!!\
	index int

	// the Receiver private / public key combination
	// it may or may not have to be the long term key of the node
	Key *config.KeyPair

	// List of Dealers. Be careful : this receiver should have the SAME index for all the Dealer's promises !!
	// otherwise we wouldn't know which index to chose from the shared public polynomial
	Dealers []*Dealer

	// When the dealers are all done, we can compute the shared secret which consists of a
	// 1. Public Polynomial which is basically the sums of all Dealers's polynomial
	// 2. Share of the global Private Polynomial (which is to never be computed directly), which is
	// 		basically SUM of fj(i) for a receiver i
	Secret SharedSecret
}

// NewDealer returns a newly created & intialized Dealer struct
func NewDealer(info PolyInfo, secret, promiser *config.KeyPair, receiverList []abstract.Point) *Dealer {
	return new(Dealer).Init(info, secret, promiser, receiverList)
}

// Dealer.Init inits a new Dealer structure :
// That basically create the promise of the dealer and the respective shares using the list of receivers
func (d *Dealer) Init(info PolyInfo, secret, promiser *config.KeyPair, receiverList []abstract.Point) *Dealer {
	d.info = info
	d.Promise = new(Promise).ConstructPromise(secret, promiser, info.T, info.R, receiverList)
	d.State = new(State).Init(*d.Promise)
	return d
}

// Basically a wrapper around Promise / Response so that a dealer can verify that all its receiver correctly received its promise and are not cheating
func (d *Dealer) AddResponse(i int, response *Response) error {
	return d.State.AddResponse(i, response)
}

// A wrapper around State.PromiseCertified for this dealer. It must have received enough Response (and/or max number of blameProof)
func (d *Dealer) Certified() error {
	return d.State.PromiseCertified()
}

func NewReceiver(info PolyInfo, key *config.KeyPair) *Receiver {
	return new(Receiver).Init(info, key)
}

// Init a new Receiver struct
// info is the info about the structure of the polynomials used
// key is the long-term public key of the receiver
func (r *Receiver) Init(info PolyInfo, key *config.KeyPair) *Receiver {
	r.index = -1 // no dealer received yet
	r.info = info
	r.Key = key
	r.Dealers = make([]*Dealer, 0, info.N)
	return r
}

// AddDealer adds a dealer to the array of dealers the receiver already has.
// You must give the index of the receiver in the promise of the dealer, and the dealer struct
// It will return a Response to be sent back to the Dealer so he can verify its promise
func (r *Receiver) AddDealer(index int, dealer *Dealer) (*Response, error) {
	if r.index == -1 {
		r.index = index
	}
	if r.index != index {
		return nil, errors.New(fmt.Sprintf("Wrong index received for receiver : %d instead of %d", index, r.index))
	}
	// produce response
	resp, err := dealer.Promise.ProduceResponse(index, r.Key)
	r.Dealers = append(r.Dealers, dealer)
	return resp, err
}

// ProduceSharedSecret will generate the sharedsecret relative to this receiver
// it will throw an error if something is wrong such as not enough Dealers received
func (r *Receiver) ProduceSharedSecret() (*SharedSecret, error) {
	if len(r.Dealers) < 1 {
		return nil, errors.New("Receiver has 0 Dealers in its data.Can't produce SharedSecret.")
	}
	pub := new(PubPoly)
	//pub.InitNull(r.info.Suite, r.info.T, r.Dealers[0].Promise.PubPoly().GetB())
	pub.InitNull(r.info.Suite, r.info.T, r.info.Suite.Point().Base())
	share := r.info.Suite.Secret()
	goodShare := 0
	for index, _ := range r.Dealers {
		// Only need T shares
		if goodShare >= r.info.T {
			break
		}
		// Compute secret shares of the shared secret = sum of the respectives shares of peer i
		// For peer i , s = SUM fj(i)
		s, e := r.Dealers[index].State.RevealShare(r.index, r.Key)
		if e != nil {
			//TODO error handling function not implemented right now. Only used for testing / comparison.
			// We must be able to tell which share failed and to implement the broadcast of that error to others receiver
			// so they reconstruct the private polynomial of the malicious dealer and set their share themself
			return nil, errors.New(fmt.Sprintf("Receiver %d could not reveal its share from Dealer %d promise : %v", r.index, index, e))
		}
		share.Add(share, s)

		// Compute shared public polynomial = SUM of indiviual public polynomials
		pub.Add(pub, r.Dealers[index].Promise.PubPoly())

		goodShare += 1
	}

	if goodShare < r.info.T {
		return nil, errors.New("Not enough shares received by the Receiver to construct its own share of the shared secret")
	}

	if val := pub.Check(r.index, share); val == false {
		return nil, errors.New("Receiver's secret share of the shared secret could not be checked against the shared polynomial")
	}

	return &SharedSecret{
		Pub:   pub,
		Share: &share,
		Index: r.index,
	}, nil
}
