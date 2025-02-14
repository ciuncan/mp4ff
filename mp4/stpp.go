package mp4

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
)

// StppBox - XMLSubtitleSampleEntryr Box (stpp)
//
// Contained in : Media Information Box (minf)
type StppBox struct {
	Namespace          string   // Mandatory
	SchemaLocation     string   // Optional
	AuxiliaryMimeTypes string   // Required if auxiliary types present
	Btrt               *BtrtBox // Optional
	Children           []Box
	DataReferenceIndex uint16
}

// NewStppBox - Create new stpp box
// namespace, schemaLocation and auxiliaryMimeType are space-separated utf8-lists without zero-termination
// schemaLocation and auxiliaryMimeTypes are optional
func NewStppBox(namespace, schemaLocation, auxiliaryMimeTypes string) *StppBox {
	return &StppBox{
		Namespace:          namespace,
		SchemaLocation:     schemaLocation,
		AuxiliaryMimeTypes: auxiliaryMimeTypes,
		DataReferenceIndex: 1,
	}
}

// AddChild - add a child box (avcC normally, but clap and pasp could be part of visual entry)
func (b *StppBox) AddChild(child Box) {
	switch box := child.(type) {
	case *BtrtBox:
		b.Btrt = box
	default:
		// Other box
	}

	b.Children = append(b.Children, child)
}

// DecodeStpp - Decode XMLSubtitleSampleEntry (stpp)
func DecodeStpp(hdr *boxHeader, startPos uint64, r io.Reader) (Box, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	b := &StppBox{}
	s := NewSliceReader(data)

	// 14496-12 8.5.2.2 Sample entry (8 bytes)
	s.SkipBytes(6) // Skip 6 reserved bytes
	b.DataReferenceIndex = s.ReadUint16()

	b.Namespace, err = s.ReadZeroTerminatedString()
	if err != nil {
		return nil, err
	}

	if s.NrRemainingBytes() > 0 {
		b.SchemaLocation, err = s.ReadZeroTerminatedString()
		if err != nil {
			return nil, err
		}
	}

	if s.NrRemainingBytes() > 0 {
		b.AuxiliaryMimeTypes, err = s.ReadZeroTerminatedString()
		if err != nil {
			return nil, err
		}
	}

	pos := startPos + uint64(boxHeaderSize+s.GetPos())
	remaining := s.RemainingBytes()
	restReader := bytes.NewReader(remaining)

	for {
		box, err := DecodeBox(pos, restReader)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if box != nil {
			b.AddChild(box)
			pos += box.Size()
		}
		if pos == startPos+hdr.size {
			break
		} else if pos > startPos+hdr.size {
			return nil, errors.New("Bad size in stpp")
		}
	}
	return b, nil
}

// Type - return box type
func (b *StppBox) Type() string {
	return "stpp"
}

// Size - return calculated size
func (b *StppBox) Size() uint64 {
	nrSampleEntryBytes := 8
	totalSize := uint64(boxHeaderSize + nrSampleEntryBytes + len(b.Namespace) + 1)
	if b.SchemaLocation != "" {
		totalSize += uint64(len(b.SchemaLocation)) + 1
	}
	if b.AuxiliaryMimeTypes != "" {
		totalSize += uint64(len(b.AuxiliaryMimeTypes)) + 1
	}
	for _, child := range b.Children {
		totalSize += child.Size()
	}
	return totalSize
}

// Encode - write box to w
func (b *StppBox) Encode(w io.Writer) error {
	err := EncodeHeader(b, w)
	if err != nil {
		return err
	}
	buf := makebuf(b)
	sw := NewSliceWriter(buf)
	sw.WriteZeroBytes(6)
	sw.WriteUint16(b.DataReferenceIndex)
	sw.WriteString(b.Namespace, true)
	if b.SchemaLocation != "" {
		sw.WriteString(b.SchemaLocation, true)
	}
	if b.AuxiliaryMimeTypes != "" {
		sw.WriteString(b.AuxiliaryMimeTypes, true)
	}
	_, err = w.Write(buf[:sw.pos]) // Only write written bytes
	if err != nil {
		return err
	}

	// Next output child boxes in order
	for _, child := range b.Children {
		err = child.Encode(w)
		if err != nil {
			return err
		}
	}
	return err
}

// Info - write specific box info to w
func (b *StppBox) Info(w io.Writer, specificBoxLevels, indent, indentStep string) error {
	bd := newInfoDumper(w, indent, b, -1, 0)
	bd.write(" - dataReferenceIndex: %d", b.DataReferenceIndex)
	bd.write(" - nameSpace: %s", b.Namespace)
	bd.write(" - schemaLocation: %s", b.SchemaLocation)
	bd.write(" - auxiliaryMimeTypes: %s", b.AuxiliaryMimeTypes)
	if bd.err != nil {
		return bd.err
	}
	var err error
	for _, child := range b.Children {
		err = child.Info(w, specificBoxLevels, indent+indentStep, indent)
		if err != nil {
			return err
		}
	}
	return nil
}
