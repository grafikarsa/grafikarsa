package media

import (
	"bytes"
	"encoding/binary"
	"net/http"
)

// detectContent inspects the first bytes of an upload and returns the media
// kind and canonical MIME type. The filename and request Content-Type are
// never trusted.
func detectContent(head []byte) (Kind, string, bool) {
	if len(head) == 0 {
		return "", "", false
	}

	switch http.DetectContentType(head) {
	case "image/jpeg":
		return KindImage, "image/jpeg", true
	case "image/png":
		return KindImage, "image/png", true
	case "image/webp":
		return KindImage, "image/webp", true
	case "image/gif":
		return KindImage, "image/gif", true
	}

	// AVIF: ISO base media file format, major brand "avif" or "avis" in the
	// ftyp box at bytes 4..12.
	if len(head) >= 12 && bytes.Equal(head[4:8], []byte("ftyp")) {
		brand := string(head[8:12])
		if brand == "avif" || brand == "avis" {
			return KindImage, "image/avif", true
		}
	}

	if bytes.HasPrefix(head, []byte("%PDF-")) {
		return KindPDF, "application/pdf", true
	}

	// GLB: "glTF" magic followed by little-endian version 2.
	if len(head) >= 8 && bytes.Equal(head[0:4], []byte("glTF")) && binary.LittleEndian.Uint32(head[4:8]) == 2 {
		return KindModel3D, "model/gltf-binary", true
	}

	return "", "", false
}

// purposeAllowsKind reports whether purpose may carry a file of kind.
func purposeAllowsKind(purpose Purpose, kind Kind) bool {
	switch purpose {
	case PurposeAvatar, PurposeBanner, PurposeGroupAvatar, PurposeGroupBanner, PurposeBadgeIcon, PurposeCurationCover:
		return kind == KindImage
	case PurposeContent, PurposeImport:
		return kind == KindImage || kind == KindPDF || kind == KindModel3D
	case PurposeExperienceAttachment:
		return kind == KindImage || kind == KindPDF
	default:
		return false
	}
}

// validPurpose reports whether purpose is one of the known upload purposes.
func validPurpose(purpose Purpose) bool {
	switch purpose {
	case PurposeContent, PurposeAvatar, PurposeBanner, PurposeGroupAvatar, PurposeGroupBanner,
		PurposeExperienceAttachment, PurposeBadgeIcon, PurposeCurationCover, PurposeImport:
		return true
	default:
		return false
	}
}
