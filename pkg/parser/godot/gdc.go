package godot

// Compiled GDScript (.gdc) tokenizer buffer decode/encode.
// Spec per godotengine/godot modules/gdscript/gdscript_tokenizer_buffer.cpp
// and core/io/marshalls.cpp decode_variant (upstream).
//
// Buffer layout (after GDSC magic + version u32 + uncompressed_size u32,
// zstd-decompressed "contents"):
//   u32 identifier_count, u32 constant_count, u32 token_line_count, u32 token_count
//   identifiers: for each: u32 len, then len UTF-32 chars, each char's 4 bytes XOR 0xB6
//   constants:   for each: one encoded Variant
//   token_lines: token_line_count * (u32 token_index, u32 line)
//   token_cols:  token_line_count * (u32 token_index, u32 column)
//   tokens:      token_count * (5 or 8 bytes; high bit of first byte marks 8)
//
// Only string constants are rewritten; every other Variant kind is copied
// byte-for-byte, so the token stream (which references constants by index)
// never changes.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/klauspost/compress/zstd"
)

const gdscXOR = 0xB6

var (
	zdec, _ = zstd.NewReader(nil)
	zenc, _ = zstd.NewWriter(nil)
)

// Variant type values (godotengine/godot core/variant/variant.h enum Type).
const (
	vpNil             = 0
	vpBool            = 1
	vpInt             = 2
	vpFloat           = 3
	vpString          = 4
	vpVector2         = 5
	vpVector2i        = 6
	vpRect2           = 7
	vpRect2i          = 8
	vpVector3         = 9
	vpVector3i        = 10
	vpTransform2D     = 11
	vpVector4         = 12
	vpVector4i        = 13
	vpPlane           = 14
	vpQuaternion      = 15
	vpAABB            = 16
	vpBasis           = 17
	vpTransform3D     = 18
	vpProjection      = 19
	vpColor           = 20
	vpStringName      = 21
	vpNodePath        = 22
	vpRID             = 23
	vpObject          = 24
	vpCallable        = 25
	vpSignal          = 26
	vpDictionary      = 27
	vpArray           = 28
	vpPackedByteArray = 29
	vpPackedInt32     = 30
	vpPackedInt64     = 31
	vpPackedFloat32   = 32
	vpPackedFloat64   = 33
	vpPackedStringArr = 34
	vpPackedVector2   = 35
	vpPackedVector3   = 36
	vpPackedColor     = 37
	vpPackedVector4   = 38
)

// floatDim maps float math types to their component count (32/64-bit per header flag).
var floatDim = map[uint32]int{
	vpVector2: 2, vpRect2: 4, vpVector3: 3, vpTransform2D: 6, vpVector4: 4,
	vpPlane: 4, vpQuaternion: 4, vpAABB: 6, vpBasis: 9, vpTransform3D: 12, vpProjection: 16,
}

// intDim maps integer math types (always u32, the 64-bit flag does not apply).
var intDim = map[uint32]int{vpVector2i: 2, vpRect2i: 4, vpVector3i: 3, vpVector4i: 4}

// packedDim maps packed vector/color arrays to float components per element.
var packedDim = map[uint32]int{vpPackedVector2: 2, vpPackedVector3: 3, vpPackedColor: 4, vpPackedVector4: 4}

// Variant is one decoded constant (only strings are materialized).
type Variant struct {
	Type   uint32
	Str    string // STRING / STRING_NAME payloads
	Mod    bool   // Str was rewritten; re-encode on output
	Header uint32 // original variant header word (reused when re-encoding)
	Raw    []byte // original encoding, verbatim
}

type gdcErr string

func (e gdcErr) Error() string { return string(e) }

