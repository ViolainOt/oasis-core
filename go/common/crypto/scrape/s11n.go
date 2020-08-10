package scrape

import (
	"fmt"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/proof/dleq"
	"go.dedis.ch/kyber/v3/share"
	"go.dedis.ch/kyber/v3/share/pvss"
)

var errInvalidPoint = fmt.Errorf("scrape/s11n: point is invalid")

// As convenient as it is to use kyber's PVSS implementation, scalars and
// points being interfaces makes s11n a huge pain, and mandates using
// wrapper types so that this can play nice with CBOR/JSON etc.
//
// Aut viam inveniam aut faciam.

// Point is an elliptic curve point.
type Point struct {
	inner kyber.Point
}

// Inner returns the actual kyber.Point.
func (p *Point) Inner() kyber.Point {
	return p.inner
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (p *Point) UnmarshalBinary(data []byte) error {
	inner := suite.Point()
	if err := inner.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("scrape/s11n: failed to deserialize point: %w", err)
	}
	if !pointIsValid(inner) {
		return errInvalidPoint
	}

	p.inner = inner

	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (p *Point) MarshalBinary() ([]byte, error) {
	data, err := p.inner.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("scrape/s11n: failed to serialize point: %w", err)
	}

	return data, nil
}

func (p *Point) isWellFormed() error {
	// Can never happen(?), but check anyway.
	if p.inner == nil {
		return fmt.Errorf("scrape/s11n: point is missing")
	}

	if !pointIsValid(p.inner) {
		return errInvalidPoint
	}

	return nil
}

func pointFromKyber(p kyber.Point) Point {
	return Point{
		inner: p,
	}
}

type validAble interface {
	Valid() bool
}

type hasSmallOrderAble interface {
	HasSmallOrder() bool
}

func pointIsValid(point kyber.Point) bool {
	switch validator := point.(type) {
	case validAble:
		// P-256 point validation (ensures point is on curve)
		//
		// Note: Kyber's idea of a valid point includes the point at
		// infinity, which does not ensure contributory behavior when
		// doing ECDH.
		return validator.Valid()
	case hasSmallOrderAble:
		// Ed25519 point validation (rejects small-order points)
		//
		// Note: This requires a recent (read: unreleased) version
		// of kyber.
		return !validator.HasSmallOrder()
	default:
		return false
	}
}

// Scalar is a scalar.
type Scalar struct {
	inner kyber.Scalar
}

// Inner returns the actual kyber.Scalar.
func (s *Scalar) Inner() kyber.Scalar {
	return s.inner
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (s *Scalar) UnmarshalBinary(data []byte) error {
	inner := suite.Scalar()
	if err := inner.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("scrape/s11n: failed to deserialize scalar: %w", err)
	}

	s.inner = inner

	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (s *Scalar) MarshalBinary() ([]byte, error) {
	data, err := s.inner.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("scrape/s11n: failed to serialize scalar: %w", err)
	}

	return data, nil
}

func (s *Scalar) isWellFormed() error {
	// Can never happen(?), but check anyway.
	if s.inner == nil {
		return fmt.Errorf("scrape/s11n: scalar is missing")
	}

	return nil
}

func scalarFromKyber(s kyber.Scalar) Scalar {
	return Scalar{
		inner: s,
	}
}

// PubVerShare is a public verifiable share (`pvss.PubVerShare`)
type PubVerShare struct {
	V Point `json:"v"` // Encrypted/decrypted share

	C  Scalar `json:"c"`  // Challenge
	R  Scalar `json:"r"`  // Response
	VG Point  `json:"vg"` // Public commitment with respect to base point G
	VH Point  `json:"vh"` // Public commitment with respect to base point H
}

func (pvs *PubVerShare) isWellFormed() error {
	if err := pvs.V.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid PubVerShare V: %w", err)
	}
	if err := pvs.C.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid PubVerShare C: %w", err)
	}
	if err := pvs.R.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid PubVerShare R: %w", err)
	}
	if err := pvs.VG.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid PubVerShare VG: %w", err)
	}
	if err := pvs.VH.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid PubVerShare VH: %w", err)
	}

	return nil
}

func (pvs *PubVerShare) toKyber(index int) *pvss.PubVerShare {
	return &pvss.PubVerShare{
		S: share.PubShare{
			I: index,
			V: pvs.V.Inner(),
		},
		P: dleq.Proof{
			C:  pvs.C.Inner(),
			R:  pvs.R.Inner(),
			VG: pvs.VG.Inner(),
			VH: pvs.VH.Inner(),
		},
	}
}

func pubVerShareFromKyber(pvs *pvss.PubVerShare) *PubVerShare {
	return &PubVerShare{
		V:  pointFromKyber(pvs.S.V),
		C:  scalarFromKyber(pvs.P.C),
		R:  scalarFromKyber(pvs.P.R),
		VG: pointFromKyber(pvs.P.VG),
		VH: pointFromKyber(pvs.P.VH),
	}
}

// CommitShare is a commit share.
type CommitShare struct {
	PolyV Point `json:"poly_v"` // Share of the public commitment polynomial
	PubVerShare
}

func (cs *CommitShare) isWellFormed() error {
	if err := cs.PolyV.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid CommitShare PolyV: %w", err)
	}
	if err := cs.PubVerShare.isWellFormed(); err != nil {
		return fmt.Errorf("scrape/s11n: invalid CommitShare PubVerShare: %w", err)
	}

	return nil
}

func (cs *CommitShare) toKyber(index int) (*share.PubShare, *pvss.PubVerShare) {
	pubShare := &share.PubShare{
		I: index,
		V: cs.PolyV.Inner(),
	}
	return pubShare, cs.PubVerShare.toKyber(index)
}

func commitShareFromKyber(pubPolyShare *share.PubShare, encShare *pvss.PubVerShare) *CommitShare {
	return &CommitShare{
		PolyV:       pointFromKyber(pubPolyShare.V),
		PubVerShare: *pubVerShareFromKyber(encShare),
	}
}

func commitSharesFromKyber(pubPolyShares []*share.PubShare, encShares []*pvss.PubVerShare) []*CommitShare {
	if len(pubPolyShares) != len(encShares) {
		panic("scrape/s11n: BUG: len(pubPolyShares != len(encShares)")
	}

	var shares []*CommitShare
	for i, pubPolyShare := range pubPolyShares {
		encShare := encShares[i]
		if pubPolyShare.I != encShare.S.I {
			panic("scrape/s11n: BUG: pubPolyShare.I != encShare.I")
		}
		shares = append(shares, commitShareFromKyber(pubPolyShare, encShare))
	}

	return shares
}
