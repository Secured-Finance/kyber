package poly

import (
	"fmt"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/config"
	"github.com/dedis/crypto/edwards"
	"github.com/dedis/crypto/random"
	"testing"
)

/////// TESTING ///////

func TestReceiverAddDealer(t *testing.T) {
	dealers, receivers := generateNDealerMReceiver(PolyInfo{edward, 2, 3, 3}, 3, 3)
	// Test adding one dealer
	_, e1 := receivers[0].AddDealer(0, dealers[0])
	if e1 != nil {
		t.Error(fmt.Sprintf("AddDealer should not return an error : %v", e1))
	}

	// Test adding another dealer with same index
	_, e2 := receivers[0].AddDealer(0, dealers[1])
	if e2 != nil {
		t.Error(fmt.Sprintf("AddDealer should not return an error : %v", e2))
	}

	// Test adding another dealer with different index !
	_, e3 := receivers[0].AddDealer(1, dealers[2])
	if e3 == nil {
		t.Error(fmt.Sprintf("AddDealer should have returned an error (adding dealer to a different index for same receiver)"))
	}
}

// Test the AddReponse func
func TestRightDealerAddResponse(t *testing.T) {
	// Test if all goes well with the right inputs
	n := 3
	m := 3
	dealers, receivers := generateNDealerMReceiver(PolyInfo{edward, 2, 3, 3}, n, m)
	// for each receiver
	for i := 0; i < m; i++ {
		// add all the dealers
		for j := 0; j < n; j++ {
			resp, err := receivers[i].AddDealer(i, dealers[j])
			if err != nil {
				t.Error("AddDealer should not generate error")
			}
			// then give the response back to the dealer
			err = dealers[j].AddResponse(i, resp)
			if err != nil {
				t.Error(fmt.Sprintf("AddResponse should not generate any error : %v", err))
			}
		}
	}
	for j := 0; j < n; j++ {
		val := dealers[j].Certified()
		if val != nil {
			t.Error(fmt.Sprintf("Dealer %d should be certified : ", j, val))
		}
	}

}

// Test the AddReponse func with wrong inputs
func TestWrongDealerAddResponse(t *testing.T) {
	n := 2
	m := 3
	dealers, receivers := generateNDealerMReceiver(PolyInfo{edward, 2, 3, 3}, n, m)
	r1, _ := receivers[0].AddDealer(0, dealers[0])
	err := dealers[0].AddResponse(1, r1)
	if err == nil {
		t.Error("AddResponse should have returned an error when given the wrong index share")
	}
	// We may do others tests but I leave it for now as a discussion because, all theses tests are based on the promise package which is already well tested
}

func TestProduceSharedSecret(t *testing.T) {
	n := 3
	m := 3
	_, receivers := generateNMSetup(PolyInfo{edward, 2, 3, 3}, n, m)
	s1, err := receivers[0].ProduceSharedSecret()
	if err != nil {
		t.Error(fmt.Sprintf("ProduceSharedSecret should not gen any error : %v", err))
	}
	s2, err := receivers[1].ProduceSharedSecret()
	if err != nil {
		t.Error(fmt.Sprintf("ProdueSharedSecret should not gen any error : %v", err))
	}

	if !s1.Pub.Equal(s2.Pub) {
		t.Error("SharedSecret's polynomials should be equals")
	}

	if v := s1.Pub.Check(receivers[1].index, *s2.Share); v == false {
		t.Error("SharedSecret's share can not be verified using another's receiver pubpoly")
	}
	if v := s2.Pub.Check(receivers[0].index, *s1.Share); v == false {
		t.Error("SharedSecret's share can not be verified using another's receiver pubpoly")
	}
}
