package messaging

import "unicode/utf16"

// Encoding is the message encoding chosen by the platform (contract Message.encoding).
type Encoding string

// Encodings.
const (
	EncodingGSM7 Encoding = "gsm7"
	EncodingUCS2 Encoding = "ucs2"
)

// Segment sizes of the GSM 03.38 / 3GPP TS 23.040 conventions: a single message holds 160 GSM-7
// septets or 70 UCS-2 code units; a concatenated message loses room to the user data header and
// holds 153 septets or 67 code units per segment.
//
// TODO(telco-confirm): each provider's own encoding support and segment counting (D-105, D-117).
// This is the platform's calculation; a provider may count differently in edge cases.
const (
	gsm7Single = 160
	gsm7Multi  = 153
	ucs2Single = 70
	ucs2Multi  = 67
)

// gsm7Basic is the GSM 03.38 default alphabet (one septet each), without the escape character.
const gsm7Basic = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
	"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"

// gsm7Extension is the GSM 03.38 extension table (two septets each: escape plus character).
const gsm7Extension = "\f^{}\\[~]|€"

var (
	gsm7BasicSet     = runeSet(gsm7Basic)
	gsm7ExtensionSet = runeSet(gsm7Extension)
)

func runeSet(s string) map[rune]bool {
	m := make(map[rune]bool)
	for _, r := range s {
		m[r] = true
	}
	return m
}

// Segmentation is the result of Segment.
type Segmentation struct {
	Encoding Encoding
	Segments int
}

// Segment chooses the encoding of body (GSM-7 if every character is in the GSM-7 alphabet,
// otherwise UCS-2) and counts its segments. It is the one shared segmentation function (AGENTS.md
// §11.9): the API response, the provider hand-off and any future usage record use it. A segment
// boundary never splits a GSM-7 escape sequence or a UTF-16 surrogate pair.
func Segment(body string) Segmentation {
	if septets, ok := gsm7Septets(body); ok {
		return Segmentation{Encoding: EncodingGSM7, Segments: countSegments(septets, gsm7Single, gsm7Multi)}
	}
	units := make([]int, 0, len(body))
	for _, r := range body {
		units = append(units, len(utf16.Encode([]rune{r})))
	}
	return Segmentation{Encoding: EncodingUCS2, Segments: countSegments(units, ucs2Single, ucs2Multi)}
}

// gsm7Septets returns the septet width of every character, or false if body is not GSM-7.
func gsm7Septets(body string) ([]int, bool) {
	widths := make([]int, 0, len(body))
	for _, r := range body {
		switch {
		case gsm7BasicSet[r]:
			widths = append(widths, 1)
		case gsm7ExtensionSet[r]:
			widths = append(widths, 2)
		default:
			return nil, false
		}
	}
	return widths, true
}

// countSegments counts the segments needed for characters of the given widths, never splitting a
// character across segments.
func countSegments(widths []int, single, multi int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	if total <= single {
		return 1
	}
	segments, used := 1, 0
	for _, w := range widths {
		if used+w > multi {
			segments++
			used = 0
		}
		used += w
	}
	return segments
}