func decodeVariant(buf []byte, pos int) (Variant, int, error) {
	if pos+4 > len(buf) {
		return Variant{}, 0, gdcErr("truncated variant header")
	}
	header := binary.LittleEndian.Uint32(buf[pos:])
	typ := header & 0xFF
	p := pos + 4
	used := 4
	str := func() (string, int, error) {
		if p+4 > len(buf) {
			return "", 0, gdcErr("truncated string length")
		}
		n := int(int32(binary.LittleEndian.Uint32(buf[p:])))
		p += 4
		pad := (4 - n%4) % 4
		if n < 0 || p+n+pad > len(buf) {
			return "", 0, gdcErr("string out of range")
		}
		s := string(buf[p : p+n])
		p += n + pad
		return s, 4 + n + pad, nil
	}
	switch typ {
	case vpNil:
	case vpBool:
		if err := adv(&p, &used, 4, len(buf)); err != nil {
			return Variant{}, 0, err
		}
	case vpInt, vpFloat:
		n := 4
		if header&(1<<16) != 0 {
			n = 8
		}
		if err := adv(&p, &used, n, len(buf)); err != nil {
			return Variant{}, 0, err
		}
	case vpString, vpStringName:
		s, n, err := str()
		if err != nil {
			return Variant{}, 0, err
		}
		used += n
		return Variant{Type: typ, Str: s, Header: header, Raw: buf[pos : pos+used]}, used, nil
	case vpNodePath:
		if p+12 > len(buf) {
			return Variant{}, 0, gdcErr("nodepath truncated")
		}
		namecount := int(binary.LittleEndian.Uint32(buf[p:]) & 0x7FFFFFFF)
		subnamecount := int(binary.LittleEndian.Uint32(buf[p+4:]))
		npflags := binary.LittleEndian.Uint32(buf[p+8:])
		if npflags&2 != 0 {
			subnamecount++
		}
		if namecount == 0 {
			return Variant{}, 0, gdcErr("nodepath legacy format")
		}
		p += 12
		used += 12
		for i := 0; i < namecount+subnamecount; i++ {
			_, n, err := str()
			if err != nil {
				return Variant{}, 0, err
			}
			used += n
		}
	case vpRID:
		if err := adv(&p, &used, 8, len(buf)); err != nil {
			return Variant{}, 0, err
		}
	case vpObject:
		if header&(1<<16) != 0 { // OBJECT_AS_ID
			if err := adv(&p, &used, 8, len(buf)); err != nil {
				return Variant{}, 0, err
			}
		} else {
			_, n, err := str() // class name
			if err != nil {
				return Variant{}, 0, err
			}
			used += n
			if p+4 > len(buf) {
				return Variant{}, 0, gdcErr("object truncated")
			}
			cnt := int(binary.LittleEndian.Uint32(buf[p:]))
			p += 4
			used += 4
			for i := 0; i < cnt; i++ {
				_, n, err := str() // property name
				if err != nil {
					return Variant{}, 0, err
				}
				used += n
				_, nu, err := decodeVariant(buf, p)
				if err != nil {
					return Variant{}, 0, err
				}
				p += nu
				used += nu
			}
		}
	case vpCallable: // empty
	case vpSignal:
		_, n, err := str()
		if err != nil {
			return Variant{}, 0, err
		}
		used += n
		if err := adv(&p, &used, 8, len(buf)); err != nil {
			return Variant{}, 0, err
		}
	case vpDictionary:
		for _, kind := range [2]uint32{(header >> 16) & 0b11, (header >> 18) & 0b11} {
			kn, err := skipContainerType(buf, p, kind)
			if err != nil {
				return Variant{}, 0, err
			}
			p += kn
			used += kn
		}
		if p+4 > len(buf) {
			return Variant{}, 0, gdcErr("dict truncated")
		}
		cnt := int(binary.LittleEndian.Uint32(buf[p:]) & 0x7FFFFFFF)
		p += 4
		used += 4
		for i := 0; i < cnt; i++ {
			for j := 0; j < 2; j++ {
				_, nu, err := decodeVariant(buf, p)
				if err != nil {
					return Variant{}, 0, err
				}
				p += nu
				used += nu
			}
		}
	case vpArray:
		kn, err := skipContainerType(buf, p, (header>>16)&0b11)
		if err != nil {
			return Variant{}, 0, err
		}
		p += kn
		used += kn
		if p+4 > len(buf) {
			return Variant{}, 0, gdcErr("array truncated")
		}
		cnt := int(binary.LittleEndian.Uint32(buf[p:]) & 0x7FFFFFFF)
		p += 4
		used += 4
		for i := 0; i < cnt; i++ {
			_, nu, err := decodeVariant(buf, p)
			if err != nil {
				return Variant{}, 0, err
			}
			p += nu
			used += nu
		}
	case vpPackedByteArray:
		if p+4 > len(buf) {
			return Variant{}, 0, gdcErr("pba truncated")
		}
		cnt := int(binary.LittleEndian.Uint32(buf[p:]))
		pad := (4 - cnt%4) % 4
		if p+4+cnt+pad > len(buf) {
			return Variant{}, 0, gdcErr("pba out of range")
		}
		p += 4 + cnt + pad
		used += 4 + cnt + pad
	case vpPackedInt32, vpPackedFloat32, vpPackedInt64, vpPackedFloat64:
		elem := 4
		if typ == vpPackedInt64 || typ == vpPackedFloat64 {
			elem = 8
		}
		if p+4 > len(buf) {
			return Variant{}, 0, gdcErr("packed truncated")
		}
		cnt := int(binary.LittleEndian.Uint32(buf[p:]))
		if p+4+cnt*elem > len(buf) {
			return Variant{}, 0, gdcErr("packed out of range")
		}
		p += 4 + cnt*elem
		used += 4 + cnt*elem
	case vpPackedStringArr:
		if p+4 > len(buf) {
			return Variant{}, 0, gdcErr("psa truncated")
		}
		cnt := int(binary.LittleEndian.Uint32(buf[p:]))
		p += 4
		used += 4
		for i := 0; i < cnt; i++ {
			_, n, err := str()
			if err != nil {
				return Variant{}, 0, err
			}
			used += n
		}
	default:
		if d, ok := packedDim[typ]; ok {
			if p+4 > len(buf) {
				return Variant{}, 0, gdcErr("packed vector truncated")
			}
			cnt := int(binary.LittleEndian.Uint32(buf[p:]))
			if p+4+cnt*d*4 > len(buf) {
				return Variant{}, 0, gdcErr("packed vector out of range")
			}
			p += 4 + cnt*d*4
			used += 4 + cnt*d*4
		} else if d, ok := intDim[typ]; ok {
			if err := adv(&p, &used, d*4, len(buf)); err != nil {
				return Variant{}, 0, err
			}
		} else if d, ok := floatDim[typ]; ok {
			sz := 4
			if header&(1<<16) != 0 {
				sz = 8
			}
			if err := adv(&p, &used, d*sz, len(buf)); err != nil {
				return Variant{}, 0, err
			}
		} else {
			return Variant{}, 0, fmt.Errorf("gdc: unknown variant type %d", typ)
		}
	}
	if p > len(buf) {
		return Variant{}, 0, gdcErr("variant overruns buffer")
	}
	return Variant{Type: typ, Header: header, Raw: buf[pos:p]}, used, nil
}

