package tg

import (
	"strings"

	"github.com/gotd/td/tg"
)

type MediaInfo struct {
	Type     string `json:"type"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitzero"`
}

// describeMedia skips link previews: the link is already in the text.
func describeMedia(media tg.MessageMediaClass) *MediaInfo {
	switch m := media.(type) {
	case nil, *tg.MessageMediaEmpty, *tg.MessageMediaWebPage:
		return nil
	case *tg.MessageMediaDocument:
		doc, ok := messageDocument(m)
		if !ok {
			return &MediaInfo{Type: "document"}
		}

		return &MediaInfo{
			Type:     "document",
			FileName: documentFileName(doc),
			MimeType: doc.MimeType,
			Size:     doc.Size,
		}
	default:
		return &MediaInfo{Type: strings.ToLower(strings.TrimPrefix(media.TypeName(), "messageMedia"))}
	}
}

func messageDocument(m *tg.MessageMediaDocument) (*tg.Document, bool) {
	d, ok := m.GetDocument()
	if !ok {
		return nil, false
	}

	doc, ok := d.(*tg.Document)

	return doc, ok
}

func documentFileName(doc *tg.Document) string {
	for _, attr := range doc.Attributes {
		if f, ok := attr.(*tg.DocumentAttributeFilename); ok {
			return f.FileName
		}
	}

	return ""
}
