package binarydist

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"errors"
	"io"
)

var ErrCorrupt = errors.New("corrupt patch")

const MaxBufferSize = int64(32 * 1024)

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func Patch(old io.ReadSeeker, new io.Writer, patch io.Reader) error {
	var hdr header
	err := binary.Read(patch, signMagLittleEndian{}, &hdr)
	if err != nil {
		return err
	}
	if hdr.Magic != magic {
		return ErrCorrupt
	}
	if hdr.CtrlLen < 0 || hdr.DiffLen < 0 || hdr.NewSize < 0 {
		return ErrCorrupt
	}

	ctrlbuf := make([]byte, hdr.CtrlLen)
	_, err = io.ReadFull(patch, ctrlbuf)
	if err != nil {
		return err
	}
	cpfbz2 := bzip2.NewReader(bytes.NewReader(ctrlbuf))

	diffbuf := make([]byte, hdr.DiffLen)
	_, err = io.ReadFull(patch, diffbuf)
	if err != nil {
		return err
	}
	dpfbz2 := bzip2.NewReader(bytes.NewReader(diffbuf))

	// The entire rest of the file is the extra block.
	epfbz2 := bzip2.NewReader(patch)

	newpos := int64(0)
	for newpos < hdr.NewSize {
		var ctrl struct{ Add, Copy, Seek int64 }
		err = binary.Read(cpfbz2, signMagLittleEndian{}, &ctrl)
		if err != nil {
			return err
		}

		// Sanity-check
		if newpos+ctrl.Add > hdr.NewSize {
			return ErrCorrupt
		}

		bytes2read := ctrl.Add
		for bytes2read > 0 {
			bufsize := MaxBufferSize
			if bytes2read < MaxBufferSize {
				bufsize = bytes2read
			}
			diffbuf := make([]byte, bufsize)
			_, err = io.ReadFull(dpfbz2, diffbuf)
			if err != nil {
				return ErrCorrupt
			}
			oldbuf := make([]byte, bufsize)
			_, err = io.ReadFull(old, oldbuf)
			if err != nil {
				return ErrCorrupt
			}
			// Add old data to diff string
			for i := int64(0); i < bufsize; i++ {
				diffbuf[i] += oldbuf[i]
			}

			written := int64(0)
			for written < bufsize {
				n, err := new.Write(diffbuf[written:])
				if err != nil {
					return err
				}
				written += int64(n)
			}
			newpos += written
			bytes2read -= bufsize
		}
		// Sanity-check
		if newpos+ctrl.Copy > hdr.NewSize {
			return ErrCorrupt
		}

		// Read extra string
		_, err = io.CopyN(new, epfbz2, ctrl.Copy)
		if err != nil {
			return ErrCorrupt
		}
		newpos += ctrl.Copy

		old.Seek(ctrl.Seek, 1)
	}

	return nil
}