func adv(p, used *int, n, bufLen int) error {
	if *p+n > bufLen {
		return gdcErr("variant out of range")
	}
	*p += n
	*used += n
	return nil
}

func skipContainerType(buf []byte, p int, kind uint32) (int, error) {
	switch kind {
	case 0b00: // untyped
		return 0, nil
	case 0b01: // builtin: u32
		if p+4 > len(buf) {
			return 0, gdcErr("container type truncated")
		}
		return 4, nil
	case 0b10: // class name: padded string
		if p+4 > len(buf) {
			return 0, gdcErr("container type truncated")
		}
		n := int(int32(binary.LittleEndian.Uint32(buf[p:])))
		pad := (4 - n%4) % 4
		if n < 0 || p+4+n+pad > len(buf) {
			return 0, gdcErr("container class out of range")
		}
		return 4 + n + pad, nil
	default:
		return 0, errors.New("gdc: script-typed containers not supported")
	}
}

// GDC is a decoded compiled GDScript buffer.
type GDC struct {
	Version     uint32
	Identifiers []string
	Constants   []Variant
	header      [4]uint32
	rawIdent    [][]byte
	tail        []byte
}

// DecodeGDC parses a .gdc buffer (GDSC magic + zstd-compressed contents).
func DecodeGDC(data []byte) (*GDC, error) {
	if len(data) < 12 || string(data[:4]) != "GDSC" {
		return nil, errors.New("not a GDSC buffer")
	}
	ver := binary.LittleEndian.Uint32(data[4:])
	unc := binary.LittleEndian.Uint32(data[8:])
	var contents []byte
	var err error
	if unc == 0 {
		contents = data[12:]
	} else {
		contents, err = zdec.DecodeAll(data[12:], nil)
		if err != nil {
			return nil, fmt.Errorf("gdc zstd: %w", err)
		}
		if uint32(len(contents)) != unc {
			return nil, fmt.Errorf("gdc size mismatch: got %d want %d", len(contents), unc)
		}
	}
	if len(contents) < 16 {
		return nil, errors.New("gdc contents too short")
	}
	g := &GDC{Version: ver}
	g.header = [4]uint32{
		binary.LittleEndian.Uint32(contents[0:]),
		binary.LittleEndian.Uint32(contents[4:]),
		binary.LittleEndian.Uint32(contents[8:]),
		binary.LittleEndian.Uint32(contents[12:]),
	}
	p := 16
	g.rawIdent = make([][]byte, g.header[0])
	g.Identifiers = make([]string, g.header[0])
	for i := range g.Identifiers {
		if p+4 > len(contents) {
			return nil, errors.New("gdc identifier truncated")
		}
		n := int(binary.LittleEndian.Uint32(contents[p:]))
		start := p
		p += 4 + n*4
		if p > len(contents) {
			return nil, errors.New("gdc identifier out of range")
		}
		g.rawIdent[i] = contents[start:p]
		raw := contents[start+4 : p]
		runes := make([]rune, n)
		for j := 0; j < n; j++ {
			runes[j] = rune(binary.LittleEndian.Uint32(raw[j*4:]) ^ (gdscXOR<<24 | gdscXOR<<16 | gdscXOR<<8 | gdscXOR))
		}
		g.Identifiers[i] = string(runes)
	}
	g.Constants = make([]Variant, g.header[1])
	for i := range g.Constants {
		v, used, err := decodeVariant(contents, p)
		if err != nil {
			return nil, fmt.Errorf("gdc constant %d: %w", i, err)
		}
		g.Constants[i] = v
		p += used
	}
	g.tail = contents[p:]
	return g, nil
}

