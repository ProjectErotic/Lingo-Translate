package godot

// Minimal reader/writer for Godot 4.x .pck pack files (format v2/v3/v4).
// Spec per godotengine/godot core/io/file_access_pack.cpp (upstream).

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PckEntry is one file inside a .pck pack.
type PckEntry struct {
	Path string
	Ofs  uint64 // absolute offset from pack start (file_base already applied)
	Size uint64
}

type pckHeader struct {
	ver, vm, vn, vp, flags uint32
	fileBase               uint64
	dirOff                 uint64
}

func leU32(b []byte, o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
func leU64(b []byte, o int) uint64 { return binary.LittleEndian.Uint64(b[o:]) }

func openPck(path string) (*os.File, pckHeader, []PckEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, pckHeader{}, nil, err
	}
	var magic [4]byte
	if _, err := f.Read(magic[:]); err != nil {
		f.Close()
		return nil, pckHeader{}, nil, err
	}
	if string(magic[:]) != "GDPC" {
		f.Close()
		return nil, pckHeader{}, nil, fmt.Errorf("%s: not a GDPC pack", path)
	}
	hdr := make([]byte, 28)
	if _, err := f.Read(hdr); err != nil {
		f.Close()
		return nil, pckHeader{}, nil, err
	}
	h := pckHeader{
		ver: leU32(hdr, 0), vm: leU32(hdr, 4), vn: leU32(hdr, 8), vp: leU32(hdr, 12),
		flags: leU32(hdr, 16), fileBase: leU64(hdr, 20),
	}
	if h.ver < 2 || h.ver > 4 {
		f.Close()
		return nil, pckHeader{}, nil, fmt.Errorf("%s: unsupported pck version %d", path, h.ver)
	}
	if h.flags&1 != 0 {
		f.Close()
		return nil, pckHeader{}, nil, fmt.Errorf("%s: encrypted pack directory not supported", path)
	}
	if h.ver >= 3 {
		tail := make([]byte, 8)
		if _, err := f.Read(tail); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
		h.dirOff = leU64(tail, 0)
	}
	// v2: directory right after the reserved block; v3/v4: at dirOff.
	if h.ver >= 3 {
		if _, err := f.Seek(int64(h.dirOff), 0); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
	} else {
		if _, err := f.Seek(4+4*5+16*4, 0); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
	}
	cntBuf := make([]byte, 4)
	if _, err := f.Read(cntBuf); err != nil {
		f.Close()
		return nil, pckHeader{}, nil, err
	}
	cnt := leU32(cntBuf, 0)
	entries := make([]PckEntry, 0, cnt)
	for i := uint32(0); i < cnt; i++ {
		lb := make([]byte, 4)
		if _, err := f.Read(lb); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
		l := leU32(lb, 0)
		pb := make([]byte, l)
		if _, err := f.Read(pb); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
		path := string(pb)
		if z := bytes.IndexByte(pb, 0); z >= 0 {
			path = string(pb[:z])
		}
		db := make([]byte, 8+8+16+4)
		if _, err := f.Read(db); err != nil {
			f.Close()
			return nil, pckHeader{}, nil, err
		}
		ofs, size := leU64(db, 0), leU64(db, 8)
		fl := leU32(db, 32)
		if fl&2 != 0 { // PACK_FILE_REMOVAL
			continue
		}
		if fl&1 != 0 { // PACK_FILE_ENCRYPTED
			f.Close()
			return nil, pckHeader{}, nil, fmt.Errorf("%s: encrypted file %s not supported", path, path)
		}
		entries = append(entries, PckEntry{Path: path, Ofs: h.fileBase + ofs, Size: size})
	}
	return f, h, entries, nil
}

// ReadPck returns the entries of a .pck pack and a closure reading one entry's bytes.
func ReadPck(path string) ([]PckEntry, func(e PckEntry) ([]byte, error), error) {
	f, _, entries, err := openPck(path)
	if err != nil {
		return nil, nil, err
	}
	read := func(e PckEntry) ([]byte, error) {
		if _, err := f.Seek(int64(e.Ofs), 0); err != nil {
			return nil, err
		}
		b := make([]byte, e.Size)
		if _, err := f.Read(b); err != nil {
			return nil, err
		}
		return b, nil
	}
	return entries, read, nil
}

// WritePck writes a new .pck pack (format v4, PACK_REL_FILEBASE) with the given files.
// Sorted by path, 16-byte aligned payloads, zero md5s (the engine does not verify them).
func WritePck(dest string, files map[string][]byte) error {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)

	type rec struct {
		name string
		ofs  uint64
		size uint64
	}
	recs := make([]rec, 0, len(names))
	pos := uint64(112)
	for _, n := range names {
		pos = (pos + 15) &^ 15
		recs = append(recs, rec{n, pos, uint64(len(files[n]))})
		pos += uint64(len(files[n]))
	}
	dirOff := pos

	buf := bytes.NewBuffer(make([]byte, 0, int(pos)+4+32*len(recs)))
	header := make([]byte, 40)
	copy(header, "GDPC")
	binary.LittleEndian.PutUint32(header[4:], 4)   // pack format version
	binary.LittleEndian.PutUint32(header[8:], 4)   // engine major
	binary.LittleEndian.PutUint32(header[12:], 4)  // engine minor
	binary.LittleEndian.PutUint32(header[16:], 0)  // engine patch
	binary.LittleEndian.PutUint32(header[20:], 2)  // flags: PACK_REL_FILEBASE
	binary.LittleEndian.PutUint64(header[24:], 112) // file base
	binary.LittleEndian.PutUint64(header[32:], dirOff)
	buf.Write(header)

	for _, r := range recs {
		buf.Write(make([]byte, int(r.ofs-uint64(buf.Len()))))
		buf.Write(files[r.name])
	}
	buf.Write(make([]byte, int(dirOff-uint64(buf.Len()))))

	var l4 [4]byte
	binary.LittleEndian.PutUint32(l4[:], uint32(len(recs)))
	buf.Write(l4[:])
	for _, r := range recs {
		binary.LittleEndian.PutUint32(l4[:], uint32(len(r.name)))
		buf.Write(l4[:])
		buf.WriteString(r.name)
		var b8 [8]byte
		binary.LittleEndian.PutUint64(b8[:], r.ofs-112)
		buf.Write(b8[:])
		binary.LittleEndian.PutUint64(b8[:], r.size)
		buf.Write(b8[:])
		buf.Write(make([]byte, 16+4)) // md5 placeholder + flags 0
	}
	return os.WriteFile(dest, buf.Bytes(), 0644)
}

// PckForGame finds the main .pck of an exported game directory.
// Embedded packs (self-contained exe) are not supported.
func PckForGame(dir string) string {
	var found string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".pck") {
			if fi, err2 := d.Info(); err2 == nil && fi.Size() > 128 {
				if found == "" || len(p) < len(found) {
					found = p
				}
			}
		}
		return nil
	})
	return found
}
