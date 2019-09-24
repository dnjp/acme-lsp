// Package text implements text editing helper routines for LSP.
package text

import (
	"io"

	"github.com/fhs/acme-lsp/internal/golang_org_x_tools/span"
	"github.com/fhs/acme-lsp/internal/lsp/protocol"
	"github.com/pkg/errors"
)

// File represents an open file in text editor.
type File interface {
	// Reader returns a reader for the entire file text buffer ("body" in acme).
	Reader() (io.Reader, error)

	// WriteAt replaces the text in rune range [q0, q1) with bytes b.
	WriteAt(q0, q1 int, b []byte) (int, error)

	// Mark marks the file for later undo.
	Mark() error

	// DisableMark turns off automatic marking (e.g. generated by WriteAt).
	DisableMark() error
}

// Edit applied edits to file f.
func Edit(f File, edits []protocol.TextEdit) error {
	if len(edits) == 0 {
		return nil
	}
	reader, err := f.Reader()
	if err != nil {
		return err
	}
	off, err := getNewlineOffsets(reader)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain newline offsets")
	}

	f.DisableMark()
	f.Mark()

	// Applying the edits in reverse order gets the job done.
	// See https://github.com/golang/go/wiki/gopls#textdocumentformatting-response
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		q0 := off.LineToOffset(int(e.Range.Start.Line), int(e.Range.Start.Character))
		q1 := off.LineToOffset(int(e.Range.End.Line), int(e.Range.End.Character))
		f.WriteAt(q0, q1, []byte(e.NewText))
	}
	return nil
}

// AddressableFile represents an open file in text editor which has a current adddress.
type AddressableFile interface {
	File

	// Filename returns the filesystem path to the file.
	Filename() (string, error)

	// CurrentAddr returns the address of current selection.
	CurrentAddr() (q0, q1 int, err error)
}

// DocumentURI returns the URI and filename of a file being edited.
func DocumentURI(f AddressableFile) (uri protocol.DocumentURI, filename string, err error) {
	name, err := f.Filename()
	if err != nil {
		return "", "", err
	}
	return ToURI(name), name, nil
}

// Position returns the current position within a file being edited.
func Position(f AddressableFile) (pos *protocol.TextDocumentPositionParams, filename string, err error) {
	name, err := f.Filename()
	if err != nil {
		return nil, "", err
	}
	q0, _, err := f.CurrentAddr()
	if err != nil {
		return nil, "", err
	}
	reader, err := f.Reader()
	if err != nil {
		return nil, "", err
	}
	off, err := getNewlineOffsets(reader)
	if err != nil {
		return nil, "", err
	}
	line, col := off.OffsetToLine(q0)
	return &protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: ToURI(name),
		},
		Position: protocol.Position{
			Line:      float64(line),
			Character: float64(col),
		},
	}, name, nil
}

// ToURI converts filename to URI.
func ToURI(filename string) protocol.DocumentURI {
	return protocol.DocumentURI(span.NewURI(filename))
}

// ToPath converts URI to filename.
func ToPath(uri protocol.DocumentURI) string {
	return span.NewURI(uri).Filename()
}