// EncodeGDC reassembles the buffer. Modified string constants are re-encoded;
// everything else is copied verbatim.
func (g *GDC) EncodeGDC() ([]byte, error) {
	var contents bytes.Buffer
	var l4 [4]byte
	for _, h := range g.header {
		binary.LittleEndian.PutUint32(l4[:], h)
		contents.Write(l4[:])
	}
	for _, raw := range g.rawIdent {
		contents.Write(raw)
	}
	for _, c := range g.Constants {
		if c.Mod {
			contents.Write(encodeStringVariant(c))
		} else {
			contents.Write(c.Raw)
		}
	}
	contents.Write(g.tail)

	out := make([]byte, 0, 12+contents.Len())
	out = append(out, 'G', 'D', 'S', 'C')
	binary.LittleEndian.PutUint32(l4[:], g.Version)
	out = append(out, l4[:]...)
	binary.LittleEndian.PutUint32(l4[:], uint32(contents.Len()))
	out = append(out, l4[:]...)
	return zenc.EncodeAll(contents.Bytes(), out), nil
}

func encodeStringVariant(v Variant) []byte {
	b := make([]byte, 0, 8+len(v.Str)+4)
	var l4 [4]byte
	binary.LittleEndian.PutUint32(l4[:], v.Header)
	b = append(b, l4[:]...)
	binary.LittleEndian.PutUint32(l4[:], uint32(len(v.Str)))
	b = append(b, l4[:]...)
	b = append(b, v.Str...)
	if pad := (4 - len(v.Str)%4) % 4; pad != 0 {
		b = append(b, make([]byte, pad)...)
	}
	return b
}
